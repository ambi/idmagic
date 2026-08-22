package tokens_jose

// DPoP proof JWT (RFC 9449) の表駆動ユニットテスト。
// VerifyDPoP の典型的な失敗ケース (typ/alg/iat/jti/htm/htu) と happy path を網羅する。

import (
	"context"
	cryptostd "crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
)

func dpopTestECKey(t *testing.T) (*ecdsa.PrivateKey, map[string]any) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// Uncompressed SEC1 point: 0x04 || X (32 bytes) || Y (32 bytes).
	point, err := key.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(point[1:33]),
		"y":   base64.RawURLEncoding.EncodeToString(point[33:65]),
	}
	return key, jwk
}

func encodeECDPoPProof(t *testing.T, key *ecdsa.PrivateKey, header, payload map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func dpopTestKey(t *testing.T) (*rsa.PrivateKey, map[string]any) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key, rsaPublicJWK(&key.PublicKey)
}

func encodeDPoPProof(t *testing.T, key *rsa.PrivateKey, header, payload map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPSS(
		rand.Reader, key, cryptostd.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyDPoPAcceptsValidProof(t *testing.T) {
	key, jwk := dpopTestKey(t)
	now := time.Now().UTC()
	proof := encodeDPoPProof(
		t, key,
		map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "jti-ok", "iat": now.Unix()},
	)
	res, err := VerifyDPoPForToken(
		context.Background(), proof, "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), now,
	)
	if err != nil {
		t.Fatalf("valid proof rejected: %v", err)
	}
	if res == nil || res.JKT == "" {
		t.Fatalf("expected non-empty thumbprint, got %+v", res)
	}
	expectedJKT, err := jwkThumbprint(jwk)
	if err != nil {
		t.Fatal(err)
	}
	if res.JKT != expectedJKT {
		t.Fatalf("jkt mismatch got=%s want=%s", res.JKT, expectedJKT)
	}
}

func TestVerifyDPoPRejectsFailureCases(t *testing.T) {
	key, jwk := dpopTestKey(t)
	now := time.Now().UTC()
	const validHTU = "https://idp.example/token"

	type proofMutator func(header, payload map[string]any)
	cases := []struct {
		name      string
		mutate    proofMutator
		wantError string // substring of expected error
		htm       string
		htu       string
	}{
		{
			name:      "typ が dpop+jwt でない",
			mutate:    func(h, _ map[string]any) { h["typ"] = "jwt" },
			wantError: "typ must be dpop+jwt",
		},
		{
			name:      "alg が PS256 / ES256 以外",
			mutate:    func(h, _ map[string]any) { h["alg"] = "HS256" },
			wantError: "alg must be",
		},
		{
			name:      "jwk header が欠落",
			mutate:    func(h, _ map[string]any) { delete(h, "jwk") },
			wantError: "jwk header required",
		},
		{
			name:      "iat が clock skew より過去",
			mutate:    func(_, p map[string]any) { p["iat"] = now.Add(-2 * time.Minute).Unix() },
			wantError: "iat outside",
		},
		{
			name:      "iat が clock skew より未来",
			mutate:    func(_, p map[string]any) { p["iat"] = now.Add(time.Minute).Unix() },
			wantError: "iat outside",
		},
		{
			name:      "jti 欠落",
			mutate:    func(_, p map[string]any) { delete(p, "jti") },
			wantError: "jti required",
		},
		{
			name:      "htm 不一致",
			mutate:    func(_, p map[string]any) { p["htm"] = "GET" },
			wantError: "htm mismatch",
		},
		{
			name:      "htu 不一致",
			mutate:    func(_, p map[string]any) { p["htu"] = "https://idp.example/other" },
			wantError: "htu mismatch",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			header := map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk}
			payload := map[string]any{"htm": "POST", "htu": validHTU, "jti": tc.name, "iat": now.Unix()}
			tc.mutate(header, payload)
			proof := encodeDPoPProof(t, key, header, payload)
			_, err := VerifyDPoPForToken(
				context.Background(), proof, "POST", validHTU,
				memory.NewDpopReplayStore(), now,
			)
			if err == nil {
				t.Fatalf("expected rejection containing %q", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantError)
			}
		})
	}
}

