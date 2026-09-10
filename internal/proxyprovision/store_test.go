package proxyprovision

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/proxyuser"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

func TestProvisioningStateCASAndOwnershipProof(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := proxyuser.Create(ctx, db, "tg_7", true); err != nil {
		t.Fatal(err)
	}

	secretA := "00112233445566778899aabbccddeeff"
	digestA, _ := SecretDigest(secretA)
	digestB, _ := SecretDigest("ffeeddccbbaa99887766554433221100")
	now := time.Unix(100, 0).UTC()
	state, created, err := Prepare(ctx, db, "tg_7", digestA, now)
	if err != nil || !created || state.Phase != PhasePrepared || state.SecretSHA256 != digestA {
		t.Fatalf("Prepare() = %#v, %v, created=%v", state, err, created)
	}

	var persisted []byte
	if err := db.QueryRowContext(ctx, `SELECT secret_sha256 FROM proxy_user_provisioning WHERE proxy_user_id = ?`, state.ProxyUserID).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 32 || string(persisted) == secretA {
		t.Fatalf("unexpected persisted provisioning secret representation: len=%d", len(persisted))
	}

	again, created, err := Prepare(ctx, db, "tg_7", digestB, now.Add(time.Second))
	if err != nil || created || again.SecretSHA256 != digestA {
		t.Fatalf("second Prepare() = %#v, %v, created=%v", again, err, created)
	}
	if _, err := ReplacePreparedDigest(ctx, db, "tg_7", digestB, digestA, now.Add(2*time.Second)); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("ReplacePreparedDigest wrong expected error = %v", err)
	}
	replaced, err := ReplacePreparedDigest(ctx, db, "tg_7", digestA, digestB, now.Add(3*time.Second))
	if err != nil || replaced.SecretSHA256 != digestB || replaced.LastErrorCode != "" {
		t.Fatalf("ReplacePreparedDigest() = %#v, %v", replaced, err)
	}
	withError, err := RecordError(ctx, db, "tg_7", digestB, "TELEMT_UNAVAILABLE", now.Add(4*time.Second))
	if err != nil || withError.LastErrorCode != "TELEMT_UNAVAILABLE" {
		t.Fatalf("RecordError() = %#v, %v", withError, err)
	}
	if _, err := RecordError(ctx, db, "tg_7", digestB, "bad error", now); err == nil {
		t.Fatal("RecordError() accepted unsafe error code")
	}

	links := telemt.UserLinks{Classic: []string{"tg://proxy?server=proxy.example&port=443&secret=ffeeddccbbaa99887766554433221100"}}
	proof, err := VerifyLinksDigest(links, digestB)
	if err != nil {
		t.Fatal(err)
	}
	owned, err := MarkOwned(ctx, db, "tg_7", proof, now.Add(5*time.Second))
	if err != nil || owned.Phase != PhaseOwned || owned.LastErrorCode != "" {
		t.Fatalf("MarkOwned() = %#v, %v", owned, err)
	}
	if _, err := MarkOwned(ctx, db, "tg_7", proof, now.Add(6*time.Second)); err != nil {
		t.Fatalf("idempotent MarkOwned() error = %v", err)
	}
	if _, err := ReplacePreparedDigest(ctx, db, "tg_7", digestB, digestA, now); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("replace after owned error = %v", err)
	}
}

func TestProvisioningCollisionAndCascade(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	user, err := proxyuser.Create(ctx, db, "tg_8", true)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := SecretDigest("00112233445566778899aabbccddeeff")
	now := time.Unix(200, 0).UTC()
	if _, _, err := Prepare(ctx, db, user.Username, digest, now); err != nil {
		t.Fatal(err)
	}
	collision, err := MarkCollision(ctx, db, user.Username, digest, now.Add(time.Second))
	if err != nil || collision.Phase != PhaseCollision {
		t.Fatalf("MarkCollision() = %#v, %v", collision, err)
	}
	if _, err := MarkOwned(ctx, db, user.Username, OwnershipProof{digest: digest}, now.Add(2*time.Second)); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("MarkOwned after collision error = %v", err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM proxy_users WHERE id = ?", user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, db, user.Username); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after cascade error = %v", err)
	}
}

func TestPrepareRequiresExistingProxyUser(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "teleproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	digest, _ := SecretDigest("00112233445566778899aabbccddeeff")
	if _, _, err := Prepare(ctx, db, "tg_9", digest, time.Now()); !errors.Is(err, proxyuser.ErrNotFound) {
		t.Fatalf("Prepare() error = %v", err)
	}
}
