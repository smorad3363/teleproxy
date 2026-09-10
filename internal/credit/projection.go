package credit

import (
	"fmt"
	"math"
	"time"
)

type Projection struct {
	AvailableBytes int64      `json:"available_bytes"`
	NextExpiry     *time.Time `json:"next_expiry,omitempty"`
}

func Project(buckets []Bucket, now time.Time) (Projection, error) {
	now = now.UTC()
	var projection Projection
	for _, bucket := range buckets {
		storedStatus := "active"
		if bucket.Status == StatusRevoked {
			storedStatus = "revoked"
		}
		if effectiveStatus(storedStatus, bucket.ConsumedBytes, bucket.OriginalBytes, bucket.StartsAt, bucket.ExpiresAt, now) != StatusActive {
			continue
		}
		remaining := bucket.OriginalBytes - bucket.ConsumedBytes
		if remaining <= 0 {
			continue
		}
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
