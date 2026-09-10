package proxyprovision

import (
	"errors"
	"strings"
	"testing"

	"github.com/smorad3363/teleproxy/internal/telemt"
)

const proofSecret = "00112233445566778899aabbccddeeff"

func TestGenerateSecretReturnsLowerHexAndMatchingDigest(t *testing.T) {
	secret, digest, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) != 32 || secret != strings.ToLower(secret) {
		t.Fatalf("secret format = %q", secret)
	}
	want, err := SecretDigest(secret)
	if err != nil {
		t.Fatal(err)
	}
	if digest != want {
		t.Fatal("generated digest does not match generated secret")
	}
}

func TestVerifyLinksDigestNormalizesClassicSecureAndTLS(t *testing.T) {
	digest, err := SecretDigest(strings.ToUpper(proofSecret))
	if err != nil {
		t.Fatal(err)
	}
	links := telemt.UserLinks{
		Classic: []string{"tg://proxy?server=proxy.example&port=443&secret=" + strings.ToUpper(proofSecret)},
		Secure:  []string{"tg://proxy?server=proxy.example&port=443&secret=dd" + proofSecret},
		TLS:     []string{"tg://proxy?server=proxy.example&port=443&secret=ee" + proofSecret + "6578616d706c652e636f6d"},
	}
	proof, err := VerifyLinksDigest(links, digest)
	if err != nil {
		t.Fatalf("VerifyLinksDigest() error = %v", err)
	}
	if proof.digest != digest {
		t.Fatal("proof digest does not match expected digest")
	}
}

func TestVerifyLinksDigestRejectsMismatchAndInconsistentLinks(t *testing.T) {
	digest, _ := SecretDigest(proofSecret)
	wrong, _ := SecretDigest("ffeeddccbbaa99887766554433221100")
	links := telemt.UserLinks{Classic: []string{"tg://proxy?server=proxy.example&port=443&secret=" + proofSecret}}
	if _, err := VerifyLinksDigest(links, wrong); !errors.Is(err, ErrOwnershipMismatch) {
		t.Fatalf("mismatch error = %v", err)
	}
	links.Secure = []string{"tg://proxy?server=proxy.example&port=443&secret=ddffeeddccbbaa99887766554433221100"}
	if _, err := VerifyLinksDigest(links, digest); err == nil || errors.Is(err, ErrOwnershipMismatch) {
		t.Fatalf("inconsistent links error = %v", err)
	}
}

func TestVerifyLinksDigestRejectsMissingOrMalformedLinks(t *testing.T) {
	digest, _ := SecretDigest(proofSecret)
	if _, err := VerifyLinksDigest(telemt.UserLinks{}, digest); err == nil {
		t.Fatal("empty links accepted")
	}
	for _, raw := range []string{
		"https://proxy.example/?server=x&port=443&secret=" + proofSecret,
		"tg://proxy?port=443&secret=" + proofSecret,
		"tg://proxy?server=x&port=0&secret=" + proofSecret,
		"tg://proxy/path?server=x&port=443&secret=" + proofSecret,
		"tg://proxy?server=x&port=443&secret=bad",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := VerifyLinksDigest(telemt.UserLinks{Classic: []string{raw}}, digest); err == nil {
				t.Fatalf("malformed link accepted: %q", raw)
			}
		})
	}
}
