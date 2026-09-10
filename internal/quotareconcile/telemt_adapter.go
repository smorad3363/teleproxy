package quotareconcile

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/smorad3363/teleproxy/internal/telemt"
)

type telemtDataPlane struct {
	client *telemt.Client
}

func NewTelemtReconciler(db *sql.DB, client *telemt.Client, options Options) (*Reconciler, error) {
	if client == nil {
		return nil, fmt.Errorf("Telemt client is required")
	}
	return New(db, telemtDataPlane{client: client}, options)
}

func (p telemtDataPlane) SetEnabled(ctx context.Context, username string, enabled bool) error {
	_, err := p.client.SetUserEnabled(ctx, username, enabled)
	return err
}

func (p telemtDataPlane) QuotaUsage(ctx context.Context, username string) (QuotaUsage, bool, error) {
	usage, found, err := p.client.GetUserQuotaUsage(ctx, username)
	if err != nil || !found {
		return QuotaUsage{}, found, err
	}
	return QuotaUsage{
		DataQuotaBytes:     usage.DataQuotaBytes,
		UsedBytes:          usage.UsedBytes,
		LastResetEpochSecs: usage.LastResetEpochSecs,
	}, true, nil
}

func (p telemtDataPlane) ResetQuota(ctx context.Context, username string) (ResetResult, error) {
	reset, err := p.client.ResetUserQuota(ctx, username)
	if err != nil {
		return ResetResult{}, err
	}
	return ResetResult{UsedBytes: reset.UsedBytes, LastResetEpochSecs: reset.LastResetEpochSecs}, nil
}

func (p telemtDataPlane) ApplyPolicy(ctx context.Context, username string, policy Policy) (PolicyState, error) {
	patch := telemt.UserPolicyPatch{DataQuota: telemt.SetDataQuotaBytes(policy.DataQuotaBytes)}
	if policy.ExpiresAt == nil {
		patch.Expiration = telemt.ClearExpiration()
	} else {
		patch.Expiration = telemt.SetExpirationRFC3339(policy.ExpiresAt.UTC().Format(time.RFC3339))
	}
	state, err := p.client.PatchUserPolicy(ctx, username, patch)
	if err != nil {
		return PolicyState{}, err
	}
	result := PolicyState{DataQuotaBytes: state.DataQuotaBytes}
	if state.ExpirationRFC3339 != nil {
		expires, err := time.Parse(time.RFC3339Nano, *state.ExpirationRFC3339)
		if err != nil {
			return PolicyState{}, fmt.Errorf("parse Telemt policy expiration: %w", err)
		}
		expires = expires.UTC()
		result.ExpiresAt = &expires
	}
	return result, nil
}
