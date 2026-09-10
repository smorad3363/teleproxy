package auth

import (
	"bytes"
	"crypto/sha256"
	"strings"
	"testing"
)

func TestSessionTokenStoresOnlyDigest(t *testing.T) {
	token, digest, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken() error = %v", err)
	}
	if token == "" || len(digest) != sha256.Size {
		t.Fatalf("token/digest lengths are invalid")
	}
	if strings.Contains(string(digest), token) {
		t.Fatal("digest contains plaintext token")
	}

	derived, err := HashSessionToken(token)
	if err != nil {
		t.Fatalf("HashSessionToken() error = %v", err)
	}
	if !bytes.Equal(derived, digest) {
		t.Fatal("session digest mismatch")
	}
}

func TestHashSessionTokenRejectsMalformedToken(t *testing.T) {
	if _, err := HashSessionToken("not-a-valid-token"); err == nil {
		t.Fatal("HashSessionToken() error = nil, want error")
	}
}
