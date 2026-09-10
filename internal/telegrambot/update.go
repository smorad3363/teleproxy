package telegrambot

import (
	"errors"
	"strings"
)

var ErrInvalidStartPayload = errors.New("Telegram /start payload is invalid")

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type CallbackQuery struct {
	ID      string        `json:"id"`
	From    *TelegramUser `json:"from,omitempty"`
	Message *Message      `json:"message,omitempty"`
	Data    string        `json:"data,omitempty"`
}

type Message struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      Chat          `json:"chat"`
	Date      int64         `json:"date"`
	Text      string        `json:"text,omitempty"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type StartCommand struct {
	Payload string
}

func ParseStartCommand(text, botUsername string) (StartCommand, bool, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return StartCommand{}, false, nil
	}

	command := fields[0]
	name, target, hasTarget := strings.Cut(command, "@")
	if !strings.EqualFold(name, "/start") {
		return StartCommand{}, false, nil
	}
	if hasTarget {
		expected := strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
		if expected == "" || target == "" || !strings.EqualFold(target, expected) {
			return StartCommand{}, false, nil
		}
	}
	if len(fields) > 2 {
		return StartCommand{}, true, ErrInvalidStartPayload
	}
	if len(fields) == 1 {
		return StartCommand{}, true, nil
	}
	payload := fields[1]
	if !validStartPayload(payload) {
		return StartCommand{}, true, ErrInvalidStartPayload
	}
	return StartCommand{Payload: payload}, true, nil
}

func validStartPayload(payload string) bool {
	if len(payload) < 1 || len(payload) > 64 {
		return false
	}
	for _, ch := range payload {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}
