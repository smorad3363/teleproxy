package forcedjoin

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

type fakeMembershipClient struct {
	members map[string]bool
	err     error
	calls   []string
}

func (f *fakeMembershipClient) IsChatMember(_ context.Context, chatRef string, _ int64) (bool, error) {
	f.calls = append(f.calls, chatRef)
	if f.err != nil {
		return false, f.err
	}
	return f.members[chatRef], nil
}

func TestCheckRequiredCollectsMissingInConfiguredOrder(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(100, 0).UTC()
	for _, input := range []CreateChannel{
		{ChatRef: "@one", DisplayName: "One", JoinURL: "https://t.me/one", Enabled: true, Required: true, Position: 1},
		{ChatRef: "@two", DisplayName: "Two", JoinURL: "https://t.me/two", Enabled: true, Required: true, Position: 2},
		{ChatRef: "@optional", DisplayName: "Optional", JoinURL: "https://t.me/optional", Enabled: true, Required: false, Position: 0},
	} {
		if _, err := Create(ctx, db, input, now); err != nil {
			t.Fatal(err)
		}
	}
	client := &fakeMembershipClient{members: map[string]bool{"@one": true, "@two": false}}
	result, err := CheckRequired(ctx, db, client, 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Missing) != 1 || result.Missing[0].ChatRef != "@two" {
		t.Fatalf("missing = %#v", result.Missing)
	}
	if len(client.calls) != 2 || client.calls[0] != "@one" || client.calls[1] != "@two" {
		t.Fatalf("calls = %#v", client.calls)
	}
}

func TestCheckRequiredFailsClosedOnMembershipError(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := Create(ctx, db, CreateChannel{ChatRef: "@one", DisplayName: "One", JoinURL: "https://t.me/one", Enabled: true, Required: true}, time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	client := &fakeMembershipClient{err: errors.New("telegram unavailable")}
	if _, err := CheckRequired(ctx, db, client, 99); err == nil {
		t.Fatal("CheckRequired() error = nil")
	}
}
