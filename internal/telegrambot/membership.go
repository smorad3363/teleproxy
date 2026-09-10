package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type chatMemberResult struct {
	Status   string `json:"status"`
	IsMember *bool  `json:"is_member"`
	User     struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

func (c *Client) IsChatMember(ctx context.Context, chatRef string, userID int64) (bool, error) {
	if c == nil {
		return false, &APIError{Code: FailureUnavailable}
	}
	chatID, ok := telegramChatIDValue(chatRef)
	if !ok || userID <= 0 {
		return false, &APIError{Code: FailureRejected}
	}
	payload, err := json.Marshal(struct {
		ChatID any   `json:"chat_id"`
		UserID int64 `json:"user_id"`
	}{ChatID: chatID, UserID: userID})
	if err != nil {
		return false, &APIError{Code: FailureInvalidOutput}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+c.token+"/getChatMember", bytes.NewReader(payload))
	if err != nil {
		return false, &APIError{Code: FailureInvalidOutput}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Never surface the URL-bearing net/http error because the URL contains the Bot token.
		return false, &APIError{Code: FailureUnavailable}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, &APIError{Code: classifyFailure(resp.StatusCode, 0)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
	if err != nil || len(body) > maxResponseBodyBytes {
		return false, &APIError{Code: FailureInvalidOutput}
	}
	var envelope struct {
		OK        bool             `json:"ok"`
		Result    chatMemberResult `json:"result"`
		ErrorCode int              `json:"error_code"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return false, &APIError{Code: FailureInvalidOutput}
	}
	if !envelope.OK {
		return false, &APIError{Code: classifyFailure(resp.StatusCode, envelope.ErrorCode)}
	}
	if envelope.Result.User.ID != userID {
		return false, &APIError{Code: FailureInvalidOutput}
	}
	switch envelope.Result.Status {
	case "creator", "administrator", "member":
		return true, nil
	case "restricted":
		if envelope.Result.IsMember == nil {
			return false, &APIError{Code: FailureInvalidOutput}
		}
		return *envelope.Result.IsMember, nil
	case "left", "kicked":
		return false, nil
	default:
		return false, &APIError{Code: FailureInvalidOutput}
	}
}

func telegramChatIDValue(chatRef string) (any, bool) {
	if chatRef == "" || chatRef != strings.TrimSpace(chatRef) || len(chatRef) > 128 {
		return nil, false
	}
	if strings.HasPrefix(chatRef, "@") {
		username := chatRef[1:]
		if len(username) < 1 || len(username) > 64 {
			return nil, false
		}
		for _, ch := range username {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
				continue
			}
			return nil, false
		}
		return chatRef, true
	}
	id, err := strconv.ParseInt(chatRef, 10, 64)
	if err != nil || id == 0 || strconv.FormatInt(id, 10) != chatRef {
		return nil, false
	}
	return id, true
}
