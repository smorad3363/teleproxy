package telemt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type patchMode uint8

const (
	patchUnchanged patchMode = iota
	patchSet
	patchClear
)

type ExpirationPatch struct {
	mode  patchMode
	value string
}

func SetExpirationRFC3339(value string) ExpirationPatch {
	return ExpirationPatch{mode: patchSet, value: value}
}

func ClearExpiration() ExpirationPatch {
	return ExpirationPatch{mode: patchClear}
}

type DataQuotaPatch struct {
	mode  patchMode
	value uint64
}

func SetDataQuotaBytes(value uint64) DataQuotaPatch {
	return DataQuotaPatch{mode: patchSet, value: value}
}

func ClearDataQuota() DataQuotaPatch {
	return DataQuotaPatch{mode: patchClear}
}

type UserPolicyPatch struct {
	Expiration ExpirationPatch
	DataQuota  DataQuotaPatch
}

type UserPolicyState struct {
	Username          string  `json:"username"`
	Enabled           bool    `json:"enabled"`
	InRuntime         bool    `json:"in_runtime"`
	ExpirationRFC3339 *string `json:"expiration_rfc3339"`
	DataQuotaBytes    *uint64 `json:"data_quota_bytes"`
	TotalOctets       uint64  `json:"total_octets"`
}

type QuotaReset struct {
	Username           string `json:"username"`
	UsedBytes          uint64 `json:"used_bytes"`
	LastResetEpochSecs uint64 `json:"last_reset_epoch_secs"`
}

func (c *Client) GetUserPolicy(ctx context.Context, username string) (UserPolicyState, error) {
	if err := validateUsername(username); err != nil {
		return UserPolicyState{}, err
	}
	var state UserPolicyState
	path := "/v1/users/" + url.PathEscape(username)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, []int{http.StatusOK}, &state); err != nil {
		return UserPolicyState{}, err
	}
	if err := validateUserPolicyState(state, username); err != nil {
		return UserPolicyState{}, err
	}
	return state, nil
}

func (c *Client) PatchUserPolicy(ctx context.Context, username string, patch UserPolicyPatch) (UserPolicyState, error) {
	if err := validateUsername(username); err != nil {
		return UserPolicyState{}, err
	}
	body := make(map[string]any, 2)

	switch patch.Expiration.mode {
	case patchUnchanged:
	case patchClear:
		body["expiration_rfc3339"] = nil
	case patchSet:
		if _, err := time.Parse(time.RFC3339Nano, patch.Expiration.value); err != nil {
			return UserPolicyState{}, fmt.Errorf("Telemt expiration must be valid RFC3339")
		}
		body["expiration_rfc3339"] = patch.Expiration.value
	default:
		return UserPolicyState{}, fmt.Errorf("Telemt expiration patch is invalid")
	}

	switch patch.DataQuota.mode {
	case patchUnchanged:
	case patchClear:
		body["data_quota_bytes"] = nil
	case patchSet:
		body["data_quota_bytes"] = patch.DataQuota.value
	default:
		return UserPolicyState{}, fmt.Errorf("Telemt data quota patch is invalid")
	}

	if len(body) == 0 {
		return UserPolicyState{}, fmt.Errorf("Telemt policy patch must change at least one field")
	}

	var state UserPolicyState
	path := "/v1/users/" + url.PathEscape(username)
	if err := c.doJSON(ctx, http.MethodPatch, path, body, []int{http.StatusOK, http.StatusAccepted}, &state); err != nil {
		return UserPolicyState{}, err
	}
	if err := validateUserPolicyState(state, username); err != nil {
		return UserPolicyState{}, err
	}
	return state, nil
}

func (c *Client) ResetUserQuota(ctx context.Context, username string) (QuotaReset, error) {
	if err := validateUsername(username); err != nil {
		return QuotaReset{}, err
	}
	var reset QuotaReset
	path := "/v1/users/" + url.PathEscape(username) + "/reset-quota"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, []int{http.StatusOK}, &reset); err != nil {
		return QuotaReset{}, err
	}
	if reset.Username != username {
		return QuotaReset{}, &APIError{Code: FailureInvalidOutput}
	}
	return reset, nil
}

func validateUserPolicyState(state UserPolicyState, username string) error {
	if state.Username != username {
		return &APIError{Code: FailureInvalidOutput}
	}
	if state.ExpirationRFC3339 != nil {
		if _, err := time.Parse(time.RFC3339Nano, *state.ExpirationRFC3339); err != nil {
			return &APIError{Code: FailureInvalidOutput}
		}
	}
	return nil
}
