package referral

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func ApproveEligibility(ctx context.Context, db *sql.DB, inviteeUserID int64, now time.Time) (EligibilityResult, error) {
	if err := validateEligibilityMutation(db, inviteeUserID, now); err != nil {
		return EligibilityResult{}, err
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
UPDATE referral_attributions
SET eligible_at = ?, updated_at = ?
WHERE invitee_user_id = ?
  AND status = 'pending'
  AND eligible_at IS NULL`, now.Unix(), now.Unix(), inviteeUserID)
	if err != nil {
		return EligibilityResult{}, fmt.Errorf("approve referral eligibility: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return EligibilityResult{}, fmt.Errorf("read referral eligibility update result: %w", err)
	}
	attribution, err := GetAttribution(ctx, db, inviteeUserID)
	if err != nil {
		return EligibilityResult{}, err
	}
	if changed == 1 {
		return EligibilityResult{Outcome: EligibilityApproved, Attribution: attribution}, nil
	}
	return existingEligibilityResult(attribution)
}

func RejectEligibility(ctx context.Context, db *sql.DB, inviteeUserID int64, reason RejectionReason, now time.Time) (EligibilityResult, error) {
	if err := validateEligibilityMutation(db, inviteeUserID, now); err != nil {
		return EligibilityResult{}, err
	}
	if !validRejectionReason(reason) {
		return EligibilityResult{}, fmt.Errorf("referral rejection reason is invalid")
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
UPDATE referral_attributions
SET status = 'rejected', rejection_reason = ?, finalized_at = ?, updated_at = ?
WHERE invitee_user_id = ?
  AND status = 'pending'
  AND eligible_at IS NULL`, string(reason), now.Unix(), now.Unix(), inviteeUserID)
	if err != nil {
		return EligibilityResult{}, fmt.Errorf("reject referral eligibility: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return EligibilityResult{}, fmt.Errorf("read referral rejection update result: %w", err)
	}
	attribution, err := GetAttribution(ctx, db, inviteeUserID)
	if err != nil {
		return EligibilityResult{}, err
	}
	if changed == 1 {
		return EligibilityResult{Outcome: EligibilityRejected, Attribution: attribution}, nil
	}
	return existingEligibilityResult(attribution)
}

func validateEligibilityMutation(db *sql.DB, inviteeUserID int64, now time.Time) error {
	if db == nil {
		return fmt.Errorf("referral database is required")
	}
	if inviteeUserID <= 0 {
		return fmt.Errorf("invitee Telegram user ID must be greater than zero")
	}
	if now.IsZero() {
		return fmt.Errorf("referral eligibility time is required")
	}
	return nil
}

func existingEligibilityResult(attribution Attribution) (EligibilityResult, error) {
	switch {
	case attribution.Status == StatusRewarded:
		return EligibilityResult{Outcome: EligibilityAlreadyRewarded, Attribution: attribution}, nil
	case attribution.Status == StatusRejected:
		return EligibilityResult{Outcome: EligibilityAlreadyRejected, Attribution: attribution}, nil
	case attribution.Status == StatusPending && attribution.EligibleAt != nil:
		return EligibilityResult{Outcome: EligibilityAlreadyApproved, Attribution: attribution}, nil
	default:
		return EligibilityResult{}, fmt.Errorf("stored referral eligibility state is inconsistent")
	}
}

func validRejectionReason(reason RejectionReason) bool {
	switch reason {
	case RejectionAntiAbuse, RejectionDailyCap, RejectionWeeklyCap, RejectionCooldown, RejectionBlacklist, RejectionSuspicious:
		return true
	default:
		return false
	}
}
