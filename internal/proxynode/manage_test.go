package proxynode

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUpdateDeleteCanonicalNodeMetadata(t *testing.T) {
	db := nodeTestDB(t)
	ctx := context.Background()
	createdAt := time.Date(2030, 2, 3, 4, 5, 6, 0, time.UTC)
	created, err := Create(ctx, db, CreateNode{
		Type: TypeProxy, Name: "Germany-1", Region: "Germany", Host: "telemt", PublicHost: "proxy.example.com",
		MTProtoPort: 443, InternalAPIEndpoint: "http://telemt:9091", Enabled: true,
	}, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	updatedAt := createdAt.Add(time.Minute)
	updated, err := Update(ctx, db, created.ID, CreateNode{
		Type: Type(" RELAY "), Name: " Iran-Relay-1 ", Region: " Iran ", Host: "2001:DB8::10", PublicHost: "RELAY.EXAMPLE.COM.",
		MTProtoPort: 8443, InternalAPIEndpoint: "HTTPS://[2001:db8::10]:9443/api", Enabled: false,
	}, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Type != TypeRelay || updated.Name != "Iran-Relay-1" || updated.Region != "Iran" || updated.Host != "2001:db8::10" ||
		updated.PublicHost != "relay.example.com" || updated.InternalAPIEndpoint != "https://[2001:db8::10]:9443/api" || updated.Enabled {
		t.Fatalf("canonical updated node = %#v", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("timestamps after update = created %v updated %v", updated.CreatedAt, updated.UpdatedAt)
	}
	if err := Delete(ctx, db, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, db, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete error = %v, want not found", err)
	}
	if err := Delete(ctx, db, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete error = %v, want not found", err)
	}
}

func TestUpdateRejectsInvalidConflictAndUnknownWithoutMutation(t *testing.T) {
	db := nodeTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	first, err := Create(ctx, db, CreateNode{Type: TypeProxy, Name: "Germany-1", Region: "Germany", Host: "telemt", PublicHost: "proxy.example.com", MTProtoPort: 443, InternalAPIEndpoint: "http://telemt:9091", Enabled: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Create(ctx, db, CreateNode{Type: TypeRelay, Name: "Iran-1", Region: "Iran", Host: "relay", PublicHost: "relay.example.com", MTProtoPort: 8443, InternalAPIEndpoint: "http://relay:9091", Enabled: false}, now)
	if err != nil {
		t.Fatal(err)
	}

	invalid := CreateNode{Type: TypeProxy, Name: "Germany-2", Region: "Germany", Host: "bad host", PublicHost: "proxy2.example.com", MTProtoPort: 443, InternalAPIEndpoint: "http://telemt2:9091", Enabled: false}
	if _, err := Update(ctx, db, first.ID, invalid, now.Add(time.Minute)); err == nil {
		t.Fatal("invalid Update error = nil")
	}
	got, err := Get(ctx, db, first.ID)
	if err != nil || got.Name != first.Name || got.Enabled != first.Enabled {
		t.Fatalf("first after invalid update = %#v, %v", got, err)
	}

	conflict := CreateNode{Type: TypeProxy, Name: "iRaN-1", Region: "Germany", Host: "telemt2", PublicHost: "proxy2.example.com", MTProtoPort: 443, InternalAPIEndpoint: "http://telemt2:9091", Enabled: true}
	if _, err := Update(ctx, db, first.ID, conflict, now.Add(time.Minute)); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict Update error = %v, want conflict", err)
	}
	got, err = Get(ctx, db, first.ID)
	if err != nil || got.Name != first.Name {
		t.Fatalf("first after conflict = %#v, %v", got, err)
	}
	if _, err := Update(ctx, db, 999999, secondToInput(second), now.Add(time.Minute)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown Update error = %v, want not found", err)
	}
	if got := nodeCount(t, db); got != 2 {
		t.Fatalf("node count = %d, want 2", got)
	}
}

func secondToInput(node Node) CreateNode {
	return CreateNode{
		Type: node.Type, Name: node.Name, Region: node.Region, Host: node.Host, PublicHost: node.PublicHost,
		MTProtoPort: node.MTProtoPort, InternalAPIEndpoint: node.InternalAPIEndpoint, Enabled: node.Enabled,
	}
}
