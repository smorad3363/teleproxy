package quotareconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

var (
	ErrQuotaUsageMissing  = errors.New("Telemt quota usage is missing for projected quota")
	ErrQuotaUsageUnstable = errors.New("Telemt quota usage did not become stable")
	ErrPolicyMismatch     = errors.New("Telemt policy does not match projected quota state")
	ErrResetNotClean      = errors.New("Telemt quota reset did not return zero usage")
	ErrStateStalled       = errors.New("quota reconciliation state machine did not converge")
)

type QuotaUsage struct {
	DataQuotaBytes     uint64
	UsedBytes          uint64
	LastResetEpochSecs uint64
}

type ResetResult struct {
	UsedBytes          uint64
	LastResetEpochSecs uint64
}

type Policy struct {
	DataQuotaBytes uint64
	ExpiresAt      *time.Time
}

type PolicyState struct {
	DataQuotaBytes *uint64
	ExpiresAt      *time.Time
}

type DataPlane interface {
	SetEnabled(context.Context, string, bool) error
	QuotaUsage(context.Context, string) (QuotaUsage, bool, error)
	ResetQuota(context.Context, string) (ResetResult, error)
	ApplyPolicy(context.Context, string, Policy) (PolicyState, error)
}

type Options struct {
	Now               func() time.Time
	StableSamples     int
	MaxUsageReads     int
	StabilityInterval time.Duration
}

type Reconciler struct {
	db                *sql.DB
	plane             DataPlane
	now               func() time.Time
	stableSamples     int
	maxUsageReads     int
	stabilityInterval time.Duration
}

func New(db *sql.DB, plane DataPlane, options Options) (*Reconciler, error) {
	if db == nil {
		return nil, fmt.Errorf("quota reconciliation database is required")
	}
	if plane == nil {
		return nil, fmt.Errorf("quota reconciliation data plane is required")
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	if options.StableSamples <= 0 {
		options.StableSamples = 3
	}
	if options.MaxUsageReads <= 0 {
		options.MaxUsageReads = 8
	}
	if options.MaxUsageReads < options.StableSamples {
		return nil, fmt.Errorf("max quota usage reads must be at least stable sample count")
	}
	if options.StabilityInterval < 0 {
		return nil, fmt.Errorf("quota usage stability interval must not be negative")
	}
	if options.StabilityInterval == 0 {
		options.StabilityInterval = 50 * time.Millisecond
	}
	return &Reconciler{
		db:                db,
		plane:             plane,
		now:               options.Now,
		stableSamples:     options.StableSamples,
		maxUsageReads:     options.MaxUsageReads,
		stabilityInterval: options.StabilityInterval,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if _, err := proxyuser.Get(ctx, r.db, username); err != nil {
		return err
	}

	snapshot, err := credit.LoadReconciliation(ctx, r.db, username)
	if errors.Is(err, credit.ErrReconciliationNotFound) {
		if err := r.bootstrap(ctx, username); err != nil {
			return err
		}
		snapshot, err = credit.LoadReconciliation(ctx, r.db, username)
	}
	if err != nil {
		return err
	}

	if snapshot.Phase == credit.PhaseActive {
		if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseActive, credit.PhaseBlocking); err != nil {
			return err
		}
		snapshot.Phase = credit.PhaseBlocking
	}

	for step := 0; step < 12; step++ {
		switch snapshot.Phase {
		case credit.PhaseBlocking:
			if err := r.plane.SetEnabled(ctx, username, false); err != nil {
				return fmt.Errorf("disable Telemt user: %w", err)
			}
			if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseBlocking, credit.PhaseBlocked); err != nil {
				return err
			}
			snapshot.Phase = credit.PhaseBlocked

		case credit.PhaseBlocked:
			observation, err := r.stableObservation(ctx, username, snapshot)
			if err != nil {
				return err
			}
			if _, err := credit.AccountObservedUsage(ctx, r.db, username, observation); err != nil {
				return err
			}
			if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseBlocked, credit.PhaseResetting); err != nil {
				return err
			}
			snapshot.Phase = credit.PhaseResetting

		case credit.PhaseResetting:
			reset, err := r.plane.ResetQuota(ctx, username)
			if err != nil {
				return fmt.Errorf("reset Telemt quota: %w", err)
			}
			if reset.UsedBytes != 0 {
				return ErrResetNotClean
			}
			next, err := credit.PrepareProjection(ctx, r.db, username, credit.PrepareProjectionInput{
				ExpectedGeneration:      snapshot.Generation,
				TelemtResetEpochSecs:    reset.LastResetEpochSecs,
				TelemtBaselineUsedBytes: 0,
				ProjectedAt:             r.now().UTC(),
			})
			if err != nil {
				return err
			}
			snapshot = next

		case credit.PhaseApplying:
			if err := r.plane.SetEnabled(ctx, username, false); err != nil {
				return fmt.Errorf("keep Telemt user disabled while applying quota: %w", err)
			}
			state, err := r.plane.ApplyPolicy(ctx, username, policyFromSnapshot(snapshot))
			if err != nil {
				return fmt.Errorf("apply Telemt quota policy: %w", err)
			}
			if !policyMatchesSnapshot(state, snapshot) {
				return ErrPolicyMismatch
			}
			user, err := proxyuser.Get(ctx, r.db, username)
			if err != nil {
				return err
			}
			if !user.DesiredEnable {
				if err := r.plane.SetEnabled(ctx, username, false); err != nil {
					return fmt.Errorf("preserve desired-disabled Telemt user: %w", err)
				}
				if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseApplying, credit.PhaseActive); err != nil {
					return err
				}
				return nil
			}
			if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseApplying, credit.PhaseEnabling); err != nil {
				return err
			}
			snapshot.Phase = credit.PhaseEnabling

		case credit.PhaseEnabling:
			user, err := proxyuser.Get(ctx, r.db, username)
			if err != nil {
				return err
			}
			if err := r.plane.SetEnabled(ctx, username, user.DesiredEnable); err != nil {
				return fmt.Errorf("restore Telemt desired enabled state: %w", err)
			}
			if err := credit.TransitionReconciliationPhase(ctx, r.db, username, snapshot.Generation, credit.PhaseEnabling, credit.PhaseActive); err != nil {
				return err
			}
			return nil

		case credit.PhaseActive:
			return nil

		default:
			return fmt.Errorf("unsupported quota reconciliation phase %q", snapshot.Phase)
		}
	}
	return ErrStateStalled
}

