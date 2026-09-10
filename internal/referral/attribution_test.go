package referral

import (
	"context"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func TestAttributeNewInviteeCreatesPendingOnceAndPreservesAttributionOnReplay(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	inviter := resolveTestUser(t, ctx, db, 12001)
	alternate := resolveTestUser(t, ctx, db, 12002)
	invitee := resolveTestUser(t, ctx, db, 12003)
	now := time.Unix(1_800_001_000, 0).UTC()

	inviterCode, err := EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	alternateCode, err := EnsureCode(ctx, db, alternate.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}

	created, err := AttributeNewInvitee(ctx, db, invitee, inviterCode.Value, now)
	if err != nil {
		t.Fatalf("AttributeNewInvitee() error = %v", err)
	}
	if created.Outcome != OutcomeCreated || created.Attribution == nil {
		t.Fatalf("created result = %#v", created)
	}
	if created.Attribution.InviterUserID != inviter.User.ID || created.Attribution.InviteeUserID != invitee.User.ID || created.Attribution.Status != StatusPending {
		t.Fatalf("created attribution = %#v", created.Attribution)
	}

	replayed, err := AttributeNewInvitee(ctx, db, invitee, alternateCode.Value, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("replayed AttributeNewInvitee() error = %v", err)
	}
	if replayed.Outcome != OutcomeExisting || replayed.Attribution == nil {
		t.Fatalf("replayed result = %#v", replayed)
	}
	if replayed.Attribution.ID != created.Attribution.ID || replayed.Attribution.InviterUserID != inviter.User.ID {
		t.Fatalf("replay changed attribution: before=%#v after=%#v", created.Attribution, replayed.Attribution)
	}

	count, err := RewardedCount(ctx, db, inviter.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("pending referral count = %d, want 0", count)
	}
	var creditBuckets int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credit_buckets WHERE proxy_user_id = ?", invitee.User.ProxyUserID).Scan(&creditBuckets); err != nil {
		t.Fatal(err)
	}
	if creditBuckets != 0 {
		t.Fatalf("referral attribution created %d credit buckets, want 0", creditBuckets)
	}
}

func TestAttributeNewInviteeRejectsUnsafeOutcomesWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	now := time.Unix(1_800_002_000, 0).UTC()
	inviter := resolveTestUser(t, ctx, db, 13001)
	inviterCode, err := EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}

	invalidInvitee := resolveTestUser(t, ctx, db, 13002)
	assertOutcome(t, ctx, db, invalidInvitee, "bad code", now, OutcomeInvalidCode)
	assertNoAttribution(t, ctx, db, invalidInvitee.User.ID)

	unknownInvitee := resolveTestUser(t, ctx, db, 13003)
	assertOutcome(t, ctx, db, unknownInvitee, "AAAAAAAAAAAAAAAAAAAAAAAA", now, OutcomeUnknownCode)
	assertNoAttribution(t, ctx, db, unknownInvitee.User.ID)

	self := resolveTestUser(t, ctx, db, 13004)
	selfCode, err := EnsureCode(ctx, db, self.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	assertOutcome(t, ctx, db, self, selfCode.Value, now, OutcomeSelfReferral)
	assertNoAttribution(t, ctx, db, self.User.ID)

	newThenExisting := resolveTestUser(t, ctx, db, 13005)
	existing, err := telegramuser.Resolve(ctx, db, newThenExisting.User.TelegramID, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if existing.Created {
		t.Fatal("second Resolve() Created = true, want false")
	}
	assertOutcome(t, ctx, db, existing, inviterCode.Value, now, OutcomeInviteeNotNew)
	assertNoAttribution(t, ctx, db, existing.User.ID)
}

func TestReferralSchemaEnforcesSingleInviteeAndNoSelfReferral(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	now := time.Unix(1_800_003_000, 0).UTC()
	inviter := resolveTestUser(t, ctx, db, 14001)
	otherInviter := resolveTestUser(t, ctx, db, 14002)
	invitee := resolveTestUser(t, ctx, db, 14003)

	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_attributions(inviter_user_id, invitee_user_id, status, created_at, updated_at)
VALUES (?, ?, 'pending', ?, ?)`, inviter.User.ID, invitee.User.ID, now.Unix(), now.Unix()); err != nil {
		t.Fatalf("insert first attribution: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_attributions(inviter_user_id, invitee_user_id, status, created_at, updated_at)
VALUES (?, ?, 'pending', ?, ?)`, otherInviter.User.ID, invitee.User.ID, now.Unix(), now.Unix()); err == nil {
		t.Fatal("duplicate invitee attribution unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO referral_attributions(inviter_user_id, invitee_user_id, status, created_at, updated_at)
VALUES (?, ?, 'pending', ?, ?)`, inviter.User.ID, inviter.User.ID, now.Unix(), now.Unix()); err == nil {
		t.Fatal("self referral unexpectedly succeeded")
	}
}

func TestRewardedCountExcludesPendingAndRejected(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	now := time.Unix(1_800_004_000, 0).UTC()
	inviter := resolveTestUser(t, ctx, db, 15001)
	code, err := EnsureCode(ctx, db, inviter.User.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	pendingInvitee := resolveTestUser(t, ctx, db, 15002)
	rejectedInvitee := resolveTestUser(t, ctx, db, 15003)
	rewardedInvitee := resolveTestUser(t, ctx, db, 15004)
	for _, invitee := range []telegramuser.ResolveResult{pendingInvitee, rejectedInvitee, rewardedInvitee} {
		result, err := AttributeNewInvitee(ctx, db, invitee, code.Value, now)
		if err != nil || result.Outcome != OutcomeCreated {
			t.Fatalf("create attribution for %d = %#v, %v", invitee.User.ID, result, err)
		}
	}
	finalized := now.Add(time.Minute).Unix()
	if _, err := db.ExecContext(ctx, `
UPDATE referral_attributions
SET status = 'rejected', rejection_reason = 'fixture', finalized_at = ?, updated_at = ?
WHERE invitee_user_id = ?`, finalized, finalized, rejectedInvitee.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE referral_attributions
SET status = 'rewarded', finalized_at = ?, updated_at = ?
WHERE invitee_user_id = ?`, finalized, finalized, rewardedInvitee.User.ID); err != nil {
		t.Fatal(err)
	}

	count, err := RewardedCount(ctx, db, inviter.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("RewardedCount() = %d, want 1", count)
	}
}
