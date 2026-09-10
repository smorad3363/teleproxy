package telegrambot

import (
	"strings"
	"testing"
)

func TestFormatStartResponseIncludesProvisionedLinkAndSafeStatus(t *testing.T) {
	link := "tg://proxy?server=proxy.example&port=443&secret=00112233445566778899aabbccddeeff"
	text := formatStartResponse(StartResponse{
		ProxyUsername:  "tg_42",
		RemainingBytes: 100000000,
		ProxyLink:      link,
		ProxySyncState: "pending",
	})
	for _, want := range []string{"tg_42", "100000000", "synchronizing", link} {
		if !strings.Contains(text, want) {
			t.Fatalf("message %q does not contain %q", text, want)
		}
	}
	if len(text) > 4096 {
		t.Fatalf("message length = %d, exceeds Telegram sendMessage limit", len(text))
	}

	ready := formatStartResponse(StartResponse{ProxyUsername: "tg_42", ProxyLink: link, ProxySyncState: "synced"})
	if !strings.Contains(ready, "Proxy status: ready.") {
		t.Fatalf("ready message = %q", ready)
	}
}
