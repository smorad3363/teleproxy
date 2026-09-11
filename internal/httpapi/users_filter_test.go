package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/telegramuser"
	"github.com/smorad3363/teleproxy/internal/useradmin"
)

func TestUserInventoryAPIExactAuthoritativeFilters(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	now := time.Now().UTC()

	first, err := telegramuser.Resolve(ctx, db, 9501, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	second, err := telegramuser.Resolve(ctx, db, 9502, now)
	if err != nil {
		t.Fatal(err)
	}

	assertSingle := func(path string, wantID int64) {
		t.Helper()
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, response.Code, response.Body.String())
		}
		var page struct {
			Users []useradmin.Entry `json:"users"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Users) != 1 || page.Users[0].TelegramUserID != wantID {
			t.Fatalf("GET %s users = %#v", path, page.Users)
		}
	}

	assertSingle("/api/users?telegram_id=9502", second.User.ID)
	assertSingle("/api/users?proxy_username="+url.QueryEscape(first.User.ProxyUsername), first.User.ID)
	assertSingle("/api/users?telegram_id=9501&proxy_username="+url.QueryEscape(first.User.ProxyUsername), first.User.ID)

	mismatch := performJSON(server.Handler(), http.MethodGet, "/api/users?telegram_id=9501&proxy_username="+url.QueryEscape(second.User.ProxyUsername), "", cookie, "")
	if mismatch.Code != http.StatusOK || !strings.Contains(mismatch.Body.String(), `"users":[]`) {
		t.Fatalf("combined mismatch = %d %s", mismatch.Code, mismatch.Body.String())
	}
	caseMismatch := performJSON(server.Handler(), http.MethodGet, "/api/users?proxy_username=TG_9501", "", cookie, "")
	if caseMismatch.Code != http.StatusOK || !strings.Contains(caseMismatch.Body.String(), `"users":[]`) {
		t.Fatalf("case mismatch = %d %s", caseMismatch.Code, caseMismatch.Body.String())
	}
}

func TestUserInventoryAPIRejectsInvalidExactFilters(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	for _, path := range []string{
		"/api/users?telegram_id=0",
		"/api/users?telegram_id=-1",
		"/api/users?telegram_id=nope",
		"/api/users?telegram_id=1&telegram_id=2",
		"/api/users?telegram_id=",
		"/api/users?proxy_username=",
		"/api/users?proxy_username=bad%20username",
		"/api/users?proxy_username=one&proxy_username=two",
	} {
		response := performJSON(server.Handler(), http.MethodGet, path, "", cookie, "")
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"USER_INVENTORY_INVALID"`) {
			t.Fatalf("invalid exact filter %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestUserPageExactFiltersAndPaginationURL(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	now := time.Now().UTC()
	first, err := telegramuser.Resolve(ctx, db, 9601, now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	second, err := telegramuser.Resolve(ctx, db, 9602, now)
	if err != nil {
		t.Fatal(err)
	}

	path := "/users?telegram_id=9602&proxy_username=" + url.QueryEscape(second.User.ProxyUsername) + "&limit=1"
	response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("filtered users page = %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`name="telegram_id"`,
		`value="9602"`,
		`name="proxy_username"`,
		`value="` + second.User.ProxyUsername + `"`,
		`type="hidden" name="limit" value="1"`,
		`Clear filters`,
		`data-user-filter-form`,
		`if (input.value === "") input.disabled = true;`,
		">9602<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("filtered users page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, ">9601<") || strings.Contains(body, first.User.ProxyUsername) {
		t.Fatalf("filtered users page included nonmatching user: %s", body)
	}

	empty := perform(server.Handler(), http.MethodGet, "/users?telegram_id=9601&proxy_username="+url.QueryEscape(second.User.ProxyUsername), nil, cookie)
	if empty.Code != http.StatusOK || !strings.Contains(empty.Body.String(), "No users match the current filters.") {
		t.Fatalf("filtered empty state = %d %s", empty.Code, empty.Body.String())
	}

	next := int64(44)
	got := userPageNextURL(useradmin.ListQuery{
		Limit:         7,
		TelegramID:    12345,
		ProxyUsername: "alpha_1",
	}, &next)
	want := "/users?before_id=44&limit=7&proxy_username=alpha_1&telegram_id=12345"
	if got != want {
		t.Fatalf("userPageNextURL() = %q, want %q", got, want)
	}
}

func TestUserPageRejectsInvalidExactFilters(t *testing.T) {
	db := testDB(t)
	server, cookie := authenticatedUserInventoryAPI(t, db)
	for _, path := range []string{
		"/users?telegram_id=0",
		"/users?telegram_id=" + strconv.FormatInt(-1, 10),
		"/users?proxy_username=bad%2Fname",
		"/users?proxy_username=a&proxy_username=b",
	} {
		response := perform(server.Handler(), http.MethodGet, path, nil, cookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"USER_INVENTORY_INVALID"`) {
			t.Fatalf("invalid users page exact filter %q = %d %s", path, response.Code, response.Body.String())
		}
	}
}
