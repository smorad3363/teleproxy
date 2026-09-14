package telegrambot

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/smorad3363/teleproxy/internal/botcontent"
)

func TestStartApplicationLoadsBotContentOverridesAtRequestTime(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2033, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := botcontent.Set(context.Background(), db, botcontent.SlotWelcome, "Welcome from panel", now); err != nil {
		t.Fatal(err)
	}
	if _, err := botcontent.Set(context.Background(), db, botcontent.SlotProxy, "Your Telegram proxy", now); err != nil {
		t.Fatal(err)
	}
	app, err := NewStartApplication(db, "TeleProxyBot", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 44}, Chat: Chat{ID: 44, Type: "private"}, Text: "/start"}}
	response, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("Handle() = %#v, %v, %v", response, handled, err)
	}
	if response.WelcomeText != "Welcome from panel" || response.ProxyText != "Your Telegram proxy" {
		t.Fatalf("content = welcome=%q proxy=%q", response.WelcomeText, response.ProxyText)
	}

	if _, err := botcontent.Set(context.Background(), db, botcontent.SlotWelcome, "Changed immediately", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response, handled, err = app.Handle(context.Background(), update)
	if err != nil || !handled || response.WelcomeText != "Changed immediately" {
		t.Fatalf("updated Handle() = %#v, %v, %v", response, handled, err)
	}
}

func TestFormatStartResponseUsesConfiguredContentAndStaysWithinTelegramLimit(t *testing.T) {
	response := StartResponse{
		ProxyUsername:  "tg_7",
		RemainingBytes: 123,
		ProxyLink:      "tg://proxy?server=203.0.113.10&port=443&secret=0123456789abcdef0123456789abcdef",
		ProxySyncState: "synced",
		ReferralCode:   "ABC",
		ReferralLink:   "https://t.me/TeleProxyBot?start=ABC",
		ReferralCount:  2,
		WelcomeText:    strings.Repeat("خوش آمدید ", 700),
		ProxyText:      "اتصال سریع",
		ReferralText:   "دعوت از دوستان",
	}
	text := formatStartResponse(response)
	if utf8.RuneCountInString(text) > maxTelegramMessageRunes {
		t.Fatalf("message has %d runes", utf8.RuneCountInString(text))
	}
	for _, want := range []string{"خوش آمدید", "tg_7", "اتصال سریع"} {
		if !strings.Contains(text, want) {
			t.Fatalf("message missing %q: %q", want, text)
		}
	}
}

func TestStartActionKeyboardBuildsOneTapProxyAndReferralButtons(t *testing.T) {
	proxyLink := "tg://proxy?server=203.0.113.10&port=443&secret=0123456789abcdef0123456789abcdef"
	referralLink := "https://t.me/TeleProxyBot?start=ABC"
	markup, ok := startActionKeyboard(StartResponse{ProxyLink: proxyLink, ReferralLink: referralLink})
	if !ok || len(markup.InlineKeyboard) != 2 {
		t.Fatalf("keyboard = %#v, %v", markup, ok)
	}
	if markup.InlineKeyboard[0][0].URL != proxyLink || markup.InlineKeyboard[1][0].URL != referralLink {
		t.Fatalf("keyboard = %#v", markup)
	}
	if err := validateInlineKeyboard(markup); err != nil {
		t.Fatalf("validateInlineKeyboard() = %v", err)
	}
}

func TestForcedJoinConfiguredTextReplacesDefaultPrompt(t *testing.T) {
	text := formatMissingChannelsWithPrefix([]StartRequiredChannel{{DisplayName: "Sponsor", JoinURL: "https://t.me/example"}}, "اول عضو کانال شوید")
	if !strings.HasPrefix(text, "اول عضو کانال شوید") || strings.Contains(text, "Join the required Telegram channels before continuing:") {
		t.Fatalf("text = %q", text)
	}
}
