package useradmin

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/referral"
	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestListProjectsAuthoritativeStateAndPaginates(t *testing.T) {
	ctx := context.Background()
	db := userAdminTestDB(t)
	now := time.Date(2032, 3, 4, 5, 6, 7, 0, time.UTC)

	invitee1 := resolveUser(t, ctx, db, 8101, now.Add(-4*time.Hour))
	invitee2 := resolveUser(t, ctx, db, 8102, now.Add(-3*time.Hour))
	inviter := resolveUser(t, ctx, db, 8103, now.Add(-2*time.Hour))
	if _, err := proxyuser.SetDesiredEnabled(ctx, db, inviter.User.ProxyUsername, false); err != nil {
		t.Fatal(err)
	}
	if _, err := proxyuser.MarkSyncError(ctx, db, inviter.User.ProxyUsername, "TELEMT_UNAVAILABLE"); err != nil {
		t.Fatal(err)
	}
	if _, created, err := proxyprovision.Prepare(ctx, db, inviter.User.ProxyUsername, [32]byte{1}, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("proxyprovision.Prepare() created = false")
	}

	expiresSoon := now.Add(2 * time.Hour)
	if _, err := credit.GrantBucket(ctx, db, inviter.User.ProxyUsername, credit.Grant{
		OriginalBytes: 1000, StartsAt: now.Add(-time.Hour), ExpiresAt: &expiresSoon, RewardType: "manual", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := credit.GrantBucket(ctx, db, inviter.User.ProxyUsername, credit.Grant{
		OriginalBytes: 500, StartsAt: now.Add(-time.Hour), RewardType: "manual", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}
	expiredAt := now.Add(-time.Hour)
	if _, err := credit.GrantBucket(ctx, db, inviter.User.ProxyUsername, credit.Grant{
		OriginalBytes: 900, StartsAt: now.Add(-2 * time.Hour), ExpiresAt: &expiredAt, RewardType: "manual", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := credit.GrantBucket(ctx, db, inviter.User.ProxyUsername, credit.Grant{
		OriginalBytes: 700, StartsAt: now.Add(time.Hour), RewardType: "manual", Source: "test",
	}); err != nil {
		t.Fatal(err)
	}

	code, err := referral.EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, invitee := range []telegramuser.ResolveResult{invitee1, invitee2} {
		result, err := referral.AttributeNewInvitee(ctx, db, invitee, code.Value, now)
		if err != nil || result.Outcome != referral.OutcomeCreated {
			t.Fatalf("attribute invitee = %#v, %v", result, err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE referral_attributions SET status = 'rejected', rejection_reason = 'COOLDOWN', finalized_at = ?, updated_at = ? WHERE invitee_user_id = ?`, now.Unix(), now.Unix(), invitee2.User.ID); err != nil {
		t.Fatal(err)
	}

	first, err := List(ctx, db, ListQuery{Limit: 1}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].TelegramUserID != inviter.User.ID {
		t.Fatalf("first page = %#v", first)
	}
	entry := first.Items[0]
	if entry.TelegramID != inviter.User.TelegramID || entry.ProxyUserID != inviter.User.ProxyUserID || entry.ProxyUsername != inviter.User.ProxyUsername {
		t.Fatalf("identity = %#v", entry)
	}
	if entry.DesiredEnabled || entry.SyncState != proxyuser.SyncError || entry.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("proxy state = %#v", entry)
	}
	if entry.ProvisioningPhase == nil || *entry.ProvisioningPhase != proxyprovision.PhasePrepared {
		t.Fatalf("provisioning phase = %v, want prepared", entry.ProvisioningPhase)
	}
	if entry.AvailableBytes != 1500 || entry.NearestExpiry == nil || !entry.NearestExpiry.Equal(expiresSoon) {
		t.Fatalf("credit projection = %#v", entry)
	}
	if entry.ReferralCount != 2 {
		t.Fatalf("referral count = %d, want 2", entry.ReferralCount)
	}
	if first.NextBeforeID == nil || *first.NextBeforeID != inviter.User.ID {
		t.Fatalf("next_before_id = %v", first.NextBeforeID)
	}

	second, err := List(ctx, db, ListQuery{BeforeID: *first.NextBeforeID, Limit: 2}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 2 || second.Items[0].TelegramUserID != invitee2.User.ID || second.Items[1].TelegramUserID != invitee1.User.ID || second.NextBeforeID != nil {
		t.Fatalf("second page = %#v", second)
	}
	if second.Items[0].ProvisioningPhase != nil || second.Items[1].ProvisioningPhase != nil {
		t.Fatalf("unprovisioned phases = %v, %v, want nil", second.Items[0].ProvisioningPhase, second.Items[1].ProvisioningPhase)
	}
}

func TestListFiltersExactProvisioningPhase(t *testing.T) {
	ctx := context.Background()
	db := userAdminTestDB(t)
	now := time.Date(2032, 4, 5, 6, 7, 8, 0, time.UTC)

	unprovisioned := resolveUser(t, ctx, db, 8301, now.Add(-3*time.Hour))
	prepared := resolveUser(t, ctx, db, 8302, now.Add(-2*time.Hour))
	collision := resolveUser(t, ctx, db, 8303, now.Add(-time.Hour))

	preparedDigest := [32]byte{2}
	if _, created, err := proxyprovision.Prepare(ctx, db, prepared.User.ProxyUsername, preparedDigest, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("prepared proxyprovision.Prepare() created = false")
	}
	collisionDigest := [32]byte{3}
	if _, created, err := proxyprovision.Prepare(ctx, db, collision.User.ProxyUsername, collisionDigest, now); err != nil {
		t.Fatal(err)
	} else if !created {
		t.Fatal("collision proxyprovision.Prepare() created = false")
	}
	if _, err := proxyprovision.MarkCollision(ctx, db, collision.User.ProxyUsername, collisionDigest, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	preparedPage, err := List(ctx, db, ListQuery{ProvisioningPhase: proxyprovision.PhasePrepared}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(preparedPage.Items) != 1 || preparedPage.Items[0].TelegramUserID != prepared.User.ID || preparedPage.Items[0].ProvisioningPhase == nil || *preparedPage.Items[0].ProvisioningPhase != proxyprovision.PhasePrepared {
		t.Fatalf("prepared phase page = %#v", preparedPage)
	}

	collisionPage, err := List(ctx, db, ListQuery{ProvisioningPhase: proxyprovision.PhaseCollision}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(collisionPage.Items) != 1 || collisionPage.Items[0].TelegramUserID != collision.User.ID || collisionPage.Items[0].ProvisioningPhase == nil || *collisionPage.Items[0].ProvisioningPhase != proxyprovision.PhaseCollision {
		t.Fatalf("collision phase page = %#v", collisionPage)
	}

	composed, err := List(ctx, db, ListQuery{TelegramID: prepared.User.TelegramID, ProvisioningPhase: proxyprovision.PhasePrepared}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(composed.Items) != 1 || composed.Items[0].TelegramUserID != prepared.User.ID {
		t.Fatalf("composed phase page = %#v", composed)
	}
	contradictory, err := List(ctx, db, ListQuery{TelegramID: unprovisioned.User.TelegramID, ProvisioningPhase: proxyprovision.PhasePrepared}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(contradictory.Items) != 0 {
		t.Fatalf("contradictory phase page = %#v", contradictory)
	}
}

func TestListEmptyAndValidation(t *testing.T) {
	db := userAdminTestDB(t)
	now := time.Now().UTC()
	page, err := List(context.Background(), db, ListQuery{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("empty items = %#v", page.Items)
	}
	for _, query := range []ListQuery{
		{BeforeID: -1},
		{Limit: -1},
		{Limit: MaxListLimit + 1},
		{ProvisioningPhase: proxyprovision.Phase("unknown")},
		{ProvisioningPhase: proxyprovision.Phase("PREPARED")},
	} {
		if _, err := List(context.Background(), db, query, now); err == nil {
			t.Fatalf("List(%#v) error = nil", query)
		}
	}
}

func TestListFailsClosedOnUnsafeStoredErrorCode(t *testing.T) {
	ctx := context.Background()
	db := userAdminTestDB(t)
	now := time.Now().UTC()
	resolved := resolveUser(t, ctx, db, 8201, now)
	if _, err := db.ExecContext(ctx, `UPDATE proxy_users SET sync_state = 'error', last_error_code = ? WHERE id = ?`, "unsafe detail", resolved.User.ProxyUserID); err != nil {
		t.Fatal(err)
	}
	if _, err := List(ctx, db, ListQuery{}, now); err == nil {
		t.Fatal("List() error = nil, want fail-closed error")
	}
}

func userAdminTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "user-admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resolveUser(t *testing.T, ctx context.Context, db *sql.DB, telegramID int64, now time.Time) telegramuser.ResolveResult {
	t.Helper()
	resolved, err := telegramuser.Resolve(ctx, db, telegramID, now)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Created {
		t.Fatalf("telegramuser.Resolve(%d) Created = false", telegramID)
	}
	return resolved
}
