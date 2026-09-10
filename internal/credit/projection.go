package credit

import (
	"fmt"
	"math"
	"time"
)

type Projection struct {
	AvailableBytes int64      `json:"available_bytes"`
	NextStart      *time.Time `json:"next_start,omitempty"`
	NextExpiry     *time.Time `json:"next_expiry,omitempty"`
}

func Project(buckets []Bucket, now time.Time) (Projection, error) {
	now = now.UTC()
	var projection Projection
	for _, bucket := range buckets {
		if bucket.OriginalBytes <= 0 || bucket.ConsumedBytes < 0 || bucket.ConsumedBytes > bucket.OriginalBytes || bucket.StartsAt.IsZero() {
			return Projection{}, fmt.Errorf("credit bucket is invalid for projection")
		}
		if bucket.ExpiresAt != nil && !bucket.ExpiresAt.After(bucket.StartsAt) {
			return Projection{}, fmt.Errorf("credit bucket expiry is invalid for projection")
		}
		storedStatus := "active"
		if bucket.Status == StatusRevoked {
			storedStatus = "revoked"
		}
		status := effectiveStatus(storedStatus, bucket.ConsumedBytes, bucket.OriginalBytes, bucket.StartsAt, bucket.ExpiresAt, now)
		switch status {
		case StatusPending:
			if projection.NextStart == nil || bucket.StartsAt.Before(*projection.NextStart) {
				starts := bucket.StartsAt.UTC()
				projection.NextStart = &starts
			}
			continue
		case StatusActive:
		case StatusExpired, StatusExhausted, StatusRevoked:
			continue
		default:
			return Projection{}, fmt.Errorf("credit bucket status is invalid for projection")
		}

		remaining := bucket.OriginalBytes - bucket.ConsumedBytes
		if projection.AvailableBytes > math.MaxInt64-remaining {
			return Projection{}, fmt.Errorf("credit projection exceeds supported byte range")
		}
		projection.AvailableBytes += remaining
		if bucket.ExpiresAt != nil && (projection.NextExpiry == nil || bucket.ExpiresAt.Before(*projection.NextExpiry)) {
			expires := bucket.ExpiresAt.UTC()
			projection.NextExpiry = &expires
		}
	}
	return projection, nil
}