func TestVerifyDPoPDetectsReplay(t *testing.T) {
	// 同一 jti の再使用は ReplayWindow 内で拒否される (DpopJtiUniquenessWithinWindow)。
	key, jwk := dpopTestKey(t)
	now := time.Now().UTC()
	store := memory.NewDpopReplayStore()
	proof := encodeDPoPProof(
		t, key,
		map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "replay-jti", "iat": now.Unix()},
	)
	if _, err := VerifyDPoPForToken(context.Background(), proof, "POST", "https://idp.example/token", store, now); err != nil {
		t.Fatalf("first attempt rejected: %v", err)
	}
	_, err := VerifyDPoPForToken(context.Background(), proof, "POST", "https://idp.example/token", store, now)
	if err == nil || !strings.Contains(err.Error(), "replay") {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}

func TestVerifyDPoPRejectsInvalidSignature(t *testing.T) {
	// 別鍵で署名された proof は jwk header と一致しないため署名検証で落ちる。
	signer, _ := dpopTestKey(t)
	_, claimedJWK := dpopTestKey(t)
	now := time.Now().UTC()
	proof := encodeDPoPProof(
		t, signer,
		map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": claimedJWK},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "sig-mismatch", "iat": now.Unix()},
	)
	_, err := VerifyDPoPForToken(
		context.Background(), proof, "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), now,
	)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature rejection, got %v", err)
	}
}

func TestVerifyDPoPForResourceBindsProofToAccessToken(t *testing.T) {
	// REQ-OAUTH2-045: a proof presented to a protected resource must carry
	// base64url(SHA-256) of the presented access token in ath.
	key, jwk := dpopTestKey(t)
	now := time.Now().UTC()
	const (
		htu         = "https://idp.example/realms/default/userinfo"
		accessToken = "AT1"
		otherToken  = "AT2"
	)
	proofWithATH := func(t *testing.T, jti string, ath any) string {
		t.Helper()
		payload := map[string]any{"htm": "GET", "htu": htu, "jti": jti, "iat": now.Unix()}
		if ath != nil {
			payload["ath"] = ath
		}
		return encodeDPoPProof(
			t, key,
			map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk},
			payload,
		)
	}

	t.Run("正しい ath を持つ proof は受理される", func(t *testing.T) {
		res, err := VerifyDPoPForResource(
			context.Background(), proofWithATH(t, "ath-ok", AccessTokenHash(accessToken)),
			"GET", htu, accessToken, memory.NewDpopReplayStore(), now,
		)
		if err != nil {
			t.Fatalf("valid ath rejected: %v", err)
		}
		if res == nil || res.JKT == "" {
			t.Fatalf("expected thumbprint, got %+v", res)
		}
	})

	t.Run("ath 欠落の proof は拒否される", func(t *testing.T) {
		_, err := VerifyDPoPForResource(
			context.Background(), proofWithATH(t, "ath-missing", nil),
			"GET", htu, accessToken, memory.NewDpopReplayStore(), now,
		)
		if err == nil || !strings.Contains(err.Error(), "ath required") {
			t.Fatalf("expected missing-ath rejection, got %v", err)
		}
	})

	t.Run("別 access token の ath を持つ proof は拒否される", func(t *testing.T) {
		_, err := VerifyDPoPForResource(
			context.Background(), proofWithATH(t, "ath-other", AccessTokenHash(otherToken)),
			"GET", htu, accessToken, memory.NewDpopReplayStore(), now,
		)
		if err == nil || !strings.Contains(err.Error(), "ath mismatch") {
			t.Fatalf("expected ath mismatch rejection, got %v", err)
		}
	})

	t.Run("proof 不在は拒否される", func(t *testing.T) {
		_, err := VerifyDPoPForResource(
			context.Background(), "", "GET", htu, accessToken,
			memory.NewDpopReplayStore(), now,
		)
		if err == nil || !strings.Contains(err.Error(), "proof required") {
			t.Fatalf("expected missing-proof rejection, got %v", err)
		}
	})

	t.Run("access token 未指定の呼び出しは拒否される", func(t *testing.T) {
		// Pins that the ath check cannot be skipped by forgetting to pass the token.
		_, err := VerifyDPoPForResource(
			context.Background(), proofWithATH(t, "ath-no-token", AccessTokenHash(accessToken)),
			"GET", htu, "", memory.NewDpopReplayStore(), now,
		)
		if err == nil || !strings.Contains(err.Error(), "access token required") {
			t.Fatalf("expected missing-token rejection, got %v", err)
		}
	})
}

