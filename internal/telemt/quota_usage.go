package telemt

import (
	"context"
	"net/http"
)

type QuotaUsage struct {
	Username           string `json:"username"`
	DataQuotaBytes     uint64 `json:"data_quota_bytes"`
	UsedBytes          uint64 `json:"used_bytes"`
	LastResetEpochSecs uint64 `json:"last_reset_epoch_secs"`
}

type quotaUsageList struct {
	Users []QuotaUsage `json:"users"`
}

func (c *Client) ListQuotaUsage(ctx context.Context) ([]QuotaUsage, error) {
	var data quotaUsageList
	if err := c.doJSON(ctx, http.MethodGet, "/v1/stats/users/quota", nil, []int{http.StatusOK}, &data); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(data.Users))
	for _, usage := range data.Users {
		if validateUsername(usage.Username) != nil || usage.DataQuotaBytes == 0 {
			return nil, &APIError{Code: FailureInvalidOutput}
		}
		if _, exists := seen[usage.Username]; exists {
			return nil, &APIError{Code: FailureInvalidOutput}
		}
		seen[usage.Username] = struct{}{}
	}
	return data.Users, nil
}

func (c *Client) GetUserQuotaUsage(ctx context.Context, username string) (QuotaUsage, bool, error) {
	if err := validateUsername(username); err != nil {
		return QuotaUsage{}, false, err
	}
	users, err := c.ListQuotaUsage(ctx)
	if err != nil {
		return QuotaUsage{}, false, err
	}
	for _, usage := range users {
		if usage.Username == username {
			return usage, true, nil
		}
	}
	return QuotaUsage{}, false, nil
}
