package telegrambot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/forcedjoin"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

type StartRequiredChannel struct {
	DisplayName string `json:"display_name"`
	JoinURL     string `json:"join_url"`
	CustomText  string `json:"custom_text,omitempty"`
}

type StartResponse struct {
	ChatID           int64                  `json:"chat_id"`
	TelegramID       int64                  `json:"telegram_id"`
	ProxyUsername    string                 `json:"proxy_username"`
	Created          bool                   `json:"created"`
	InitialGiftBytes int64                  `json:"initial_gift_bytes"`
	RemainingBytes   int64                  `json:"remaining_bytes"`
	Payload          string                 `json:"payload,omitempty"`
	ProxyLink        string                 `json:"proxy_link,omitempty"`
	ProxySyncState   string                 `json:"proxy_sync_state,omitempty"`
	MissingChannels  []StartRequiredChannel `json:"missing_channels,omitempty"`
}

type proxyProvisioner interface {
	Ensure(context.Context, string) (proxyprovision.Result, error)
}

type StartApplication struct {
	db          *sql.DB
	botUsername string
	now         func() time.Time
	provisioner proxyProvisioner
	membership  forcedjoin.MembershipClient
}

func NewStartApplication(db *sql.DB, botUsername string, now func() time.Time) (*StartApplication, error) {
	return newStartApplication(db, botUsername, now, nil, nil)
}

func NewStartApplicationWithProvisioner(db *sql.DB, botUsername string, now func() time.Time, provisioner proxyProvisioner) (*StartApplication, error) {
	if provisioner == nil {
		return nil, fmt.Errorf("Telegram proxy provisioner is required")
	}
	return newStartApplication(db, botUsername, now, provisioner, nil)
}

func NewStartApplicationWithForcedJoin(db *sql.DB, botUsername string, now func() time.Time, provisioner proxyProvisioner, membership forcedjoin.MembershipClient) (*StartApplication, error) {
	if provisioner == nil {
		return nil, fmt.Errorf("Telegram proxy provisioner is required")
	}
	if membership == nil {
		return nil, fmt.Errorf("Telegram forced join membership client is required")
	}
	return newStartApplication(db, botUsername, now, provisioner, membership)
}

func newStartApplication(db *sql.DB, botUsername string, now func() time.Time, provisioner proxyProvisioner, membership forcedjoin.MembershipClient) (*StartApplication, error) {
	if db == nil {
		return nil, fmt.Errorf("Telegram start database is required")
	}
	botUsername = strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &StartApplication{db: db, botUsername: botUsername, now: now, provisioner: provisioner, membership: membership}, nil
}

func (a *StartApplication) Handle(ctx context.Context, update Update) (StartResponse, bool, error) {
	if a == nil || a.db == nil {
		return StartResponse{}, false, fmt.Errorf("Telegram start application is not configured")
	}
	if update.Message == nil || update.Message.From == nil || update.Message.From.IsBot || update.Message.Text == "" {
		return StartResponse{}, false, nil
	}
	command, handled, err := ParseStartCommand(update.Message.Text, a.botUsername)
	if !handled || err != nil {
		return StartResponse{}, handled, err
	}
	if update.Message.Chat.Type != "private" || update.Message.Chat.ID == 0 || update.Message.From.ID <= 0 {
		return StartResponse{}, false, nil
	}

	now := a.now().UTC()
	if a.membership != nil {
		return a.handleGatedStart(ctx, update, command, now)
	}
	started, err := telegramuser.Start(ctx, a.db, update.Message.From.ID, now)
	if err != nil {
		return StartResponse{}, true, err
	}
	return a.finishStart(ctx, update.Message.Chat.ID, command.Payload, started.User, started.Created, started.InitialGiftBytes, now)
}

func (a *StartApplication) handleGatedStart(ctx context.Context, update Update, command StartCommand, now time.Time) (StartResponse, bool, error) {
	resolved, err := telegramuser.Resolve(ctx, a.db, update.Message.From.ID, now)
	if err != nil {
		return StartResponse{}, true, err
	}
	check, err := forcedjoin.CheckRequired(ctx, a.db, a.membership, resolved.User.TelegramID)
	if err != nil {
		return StartResponse{}, true, fmt.Errorf("check Telegram forced join: %w", err)
	}
	if len(check.Missing) > 0 {
		return StartResponse{
			ChatID:          update.Message.Chat.ID,
			TelegramID:      resolved.User.TelegramID,
			ProxyUsername:   resolved.User.ProxyUsername,
			Created:         resolved.Created,
			Payload:         command.Payload,
			MissingChannels: startRequiredChannels(check.Missing),
		}, true, nil
	}
	gift, err := telegramuser.EnsureStartGift(ctx, a.db, resolved.User.TelegramID, now)
	if err != nil {
		return StartResponse{}, true, err
	}
	return a.finishStart(ctx, update.Message.Chat.ID, command.Payload, gift.User, resolved.Created, gift.InitialGiftBytes, now)
}

func (a *StartApplication) finishStart(ctx context.Context, chatID int64, payload string, user telegramuser.User, created bool, initialGiftBytes int64, now time.Time) (StartResponse, bool, error) {
	remaining, err := credit.Balance(ctx, a.db, user.ProxyUsername, now)
	if err != nil {
		return StartResponse{}, true, fmt.Errorf("read Telegram user credit balance: %w", err)
	}
	response := StartResponse{
		ChatID:           chatID,
		TelegramID:       user.TelegramID,
		ProxyUsername:    user.ProxyUsername,
		Created:          created,
		InitialGiftBytes: initialGiftBytes,
		RemainingBytes:   remaining,
		Payload:          payload,
	}
	if a.provisioner == nil {
		return response, true, nil
	}
	provisioned, err := a.provisioner.Ensure(ctx, user.ProxyUsername)
	if err != nil {
		return StartResponse{}, true, fmt.Errorf("provision Telegram proxy: %w", err)
	}
	response.ProxyLink = provisioned.Link
	response.ProxySyncState = string(provisioned.SyncState)
	return response, true, nil
}

func startRequiredChannels(channels []forcedjoin.Channel) []StartRequiredChannel {
	result := make([]StartRequiredChannel, 0, len(channels))
	for _, channel := range channels {
		result = append(result, StartRequiredChannel{
			DisplayName: channel.DisplayName,
			JoinURL:     channel.JoinURL,
			CustomText:  channel.CustomText,
		})
	}
	return result
}
