package telegrambot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/credit"
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
)

type fakeStartProvisioner struct {
	calls  int
	result proxyprovision.Result
	err    error
}

func (f *fakeStartProvisioner) Ensure(_ context.Context, username string) (proxyprovision.Result, error) {
	f.calls++
	if f.err != nil {
		return proxyprovision.Result{}, f.err
	}
	result := f.result
	if result.Username == "" {
		result.Username = username
	}
	return result, nil
}

func TestStartApplicationWithProvisionerReturnsLinkState(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2032, 3, 1, 0, 0, 0, 0, time.UTC)
	provisioner := &fakeStartProvisioner{result: proxyprovision.Result{
		Link:      "tg://proxy?server=proxy.example&port=443&secret=00112233445566778899aabbccddeeff",
		SyncState: proxyuser.SyncPending,
	}}
	app, err := NewStartApplicationWithProvisioner(db, "TeleProxyBot", func() time.Time { return now }, provisioner)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{UpdateID: 1, Message: &Message{
		From: &TelegramUser{ID: 9001}, Chat: Chat{ID: 9001, Type: "private"}, Text: "/start",
	}}
	response, handled, err := app.Handle(context.Background(), update)
	if err != nil || !handled {
		t.Fatalf("Handle() = %#v, handled=%v err=%v", response, handled, err)
	}
	if provisioner.calls != 1 || response.ProxyLink != provisioner.result.Link || response.ProxySyncState != string(proxyuser.SyncPending) {
		t.Fatalf("provisioned response = %#v calls=%d", response, provisioner.calls)
	}
}

func TestStartApplicationProvisionFailureKeepsIdempotentGift(t *testing.T) {
	db := startAppTestDB(t)
	now := time.Date(2032, 3, 2, 0, 0, 0, 0, time.UTC)
	provisioner := &fakeStartProvisioner{err: errors.New("temporary provisioning failure")}
	app, err := NewStartApplicationWithProvisioner(db, "", func() time.Time { return now }, provisioner)
	if err != nil {
		t.Fatal(err)
	}
	update := Update{Message: &Message{From: &TelegramUser{ID: 9002}, Chat: Chat{ID: 9002, Type: "private"}, Text: "/start"}}
	for i := 0; i < 2; i++ {
		if _, handled, err := app.Handle(context.Background(), update); err == nil || !handled {
			t.Fatalf("attempt %d handled=%v err=%v", i, handled, err)
		}
	}
	buckets, err := credit.List(context.Background(), db, "tg_9002", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 {
		t.Fatalf("provision retries created %d start gifts, want 1", len(buckets))
	}
}

func TestStartApplicationWithProvisionerRequiresProvisioner(t *testing.T) {
	db := startAppTestDB(t)
	if _, err := NewStartApplicationWithProvisioner(db, "", nil, nil); err == nil {
		t.Fatal("NewStartApplicationWithProvisioner() accepted nil provisioner")
	}
}
