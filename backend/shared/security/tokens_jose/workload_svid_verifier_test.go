package tokens_jose

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

const testWorkloadSubject = "spiffe://example.org/ns/prod/sa/worker-1"

func signWorkloadSVID(
	t *testing.T,
	key *rsa.PrivateKey,
	alg, iss, aud string,
	iat, exp time.Time,
) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": alg, "kid": "workload-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": iss, "sub": testWorkloadSubject, "aud": aud,
		"iat": iat.Unix(), "exp": exp.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	var sig []byte
	switch alg {
	case "RS256":
		sig, err = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	case "PS256":
		sig, err = rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	default:
		t.Fatalf("unsupported test alg %q", alg)
	}
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyWorkloadSVID(t *testing.T) {
	const (
		issuer   = "https://issuer.example"
		audience = "https://idmagic.example/token"
		kid      = "workload-key"
		subject  = "spiffe://example.org/ns/prod/sa/worker-1"
	)
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaPublicJWK(&key.PublicKey)
	jwk["kid"] = kid
	jwks := []map[string]any{jwk}

	t.Run("accepts a valid RS256 workload SVID", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now, now.Add(10*time.Minute))
		claims, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if err != nil {
			t.Fatalf("valid SVID rejected: %v", err)
		}
		if claims.Subject != subject {
			t.Fatalf("Subject = %q, want %q", claims.Subject, subject)
		}
		if claims.Issuer != issuer {
			t.Fatalf("Issuer = %q, want %q", claims.Issuer, issuer)
		}
	})

	t.Run("rejects spoofed signature", func(t *testing.T) {
		attacker, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		token := signWorkloadSVID(t, attacker, "RS256", issuer, audience, now, now.Add(10*time.Minute))
		_, err = VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDInvalidSignature) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDInvalidSignature", err)
		}
	})

	t.Run("rejects unregistered issuer", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", "https://unknown-issuer.example", audience, now, now.Add(10*time.Minute))
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDIssuerMismatch) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDIssuerMismatch", err)
		}
	})

	t.Run("rejects audience mismatch", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, "https://other.example", now, now.Add(10*time.Minute))
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDAudienceMismatch) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDAudienceMismatch", err)
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now.Add(-2*time.Hour), now.Add(-time.Hour))
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDExpired) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDExpired", err)
		}
	})

	t.Run("rejects a token whose lifetime exceeds the trust bundle max TTL", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now, now.Add(2*time.Hour))
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDTTLExceeded) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDTTLExceeded", err)
		}
	})

	t.Run("accepts PS256 as well as RS256", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "PS256", issuer, audience, now, now.Add(10*time.Minute))
		if _, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now); err != nil {
			t.Fatalf("PS256 SVID rejected: %v", err)
		}
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		_, err := VerifyWorkloadSVID("not-a-jwt", jwks, oidcExpectation(), now)
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}

func TestWorkloadJWKSCache(t *testing.T) {
	now := time.Now().UTC()
	goodKeys := []map[string]any{{"kid": "k1"}}

	t.Run("returns freshly fetched keys and remembers them", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		calls := 0
		fetch := func(context.Context) ([]map[string]any, error) {
			calls++
			return goodKeys, nil
		}
		keys, stale, err := cache.Get(context.Background(), "bundle-1", fetch, now)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if stale {
			t.Fatal("freshly fetched keys must not be reported stale")
		}
		if len(keys) != 1 || calls != 1 {
			t.Fatalf("keys=%v calls=%d", keys, calls)
		}
	})

	t.Run("falls back to last-known-good within the staleness window on fetch failure", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		fetch := func(context.Context) ([]map[string]any, error) { return goodKeys, nil }
		if _, _, err := cache.Get(context.Background(), "bundle-1", fetch, now); err != nil {
			t.Fatalf("seed fetch: %v", err)
		}
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		keys, stale, err := cache.Get(context.Background(), "bundle-1", failing, now.Add(time.Hour))
		if err != nil {
			t.Fatalf("expected fallback to last-known-good, got error: %v", err)
		}
		if !stale {
			t.Fatal("fallback keys must be reported stale")
		}
		if len(keys) != 1 {
			t.Fatalf("keys=%v", keys)
		}
	})

	t.Run("fails closed once the last-known-good entry exceeds the staleness window", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		fetch := func(context.Context) ([]map[string]any, error) { return goodKeys, nil }
		if _, _, err := cache.Get(context.Background(), "bundle-1", fetch, now); err != nil {
			t.Fatalf("seed fetch: %v", err)
		}
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		_, _, err := cache.Get(context.Background(), "bundle-1", failing, now.Add(25*time.Hour))
		if err == nil {
			t.Fatal("expected fail-closed once last-known-good is stale beyond the window")
		}
	})

	t.Run("fails closed with no prior successful fetch", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		_, _, err := cache.Get(context.Background(), "bundle-1", failing, now)
		if err == nil {
			t.Fatal("expected fail-closed with no cache entry")
		}
	})
}

