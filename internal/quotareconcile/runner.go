package quotareconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

var ErrRunnerClosed = errors.New("quota reconciliation runner is closed")

const reconcileFailureCode = "QUOTA_RECONCILE_FAILED"

type reconcileOperation interface {
	Reconcile(context.Context, string) error
}

type Timer interface {
	Stop() bool
}

type RunnerOptions struct {
	Concurrency      int
	ReconcileTimeout time.Duration
	Now              func() time.Time
	AfterFunc        func(time.Duration, func()) Timer
}

type Runner struct {
	db               *sql.DB
	reconciler       reconcileOperation
	ctx              context.Context
	cancel           context.CancelFunc
	reconcileTimeout time.Duration
	now              func() time.Time
	afterFunc        func(time.Duration, func()) Timer
	triggerCh        chan triggerRequest
	resultCh         chan runResult
	workCh           chan string
	wg               sync.WaitGroup
}

type triggerRequest struct {
	username string
	ack      chan error
}

type runResult struct {
	username string
	boundary *time.Time
	err      error
}

type runnerUserState struct {
	queued  bool
	running bool
	pending bool
	timer   Timer
}

func NewRunner(parent context.Context, db *sql.DB, reconciler reconcileOperation, options RunnerOptions) (*Runner, error) {
	if db == nil {
		return nil, fmt.Errorf("quota reconciliation runner database is required")
	}
	if reconciler == nil {
		return nil, fmt.Errorf("quota reconciliation operation is required")
	}
	if parent == nil {
		parent = context.Background()
	}
	if options.Concurrency == 0 {
		options.Concurrency = 2
	}
	if options.Concurrency < 1 || options.Concurrency > 32 {
		return nil, fmt.Errorf("quota reconciliation concurrency must be between 1 and 32")
	}
	if options.ReconcileTimeout == 0 {
		options.ReconcileTimeout = 30 * time.Second
	}
	if options.ReconcileTimeout < 0 {
		return nil, fmt.Errorf("quota reconciliation timeout must not be negative")
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	if options.AfterFunc == nil {
		options.AfterFunc = func(delay time.Duration, fn func()) Timer { return time.AfterFunc(delay, fn) }
	}

	ctx, cancel := context.WithCancel(parent)
	r := &Runner{
		db:               db,
		reconciler:       reconciler,
		ctx:              ctx,
		cancel:           cancel,
		reconcileTimeout: options.ReconcileTimeout,
		now:              options.Now,
		afterFunc:        options.AfterFunc,
		triggerCh:        make(chan triggerRequest),
		resultCh:         make(chan runResult, options.Concurrency),
		workCh:           make(chan string, options.Concurrency),
	}

	r.wg.Add(1)
	go r.scheduler(options.Concurrency)
	for i := 0; i < options.Concurrency; i++ {
		r.wg.Add(1)
		go r.worker()
	}
	return r, nil
}

func (r *Runner) Trigger(username string) error {
	username = strings.TrimSpace(username)
	if err := proxyuser.ValidateUsername(username); err != nil {
		return err
	}
	ack := make(chan error, 1)
	request := triggerRequest{username: username, ack: ack}
	select {
	case <-r.ctx.Done():
		return ErrRunnerClosed
	case r.triggerCh <- request:
	}
	select {
	case <-r.ctx.Done():
		return ErrRunnerClosed
	case err := <-ack:
		return err
	}
}

func (r *Runner) TriggerAll(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	users, err := proxyuser.List(ctx, r.db)
	if err != nil {
		return err
	}
	for _, user := range users {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := r.Trigger(user.Username); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) Stop() {
	r.cancel()
}

func (r *Runner) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (r *Runner) Close(ctx context.Context) error {
	r.Stop()
	return r.Wait(ctx)
}

func (r *Runner) scheduler(concurrency int) {
	defer r.wg.Done()
	states := make(map[string]*runnerUserState)
	queue := make([]string, 0)
	active := 0

	dispatch := func() bool {
		for active < concurrency && len(queue) > 0 {
			username := queue[0]
			queue = queue[1:]
			state := states[username]
			state.queued = false
			state.running = true
			active++
			select {
			case <-r.ctx.Done():
				return false
			case r.workCh <- username:
			}
		}
		return true
	}

	for {
		if !dispatch() {
			for _, state := range states {
				if state.timer != nil {
					state.timer.Stop()
				}
			}
			close(r.workCh)
			return
		}

		select {
		case <-r.ctx.Done():
			for _, state := range states {
				if state.timer != nil {
					state.timer.Stop()
				}
			}
			close(r.workCh)
			return

		case request := <-r.triggerCh:
			state := states[request.username]
			if state == nil {
				state = &runnerUserState{}
				states[request.username] = state
			}
			if state.timer != nil {
				state.timer.Stop()
				state.timer = nil
			}
			switch {
			case state.running:
				state.pending = true
			case state.queued:
				// Already coalesced into the pending queue.
			default:
				state.queued = true
				queue = append(queue, request.username)
			}
			request.ack <- nil

		case result := <-r.resultCh:
			if active > 0 {
				active--
			}
			state := states[result.username]
			if state == nil {
				continue
			}
			state.running = false
			if state.pending {
				state.pending = false
				state.queued = true
				queue = append(queue, result.username)
				continue
			}
			if result.err == nil && result.boundary != nil && r.ctx.Err() == nil {
				delay := result.boundary.Sub(r.now().UTC())
				if delay < 0 {
					delay = 0
				}
				username := result.username
				state.timer = r.afterFunc(delay, func() {
					_ = r.Trigger(username)
				})
			}
		}
	}
}

func (r *Runner) worker() {
	defer r.wg.Done()
	for username := range r.workCh {
		result := r.runOne(username)
		select {
		case r.resultCh <- result:
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *Runner) runOne(username string) runResult {
	ctx, cancel := context.WithTimeout(r.ctx, r.reconcileTimeout)
	defer cancel()

	if err := r.reconciler.Reconcile(ctx, username); err != nil {
		r.markFailure(username)
		return runResult{username: username, err: err}
	}

	snapshot, err := credit.LoadReconciliation(ctx, r.db, username)
	if err != nil {
		r.markFailure(username)
		return runResult{username: username, err: err}
	}
	if snapshot.Phase != credit.PhaseActive {
		err := fmt.Errorf("quota reconciliation returned in phase %q", snapshot.Phase)
		r.markFailure(username)
		return runResult{username: username, err: err}
	}
	if _, err := proxyuser.MarkSynced(ctx, r.db, username); err != nil {
		return runResult{username: username, err: err}
	}
	boundary := nearestBoundary(snapshot)
	return runResult{username: username, boundary: boundary}
}

func (r *Runner) markFailure(username string) {
	if r.ctx.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()
	_, _ = proxyuser.MarkSyncError(ctx, r.db, username, reconcileFailureCode)
}

func nearestBoundary(snapshot credit.ReconciliationSnapshot) *time.Time {
	var boundary *time.Time
	for _, candidate := range []*time.Time{snapshot.EnforcedExpiry, snapshot.NextStart} {
		if candidate == nil {
			continue
		}
		value := candidate.UTC()
		if boundary == nil || value.Before(*boundary) {
			copy := value
			boundary = &copy
		}
	}
	return boundary
}