func TestVerifyDPoPForTokenAcceptsProofWithoutATH(t *testing.T) {
	// REQ-OAUTH2-045: the token endpoint requires no ath because the target access
	// token does not exist yet. Guards against leaking the requirement onto that path.
	key, jwk := dpopTestKey(t)
	now := time.Now().UTC()
	proof := encodeDPoPProof(
		t, key,
		map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "token-no-ath", "iat": now.Unix()},
	)
	res, err := VerifyDPoPForToken(
		context.Background(), proof, "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), now,
	)
	if err != nil || res == nil {
		t.Fatalf("token endpoint proof without ath: res=%v err=%v", res, err)
	}
}

func TestVerifyDPoPES256ProofFailsAtThumbprint(t *testing.T) {
	// Pins a known limitation rather than the RFC 9449-intended behavior:
	// alg validation accepts ES256 ("alg must be PS256 or ES256" above), and
	// signature verification over the EC key succeeds, but jwkThumbprint
	// hardcodes the RSA member set (e/kty/n) for the RFC 7638 canonical
	// form. An EC jwk has none of those, so every ES256 DPoP proof is
	// rejected at the last step regardless of a valid signature. Coverage
	// task wi-129 documents current behavior rather than changing it
	// (spec_impact: none); fixing this is tracked separately.
	key, jwk := dpopTestECKey(t)
	now := time.Now().UTC()
	proof := encodeECDPoPProof(
		t, key,
		map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": jwk},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "es256-ok", "iat": now.Unix()},
	)
	_, err := VerifyDPoPForToken(
		context.Background(), proof, "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), now,
	)
	if err == nil || !strings.Contains(err.Error(), "missing required member") {
		t.Fatalf("expected thumbprint rejection for EC jwk, got %v", err)
	}
}

func TestVerifyDPoPRejectsES256WithWrongKey(t *testing.T) {
	signer, _ := dpopTestECKey(t)
	_, claimedJWK := dpopTestECKey(t)
	now := time.Now().UTC()
	proof := encodeECDPoPProof(
		t, signer,
		map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": claimedJWK},
		map[string]any{"htm": "POST", "htu": "https://idp.example/token", "jti": "es256-bad", "iat": now.Unix()},
	)
	_, err := VerifyDPoPForToken(
		context.Background(), proof, "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), now,
	)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature rejection, got %v", err)
	}
}

