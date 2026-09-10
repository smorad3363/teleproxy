package proxyprovision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

var ErrProvisioningCollision = errors.New("Telemt username belongs to another provisioning identity")

type dataPlane interface {
	CreateUserWithSecret(context.Context, string, bool, string) (telemt.Credential, error)
	GetUserLinks(context.Context, string) (telemt.UserLinks, error)
}

type quotaTrigger interface {
	Trigger(string) error
}

type Result struct {
	Username  string
	Link      string
	SyncState proxyuser.SyncState
}

type Service struct {
	db      *sql.DB
	data    dataPlane
	trigger quotaTrigger
	now     func() time.Time
	locksMu sync.Mutex
	locks   map[string]*sync.Mutex
}

func NewService(db *sql.DB, data dataPlane, trigger quotaTrigger, now func() time.Time) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("proxy provisioning database is required")
	}
	if data == nil {
		return nil, fmt.Errorf("Telemt provisioning client is required")
	}
	if trigger == nil {
		return nil, fmt.Errorf("quota reconciliation trigger is required")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{db: db, data: data, trigger: trigger, now: now, locks: make(map[string]*sync.Mutex)}, nil
}

func (s *Service) Ensure(ctx context.Context, username string) (Result, error) {
	if s == nil || s.db == nil || s.data == nil || s.trigger == nil {
		return Result{}, fmt.Errorf("proxy provisioning service is not configured")
	}
	if err := proxyuser.ValidateUsername(username); err != nil {
		return Result{}, err
	}
	unlock := s.lock(username)
	defer unlock()

	if _, err := proxyuser.Get(ctx, s.db, username); err != nil {
		return Result{}, err
	}
	if _, err := s.ensureOwned(ctx, username); err != nil {
		return Result{}, err
	}
	if err := s.trigger.Trigger(username); err != nil {
		return Result{}, fmt.Errorf("queue quota reconciliation: %w", err)
	}

	// Fetch only after the quota reconciliation has been queued. The earlier
	// reads are ownership/existence checks and are never returned to the user.
	links, err := s.data.GetUserLinks(ctx, username)
	if err != nil {
		return Result{}, fmt.Errorf("read provisioned proxy links: %w", err)
	}
	link, err := primaryLink(links)
	if err != nil {
		return Result{}, err
	}
	user, err := proxyuser.Get(ctx, s.db, username)
	if err != nil {
		return Result{}, err
	}
	return Result{Username: username, Link: link, SyncState: user.SyncState}, nil
}

func (s *Service) ensureOwned(ctx context.Context, username string) (telemt.UserLinks, error) {
	state, err := Get(ctx, s.db, username)
	var attemptSecret string
	if errors.Is(err, ErrNotFound) {
		secret, digest, generateErr := GenerateSecret()
		if generateErr != nil {
			return telemt.UserLinks{}, generateErr
		}
		var created bool
		state, created, err = Prepare(ctx, s.db, username, digest, s.now())
		if err != nil {
			return telemt.UserLinks{}, err
		}
		if created && state.SecretSHA256 == digest {
			attemptSecret = secret
		}
	} else if err != nil {
		return telemt.UserLinks{}, err
	}

	switch state.Phase {
	case PhaseCollision:
		return telemt.UserLinks{}, ErrProvisioningCollision
	case PhaseOwned:
		links, getErr := s.data.GetUserLinks(ctx, username)
		if getErr == nil {
			return links, nil
		}
		if telemt.FailureCodeOf(getErr) != telemt.FailureNotFound {
			return telemt.UserLinks{}, fmt.Errorf("check owned Telemt user: %w", getErr)
		}
		secret, digest, generateErr := GenerateSecret()
		if generateErr != nil {
			return telemt.UserLinks{}, generateErr
		}
		state, err = RestartOwned(ctx, s.db, username, state.SecretSHA256, digest, s.now())
		if err != nil {
			return telemt.UserLinks{}, err
		}
		attemptSecret = secret
	case PhasePrepared:
	default:
		return telemt.UserLinks{}, fmt.Errorf("proxy provisioning phase is invalid")
	}

	if state.Phase != PhasePrepared {
		return telemt.UserLinks{}, fmt.Errorf("proxy provisioning state is invalid")
	}

	links, getErr := s.data.GetUserLinks(ctx, username)
	if getErr == nil {
		return s.provePrepared(ctx, username, state, links)
	}
	if telemt.FailureCodeOf(getErr) != telemt.FailureNotFound {
		s.recordError(ctx, username, state.SecretSHA256, getErr)
		return telemt.UserLinks{}, fmt.Errorf("check prepared Telemt user: %w", getErr)
	}

	if attemptSecret == "" {
		secret, digest, generateErr := GenerateSecret()
		if generateErr != nil {
			return telemt.UserLinks{}, generateErr
		}
		state, err = ReplacePreparedDigest(ctx, s.db, username, state.SecretSHA256, digest, s.now())
		if err != nil {
			return telemt.UserLinks{}, err
		}
		attemptSecret = secret
	}

	_, createErr := s.data.CreateUserWithSecret(ctx, username, false, attemptSecret)
	attemptSecret = ""
	if createErr != nil {
		// A timeout can occur after Telemt committed the create. Probe before
		// returning so ownership can be established without a second create.
		links, verifyErr := s.data.GetUserLinks(ctx, username)
		if verifyErr == nil {
			return s.provePrepared(ctx, username, state, links)
		}
		s.recordError(ctx, username, state.SecretSHA256, createErr)
		return telemt.UserLinks{}, fmt.Errorf("create disabled Telemt user: %w", createErr)
	}

	links, err = s.data.GetUserLinks(ctx, username)
	if err != nil {
		s.recordError(ctx, username, state.SecretSHA256, err)
		return telemt.UserLinks{}, fmt.Errorf("verify created Telemt user: %w", err)
	}
	return s.provePrepared(ctx, username, state, links)
}

