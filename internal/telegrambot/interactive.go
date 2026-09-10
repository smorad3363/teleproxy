package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	forcedJoinRecheckCallbackData = "forced_join_recheck"
	maxInlineKeyboardBodyBytes    = 64 << 10
)

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func (c *Client) SendMessageWithInlineKeyboard(ctx context.Context, chatID int64, text string, markup InlineKeyboardMarkup) (SentMessage, error) {
	if c == nil {
		return SentMessage{}, &APIError{Code: FailureUnavailable}
	}
	if chatID == 0 {
		return SentMessage{}, fmt.Errorf("Telegram chat ID must not be zero")
	}
	if !utf8.ValidString(text) {
		return SentMessage{}, fmt.Errorf("Telegram message text must be valid UTF-8")
	}
	characters := utf8.RuneCountInString(text)
	if characters < 1 || characters > 4096 {
		return SentMessage{}, fmt.Errorf("Telegram message text must contain between 1 and 4096 characters")
	}
	if err := validateInlineKeyboard(markup); err != nil {
		return SentMessage{}, err
	}

	payload, err := json.Marshal(struct {
		ChatID      int64                `json:"chat_id"`
		Text        string               `json:"text"`
		ReplyMarkup InlineKeyboardMarkup `json:"reply_markup"`
	}{ChatID: chatID, Text: text, ReplyMarkup: markup})
	if err != nil || len(payload) > maxInlineKeyboardBodyBytes {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	body, err := c.callBotAPI(ctx, "sendMessage", payload)
	if err != nil {
		return SentMessage{}, err
	}
	var message SentMessage
	if json.Unmarshal(body, &message) != nil || message.MessageID <= 0 || message.Chat.ID != chatID {
		return SentMessage{}, &APIError{Code: FailureInvalidOutput}
	}
	return message, nil
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackID, text string) error {
	if c == nil {
		return &APIError{Code: FailureUnavailable}
	}
	if !validCallbackQueryID(callbackID) {
		return fmt.Errorf("Telegram callback query ID is invalid")
	}
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > 200 {
		return fmt.Errorf("Telegram callback answer text is invalid")
	}
	payload, err := json.Marshal(struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text,omitempty"`
	}{CallbackQueryID: callbackID, Text: text})
	if err != nil {
		return &APIError{Code: FailureInvalidOutput}
	}
	body, err := c.callBotAPI(ctx, "answerCallbackQuery", payload)
	if err != nil {
		return err
	}
	var ok bool
	if json.Unmarshal(body, &ok) != nil || !ok {
		return &APIError{Code: FailureInvalidOutput}
	}
	return nil
}

func (c *Client) callBotAPI(ctx context.Context, method string, payload []byte) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return nil, &APIError{Code: FailureInvalidOutput}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &APIError{Code: FailureUnavailable}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Code: classifyFailure(resp.StatusCode, 0)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil || len(body) > maxResponseBodyBytes {
		return nil, &APIError{Code: FailureInvalidOutput}
	}
	var envelope struct {
		OK        bool            `json:"ok"`
		Result    json.RawMessage `json:"result"`
		ErrorCode int             `json:"error_code"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return nil, &APIError{Code: FailureInvalidOutput}
	}
	if !envelope.OK {
		return nil, &APIError{Code: classifyFailure(resp.StatusCode, envelope.ErrorCode)}
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil, &APIError{Code: FailureInvalidOutput}
	}
	return envelope.Result, nil
}

func validCallbackQueryID(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= 256 && utf8.ValidString(value)
}

func validateInlineKeyboard(markup InlineKeyboardMarkup) error {
	if len(markup.InlineKeyboard) == 0 {
		return fmt.Errorf("Telegram inline keyboard must contain at least one row")
	}
	for _, row := range markup.InlineKeyboard {
		if len(row) == 0 {
			return fmt.Errorf("Telegram inline keyboard row must not be empty")
		}
		for _, button := range row {
			if !utf8.ValidString(button.Text) || strings.TrimSpace(button.Text) == "" || utf8.RuneCountInString(button.Text) > 128 {
				return fmt.Errorf("Telegram inline keyboard button text is invalid")
			}
			hasURL := button.URL != ""
			hasCallback := button.CallbackData != ""
			if hasURL == hasCallback {
				return fmt.Errorf("Telegram inline keyboard button must contain exactly one action")
			}
			if hasURL {
				if err := validateTelegramJoinURL(button.URL); err != nil {
					return err
				}
			}
			if hasCallback && (len(button.CallbackData) < 1 || len(button.CallbackData) > 64 || !utf8.ValidString(button.CallbackData)) {
				return fmt.Errorf("Telegram inline keyboard callback data is invalid")
			}
		}
	}
	return nil
}

func validateTelegramJoinURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Fragment != "" || parsed.Port() != "" {
		return fmt.Errorf("Telegram inline keyboard URL is invalid")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "t.me" && host != "telegram.me" || parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("Telegram inline keyboard URL is invalid")
	}
	return nil
}

func forcedJoinKeyboard(channels []StartRequiredChannel) InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(channels)+1)
	for _, channel := range channels {
		rows = append(rows, []InlineKeyboardButton{{Text: channel.DisplayName, URL: channel.JoinURL}})
	}
	rows = append(rows, []InlineKeyboardButton{{Text: "Recheck membership", CallbackData: forcedJoinRecheckCallbackData}})
	return InlineKeyboardMarkup{InlineKeyboard: rows}
}
