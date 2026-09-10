package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password || strings.Contains(hash, password) {
		t.Fatal("password hash contains plaintext password")
	}
	if !VerifyPassword(hash, password) {
		t.Fatal("VerifyPassword(correct) = false")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("VerifyPassword(wrong) = true")
	}
}

func TestPasswordHashUsesRandomSalt(t *testing.T) {
	first, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("same password produced identical encoded hash")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, encoded := range []string{"", "plaintext", "$pbkdf2-sha256$i=999999999$bad$bad"} {
		if VerifyPassword(encoded, "password") {
			t.Fatalf("VerifyPassword(%q) = true", encoded)
		}
	}
}