func TestPublicKeyFromJWK(t *testing.T) {
	_, rsaJWK := dpopTestKey(t)
	_, ecJWK := dpopTestECKey(t)

	t.Run("parses RSA jwk", func(t *testing.T) {
		pub, err := publicKeyFromJWK(rsaJWK)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := pub.(*rsa.PublicKey); !ok {
			t.Fatalf("got %T, want *rsa.PublicKey", pub)
		}
	})

	t.Run("parses EC P-256 jwk", func(t *testing.T) {
		pub, err := publicKeyFromJWK(ecJWK)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := pub.(*ecdsa.PublicKey); !ok {
			t.Fatalf("got %T, want *ecdsa.PublicKey", pub)
		}
	})

	t.Run("rejects unsupported kty", func(t *testing.T) {
		if _, err := publicKeyFromJWK(map[string]any{"kty": "OKP"}); err == nil {
			t.Fatal("expected error for unsupported kty")
		}
	})

	t.Run("rejects unsupported EC curve", func(t *testing.T) {
		jwk := map[string]any{"kty": "EC", "crv": "P-384", "x": "", "y": ""}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for unsupported curve")
		}
	})

	t.Run("rejects malformed RSA n", func(t *testing.T) {
		jwk := map[string]any{"kty": "RSA", "n": "not-base64!", "e": rsaJWK["e"]}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for malformed n")
		}
	})

	t.Run("rejects malformed RSA e", func(t *testing.T) {
		jwk := map[string]any{"kty": "RSA", "n": rsaJWK["n"], "e": "not-base64!"}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for malformed e")
		}
	})

	t.Run("rejects malformed EC x", func(t *testing.T) {
		jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": "not-base64!", "y": ecJWK["y"]}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for malformed x")
		}
	})

	t.Run("rejects malformed EC y", func(t *testing.T) {
		jwk := map[string]any{"kty": "EC", "crv": "P-256", "x": ecJWK["x"], "y": "not-base64!"}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for malformed y")
		}
	})

	t.Run("rejects wrong-length EC coordinates", func(t *testing.T) {
		jwk := map[string]any{
			"kty": "EC", "crv": "P-256",
			"x": base64.RawURLEncoding.EncodeToString([]byte{1, 2, 3}),
			"y": ecJWK["y"],
		}
		if _, err := publicKeyFromJWK(jwk); err == nil {
			t.Fatal("expected error for short coordinate")
		}
	})
}

func TestVerifyJWTSignatureRejectsAlgKeyMismatch(t *testing.T) {
	_, rsaJWK := dpopTestKey(t)
	rsaPub, err := publicKeyFromJWK(rsaJWK)
	if err != nil {
		t.Fatal(err)
	}
	_, ecJWK := dpopTestECKey(t)
	ecPub, err := publicKeyFromJWK(ecJWK)
	if err != nil {
		t.Fatal(err)
	}
	parts := []string{
		base64.RawURLEncoding.EncodeToString([]byte("h")),
		base64.RawURLEncoding.EncodeToString([]byte("p")),
		base64.RawURLEncoding.EncodeToString(make([]byte, 64)),
	}

	if err := verifyJWTSignature(parts, "ES256", rsaPub); err == nil {
		t.Fatal("expected error: ES256 requires an EC public key")
	}
	if err := verifyJWTSignature(parts, "PS256", ecPub); err == nil {
		t.Fatal("expected error: PS256 requires an RSA public key")
	}
	if err := verifyJWTSignature(parts, "RS256", ecPub); err == nil {
		t.Fatal("expected error: RS256 requires an RSA public key")
	}
	if err := verifyJWTSignature(parts, "none", rsaPub); err == nil {
		t.Fatal("expected error for unsupported alg")
	}
	shortSig := []string{parts[0], parts[1], base64.RawURLEncoding.EncodeToString(make([]byte, 10))}
	if err := verifyJWTSignature(shortSig, "ES256", ecPub); err == nil {
		t.Fatal("expected error for wrong-length ES256 signature")
	}
}

func TestVerifyDPoPMissingHeaderIsNoOp(t *testing.T) {
	// 空ヘッダーは「proof 不在」を意味する。エラー無しで nil を返し、呼び出し側の責務に委ねる。
	res, err := VerifyDPoPForToken(
		context.Background(), "", "POST", "https://idp.example/token",
		memory.NewDpopReplayStore(), time.Now().UTC(),
	)
	if err != nil || res != nil {
		t.Fatalf("empty header: got res=%v err=%v, want nil/nil", res, err)
	}
}
