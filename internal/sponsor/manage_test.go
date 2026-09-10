package sponsor

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUpdateDeleteProfilePreservesValidationAndConflict(t *testing.T) {
	ctx := context.Background()
	db := sponsorTestDB(t)
	now := time.Unix(1_831_000_000, 0).UTC()
	first, err := Create(ctx, db, CreateProfile{Name: "First", ChannelRef: "@first", AdTag: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Enabled: true, Weight: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Create(ctx, db, CreateProfile{Name: "Second", ChannelRef: "@second", AdTag: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Enabled: true, Weight: 2}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	start := now.Add(time.Hour)
	end := start.Add(2 * time.Hour)
	updated, err := Update(ctx, db, first.ID, CreateProfile{
		Name:       "  First Updated ",
		ChannelRef: "https://T.ME/First_Updated",
		AdTag:      "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
		Enabled:    false,
		Weight:     9,
		StartsAt:   &start,
		EndsAt:     &end,
		Notes:      "updated",
	}, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "First Updated" || updated.ChannelRef != "https://t.me/First_Updated" || updated.AdTag != "cccccccccccccccccccccccccccccccc" || updated.Enabled || updated.Weight != 9 {
		t.Fatalf("updated profile = %#v", updated)
	}
	if !updated.CreatedAt.Equal(first.CreatedAt) || !updated.UpdatedAt.Equal(now.Add(2*time.Second)) {
		t.Fatalf("update timestamps = created %v updated %v", updated.CreatedAt, updated.UpdatedAt)
	}

	if _, err := Update(ctx, db, first.ID, CreateProfile{Name: "Conflict", ChannelRef: "@conflict", AdTag: second.AdTag, Enabled: true, Weight: 1}, now.Add(3*time.Second)); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting Update() error = %v", err)
	}
	persisted, err := Get(ctx, db, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.AdTag != updated.AdTag || persisted.Name != updated.Name {
		t.Fatalf("conflicting update mutated profile = %#v", persisted)
	}

	if err := Delete(ctx, db, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, db, second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
	if err := Delete(ctx, db, second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete() error = %v, want ErrNotFound", err)
	}
	if _, err := Update(ctx, db, 999999, CreateProfile{Name: "Missing", ChannelRef: "@missing", AdTag: "dddddddddddddddddddddddddddddddd", Enabled: true, Weight: 1}, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want ErrNotFound", err)
	}
}
