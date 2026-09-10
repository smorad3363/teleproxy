package referral

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resolveTestUser(t *testing.T, ctx context.Context, db *sql.DB, telegramID int64) telegramuser.ResolveResult {
	t.Helper()
	resolved, err := telegramuser.Resolve(ctx, db, telegramID, time.Unix(1_799_999_000, 0).UTC())
	if err != nil {
		t.Fatalf("telegramuser.Resolve(%d) error = %v", telegramID, err)
	}
	if !resolved.Created {
		t.Fatalf("telegramuser.Resolve(%d) Created = false, want true", telegramID)
	}
	return resolved
}

func assertOutcome(t *testing.T, ctx context.Context, db *sql.DB, resolved telegramuser.ResolveResult, code string, now time.Time, want AttributionOutcome) {
	t.Helper()
	result, err := AttributeNewInvitee(ctx, db, resolved, code, now)
	if err != nil {
		t.Fatalf("AttributeNewInvitee() error = %v", err)
	}
	if result.Outcome != want || result.Attribution != nil {
		t.Fatalf("AttributeNewInvitee() = %#v, want outcome %q without attribution", result, want)
	}
}

func assertNoAttribution(t *testing.T, ctx context.Context, db *sql.DB, inviteeUserID int64) {
	t.Helper()
	if _, err := GetAttribution(ctx, db, inviteeUserID); !errors.Is(err, ErrAttributionNotFound) {
		t.Fatalf("GetAttribution(%d) error = %v, want ErrAttributionNotFound", inviteeUserID, err)
	}
}
