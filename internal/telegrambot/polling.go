package telegrambot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	defaultPollingTimeoutSeconds = 25
	defaultPollingRetryDelay     = 2 * time.Second
	maxPollingUpdates            = 100
)

type Poller struct {
	client       *Client
	start        startUpdateHandler
	sender       messageSender
	pollSeconds  int
	retryDelay   time.Duration
	webhookReady bool
}

func NewPoller(client *Client, start startUpdateHandler, sender messageSender) (*Poller, error) {
	return newPoller(client, start, sender, defaultPollingTimeoutSeconds, defaultPollingRetryDelay)
}

func newPoller(client *Client, start startUpdateHandler, sender messageSender, pollSeconds int, retryDelay time.Duration) (*Poller, error) {
	if client == nil {
		return nil, fmt.Errorf("Telegram polling client is required")
	}
	if start == nil {
		return nil, fmt.Errorf("Telegram polling start handler is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("Telegram polling message sender is required")
	}
	if pollSeconds < 0 || pollSeconds > 50 {
		return nil, fmt.Errorf("Telegram polling timeout is invalid")
	}
	if retryDelay <= 0 {
		retryDelay = defaultPollingRetryDelay
	}
	return &Poller{client: client, start: start, sender: sender, pollSeconds: pollSeconds, retryDelay: retryDelay}, nil
}

func (p *Poller) Run(ctx context.Context) error {
	if p == nil || p.client == nil || p.start == nil || p.sender == nil {
		return fmt.Errorf("Telegram poller is not configured")
	}
	offset := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if !p.webhookReady {
			if err := p.client.DeleteWebhook(ctx); err != nil {
				if isTerminalPollingFailure(err) {
					return fmt.Errorf("prepare Telegram polling: %w", err)
				}
				if err := waitPollingRetry(ctx, p.retryDelay); err != nil {
					return nil
				}
				continue
			}
			p.webhookReady = true
		}
		updates, err := p.client.GetUpdates(ctx, offset, p.pollSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if isTerminalPollingFailure(err) {
				return fmt.Errorf("receive Telegram polling updates: %w", err)
			}
			if err := waitPollingRetry(ctx, p.retryDelay); err != nil {
				return nil
			}
			continue
		}
		retryBatch := false
		for _, update := range updates {
			if update.UpdateID < offset {
				continue
			}
			if err := p.dispatch(ctx, update); err != nil {
				retryBatch = true
				break
			}
			offset = update.UpdateID + 1
		}
		if retryBatch {
			if err := waitPollingRetry(ctx, p.retryDelay); err != nil {
				return nil
			}
		}
	}
}

func (p *Poller) dispatch(ctx context.Context, update Update) error {
	response, handled, err := p.start.Handle(ctx, update)
	if !handled {
		return nil
	}
	if update.CallbackQuery != nil {
		handleCallbackResult(ctx, p.sender, *update.CallbackQuery, response, err)
		return nil
	}
	if errors.Is(err, ErrInvalidStartPayload) {
		if update.Message != nil {
			_, _ = p.sender.SendMessage(ctx, update.Message.Chat.ID, "This start link is invalid.")
		}
		return nil
	}
	if err != nil {
		return err
	}
	return sendPollingStartResponse(ctx, p.sender, response)
}

func sendPollingStartResponse(ctx context.Context, sender messageSender, response StartResponse) error {
	text := formatStartResponse(response)
	if keyboardSender, ok := sender.(inlineKeyboardSender); ok {
		var markup InlineKeyboardMarkup
		var hasKeyboard bool
		if len(response.MissingChannels) > 0 {
			markup = forcedJoinKeyboard(response.MissingChannels)
			hasKeyboard = true
		} else if actionMarkup, ok := startActionKeyboard(response); ok {
			markup = actionMarkup
			hasKeyboard = true
		}
		if hasKeyboard {
			if _, err := keyboardSender.SendMessageWithInlineKeyboard(ctx, response.ChatID, text, markup); err == nil {
				return nil
			}
			// Keyboard validation or Bot API rejection must not make /start silent.
			// Fall back to the same safe text without buttons before retrying the update.
		}
	}
	_, err := sender.SendMessage(ctx, response.ChatID, text)
	return err
}

func isTerminalPollingFailure(err error) bool {
	switch FailureCodeOf(err) {
	case FailureUnauthorized, FailureRejected, FailureInvalidOutput:
		return true
	default:
		return false
	}
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	payload, err := json.Marshal(struct {
		DropPendingUpdates bool `json:"drop_pending_updates"`
	}{DropPendingUpdates: false})
	if err != nil {
		return &APIError{Code: FailureInvalidOutput}
	}
	body, err := c.callBotAPI(ctx, "deleteWebhook", payload)
	if err != nil {
		return err
	}
	var ok bool
	if json.Unmarshal(body, &ok) != nil || !ok {
		return &APIError{Code: FailureInvalidOutput}
	}
	return nil
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	if c == nil || c.httpClient == nil {
		return nil, &APIError{Code: FailureUnavailable}
	}
	if offset < 0 || timeoutSeconds < 0 || timeoutSeconds > 50 {
		return nil, fmt.Errorf("Telegram polling request is invalid")
	}
	payload, err := json.Marshal(struct {
		Offset         int64    `json:"offset,omitempty"`
		Limit          int      `json:"limit"`
		Timeout        int      `json:"timeout"`
		AllowedUpdates []string `json:"allowed_updates"`
	}{
		Offset:         offset,
		Limit:          maxPollingUpdates,
		Timeout:        timeoutSeconds,
		AllowedUpdates: []string{"message", "callback_query"},
	})
	if err != nil {
		return nil, &APIError{Code: FailureInvalidOutput}
	}

	pollHTTPClient := *c.httpClient
	minimumTimeout := time.Duration(timeoutSeconds+5) * time.Second
	if pollHTTPClient.Timeout <= 0 || pollHTTPClient.Timeout < minimumTimeout {
		pollHTTPClient.Timeout = minimumTimeout
	}
	body, err := c.callBotAPIWithHTTPClient(ctx, &pollHTTPClient, "getUpdates", payload)
	if err != nil {
		return nil, err
	}
	var updates []Update
	if json.Unmarshal(body, &updates) != nil || len(updates) > maxPollingUpdates {
		return nil, &APIError{Code: FailureInvalidOutput}
	}
	last := int64(0)
	for index, update := range updates {
		if update.UpdateID <= 0 || (index > 0 && update.UpdateID <= last) {
			return nil, &APIError{Code: FailureInvalidOutput}
		}
		last = update.UpdateID
	}
	return updates, nil
}

func waitPollingRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