func (r *Reconciler) bootstrap(ctx context.Context, username string) error {
	if err := r.plane.SetEnabled(ctx, username, false); err != nil {
		return fmt.Errorf("disable Telemt user for initial quota bootstrap: %w", err)
	}
	reset, err := r.plane.ResetQuota(ctx, username)
	if err != nil {
		return fmt.Errorf("reset Telemt quota for initial bootstrap: %w", err)
	}
	if reset.UsedBytes != 0 {
		return ErrResetNotClean
	}
	_, err = credit.PrepareProjection(ctx, r.db, username, credit.PrepareProjectionInput{
		ExpectedGeneration:      0,
		TelemtResetEpochSecs:    reset.LastResetEpochSecs,
		TelemtBaselineUsedBytes: 0,
		ProjectedAt:             r.now().UTC(),
	})
	return err
}

func (r *Reconciler) stableObservation(ctx context.Context, username string, snapshot credit.ReconciliationSnapshot) (credit.QuotaUsageObservation, error) {
	var previous QuotaUsage
	stable := 0
	for read := 0; read < r.maxUsageReads; read++ {
		usage, found, err := r.plane.QuotaUsage(ctx, username)
		if err != nil {
			return credit.QuotaUsageObservation{}, fmt.Errorf("read Telemt quota usage: %w", err)
		}
		if !found {
			if snapshot.ProjectedQuotaBytes == 0 && snapshot.TelemtBaselineUsedBytes == 0 {
				return credit.QuotaUsageObservation{
					ResetEpochSecs: snapshot.TelemtResetEpochSecs,
					UsedBytes:      0,
				}, nil
			}
			return credit.QuotaUsageObservation{}, ErrQuotaUsageMissing
		}
		if usage.DataQuotaBytes != snapshot.ProjectedQuotaBytes {
			return credit.QuotaUsageObservation{}, ErrPolicyMismatch
		}
		if usage.LastResetEpochSecs != snapshot.TelemtResetEpochSecs {
			return credit.QuotaUsageObservation{}, credit.ErrQuotaResetMismatch
		}
		if stable == 0 || usage != previous {
			previous = usage
			stable = 1
		} else {
			stable++
		}
		if stable >= r.stableSamples {
			return credit.QuotaUsageObservation{
				ResetEpochSecs: usage.LastResetEpochSecs,
				UsedBytes:      usage.UsedBytes,
			}, nil
		}
		if read+1 < r.maxUsageReads {
			if err := waitContext(ctx, r.stabilityInterval); err != nil {
				return credit.QuotaUsageObservation{}, err
			}
		}
	}
	return credit.QuotaUsageObservation{}, ErrQuotaUsageUnstable
}

func policyFromSnapshot(snapshot credit.ReconciliationSnapshot) Policy {
	policy := Policy{DataQuotaBytes: snapshot.ProjectedQuotaBytes}
	if snapshot.EnforcedExpiry != nil {
		expires := snapshot.EnforcedExpiry.UTC()
		policy.ExpiresAt = &expires
	}
	return policy
}

func policyMatchesSnapshot(state PolicyState, snapshot credit.ReconciliationSnapshot) bool {
	if state.DataQuotaBytes == nil || *state.DataQuotaBytes != snapshot.ProjectedQuotaBytes {
		return false
	}
	if snapshot.EnforcedExpiry == nil {
		return state.ExpiresAt == nil
	}
	return state.ExpiresAt != nil && state.ExpiresAt.Equal(snapshot.EnforcedExpiry.UTC())
}

func waitContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
