package telegrambot

import (
	"errors"
	"strings"
	"testing"
)

func TestParseStartCommand(t *testing.T) {
	for _, test := range []struct {
		name        string
		text        string
		botUsername string
		handled     bool
		payload     string
	}{
		{name: "bare", text: "/start", botUsername: "TeleProxyBot", handled: true},
		{name: "payload", text: "/start ref_ABC-123", botUsername: "TeleProxyBot", handled: true, payload: "ref_ABC-123"},
		{name: "addressed", text: "/start@teleproxybot campaign_1", botUsername: "TeleProxyBot", handled: true, payload: "campaign_1"},
		{name: "at-prefixed config", text: "/start@TeleProxyBot", botUsername: "@TeleProxyBot", handled: true},
		{name: "other bot", text: "/start@OtherBot ref", botUsername: "TeleProxyBot", handled: false},
		{name: "other command", text: "/help", botUsername: "TeleProxyBot", handled: false},
		{name: "near prefix", text: "/starter", botUsername: "TeleProxyBot", handled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			command, handled, err := ParseStartCommand(test.text, test.botUsername)
			if err != nil {
				t.Fatal(err)
			}
			if handled != test.handled || command.Payload != test.payload {
				t.Fatalf("ParseStartCommand() = %#v, %v", command, handled)
			}
		})
	}
}

func TestParseStartCommandRejectsInvalidDeepLinkPayload(t *testing.T) {
	for _, text := range []string{
		"/start bad!",
		"/start first second",
		"/start " + strings.Repeat("a", 65),
	} {
		_, handled, err := ParseStartCommand(text, "TeleProxyBot")
		if !handled || !errors.Is(err, ErrInvalidStartPayload) {
			t.Fatalf("%q => handled=%v err=%v", text, handled, err)
		}
	}
}