func TestAudienceStrings(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"single string", "aud-1", []string{"aud-1"}},
		{"string array", []any{"a", "b"}, []string{"a", "b"}},
		{"array with non-string entries", []any{"a", 1, false}, []string{"a"}},
		{"empty array", []any{}, []string{}},
		{"unsupported type", 42, nil},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := audienceStrings(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("audienceStrings(%#v) = %#v, want %#v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("audienceStrings(%#v) = %#v, want %#v", tc.in, got, tc.want)
				}
			}
		})
	}
}

func oidcExpectation() WorkloadSVIDExpectation {
	return WorkloadSVIDExpectation{Issuer: "https://issuer.example", AcceptedAudiences: []string{"https://idmagic.example/token"}, MaxTTL: time.Hour}
}

// signWorkloadJWT は header と payload をそのまま JWS にする。alg ごとのハッシュと
// 署名形式は RFC 7518 の 3 節に従う (検証側の実装から計算しない)。
func signWorkloadJWT(t *testing.T, key crypto.Signer, header, payload map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	alg, _ := header["alg"].(string)
	hashes := map[string]crypto.Hash{"256": crypto.SHA256, "384": crypto.SHA384, "512": crypto.SHA512}
	hash := hashes[alg[2:]]
	h := hash.New()
	h.Write([]byte(input))
	digest := h.Sum(nil)
	var sig []byte
	switch alg[:2] {
	case "RS":
		sig, err = rsa.SignPKCS1v15(rand.Reader, key.(*rsa.PrivateKey), hash, digest)
	case "PS":
		sig, err = rsa.SignPSS(rand.Reader, key.(*rsa.PrivateKey), hash, digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	case "ES":
		ecKey := key.(*ecdsa.PrivateKey)
		var r, s *big.Int
		r, s, err = ecdsa.Sign(rand.Reader, ecKey, digest)
		size := (ecKey.Curve.Params().BitSize + 7) / 8
		sig = make([]byte, 2*size)
		if err == nil {
			r.FillBytes(sig[:size])
			s.FillBytes(sig[size:])
		}
	default:
		t.Fatalf("unsupported test alg %q", alg)
	}
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func ecPublicWorkloadJWK(t *testing.T, key *ecdsa.PrivateKey, crv, kid string) map[string]any {
	t.Helper()
	size := (key.Curve.Params().BitSize + 7) / 8
	x, y := make([]byte, size), make([]byte, size)
	key.X.FillBytes(x)
	key.Y.FillBytes(y)
	return map[string]any{
		"kty": "EC", "crv": crv, "kid": kid,
		"x": base64.RawURLEncoding.EncodeToString(x), "y": base64.RawURLEncoding.EncodeToString(y),
	}
}

// SPIFFE の種別は JWT-SVID の JOSE の規則 (kid 必須、typ、RS/ES/PS の 256/384/512、
// iss 任意) を掛け、OIDC の種別は従来どおりの規則のままにする。
//
//spec:covers REQ-WORKLOADIDENTITY-011
func TestVerifyWorkloadSVID_SPIFFEProfile(t *testing.T) {
	const (
		audience = "https://idmagic.example/token"
		subject  = "spiffe://example.org/ns/prod/sa/worker-1"
	)
	now := time.Now().UTC()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaJWK := rsaPublicJWK(&rsaKey.PublicKey)
	rsaJWK["kid"] = "rsa"
	keys := map[string]crypto.Signer{"rsa": rsaKey}
	jwks := []map[string]any{rsaJWK}
	for kid, curve := range map[string]elliptic.Curve{"P-256": elliptic.P256(), "P-384": elliptic.P384(), "P-521": elliptic.P521()} {
		ecKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		keys[kid] = ecKey
		jwks = append(jwks, ecPublicWorkloadJWK(t, ecKey, kid, kid))
	}
	spiffe := WorkloadSVIDExpectation{Profile: WorkloadSVIDProfileSPIFFE, AcceptedAudiences: []string{audience}, MaxTTL: time.Hour}
	claims := func() map[string]any {
		return map[string]any{"sub": subject, "aud": audience, "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix()}
	}

	for alg, kid := range map[string]string{
		"RS256": "rsa", "RS384": "rsa", "RS512": "rsa", "PS256": "rsa", "PS384": "rsa", "PS512": "rsa",
		"ES256": "P-256", "ES384": "P-384", "ES512": "P-521",
	} {
		t.Run("accepts "+alg+" without iss", func(t *testing.T) {
			token := signWorkloadJWT(t, keys[kid], map[string]any{"alg": alg, "kid": kid, "typ": "JWT"}, claims())
			got, err := VerifyWorkloadSVID(token, jwks, spiffe, now)
			if err != nil {
				t.Fatalf("%s JWT-SVID rejected: %v", alg, err)
			}
			if got.Subject != subject || got.Issuer != "" {
				t.Fatalf("claims = %+v", got)
			}
		})
	}

	t.Run("OIDC keeps rejecting ES384", func(t *testing.T) {
		payload := claims()
		payload["iss"] = "https://issuer.example"
		token := signWorkloadJWT(t, keys["P-384"], map[string]any{"alg": "ES384", "kid": "P-384"}, payload)
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDInvalidSignature) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDInvalidSignature", err)
		}
	})

	t.Run("OIDC keeps requiring iss", func(t *testing.T) {
		token := signWorkloadJWT(t, rsaKey, map[string]any{"alg": "RS256", "kid": "rsa"}, claims())
		_, err := VerifyWorkloadSVID(token, jwks, oidcExpectation(), now)
		if !errors.Is(err, ErrWorkloadSVIDIssuerMismatch) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDIssuerMismatch", err)
		}
	})

	t.Run("OIDC keeps accepting a missing kid with a single key", func(t *testing.T) {
		payload := claims()
		payload["iss"] = "https://issuer.example"
		token := signWorkloadJWT(t, rsaKey, map[string]any{"alg": "RS256"}, payload)
		if _, err := VerifyWorkloadSVID(token, []map[string]any{rsaJWK}, oidcExpectation(), now); err != nil {
			t.Fatalf("OIDC token without kid rejected: %v", err)
		}
	})

	t.Run("checks iss when the SPIFFE trust bundle registers an issuer", func(t *testing.T) {
		withIssuer := spiffe
		withIssuer.Issuer = "https://spire.example"
		token := signWorkloadJWT(t, rsaKey, map[string]any{"alg": "RS256", "kid": "rsa"}, claims())
		if _, err := VerifyWorkloadSVID(token, jwks, withIssuer, now); !errors.Is(err, ErrWorkloadSVIDIssuerMismatch) {
			t.Fatalf("missing iss: err = %v, want ErrWorkloadSVIDIssuerMismatch", err)
		}
		payload := claims()
		payload["iss"] = "https://spire.example"
		token = signWorkloadJWT(t, rsaKey, map[string]any{"alg": "RS256", "kid": "rsa"}, payload)
		if _, err := VerifyWorkloadSVID(token, jwks, withIssuer, now); err != nil {
			t.Fatalf("matching iss rejected: %v", err)
		}
	})

	rejected := map[string]map[string]any{
		"missing kid even with a single key": {"alg": "RS256"},
		"typ other than JWT and JOSE":        {"alg": "RS256", "kid": "rsa", "typ": "at+jwt"},
		"non-string typ":                     {"alg": "RS256", "kid": "rsa", "typ": 1},
		"alg outside the JWT-SVID set":       {"alg": "HS256", "kid": "rsa"},
		"ES384 header over a P-256 key":      {"alg": "ES384", "kid": "P-256"},
	}
	for name, header := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			signer := crypto.Signer(rsaKey)
			signHeader := header
			if name == "ES384 header over a P-256 key" {
				signer = keys["P-256"]
			}
			if header["alg"] == "HS256" {
				signHeader = map[string]any{"alg": "RS256", "kid": "rsa"}
			}
			token := signWorkloadJWT(t, signer, signHeader, claims())
			if header["alg"] == "HS256" || header["alg"] == "ES384" {
				token = replaceJWTHeader(t, token, header)
			}
			jwkSet := jwks
			if _, hasKid := header["kid"]; !hasKid {
				jwkSet = []map[string]any{rsaJWK}
			}
			_, err := VerifyWorkloadSVID(token, jwkSet, spiffe, now)
			if !errors.Is(err, ErrWorkloadSVIDInvalidSignature) {
				t.Fatalf("err = %v, want ErrWorkloadSVIDInvalidSignature", err)
			}
		})
	}

	t.Run("accepts typ JOSE and an omitted typ", func(t *testing.T) {
		for _, header := range []map[string]any{{"alg": "RS256", "kid": "rsa", "typ": "JOSE"}, {"alg": "RS256", "kid": "rsa"}} {
			token := signWorkloadJWT(t, rsaKey, header, claims())
			if _, err := VerifyWorkloadSVID(token, jwks, spiffe, now); err != nil {
				t.Fatalf("header %v rejected: %v", header, err)
			}
		}
	})
}

// replaceJWTHeader は署名を変えずに header だけを差し替え、header の検査が署名より
// 先に、または署名と独立に拒否することを観測できるようにする。
func replaceJWTHeader(t *testing.T, token string, header map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(token, ".", 2)
	return base64.RawURLEncoding.EncodeToString(hb) + "." + parts[1]
}
