package proxyprovision

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceRejectsPreexistingSameNameCollision(t *testing.T) {
	db := provisionServiceDB(t, "tg_104")
	data := &fakeProvisionData{}
	data.setRemote("ffeeddccbbaa99887766554433221100")
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Ensure(context.Background(), "tg_104")
	if !errors.Is(err, ErrProvisioningCollision) {
		t.Fatalf("Ensure() error = %v, want collision", err)
	}
	state, getErr := Get(context.Background(), db, "tg_104")
	if getErr != nil || state.Phase != PhaseCollision {
		t.Fatalf("collision state = %#v, %v", state, getErr)
	}
	if data.calls() != 0 || trigger.count() != 0 {
		t.Fatalf("collision performed unsafe work create=%d trigger=%d", data.calls(), trigger.count())
	}
}

func TestServiceRunnerFailureLeavesOwnedAndReplayable(t *testing.T) {
	db := provisionServiceDB(t, "tg_105")
	data := &fakeProvisionData{}
	trigger := &fakeQuotaTrigger{err: errors.New("runner closed")}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ensure(context.Background(), "tg_105"); err == nil {
		t.Fatal("Ensure() error = nil, want trigger failure")
	}
	state, err := Get(context.Background(), db, "tg_105")
	if err != nil || state.Phase != PhaseOwned {
		t.Fatalf("state after trigger failure = %#v, %v", state, err)
	}
	trigger.mu.Lock()
	trigger.err = nil
	trigger.mu.Unlock()
	if _, err := service.Ensure(context.Background(), "tg_105"); err != nil {
		t.Fatalf("replay Ensure() error = %v", err)
	}
	if data.calls() != 1 {
		t.Fatalf("create calls = %d, want 1", data.calls())
	}
}

func TestServiceOwnedStateSurvivesIntentionalSecretRotation(t *testing.T) {
	db := provisionServiceDB(t, "tg_106")
	data := &fakeProvisionData{}
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Ensure(context.Background(), "tg_106")
	if err != nil {
		t.Fatal(err)
	}
	data.setRemote("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	second, err := service.Ensure(context.Background(), "tg_106")
	if err != nil {
		t.Fatal(err)
	}
	if first.Link == second.Link || !containsSecret(second.Link, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatalf("rotated link not returned: first=%q second=%q", first.Link, second.Link)
	}
	if data.calls() != 1 {
		t.Fatalf("secret rotation caused recreate: calls=%d", data.calls())
	}
}

func TestServiceReprovisionsOwnedUserAfterDefinitiveDeletion(t *testing.T) {
	db := provisionServiceDB(t, "tg_107")
	data := &fakeProvisionData{}
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Ensure(context.Background(), "tg_107"); err != nil {
		t.Fatal(err)
	}
	before, err := Get(context.Background(), db, "tg_107")
	if err != nil {
		t.Fatal(err)
	}
	data.deleteRemote()
	if _, err := service.Ensure(context.Background(), "tg_107"); err != nil {
		t.Fatal(err)
	}
	after, err := Get(context.Background(), db, "tg_107")
	if err != nil {
		t.Fatal(err)
	}
	if after.Phase != PhaseOwned || after.SecretSHA256 == before.SecretSHA256 || data.calls() != 2 {
		t.Fatalf("reprovision result before=%#v after=%#v calls=%d", before, after, data.calls())
	}
}

func TestServiceSerializesConcurrentSameUserProvisioning(t *testing.T) {
	db := provisionServiceDB(t, "tg_108")
	data := &fakeProvisionData{createDelay: 20 * time.Millisecond}
	trigger := &fakeQuotaTrigger{}
	service, err := NewService(db, data, trigger, nil)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.Ensure(context.Background(), "tg_108")
			if err == nil && result.Link == "" {
				err = errors.New("empty link")
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Ensure() error = %v", err)
		}
	}
	if data.calls() != 1 {
		t.Fatalf("create calls = %d, want 1", data.calls())
	}
}
