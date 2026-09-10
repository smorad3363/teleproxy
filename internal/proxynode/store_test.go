package proxynode

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
)

func TestCreateGetListCanonicalizesStaticNodeMetadata(t *testing.T) {
	db := nodeTestDB(t)
	ctx := context.Background()
	now := time.Date(2030, 2, 3, 4, 5, 6, 0, time.UTC)
	first, err := Create(ctx, db, CreateNode{
		Type: Type(" PROXY "), Name: " Germany-1 ", Region: " Germany ", Host: "TELEMT.INTERNAL.", PublicHost: "203.0.113.10",
		MTProtoPort: 443, InternalAPIEndpoint: "HTTP://TELEMT.INTERNAL.:9091/api", Enabled: true,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Type != TypeProxy || first.Name != "Germany-1" || first.Region != "Germany" || first.Host != "telemt.internal" ||
		first.PublicHost != "203.0.113.10" || first.InternalAPIEndpoint != "http://telemt.internal:9091/api" || !first.Enabled {
		t.Fatalf("canonical first node = %#v", first)
	}
	second, err := Create(ctx, db, CreateNode{
		Type: TypeRelay, Name: "Iran-Relay-1", Region: "Iran", Host: "2001:db8::10", PublicHost: "relay.example.com",
		MTProtoPort: 8443, InternalAPIEndpoint: "https://[2001:db8::10]:9443", Enabled: false,
	}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if second.Host != "2001:db8::10" || second.InternalAPIEndpoint != "https://[2001:db8::10]:9443" {
		t.Fatalf("canonical relay = %#v", second)
	}

	listed, err := List(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].ID != first.ID || listed[1].ID != second.ID {
		t.Fatalf("node list = %#v", listed)
	}
	got, err := Get(ctx, db, second.ID)
	if err != nil || got.Name != second.Name {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	if _, err := Get(ctx, db, 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing error = %v", err)
	}
}

func TestCreateRejectsInvalidAndDuplicateNodeMetadataWithoutMutation(t *testing.T) {
	db := nodeTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	valid := CreateNode{Type: TypeProxy, Name: "Germany-1", Region: "Germany", Host: "telemt", PublicHost: "proxy.example.com", MTProtoPort: 443, InternalAPIEndpoint: "http://telemt:9091", Enabled: true}
	if _, err := Create(ctx, db, valid, now); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*CreateNode)
	}{
		{name: "type", mutate: func(v *CreateNode) { v.Type = "unknown" }},
		{name: "name", mutate: func(v *CreateNode) { v.Name = "   " }},
		{name: "region", mutate: func(v *CreateNode) { v.Region = strings.Repeat("x", 129) }},
		{name: "host scheme", mutate: func(v *CreateNode) { v.Host = "https://telemt" }},
		{name: "host whitespace", mutate: func(v *CreateNode) { v.PublicHost = "bad host" }},
		{name: "port zero", mutate: func(v *CreateNode) { v.MTProtoPort = 0 }},
		{name: "port high", mutate: func(v *CreateNode) { v.MTProtoPort = 65536 }},
		{name: "api credentials", mutate: func(v *CreateNode) { v.InternalAPIEndpoint = "http://user:pass@telemt:9091" }},
		{name: "api fragment", mutate: func(v *CreateNode) { v.InternalAPIEndpoint = "http://telemt:9091/#secret" }},
		{name: "api query", mutate: func(v *CreateNode) { v.InternalAPIEndpoint = "http://telemt:9091/?token=x" }},
		{name: "api scheme", mutate: func(v *CreateNode) { v.InternalAPIEndpoint = "ftp://telemt:9091" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			candidate.Name = "candidate-" + strings.ReplaceAll(test.name, " ", "-")
			test.mutate(&candidate)
			if _, err := Create(ctx, db, candidate, now); err == nil {
				t.Fatal("Create() error = nil, want error")
			}
			if got := nodeCount(t, db); got != 1 {
				t.Fatalf("node count = %d, want 1", got)
			}
		})
	}

	duplicate := valid
	duplicate.Name = "gErMaNy-1"
	duplicate.Host = "telemt-2"
	duplicate.PublicHost = "proxy2.example.com"
	duplicate.InternalAPIEndpoint = "http://telemt-2:9091"
	if _, err := Create(ctx, db, duplicate, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate Create() error = %v, want conflict", err)
	}
	if got := nodeCount(t, db); got != 1 {
		t.Fatalf("node count after conflict = %d, want 1", got)
	}
}

func TestProxyNodesSchemaContainsNoSecretColumns(t *testing.T) {
	db := nodeTestDB(t)
	rows, err := db.Query("PRAGMA table_info(proxy_nodes)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(name)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			t.Fatalf("secret-like Node column persisted: %q", name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func nodeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "node.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func nodeCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM proxy_nodes").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
