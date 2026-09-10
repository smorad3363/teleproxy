package quotareconcile

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

type blockingOperation struct {
	mu        sync.Mutex
	active    int
	maxActive int
	calls     map[string]int
	perUser   map[string]int
	maxUser   map[string]int
	started   chan string
	release   <-chan struct{}
	err       error
}

func (f *blockingOperation) Reconcile(ctx context.Context, username string) error {
	f.mu.Lock()
	if f.calls == nil {
		f.calls = make(map[string]int)
		f.perUser = make(map[string]int)
		f.maxUser = make(map[string]int)
	}
	f.calls[username]++
	f.active++
	f.perUser[username]++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	if f.perUser[username] > f.maxUser[username] {
		f.maxUser[username] = f.perUser[username]
	}
	f.mu.Unlock()

	if f.started != nil {
		select {
		case f.started <- username:
		case <-ctx.Done():
		}
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			f.finish(username)
			return ctx.Err()
		}
	}
	f.finish(username)
	return f.err
}

func (f *blockingOperation) finish(username string) {
	f.mu.Lock()
	f.active--
	f.perUser[username]--
	f.mu.Unlock()
}

func (f *blockingOperation) snapshot() (int, map[string]int, map[string]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := make(map[string]int, len(f.calls))
	for key, value := range f.calls {
		calls[key] = value
	}
	maxUser := make(map[string]int, len(f.maxUser))
	for key, value := range f.maxUser {
		maxUser[key] = value
	}
	return f.maxActive, calls, maxUser
}

type scheduledCall struct {
	delay time.Duration
	fn    func()
	timer *fakeTimer
}

type fakeTimer struct {
	mu      sync.Mutex
	stopped bool
}

func (t *fakeTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = true
	return wasActive
}

