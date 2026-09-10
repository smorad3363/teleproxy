package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	passwordAlgorithm  = "pbkdf2-sha256"
	passwordIterations = 600_000
	passwordSaltBytes  = 16
	passwordKeyBytes   = 32
	maxPasswordBytes   = 1024
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is required")
	}
	if len(password) > maxPasswordBytes {
		return "", fmt.Errorf("password is too long")
	}

	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyBytes)
	if err != nil {
		return "", fmt.Errorf("derive password key: %w", err)
	}

	return fmt.Sprintf(
		"$%s$i=%d$%s$%s",
		passwordAlgorithm,
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	if password == "" || len(password) > maxPasswordBytes {
		return false
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "" || parts[1] != passwordAlgorithm {
		return false
	}

	iterationText, ok := strings.CutPrefix(parts[2], "i=")
	if !ok {
		return false
	}
	iterations, err := strconv.Atoi(iterationText)
	if err != nil || iterations < 100_000 || iterations > 2_000_000 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) != passwordSaltBytes {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(want) != passwordKeyBytes {
		return false
	}

	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
