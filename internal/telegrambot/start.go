package telegrambot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

type StartResponse struct {
	ChatID           int64  `json:"chat_id"`
	TelegramID       int64  `json:"telegram_id"`
	ProxyUsername    string `json:"proxy_username"`
	Created          bool   `json:"created"`
	InitialGiftBytes int64  `json:"initial_gift_bytes"`
	RemainingBytes   int64  `json:"remaining_bytes"`
	Payload          string `json:"payload,omitempty"`
}

type StartApplication struct {
	db          *sql.DB
	botUsername string
	now         func() time.Time
}

func NewStartApplication(db *sql.DB, botUsername string, now func() time.Time) (*StartApplication, error) {
	if db == nil {
		return nil, fmt.Errorf("Telegram start database is required")
	}
	botUsername = strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &StartApplication{db: db, botUsername: botUsername, now: now}, nil
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
	started, err := telegramuser.Start(ctx, a.db, update.Message.From.ID, now)
	if err != nil {
		return StartResponse{}, true, err
	}
	remaining, err := credit.Balance(ctx, a.db, started.User.ProxyUsername, now)
	if err != nil {
		return StartResponse{}, true, fmt.Errorf("read Telegram user credit balance: %w", err)
	}
	return StartResponse{
		ChatID:           update.Message.Chat.ID,
		TelegramID:       started.User.TelegramID,
		ProxyUsername:    started.User.ProxyUsername,
		Created:          started.Created,
		InitialGiftBytes: started.InitialGiftBytes,
		RemainingBytes:   remaining,
		Payload:          command.Payload,
	}, true, nil
}
