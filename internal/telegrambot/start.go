package telegrambot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/forcedjoin"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/referral"
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
	ReferralCode     string                 `json:"referral_code,omitempty"`
	ReferralLink     string                 `json:"referral_link,omitempty"`
	ReferralCount    int64                  `json:"referral_count"`
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
	if update.CallbackQuery != nil {
		return a.handleForcedJoinRecheck(ctx, *update.CallbackQuery)
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
	if command.Payload != "" {
		if _, err := referral.AttributeNewInvitee(ctx, a.db, resolved, command.Payload, now); err != nil {
			return StartResponse{}, true, fmt.Errorf("resolve Telegram referral attribution: %w", err)
		}
	}
	return a.finishGatedIdentity(ctx, update.Message.Chat.ID, command.Payload, resolved.User, resolved.Created, now)
}

func (a *StartApplication) handleForcedJoinRecheck(ctx context.Context, query CallbackQuery) (StartResponse, bool, error) {
	if a.membership == nil || a.provisioner == nil || query.Data != forcedJoinRecheckCallbackData {
		return StartResponse{}, false, nil
	}
	if !validCallbackQueryID(query.ID) || query.From == nil || query.From.IsBot || query.From.ID <= 0 || query.Message == nil {
		return StartResponse{}, false, nil
	}
	if query.Message.MessageID <= 0 || query.Message.Chat.Type != "private" || query.Message.Chat.ID == 0 || query.Message.Chat.ID != query.From.ID {
		return StartResponse{}, false, nil
	}
	user, err := telegramuser.Get(ctx, a.db, query.From.ID)
	if errors.Is(err, telegramuser.ErrNotFound) {
		return StartResponse{}, false, nil
	}
	if err != nil {
		return StartResponse{}, true, err
	}
	return a.finishGatedIdentity(ctx, query.Message.Chat.ID, "", user, false, a.now().UTC())
}

func (a *StartApplication) finishGatedIdentity(ctx context.Context, chatID int64, payload string, user telegramuser.User, created bool, now time.Time) (StartResponse, bool, error) {
	check, err := forcedjoin.CheckRequired(ctx, a.db, a.membership, user.TelegramID)
	if err != nil {
		return StartResponse{}, true, fmt.Errorf("check Telegram forced join: %w", err)
	}
	if len(check.Missing) > 0 {
		return StartResponse{
			ChatID:          chatID,
			TelegramID:      user.TelegramID,
			ProxyUsername:   user.ProxyUsername,
			Created:         created,
			Payload:         payload,
			MissingChannels: startRequiredChannels(check.Missing),
		}, true, nil
	}
	gift, err := telegramuser.EnsureStartGift(ctx, a.db, user.TelegramID, now)
	if err != nil {
		return StartResponse{}, true, err
	}
	response, handled, err := a.finishStart(ctx, chatID, payload, gift.User, created, gift.InitialGiftBytes, now)
	if err != nil || !handled {
		return response, handled, err
	}
	if err := a.attachReferralSummary(ctx, &response, gift.User.ID, now); err != nil {
		return StartResponse{}, true, err
	}
	return response, true, nil
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

func (a *StartApplication) attachReferralSummary(ctx context.Context, response *StartResponse, telegramUserID int64, now time.Time) error {
	code, err := referral.EnsureCode(ctx, a.db, telegramUserID, now)
	if err != nil {
		return fmt.Errorf("ensure Telegram referral code: %w", err)
	}
	count, err := referral.RewardedCount(ctx, a.db, telegramUserID)
	if err != nil {
		return fmt.Errorf("read Telegram referral count: %w", err)
	}
	response.ReferralCode = code.Value
	response.ReferralCount = count
	if a.botUsername != "" {
		query := url.Values{}
		query.Set("start", code.Value)
		link := url.URL{Scheme: "https", Host: "t.me", Path: "/" + a.botUsername, RawQuery: query.Encode()}
		response.ReferralLink = link.String()
	}
	return nil
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