func (s *Service) provePrepared(ctx context.Context, username string, state State, links telemt.UserLinks) (telemt.UserLinks, error) {
	proof, err := VerifyLinksDigest(links, state.SecretSHA256)
	if errors.Is(err, ErrOwnershipMismatch) {
		if _, markErr := MarkCollision(ctx, s.db, username, state.SecretSHA256, s.now()); markErr != nil {
			return telemt.UserLinks{}, markErr
		}
		return telemt.UserLinks{}, ErrProvisioningCollision
	}
	if err != nil {
		s.recordErrorCode(ctx, username, state.SecretSHA256, "TELEMT_INVALID_RESPONSE")
		return telemt.UserLinks{}, fmt.Errorf("verify Telemt provisioning ownership: %w", err)
	}
	if _, err := MarkOwned(ctx, s.db, username, proof, s.now()); err != nil {
		return telemt.UserLinks{}, err
	}
	return links, nil
}

func (s *Service) recordError(ctx context.Context, username string, digest [32]byte, err error) {
	code := "TELEMT_UNAVAILABLE"
	switch telemt.FailureCodeOf(err) {
	case telemt.FailureUnauthorized:
		code = "TELEMT_UNAUTHORIZED"
	case telemt.FailureForbidden:
		code = "TELEMT_FORBIDDEN"
	case telemt.FailureNotFound:
		code = "TELEMT_NOT_FOUND"
	case telemt.FailureConflict:
		code = "TELEMT_CONFLICT"
	case telemt.FailureRejected:
		code = "TELEMT_REJECTED"
	case telemt.FailureInvalidOutput:
		code = "TELEMT_INVALID_RESPONSE"
	case telemt.FailureUnavailable:
		code = "TELEMT_UNAVAILABLE"
	}
	s.recordErrorCode(ctx, username, digest, code)
}

func (s *Service) recordErrorCode(ctx context.Context, username string, digest [32]byte, code string) {
	_, _ = RecordError(ctx, s.db, username, digest, code, s.now())
}

func primaryLink(links telemt.UserLinks) (string, error) {
	all := links.All()
	if len(all) == 0 {
		return "", fmt.Errorf("Telemt user has no proxy link")
	}
	best := ""
	for _, link := range all {
		if link == "" || len(link) > 3500 {
			continue
		}
		if best == "" || len(link) < len(best) {
			best = link
		}
	}
	if best == "" {
		return "", fmt.Errorf("Telemt proxy link is too large for Telegram response")
	}
	return best, nil
}

func (s *Service) lock(username string) func() {
	s.locksMu.Lock()
	mutex := s.locks[username]
	if mutex == nil {
		mutex = &sync.Mutex{}
		s.locks[username] = mutex
	}
	s.locksMu.Unlock()
	mutex.Lock()
	return mutex.Unlock
}
