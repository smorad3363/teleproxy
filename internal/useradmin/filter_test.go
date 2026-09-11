package useradmin

import (
	"context"
	"testing"
	"time"
)

func TestListExactAuthoritativeFilters(t *testing.T) {
	ctx := context.Background()
	db := userAdminTestDB(t)
	now := time.Date(2033, 4, 5, 6, 7, 8, 0, time.UTC)

	first := resolveUser(t, ctx, db, 8301, now.Add(-3*time.Minute))
	second := resolveUser(t, ctx, db, 8302, now.Add(-2*time.Minute))
	resolveUser(t, ctx, db, 8303, now.Add(-time.Minute))

	byTelegram, err := List(ctx, db, ListQuery{TelegramID: second.User.TelegramID}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(byTelegram.Items) != 1 || byTelegram.Items[0].TelegramUserID != second.User.ID {
		t.Fatalf("Telegram ID filter = %#v", byTelegram)
	}

	byProxy, err := List(ctx, db, ListQuery{ProxyUsername: first.User.ProxyUsername}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(byProxy.Items) != 1 || byProxy.Items[0].TelegramUserID != first.User.ID {
		t.Fatalf("proxy username filter = %#v", byProxy)
	}

	combined, err := List(ctx, db, ListQuery{TelegramID: first.User.TelegramID, ProxyUsername: first.User.ProxyUsername}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(combined.Items) != 1 || combined.Items[0].TelegramUserID != first.User.ID {
		t.Fatalf("combined filter = %#v", combined)
	}

	mismatch, err := List(ctx, db, ListQuery{TelegramID: first.User.TelegramID, ProxyUsername: second.User.ProxyUsername}, now)
	if err != nil {
		t.Fatal(err)
	}
	if mismatch.Items == nil || len(mismatch.Items) != 0 || mismatch.NextBeforeID != nil {
		t.Fatalf("mismatched combined filter = %#v", mismatch)
	}

	caseMismatch, err := List(ctx, db, ListQuery{ProxyUsername: "TG_8301"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(caseMismatch.Items) != 0 {
		t.Fatalf("case-folded proxy username unexpectedly matched = %#v", caseMismatch)
	}
}

func TestListRejectsInvalidExactFilters(t *testing.T) {
	db := userAdminTestDB(t)
	now := time.Now().UTC()
	for _, query := range []ListQuery{
		{TelegramID: -1},
		{ProxyUsername: "bad username"},
		{ProxyUsername: "slash/name"},
	} {
		if _, err := List(context.Background(), db, query, now); err == nil {
			t.Fatalf("List(%#v) error = nil", query)
		}
	}
}
