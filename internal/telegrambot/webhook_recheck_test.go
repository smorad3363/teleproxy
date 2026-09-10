package telegrambot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type interactiveWebhookSender struct {
	plainCalls    int
	keyboardCalls int
	answerCalls   int
	chatID        int64
	text          string
	markup        InlineKeyboardMarkup
	callbackID    string
	answerText    string
	answerErr     error
}

func (s *interactiveWebhookSender) SendMessage(_ context.Context, chatID int64, text string) (SentMessage, error) {
	s.plainCalls++
	s.chatID = chatID
	s.text = text
	return SentMessage{MessageID: int64(s.plainCalls), Chat: Chat{ID: chatID}, Text: text}, nil
}

func (s *interactiveWebhookSender) SendMessageWithInlineKeyboard(_ context.Context, chatID int64, text string, markup InlineKeyboardMarkup) (SentMessage, error) {
	s.keyboardCalls++
	s.chatID = chatID
	s.text = text
	s.markup = markup
	return SentMessage{MessageID: int64(s.keyboardCalls), Chat: Chat{ID: chatID}, Text: text}, nil
}

func (s *interactiveWebhookSender) AnswerCallbackQuery(_ context.Context, callbackID, text string) error {
	s.answerCalls++
	s.callbackID = callbackID
	s.answerText = text
	return s.answerErr
}

func TestWebhookMissingStartUsesJoinAndRecheckKeyboard(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{
		ChatID: 42,
		MissingChannels: []StartRequiredChannel{{
			DisplayName: "Required",
			JoinURL:     "https://t.me/required",
		}},
	}}
	sender := &interactiveWebhookSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptestResponse(handler, `{"update_id":30,"message":{"message_id":5,"from":{"id":42,"is_bot":false,"first_name":"A"},"chat":{"id":42,"type":"private"},"date":1,"text":"/start"}}`)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body=%q", recorder.Code, recorder.Body.String())
	}
	if sender.keyboardCalls != 1 || sender.plainCalls != 0 || sender.answerCalls != 0 {
		t.Fatalf("message state = %#v", sender)
	}
	if len(sender.markup.InlineKeyboard) != 2 || sender.markup.InlineKeyboard[0][0].URL != "https://t.me/required" || sender.markup.InlineKeyboard[1][0].CallbackData != forcedJoinRecheckCallbackData {
		t.Fatalf("markup = %#v", sender.markup)
	}
}

func TestWebhookMissingMembershipUsesKeyboardAndAnswersRecheck(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{
		ChatID: 42,
		MissingChannels: []StartRequiredChannel{{
			DisplayName: "Required",
			JoinURL:     "https://t.me/required",
		}},
	}}
	sender := &interactiveWebhookSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptestResponse(handler, callbackWebhookJSON(31, "cb-31", 42))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body=%q", recorder.Code, recorder.Body.String())
	}
	if sender.answerCalls != 1 || sender.callbackID != "cb-31" || !strings.Contains(sender.answerText, "Join all required") {
		t.Fatalf("callback answer = %#v", sender)
	}
	if sender.keyboardCalls != 1 || sender.plainCalls != 0 || sender.chatID != 42 {
		t.Fatalf("message calls = %#v", sender)
	}
	if len(sender.markup.InlineKeyboard) != 2 || sender.markup.InlineKeyboard[0][0].URL != "https://t.me/required" || sender.markup.InlineKeyboard[1][0].CallbackData != forcedJoinRecheckCallbackData {
		t.Fatalf("markup = %#v", sender.markup)
	}
}

func TestWebhookSuccessfulRecheckAnswersAndSendsReadyMessage(t *testing.T) {
	start := &fakeStartHandler{handled: true, response: StartResponse{
		ChatID: 42, ProxyUsername: "tg_42", RemainingBytes: 100, ProxyLink: "tg://proxy?server=proxy.example&port=443&secret=00112233445566778899aabbccddeeff", ProxySyncState: "synced",
	}}
	sender := &interactiveWebhookSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptestResponse(handler, callbackWebhookJSON(32, "cb-32", 42))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if sender.answerCalls != 1 || sender.answerText != "Membership verified." {
		t.Fatalf("callback answer = %#v", sender)
	}
	if sender.plainCalls != 1 || sender.keyboardCalls != 0 || !strings.Contains(sender.text, "tg_42") {
		t.Fatalf("message state = %#v", sender)
	}
}

func TestWebhookRecheckApplicationErrorIsAnsweredWithoutTelegramReplay(t *testing.T) {
	start := &fakeStartHandler{handled: true, err: errors.New("membership API unavailable")}
	sender := &interactiveWebhookSender{answerErr: &APIError{Code: FailureUnavailable}}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptestResponse(handler, callbackWebhookJSON(33, "cb-33", 42))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body=%q", recorder.Code, recorder.Body.String())
	}
	if sender.answerCalls != 1 || !strings.Contains(sender.answerText, "failed") {
		t.Fatalf("callback answer = %#v", sender)
	}
	if sender.plainCalls != 0 || sender.keyboardCalls != 0 {
		t.Fatalf("application error sent message: %#v", sender)
	}
}

func TestWebhookUnhandledCallbackDoesNotAnswerOrSend(t *testing.T) {
	start := &fakeStartHandler{handled: false}
	sender := &interactiveWebhookSender{}
	handler, err := NewWebhookHandler("secret_123", start, sender)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptestResponse(handler, callbackWebhookJSON(34, "cb-34", 42))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if sender.answerCalls != 0 || sender.plainCalls != 0 || sender.keyboardCalls != 0 {
		t.Fatalf("unhandled callback caused output: %#v", sender)
	}
}

func httptestResponse(handler http.Handler, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, webhookRequest(body, "secret_123"))
	return recorder
}

func callbackWebhookJSON(updateID int64, callbackID string, telegramID int64) string {
	return fmt.Sprintf(`{"update_id":%d,"callback_query":{"id":%q,"from":{"id":%d,"is_bot":false,"first_name":"A"},"message":{"message_id":7,"chat":{"id":%d,"type":"private"},"date":1},"data":%q}}`, updateID, callbackID, telegramID, telegramID, forcedJoinRecheckCallbackData)
}
