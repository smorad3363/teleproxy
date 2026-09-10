package proxyprovision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

type fakeProvisionData struct {
	mu                sync.Mutex
	exists            bool
	secret            string
	createCalls       int
	createDelay       time.Duration
	createErrOnce     error
	postCreateGetErrs int
}

func (f *fakeProvisionData) CreateUserWithSecret(_ context.Context, username string, enabled bool, secret string) (telemt.Credential, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	if enabled {
		return telemt.Credential{}, fmt.Errorf("test create unexpectedly enabled user")
	}
	if f.createDelay > 0 {
		time.Sleep(f.createDelay)
	}
	f.exists = true
	f.secret = secret
	if f.createErrOnce != nil {
		err := f.createErrOnce
		f.createErrOnce = nil
		return telemt.Credential{}, err
	}
	return telemt.Credential{User: telemt.User{Username: username, Enabled: false, InRuntime: true}, Secret: secret}, nil
}

func (f *fakeProvisionData) GetUserLinks(_ context.Context, _ string) (telemt.UserLinks, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.exists {
		return telemt.UserLinks{}, &telemt.APIError{Code: telemt.FailureNotFound}
	}
	if f.postCreateGetErrs > 0 {
		f.postCreateGetErrs--
		return telemt.UserLinks{}, &telemt.APIError{Code: telemt.FailureUnavailable}
	}
	return telemt.UserLinks{Classic: []string{proxyLink(f.secret)}}, nil
}

func (f *fakeProvisionData) setRemote(secret string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exists = true
	f.secret = secret
}

func (f *fakeProvisionData) deleteRemote() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exists = false
	f.secret = ""
}

func (f *fakeProvisionData) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls
}

type fakeQuotaTrigger struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeQuotaTrigger) Trigger(string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.err
}

func (f *fakeQuotaTrigger) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestServiceFirstProvisionAndReplay(t *testing.T) {
	db := provisionServiceDB(t, "tg_101")
	data := &fakeProvisionData{}
	trigger := &fakeQuotaTrigger{}
	now := time.Unix(1000, 0).UTC()
	service, err := NewService(db, data, trigger, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.Ensure(context.Background(), "tg_101")
	if err != nil {
		t.Fatal(err)
	}
	if first.Username != "tg_101" || first.Link == "" || first.SyncState != proxyuser.SyncPending {
		t.Fatalf("first result = %#v", first)
	}
	if data.calls() != 1 || trigger.count() != 1 {
		t.Fatalf("calls create=%d trigger=%d", data.calls(), trigger.count())
	}
	state, err := Get(context.Background(), db, "tg_101")
	if err != nil || state.Phase != PhaseOwned {
		t.Fatalf("state = %#v, %v", state, err)
	}

	second, err := service.Ensure(context.Background(), "tg_101")
	if err != nil {
		t.Fatal(err)
	}
	if second.Link != first.Link || data.calls() != 1 || trigger.count() != 2 {
		t.Fatalf("replay result=%#v create=%d trigger=%d", second, data.calls(), trigger.count())
	}
}

func TestServiceRecoversAmbiguousCreateWithoutSecondCreate(t *testing.T) {
	db := provisionServiceDB(t, "tg_102")
	data := &fakeProvisionData{
		createErrOnce:     &telemt.APIError{Code: telemt.FailureUnavailable},
		postCreateGetErrs: 1,
	}
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Ensure(context.Background(), "tg_102"); err == nil {
		t.Fatal("first Ensure() error = nil, want ambiguous failure")
	}
	state, err := Get(context.Background(), db, "tg_102")
	if err != nil || state.Phase != PhasePrepared || state.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("prepared state = %#v, %v", state, err)
	}
	result, err := service.Ensure(context.Background(), "tg_102")
	if err != nil || result.Link == "" {
		t.Fatalf("recovered Ensure() = %#v, %v", result, err)
	}
	if data.calls() != 1 {
		t.Fatalf("create calls = %d, want 1", data.calls())
	}
	state, err = Get(context.Background(), db, "tg_102")
	if err != nil || state.Phase != PhaseOwned {
		t.Fatalf("owned state = %#v, %v", state, err)
	}
}

func TestServiceReplacesStalePreparedDigestOnlyAfterAbsence(t *testing.T) {
	db := provisionServiceDB(t, "tg_103")
	oldDigest, _ := SecretDigest("00112233445566778899aabbccddeeff")
	if _, _, err := Prepare(context.Background(), db, "tg_103", oldDigest, time.Unix(10, 0)); err != nil {
		t.Fatal(err)
	}
	data := &fakeProvisionData{}
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ensure(context.Background(), "tg_103"); err != nil {
		t.Fatal(err)
	}
	state, err := Get(context.Background(), db, "tg_103")
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseOwned || state.SecretSHA256 == oldDigest {
		t.Fatalf("state after stale restart = %#v", state)
	}
	data.mu.Lock()
	createdSecret := data.secret
	data.mu.Unlock()
	createdDigest, _ := SecretDigest(createdSecret)
	if state.SecretSHA256 != createdDigest {
		t.Fatal("persisted digest does not match replacement create attempt")
	}
}

func provisionServiceDB(t *testing.T, username string) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := proxyuser.Create(context.Background(), db, username, true); err != nil {
		t.Fatal(err)
	}
	return db
}

func proxyLink(secret string) string {
	return "tg://proxy?server=proxy.example&port=443&secret=" + secret
}

func containsSecret(link, secret string) bool {
	return len(secret) > 0 && len(link) >= len(secret) && link[len(link)-len(secret):] == secret
}
