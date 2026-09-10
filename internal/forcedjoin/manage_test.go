package forcedjoin

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestListAndUpdateManageAllChannelStates(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	createdAt := time.Unix(100, 0).UTC()
	later, err := Create(ctx, db, CreateChannel{
		ChatRef: "@later", DisplayName: "Later", JoinURL: "https://t.me/later",
		Enabled: false, Required: false, Position: 20,
	}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Create(ctx, db, CreateChannel{
		ChatRef: "@first", DisplayName: "First", JoinURL: "https://t.me/first",
		Enabled: true, Required: true, Position: 10,
	}, createdAt)
	if err != nil {
		t.Fatal(err)
	}

	channels, err := List(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 || channels[0].ID != first.ID || channels[1].ID != later.ID {
		t.Fatalf("initial management order = %#v", channels)
	}
	if channels[1].Enabled || channels[1].Required {
		t.Fatalf("disabled optional channel lost state: %#v", channels[1])
	}

	updatedAt := createdAt.Add(time.Hour)
	updated, err := Update(ctx, db, later.ID, CreateChannel{
		ChatRef: "@later_new", DisplayName: "Later renamed", JoinURL: "https://t.me/later_new",
		Enabled: true, Required: true, Position: 5, CustomText: "Join this channel",
	}, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != later.ID || !updated.CreatedAt.Equal(later.CreatedAt) || !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("update changed identity/timestamps: before=%#v after=%#v", later, updated)
	}
	if updated.ChatRef != "@later_new" || !updated.Enabled || !updated.Required || updated.Position != 5 || updated.CustomText != "Join this channel" {
		t.Fatalf("updated channel = %#v", updated)
	}

	channels, err = List(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 || channels[0].ID != later.ID || channels[1].ID != first.ID {
		t.Fatalf("updated management order = %#v", channels)
	}
}

func TestManagementConflictAndDeleteAreSafe(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(100, 0).UTC()

	one, err := Create(ctx, db, CreateChannel{ChatRef: "@one", DisplayName: "One", JoinURL: "https://t.me/one", Enabled: true, Required: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	two, err := Create(ctx, db, CreateChannel{ChatRef: "@two", DisplayName: "Two", JoinURL: "https://t.me/two", Enabled: false, Required: false}, now)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Create(ctx, db, CreateChannel{ChatRef: "@one", DisplayName: "Duplicate", JoinURL: "https://t.me/one", Enabled: true, Required: true}, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate create error = %v", err)
	}
	if _, err := Update(ctx, db, two.ID, CreateChannel{ChatRef: "@one", DisplayName: "Two", JoinURL: "https://t.me/two", Enabled: true, Required: true}, now.Add(time.Minute)); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate update error = %v", err)
	}
	unchanged, err := Get(ctx, db, two.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.ChatRef != "@two" || unchanged.Enabled || unchanged.Required {
		t.Fatalf("conflicting update changed row: %#v", unchanged)
	}

	if err := Delete(ctx, db, one.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, db, one.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted Get() error = %v", err)
	}
	if err := Delete(ctx, db, one.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete() error = %v", err)
	}
	if _, err := Get(ctx, db, two.ID); err != nil {
		t.Fatalf("deleting one channel affected another: %v", err)
	}
}
