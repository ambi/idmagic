package tokens_jose

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func encryptionKey(tb testing.TB) (*ecdsa.PrivateKey, map[string]any, string) {
	tb.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	point, err := key.PublicKey.Bytes()
	if err != nil {
		tb.Fatal(err)
	}
	jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": base64.RawURLEncoding.EncodeToString(point[1:33]), "y": base64.RawURLEncoding.EncodeToString(point[33:65])}
	kid, err := ValidateEncryptionJWK(jwk)
	if err != nil {
		tb.Fatal(err)
	}
	return key, jwk, kid
}

//spec:covers REQ-SOURCING-026 REQ-DIRECTORYCONNECTOR-007: 正しい鍵だけが復号でき、改ざん、別の kid と秘密成分を拒否する。
func TestEncryptionRequiresTheBoundPublicKey(t *testing.T) {
	key, jwk, kid := encryptionKey(t)
	plain := []byte(`{"password":"secret-directory-password"}`)
	cipher, err := EncryptCompact(jwk, plain)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cipher, "secret-directory-password") {
		t.Fatal("plaintext leaked")
	}
	got, err := DecryptCompact(key, kid, cipher)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("decrypt = %s %v", got, err)
	}
	other, _, _ := encryptionKey(t)
	if _, err := DecryptCompact(other, kid, cipher); err == nil {
		t.Fatal("another key decrypted")
	}
	if _, err := DecryptCompact(key, "other-key", cipher); err == nil {
		t.Fatal("another kid decrypted")
	}
	parts := strings.Split(cipher, ".")
	parts[3] = "AAAA"
	if _, err := DecryptCompact(key, kid, strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
	jwk["d"] = "secret"
	if _, err := ValidateEncryptionJWK(jwk); err == nil {
		t.Fatal("private JWK accepted")
	}
}

func FuzzDecryptCompact(f *testing.F) {
	key, jwk, kid := encryptionKey(f)
	f.Add("credential")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 16<<10 {
			t.Skip()
		}
		cipher, err := EncryptCompact(jwk, []byte(input))
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecryptCompact(key, kid, cipher)
		if err != nil || string(got) != input {
			t.Fatalf("round trip changed the plaintext: %v", err)
		}
		if _, err := DecryptCompact(key, "wrong-key-id", cipher); err == nil {
			t.Fatal("unbound key id accepted")
		}
		parts := strings.Split(cipher, ".")
		tag, err := base64.RawURLEncoding.DecodeString(parts[4])
		if err != nil {
			t.Fatal(err)
		}
		tag[0] ^= 1
		parts[4] = base64.RawURLEncoding.EncodeToString(tag)
		if _, err := DecryptCompact(key, kid, strings.Join(parts, ".")); err == nil {
			t.Fatal("tampered tag accepted")
		}
	})
}
