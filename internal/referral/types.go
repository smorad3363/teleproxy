package referral

import (
	"errors"
	"time"
)

var (
	ErrCodeNotFound         = errors.New("referral code not found")
	ErrAttributionNotFound  = errors.New("referral attribution not found")
	ErrTelegramUserNotFound = errors.New("referral Telegram user not found")
)

const (
	codeRandomBytes    = 18
	codeCreateAttempts = 8
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusRewarded Status = "rewarded"
	StatusRejected Status = "rejected"
)

type AttributionOutcome string

const (
	OutcomeCreated       AttributionOutcome = "created"
	OutcomeExisting      AttributionOutcome = "existing"
	OutcomeInviteeNotNew AttributionOutcome = "invitee_not_new"
	OutcomeInvalidCode   AttributionOutcome = "invalid_code"
	OutcomeUnknownCode   AttributionOutcome = "unknown_code"
	OutcomeSelfReferral  AttributionOutcome = "self_referral"
)

type Code struct {
	TelegramUserID int64     `json:"telegram_user_id"`
	Value          string    `json:"code"`
	CreatedAt      time.Time `json:"created_at"`
}

type Attribution struct {
	ID              int64      `json:"id"`
	InviterUserID   int64      `json:"inviter_user_id"`
	InviteeUserID   int64      `json:"invitee_user_id"`
	Status          Status     `json:"status"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	FinalizedAt     *time.Time `json:"finalized_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AttributionResult struct {
	Outcome     AttributionOutcome `json:"outcome"`
	Attribution *Attribution       `json:"attribution,omitempty"`
}
