package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/smorad3363/teleproxy/internal/telegramuser"
)

func AttributeNewInvitee(ctx context.Context, db *sql.DB, resolved telegramuser.ResolveResult, codeValue string, now time.Time) (AttributionResult, error) {
	if db == nil {
		return AttributionResult{}, fmt.Errorf("referral database is required")
	}
	if now.IsZero() {
		return AttributionResult{}, fmt.Errorf("referral attribution time is required")
	}
	if resolved.User.ID <= 0 || resolved.User.TelegramID <= 0 {
		return AttributionResult{}, fmt.Errorf("resolved Telegram user is required")
	}
	if !resolved.Created {
		return AttributionResult{Outcome: OutcomeInviteeNotNew}, nil
	}
	if err := verifyResolvedUser(ctx, db, resolved.User); err != nil {
		return AttributionResult{}, err
	}
	if existing, err := GetAttribution(ctx, db, resolved.User.ID); err == nil {
		return AttributionResult{Outcome: OutcomeExisting, Attribution: &existing}, nil
	} else if !errors.Is(err, ErrAttributionNotFound) {
		return AttributionResult{}, err
	}
	if !validCode(codeValue) {
		return AttributionResult{Outcome: OutcomeInvalidCode}, nil
	}

	inviterCode, err := getCodeByValue(ctx, db, codeValue)
	if errors.Is(err, ErrCodeNotFound) {
		return AttributionResult{Outcome: OutcomeUnknownCode}, nil
	}
	if err != nil {
		return AttributionResult{}, err
	}
	if inviterCode.TelegramUserID == resolved.User.ID {
		return AttributionResult{Outcome: OutcomeSelfReferral}, nil
	}

	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
INSERT INTO referral_attributions(
    inviter_user_id, invitee_user_id, status, rejection_reason, finalized_at, created_at, updated_at
) VALUES (?, ?, 'pending', NULL, NULL, ?, ?)`,
		inviterCode.TelegramUserID, resolved.User.ID, now.Unix(), now.Unix(),
	)
	if isUniqueConstraint(err) {
		if existing, getErr := GetAttribution(ctx, db, resolved.User.ID); getErr == nil {
			return AttributionResult{Outcome: OutcomeExisting, Attribution: &existing}, nil
		}
	}
	if err != nil {
		return AttributionResult{}, fmt.Errorf("create referral attribution: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return AttributionResult{}, fmt.Errorf("read referral attribution id: %w", err)
	}
	attribution, err := getAttributionByID(ctx, db, id)
	if err != nil {
		return AttributionResult{}, err
	}
	return AttributionResult{Outcome: OutcomeCreated, Attribution: &attribution}, nil
}

func GetAttribution(ctx context.Context, db *sql.DB, inviteeUserID int64) (Attribution, error) {
	if db == nil {
		return Attribution{}, fmt.Errorf("referral database is required")
	}
	if inviteeUserID <= 0 {
		return Attribution{}, fmt.Errorf("invitee Telegram user ID must be greater than zero")
	}
	row := db.QueryRowContext(ctx, `
SELECT id, inviter_user_id, invitee_user_id, status, rejection_reason, finalized_at, created_at, updated_at
FROM referral_attributions
WHERE invitee_user_id = ?`, inviteeUserID)
	return scanAttribution(row)
}

func RewardedCount(ctx context.Context, db *sql.DB, inviterUserID int64) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("referral database is required")
	}
	if inviterUserID <= 0 {
		return 0, fmt.Errorf("inviter Telegram user ID must be greater than zero")
	}
	var count int64
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM referral_attributions
WHERE inviter_user_id = ? AND status = 'rewarded'`, inviterUserID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rewarded referrals: %w", err)
	}
	return count, nil
}

func getAttributionByID(ctx context.Context, db *sql.DB, id int64) (Attribution, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, inviter_user_id, invitee_user_id, status, rejection_reason, finalized_at, created_at, updated_at
FROM referral_attributions
WHERE id = ?`, id)
	return scanAttribution(row)
}

type scanner interface {
	Scan(...any) error
}

func scanAttribution(row scanner) (Attribution, error) {
	var attribution Attribution
	var status string
	var rejectionReason sql.NullString
	var finalizedAt sql.NullInt64
	var createdAt, updatedAt int64
	if err := row.Scan(
		&attribution.ID,
		&attribution.InviterUserID,
		&attribution.InviteeUserID,
		&status,
		&rejectionReason,
		&finalizedAt,
		&createdAt,
		&updatedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return Attribution{}, ErrAttributionNotFound
	} else if err != nil {
		return Attribution{}, fmt.Errorf("read referral attribution: %w", err)
	}
	attribution.Status = Status(status)
	if attribution.Status != StatusPending && attribution.Status != StatusRewarded && attribution.Status != StatusRejected {
		return Attribution{}, fmt.Errorf("stored referral attribution status is invalid")
	}
	if rejectionReason.Valid {
		reason := rejectionReason.String
		attribution.RejectionReason = &reason
	}
	if finalizedAt.Valid {
		value := time.Unix(finalizedAt.Int64, 0).UTC()
		attribution.FinalizedAt = &value
	}
	attribution.CreatedAt = time.Unix(createdAt, 0).UTC()
	attribution.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return attribution, nil
}

func verifyResolvedUser(ctx context.Context, db *sql.DB, user telegramuser.User) error {
	var telegramID int64
	err := db.QueryRowContext(ctx, "SELECT telegram_id FROM telegram_users WHERE id = ?", user.ID).Scan(&telegramID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTelegramUserNotFound
	}
	if err != nil {
		return fmt.Errorf("verify resolved Telegram user: %w", err)
	}
	if telegramID != user.TelegramID {
		return fmt.Errorf("resolved Telegram user does not match persisted identity")
	}
	return nil
}
