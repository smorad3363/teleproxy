package forcedjoin

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestRequiredChannelsAreOrderedAndFiltered(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(100, 0).UTC()
	inputs := []CreateChannel{
		{ChatRef: "@later", DisplayName: "Later", JoinURL: "https://t.me/later", Enabled: true, Required: true, Position: 20},
		{ChatRef: "-1001234567890", DisplayName: "First", JoinURL: "https://t.me/+invite", Enabled: true, Required: true, Position: 10, CustomText: "Join first"},
		{ChatRef: "@disabled", DisplayName: "Disabled", JoinURL: "https://t.me/disabled", Enabled: false, Required: true, Position: 1},
		{ChatRef: "@optional", DisplayName: "Optional", JoinURL: "https://t.me/optional", Enabled: true, Required: false, Position: 1},
	}
	for _, input := range inputs {
		if _, err := Create(ctx, db, input, now); err != nil {
			t.Fatal(err)
		}
	}
	channels, err := ListRequired(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 || channels[0].ChatRef != "-1001234567890" || channels[1].ChatRef != "@later" {
		t.Fatalf("required channels = %#v", channels)
	}
	if channels[0].CustomText != "Join first" || !channels[0].Enabled || !channels[0].Required {
		t.Fatalf("first channel = %#v", channels[0])
	}
}

func TestCreateChannelRejectsUnsafeConfiguration(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(100, 0).UTC()
	base := CreateChannel{ChatRef: "@channel", DisplayName: "Channel", JoinURL: "https://t.me/channel", Enabled: true, Required: true}
	tests := []CreateChannel{
		{ChatRef: "channel", DisplayName: base.DisplayName, JoinURL: base.JoinURL, Enabled: true, Required: true},
		{ChatRef: "@bad-name", DisplayName: base.DisplayName, JoinURL: base.JoinURL, Enabled: true, Required: true},
		{ChatRef: base.ChatRef, DisplayName: "", JoinURL: base.JoinURL, Enabled: true, Required: true},
		{ChatRef: base.ChatRef, DisplayName: base.DisplayName, JoinURL: "http://t.me/channel", Enabled: true, Required: true},
		{ChatRef: base.ChatRef, DisplayName: base.DisplayName, JoinURL: "https://example.com/channel", Enabled: true, Required: true},
		{ChatRef: base.ChatRef, DisplayName: base.DisplayName, JoinURL: "https://t.me:444/channel", Enabled: true, Required: true},
		{ChatRef: base.ChatRef, DisplayName: base.DisplayName, JoinURL: base.JoinURL, Enabled: true, Required: true, Position: -1},
	}
	for index, input := range tests {
		if _, err := Create(ctx, db, input, now); err == nil {
			t.Fatalf("case %d accepted invalid channel", index)
		}
	}
}