func TestRunnerCoalescesPerUserAndBoundsConcurrency(t *testing.T) {
	db := runnerTestDB(t)
	now := time.Date(2030, 4, 1, 12, 0, 0, 0, time.UTC)
	for _, username := range []string{"alice", "bob", "carol"} {
		prepareRunnerUser(t, db, username, now)
	}

	release := make(chan struct{})
	operation := &blockingOperation{started: make(chan string, 16), release: release}
	runner, err := NewRunner(context.Background(), db, operation, RunnerOptions{Concurrency: 2, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer closeRunner(t, runner)

	if err := runner.Trigger("alice"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Trigger("bob"); err != nil {
		t.Fatal(err)
	}
	first := readStarted(t, operation.started)
	second := readStarted(t, operation.started)
	if first == second {
		t.Fatalf("concurrent starts = %q and %q", first, second)
	}
	select {
	case username := <-operation.started:
		t.Fatalf("third user started above concurrency bound: %s", username)
	case <-time.After(30 * time.Millisecond):
	}

	for i := 0; i < 8; i++ {
		if err := runner.Trigger("alice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := runner.Trigger("carol"); err != nil {
		t.Fatal(err)
	}
	close(release)

	waitFor(t, time.Second, func() bool {
		maxActive, calls, maxUser := operation.snapshot()
		return maxActive <= 2 && calls["alice"] == 2 && calls["bob"] == 1 && calls["carol"] == 1 && maxUser["alice"] == 1
	})
	maxActive, calls, maxUser := operation.snapshot()
	if maxActive != 2 {
		t.Fatalf("max active = %d, want 2; calls=%v", maxActive, calls)
	}
	if maxUser["alice"] != 1 {
		t.Fatalf("alice concurrent runs = %d", maxUser["alice"])
	}
}

func TestRunnerTriggerAllAndFailureUsesNarrowSyncCode(t *testing.T) {
	db := runnerTestDB(t)
	now := time.Date(2030, 4, 2, 12, 0, 0, 0, time.UTC)
	for _, username := range []string{"alice", "bob"} {
		prepareRunnerUser(t, db, username, now)
	}
	operation := &blockingOperation{err: errors.New("upstream body with secret-token")}
	runner, err := NewRunner(context.Background(), db, operation, RunnerOptions{Concurrency: 1, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer closeRunner(t, runner)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.TriggerAll(ctx); err != nil {
		t.Fatal(err)
	}
	waitFor(t, time.Second, func() bool {
		alice, aerr := proxyuser.Get(context.Background(), db, "alice")
		bob, berr := proxyuser.Get(context.Background(), db, "bob")
		return aerr == nil && berr == nil && alice.SyncState == proxyuser.SyncError && bob.SyncState == proxyuser.SyncError
	})
	for _, username := range []string{"alice", "bob"} {
		user, err := proxyuser.Get(context.Background(), db, username)
		if err != nil {
			t.Fatal(err)
		}
		if user.LastErrorCode != reconcileFailureCode {
			t.Fatalf("%s error code = %q", username, user.LastErrorCode)
		}
	}
}

func TestRunnerSchedulesNearestBoundaryAndRetriggers(t *testing.T) {
	db := runnerTestDB(t)
	now := time.Date(2030, 4, 3, 12, 0, 0, 0, time.UTC)
	createRunnerUser(t, db, "alice")
	start := now.Add(10 * time.Minute)
	expires := now.Add(20 * time.Minute)
	if _, err := credit.GrantBucket(context.Background(), db, "alice", credit.Grant{OriginalBytes: 100, StartsAt: start, ExpiresAt: &expires, RewardType: "scheduled", Source: "boundary"}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := credit.PrepareProjection(context.Background(), db, "alice", credit.PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	activateRunnerSnapshot(t, db, snapshot)

	operation := &blockingOperation{}
	scheduled := make(chan scheduledCall, 4)
	runner, err := NewRunner(context.Background(), db, operation, RunnerOptions{
		Concurrency: 1,
		Now:         func() time.Time { return now },
		AfterFunc: func(delay time.Duration, fn func()) Timer {
			timer := &fakeTimer{}
			scheduled <- scheduledCall{delay: delay, fn: fn, timer: timer}
			return timer
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer closeRunner(t, runner)

	if err := runner.Trigger("alice"); err != nil {
		t.Fatal(err)
	}
	var call scheduledCall
	select {
	case call = <-scheduled:
	case <-time.After(time.Second):
		t.Fatal("boundary timer was not scheduled")
	}
	if call.delay != 10*time.Minute {
		t.Fatalf("boundary delay = %v, want 10m", call.delay)
	}
	call.fn()
	waitFor(t, time.Second, func() bool {
		_, calls, _ := operation.snapshot()
		return calls["alice"] >= 2
	})
}

func TestRunnerShutdownCancelsInFlightWorkAndRejectsNewTriggers(t *testing.T) {
	db := runnerTestDB(t)
	now := time.Date(2030, 4, 4, 12, 0, 0, 0, time.UTC)
	prepareRunnerUser(t, db, "alice", now)
	never := make(chan struct{})
	operation := &blockingOperation{started: make(chan string, 1), release: never}
	runner, err := NewRunner(context.Background(), db, operation, RunnerOptions{Concurrency: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Trigger("alice"); err != nil {
		t.Fatal(err)
	}
	_ = readStarted(t, operation.started)

	runner.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := runner.Trigger("alice"); !errors.Is(err, ErrRunnerClosed) {
		t.Fatalf("Trigger after shutdown error = %v", err)
	}
}

func TestNearestBoundaryChoosesExpiryBeforeFutureStart(t *testing.T) {
	now := time.Date(2030, 4, 5, 12, 0, 0, 0, time.UTC)
	expiry := now.Add(time.Minute)
	start := now.Add(2 * time.Minute)
	got := nearestBoundary(credit.ReconciliationSnapshot{EnforcedExpiry: &expiry, NextStart: &start})
	if got == nil || !got.Equal(expiry) {
		t.Fatalf("nearest boundary = %v, want %v", got, expiry)
	}
}

func runnerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "runner.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createRunnerUser(t *testing.T, db *sql.DB, username string) {
	t.Helper()
	if _, err := proxyuser.Create(context.Background(), db, username, true); err != nil {
		t.Fatal(err)
	}
	if _, err := proxyuser.MarkSynced(context.Background(), db, username); err != nil {
		t.Fatal(err)
	}
}

func prepareRunnerUser(t *testing.T, db *sql.DB, username string, now time.Time) {
	t.Helper()
	createRunnerUser(t, db, username)
	snapshot, err := credit.PrepareProjection(context.Background(), db, username, credit.PrepareProjectionInput{ProjectedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	activateRunnerSnapshot(t, db, snapshot)
}

func activateRunnerSnapshot(t *testing.T, db *sql.DB, snapshot credit.ReconciliationSnapshot) {
	t.Helper()
	if err := credit.TransitionReconciliationPhase(context.Background(), db, snapshot.Username, snapshot.Generation, credit.PhaseApplying, credit.PhaseActive); err != nil {
		t.Fatal(err)
	}
}

func readStarted(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case username := <-ch:
		return username
	case <-time.After(time.Second):
		t.Fatal("reconciliation did not start")
		return ""
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

func closeRunner(t *testing.T, runner *Runner) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Close(ctx); err != nil {
		t.Fatalf("close runner: %v", err)
	}
}
