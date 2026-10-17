package tokens_jose

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSignPS256PreservesHeaderClaimsAndSignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	token, err := SignPS256(key, "key-1", map[string]string{"typ": "secevent+jwt"}, map[string]any{"sub": "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if err := verifyJWTSignature(parts, "PS256", &key.PublicKey); err != nil {
		t.Fatalf("signature: %v", err)
	}
	for index, want := range []map[string]any{
		{"alg": "PS256", "kid": "key-1", "typ": "secevent+jwt"},
		{"sub": "user-1"},
	} {
		decoded, err := base64.RawURLEncoding.DecodeString(parts[index])
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(decoded, &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("JWT part %d = %v, want %v", index, got, want)
		}
		for name, value := range want {
			if got[name] != value {
				t.Fatalf("JWT part %d field %s = %v, want %v", index, name, got[name], value)
			}
		}
	}
}

type unavailableSigner struct {
	crypto.Signer
	err error
}

func (s unavailableSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	return nil, s.err
}

func TestSignPS256PropagatesSigningAndEncodingFailures(t *testing.T) {
	signErr := errors.New("remote signing unavailable")
	for _, tc := range []struct {
		name   string
		key    any
		claims map[string]any
		want   error
	}{
		{name: "missing signer"},
		{name: "unencodable claim", key: unavailableSigner{err: signErr}, claims: map[string]any{"invalid": make(chan int)}},
		{name: "signing failure", key: unavailableSigner{err: signErr}, want: signErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := SignPS256(tc.key, "key-1", nil, tc.claims)
			if token != "" || err == nil {
				t.Fatalf("token=%q error=%v, want no token and an error", token, err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
		})
	}
}
