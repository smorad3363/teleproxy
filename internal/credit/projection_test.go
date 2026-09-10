package credit

import (
	"math"
	"testing"
	"time"
)

func TestProjectSumsActiveRemainingAndChoosesBoundaries(t *testing.T) {
	now := time.Date(2030, 4, 1, 12, 0, 0, 0, time.UTC)
	early := now.Add(6 * time.Hour)
	late := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)
	futureStart := now.Add(time.Hour)
	projection, err := Project([]Bucket{
		{OriginalBytes: 100, ConsumedBytes: 40, StartsAt: now.Add(-time.Hour), ExpiresAt: &late, Status: StatusActive},
		{OriginalBytes: 200, ConsumedBytes: 50, StartsAt: now.Add(-time.Hour), ExpiresAt: &early, Status: StatusActive},
		{OriginalBytes: 300, ConsumedBytes: 0, StartsAt: now.Add(-time.Hour), Status: StatusActive},
		{OriginalBytes: 400, ConsumedBytes: 0, StartsAt: futureStart, Status: StatusPending},
		{OriginalBytes: 500, ConsumedBytes: 0, StartsAt: now.Add(-24 * time.Hour), ExpiresAt: &past, Status: StatusExpired},
		{OriginalBytes: 600, ConsumedBytes: 0, StartsAt: now.Add(-time.Hour), Status: StatusRevoked},
		{OriginalBytes: 700, ConsumedBytes: 700, StartsAt: now.Add(-time.Hour), Status: StatusExhausted},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if projection.AvailableBytes != 510 {
		t.Fatalf("AvailableBytes = %d, want 510", projection.AvailableBytes)
	}
	if projection.NextStart == nil || !projection.NextStart.Equal(futureStart) {
		t.Fatalf("NextStart = %v, want %v", projection.NextStart, futureStart)
	}
	if projection.NextExpiry == nil || !projection.NextExpiry.Equal(early) {
		t.Fatalf("NextExpiry = %v, want %v", projection.NextExpiry, early)
	}
}

func TestProjectNonExpiringOnlyHasNoBoundary(t *testing.T) {
	now := time.Date(2030, 4, 2, 0, 0, 0, 0, time.UTC)
	projection, err := Project([]Bucket{{OriginalBytes: 100, ConsumedBytes: 25, StartsAt: now.Add(-time.Hour), Status: StatusActive}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if projection.AvailableBytes != 75 || projection.NextStart != nil || projection.NextExpiry != nil {
		t.Fatalf("Project() = %#v", projection)
	}
}

func TestProjectFutureCreditReturnsActivationBoundary(t *testing.T) {
	now := time.Date(2030, 4, 3, 0, 0, 0, 0, time.UTC)
	starts := now.Add(2 * time.Hour)
	expires := starts.Add(24 * time.Hour)
	projection, err := Project([]Bucket{{OriginalBytes: 100, StartsAt: starts, ExpiresAt: &expires, Status: StatusPending}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if projection.AvailableBytes != 0 || projection.NextStart == nil || !projection.NextStart.Equal(starts) || projection.NextExpiry != nil {
		t.Fatalf("Project() = %#v", projection)
	}
}

func TestProjectNoEligibleCreditReturnsZero(t *testing.T) {
	now := time.Date(2030, 4, 4, 0, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	projection, err := Project([]Bucket{{OriginalBytes: 100, StartsAt: now.Add(-2 * time.Hour), ExpiresAt: &past, Status: StatusExpired}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if projection.AvailableBytes != 0 || projection.NextStart != nil || projection.NextExpiry != nil {
		t.Fatalf("Project() = %#v", projection)
	}
}

func TestProjectRejectsOverflowAndInvalidBuckets(t *testing.T) {
	now := time.Date(2030, 4, 5, 0, 0, 0, 0, time.UTC)
	if _, err := Project([]Bucket{
		{OriginalBytes: math.MaxInt64, StartsAt: now.Add(-time.Hour), Status: StatusActive},
		{OriginalBytes: 1, StartsAt: now.Add(-time.Hour), Status: StatusActive},
	}, now); err == nil {
		t.Fatal("Project() accepted int64 overflow")
	}
	if _, err := Project([]Bucket{{OriginalBytes: 10, ConsumedBytes: 11, StartsAt: now.Add(-time.Hour), Status: StatusActive}}, now); err == nil {
		t.Fatal("Project() accepted consumed bytes above original bytes")
	}
}
