package tokens_jose

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"maps"
	"math/big"
	"testing"
)

func rsaTestJWK(t *testing.T, bits int) map[string]any {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	return rsaPublicJWK(&key.PublicKey)
}

//spec:covers REQ-SOURCING-009
func TestValidatePublicSigningJWK(t *testing.T) {
	_, ec := dpopTestECKey(t)
	rsa2048 := rsaTestJWK(t, 2048)
	with := func(jwk map[string]any, member string, value any) map[string]any {
		copied := maps.Clone(jwk)
		copied[member] = value
		return copied
	}
	tests := []struct {
		name string
		jwk  map[string]any
		ok   bool
	}{
		{"P-256 の EC 鍵", ec, true},
		{"2048 ビットの RSA 鍵", rsa2048, true},
		{"2048 ビット未満の RSA 鍵", rsaTestJWK(t, 1024), false},
		{"P-384 の EC 鍵", with(ec, "crv", "P-384"), false},
		{"曲線上にない点", with(ec, "y", base64.RawURLEncoding.EncodeToString(make([]byte, 32))), false},
		{"EC の秘密鍵の成分", with(ec, "d", "AAAA"), false},
		{"RSA の秘密鍵の成分", with(rsa2048, "p", "AAAA"), false},
		{"対称鍵", map[string]any{"kty": "oct", "k": "AAAA"}, false},
		{"公開指数が 1", with(rsa2048, "e", base64.RawURLEncoding.EncodeToString(big.NewInt(1).Bytes())), false},
		{"公開指数が 3", with(rsa2048, "e", base64.RawURLEncoding.EncodeToString(big.NewInt(3).Bytes())), true},
		{"公開指数が偶数", with(rsa2048, "e", base64.RawURLEncoding.EncodeToString(big.NewInt(65538).Bytes())), false},
		{"空の JWK", map[string]any{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			thumbprint, err := ValidatePublicSigningJWK(tt.jwk)
			if tt.ok != (err == nil) {
				t.Fatalf("ValidatePublicSigningJWK() error = %v, want ok=%v", err, tt.ok)
			}
			if tt.ok {
				want, _ := jwkThumbprint(tt.jwk)
				if thumbprint == "" || thumbprint != want {
					t.Fatalf("thumbprint = %q, want %q", thumbprint, want)
				}
			}
		})
	}
}
