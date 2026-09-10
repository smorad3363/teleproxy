package sponsor

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestCreateGetListProfilesNormalizesAndOrders(t *testing.T) {
	ctx := context.Background()
	db := sponsorTestDB(t)
	start := time.Unix(1_830_000_100, 987654321).UTC()
	end := time.Unix(1_830_100_100, 123456789).UTC()
	created, err := Create(ctx, db, CreateProfile{
		Name:       "  Main Sponsor  ",
		ChannelRef: "@Sponsor_Channel",
		AdTag:      "ABCDEF0123456789ABCDEF0123456789",
		Enabled:    true,
		Weight:     5,
		StartsAt:   &start,
		EndsAt:     &end,
		Notes:      "primary campaign",
	}, time.Unix(1_830_000_000, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Main Sponsor" || created.ChannelRef != "@sponsor_channel" || created.AdTag != "abcdef0123456789abcdef0123456789" || !created.Enabled || created.Weight != 5 {
		t.Fatalf("created profile = %#v", created)
	}
	if created.StartsAt == nil || !created.StartsAt.Equal(start.Truncate(time.Second)) || created.EndsAt == nil || !created.EndsAt.Equal(end.Truncate(time.Second)) {
		t.Fatalf("created time window = %#v", created)
	}

	second, err := Create(ctx, db, CreateProfile{
		Name:       "Backup",
		ChannelRef: "https://T.ME/backup_channel",
		AdTag:      "11111111111111111111111111111111",
		Enabled:    false,
		Weight:     1,
	}, time.Unix(1_830_000_001, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if second.ChannelRef != "https://t.me/backup_channel" {
		t.Fatalf("second channel ref = %q", second.ChannelRef)
	}

	got, err := Get(ctx, db, created.ID)
	if err != nil || got.ID != created.ID || got.AdTag != created.AdTag {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	profiles, err := List(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || profiles[0].ID != created.ID || profiles[1].ID != second.ID {
		t.Fatalf("List() = %#v", profiles)
	}
}

func TestCreateRejectsDuplicateAdTagCaseInsensitively(t *testing.T) {
	ctx := context.Background()
	db := sponsorTestDB(t)
	now := time.Unix(1_830_001_000, 0).UTC()
	if _, err := Create(ctx, db, CreateProfile{Name: "One", ChannelRef: "@one", AdTag: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Enabled: true, Weight: 1}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, db, CreateProfile{Name: "Two", ChannelRef: "@two", AdTag: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", Enabled: true, Weight: 2}, now.Add(time.Second)); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate Create() error = %v, want ErrConflict", err)
	}
	if countSponsorProfiles(t, db) != 1 {
		t.Fatal("duplicate ad tag mutated sponsor profile count")
	}
}

func TestCreateRejectsInvalidProfilesWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := sponsorTestDB(t)
	now := time.Unix(1_830_002_000, 0).UTC()
	start := now.Add(time.Hour)
	equalEnd := start
	zero := time.Time{}
	invalidUTF8 := string([]byte{0xff})

	tests := []CreateProfile{
		{Name: "", ChannelRef: "@ok", AdTag: "0123456789abcdef0123456789abcdef", Enabled: true, Weight: 1},
		{Name: "Bad channel", ChannelRef: "https://example.com/channel", AdTag: "1123456789abcdef0123456789abcdef", Enabled: true, Weight: 1},
		{Name: "Bad tag short", ChannelRef: "@ok", AdTag: "abcdef", Enabled: true, Weight: 1},
		{Name: "Bad tag chars", ChannelRef: "@ok", AdTag: "gggggggggggggggggggggggggggggggg", Enabled: true, Weight: 1},
		{Name: "Bad weight", ChannelRef: "@ok", AdTag: "2123456789abcdef0123456789abcdef", Enabled: true, Weight: 0},
		{Name: "Bad window", ChannelRef: "@ok", AdTag: "3123456789abcdef0123456789abcdef", Enabled: true, Weight: 1, StartsAt: &start, EndsAt: &equalEnd},
		{Name: "Bad zero time", ChannelRef: "@ok", AdTag: "4123456789abcdef0123456789abcdef", Enabled: true, Weight: 1, StartsAt: &zero},
		{Name: "Bad notes", ChannelRef: "@ok", AdTag: "5123456789abcdef0123456789abcdef", Enabled: true, Weight: 1, Notes: invalidUTF8},
		{Name: "Long notes", ChannelRef: "@ok", AdTag: "6123456789abcdef0123456789abcdef", Enabled: true, Weight: 1, Notes: strings.Repeat("x", 4097)},
	}
	for index, input := range tests {
		if _, err := Create(ctx, db, input, now.Add(time.Duration(index)*time.Second)); err == nil {
			t.Fatalf("invalid Create(%d) unexpectedly succeeded: %#v", index, input)
		}
	}
	if countSponsorProfiles(t, db) != 0 {
		t.Fatal("invalid profiles mutated sponsor table")
	}
}

func TestSponsorSchemaRejectsNonCanonicalAdTagAndInvalidWindow(t *testing.T) {
	db := sponsorTestDB(t)
	if _, err := db.Exec(`
INSERT INTO sponsor_profiles(name, channel_ref, ad_tag, enabled, weight, starts_at, ends_at, notes, created_at, updated_at)
VALUES ('Uppercase', '@upper', 'ABCDEF0123456789ABCDEF0123456789', 1, 1, NULL, NULL, '', 1, 1)`); err == nil {
		t.Fatal("schema accepted non-canonical uppercase ad_tag")
	}
	if _, err := db.Exec(`
INSERT INTO sponsor_profiles(name, channel_ref, ad_tag, enabled, weight, starts_at, ends_at, notes, created_at, updated_at)
VALUES ('Window', '@window', 'abcdef0123456789abcdef0123456789', 1, 1, 10, 10, '', 1, 1)`); err == nil {
		t.Fatal("schema accepted invalid sponsor time window")
	}
}

func sponsorTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countSponsorProfiles(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sponsor_profiles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
