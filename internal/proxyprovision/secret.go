package proxyprovision

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/smorad3363/teleproxy/internal/telemt"
)

var ErrOwnershipMismatch = errors.New("Telemt user ownership does not match provisioning attempt")

type OwnershipProof struct {
	digest [32]byte
}

func GenerateSecret() (string, [32]byte, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", [32]byte{}, fmt.Errorf("generate proxy provisioning secret: %w", err)
	}
	secret := hex.EncodeToString(raw[:])
	digest, err := SecretDigest(secret)
	if err != nil {
		return "", [32]byte{}, err
	}
	return secret, digest, nil
}

func SecretDigest(secret string) ([32]byte, error) {
	if len(secret) != 32 {
		return [32]byte{}, fmt.Errorf("proxy provisioning secret must be 32 hexadecimal characters")
	}
	decoded, err := hex.DecodeString(secret)
	if err != nil || len(decoded) != 16 {
		return [32]byte{}, fmt.Errorf("proxy provisioning secret must be 32 hexadecimal characters")
	}
	return sha256.Sum256([]byte(strings.ToLower(secret))), nil
}

func VerifyLinksDigest(links telemt.UserLinks, expected [32]byte) (OwnershipProof, error) {
	all := links.All()
	if len(all) == 0 {
		return OwnershipProof{}, fmt.Errorf("Telemt user has no proxy links")
	}
	var observed [32]byte
	for index, raw := range all {
		secret, err := rawSecretFromLink(raw)
		if err != nil {
			return OwnershipProof{}, err
		}
		digest, err := SecretDigest(secret)
		if err != nil {
			return OwnershipProof{}, err
		}
		if index == 0 {
			observed = digest
			continue
		}
		if subtle.ConstantTimeCompare(observed[:], digest[:]) != 1 {
			return OwnershipProof{}, fmt.Errorf("Telemt proxy links contain inconsistent user secrets")
		}
	}
	if subtle.ConstantTimeCompare(observed[:], expected[:]) != 1 {
		return OwnershipProof{}, ErrOwnershipMismatch
	}
	return OwnershipProof{digest: expected}, nil
}

func rawSecretFromLink(raw string) (string, error) {
	if len(raw) < 1 || len(raw) > 4096 {
		return "", fmt.Errorf("invalid Telemt proxy link")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "tg" || parsed.Host != "proxy" || parsed.Path != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", fmt.Errorf("invalid Telemt proxy link")
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("server")) == "" {
		return "", fmt.Errorf("invalid Telemt proxy link")
	}
	port, err := strconv.ParseUint(query.Get("port"), 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("invalid Telemt proxy link")
	}
	encoded := query.Get("secret")
	switch {
	case len(encoded) == 32:
		return normalizeHexSecret(encoded)
	case len(encoded) == 34 && strings.EqualFold(encoded[:2], "dd"):
		return normalizeHexSecret(encoded[2:])
	case len(encoded) >= 34 && strings.EqualFold(encoded[:2], "ee"):
		return normalizeHexSecret(encoded[2:34])
	default:
		return "", fmt.Errorf("invalid Telemt proxy link secret")
	}
}

func normalizeHexSecret(secret string) (string, error) {
	if _, err := SecretDigest(secret); err != nil {
		return "", err
	}
	return strings.ToLower(secret), nil
}
