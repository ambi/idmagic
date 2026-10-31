package tokens_jose

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"
)

const (
	idJagTestIssuer   = "https://acme.idp.example/"
	idJagTestAudience = "https://acme.chat.example/"
)

func idJagTestJWK(key *rsa.PrivateKey, kid, alg string) map[string]any {
	return map[string]any{
		"kty": "RSA", "kid": kid, "alg": alg,
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}

// 期待値の typ と URN は draft の 3.1 節の例から写す。製品の定数から作らない。
func idJagTestClaims(now time.Time) map[string]any {
	return map[string]any{
		"jti": "9e43f81b64a33f20116179", "iss": idJagTestIssuer, "sub": "U019488227",
		"aud": idJagTestAudience, "client_id": "f53f191f9311af35",
		"iat": now.Unix(), "exp": now.Add(300 * time.Second).Unix(),
		"resource": "https://api.chat.example/", "scope": "chat.read chat.history",
	}
}

//spec:covers IDJAG-REDEMPTION: typ、alg、aud、期限の境界、resource の形、cnf の読み取り
func TestVerifyIDJAG_IDJAG_REDEMPTION(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1311280970, 0).UTC()
	jwks := []map[string]any{idJagTestJWK(key, "k1", "RS256")}
	typed := map[string]string{"typ": "oauth-id-jag+jwt"}
	sign := func(t *testing.T, mutate func(map[string]any)) string {
		t.Helper()
		claims := idJagTestClaims(now)
		if mutate != nil {
			mutate(claims)
		}
		token, err := SignRS256(key, "k1", typed, claims)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	t.Run("accepts the draft example", func(t *testing.T) {
		claims, err := VerifyIDJAG(sign(t, nil), jwks, idJagTestIssuer, idJagTestAudience, now)
		if err != nil {
			t.Fatal(err)
		}
		if claims.Subject != "U019488227" || claims.ClientID != "f53f191f9311af35" || len(claims.Resources) != 1 || claims.Resources[0] != "https://api.chat.example/" {
			t.Fatalf("claims = %+v", claims)
		}
	})
	t.Run("aud as a single element array", func(t *testing.T) {
		token := sign(t, func(c map[string]any) { c["aud"] = []string{idJagTestAudience} })
		if _, err := VerifyIDJAG(token, jwks, idJagTestIssuer, idJagTestAudience, now); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("reads cnf members", func(t *testing.T) {
		token := sign(t, func(c map[string]any) { c["cnf"] = map[string]any{"jkt": "abc"} })
		claims, err := VerifyIDJAG(token, jwks, idJagTestIssuer, idJagTestAudience, now)
		if err != nil || claims.Confirmation == nil || claims.Confirmation.JKT != "abc" || claims.Confirmation.Other {
			t.Fatalf("claims = %+v err = %v", claims, err)
		}
		token = sign(t, func(c map[string]any) { c["cnf"] = map[string]any{"jwk": map[string]any{}} })
		claims, err = VerifyIDJAG(token, jwks, idJagTestIssuer, idJagTestAudience, now)
		if err != nil || !claims.Confirmation.Other {
			t.Fatalf("claims = %+v err = %v", claims, err)
		}
	})

	refusals := map[string]func(t *testing.T) string{
		"another issuer": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["iss"] = "https://evil.example/" })
		},
		"another audience": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["aud"] = "https://other.example/" })
		},
		"two audiences": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["aud"] = []string{idJagTestAudience, "https://other.example/"} })
		},
		"missing jti": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				delete(c, "jti")
			})
		},
		"missing sub": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				delete(c, "sub")
			})
		},
		"missing client_id": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				delete(c, "client_id")
			})
		},
		"missing iat": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				delete(c, "iat")
			})
		},
		"iat just past the skew": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				c["iat"] = now.Add(61 * time.Second).Unix()
				c["exp"] = now.Add(300 * time.Second).Unix()
			})
		},
		"exp at the skew": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) {
				c["iat"] = now.Add(-120 * time.Second).Unix()
				c["exp"] = now.Add(-60 * time.Second).Unix()
			})
		},
		"lifetime just over the limit": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["exp"] = now.Add(601 * time.Second).Unix() })
		},
		"duplicate resources": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["resource"] = []string{"https://a.example/", "https://a.example/"} })
		},
		"empty resources": func(t *testing.T) string {
			t.Helper()
			return sign(t, func(c map[string]any) { c["resource"] = []string{} })
		},
		"wrong typ": func(t *testing.T) string {
			t.Helper()
			token, err := SignRS256(key, "k1", map[string]string{"typ": "at+jwt"}, idJagTestClaims(now))
			if err != nil {
				t.Fatal(err)
			}
			return token
		},
		"no typ": func(t *testing.T) string {
			t.Helper()
			token, err := SignRS256(key, "k1", nil, idJagTestClaims(now))
			if err != nil {
				t.Fatal(err)
			}
			return token
		},
		"signed by another key": func(t *testing.T) string {
			t.Helper()
			token, err := SignRS256(other, "k1", typed, idJagTestClaims(now))
			if err != nil {
				t.Fatal(err)
			}
			return token
		},
		"alg differs from the key": func(t *testing.T) string {
			t.Helper()
			token, err := SignPS256(key, "k1", typed, idJagTestClaims(now))
			if err != nil {
				t.Fatal(err)
			}
			return token
		},
		"alg none": func(t *testing.T) string {
			t.Helper()
			header, err := json.Marshal(map[string]string{"alg": "none", "typ": "oauth-id-jag+jwt", "kid": "k1"})
			if err != nil {
				t.Fatal(err)
			}
			payload, err := json.Marshal(idJagTestClaims(now))
			if err != nil {
				t.Fatal(err)
			}
			return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
		},
	}
	for name, build := range refusals {
		t.Run(name, func(t *testing.T) {
			if _, err := VerifyIDJAG(build(t), jwks, idJagTestIssuer, idJagTestAudience, now); !errors.Is(err, ErrIDJAGInvalid) {
				t.Fatalf("err = %v, want ErrIDJAGInvalid", err)
			}
		})
	}

	t.Run("boundaries inside the limits are accepted", func(t *testing.T) {
		for name, mutate := range map[string]func(map[string]any){
			"iat at the skew": func(c map[string]any) { c["iat"] = now.Add(60 * time.Second).Unix() },
			"exp one second before the skew": func(c map[string]any) {
				c["iat"] = now.Add(-120 * time.Second).Unix()
				c["exp"] = now.Add(-59 * time.Second).Unix()
			},
			"lifetime at the limit": func(c map[string]any) { c["exp"] = now.Add(600 * time.Second).Unix() },
		} {
			if _, err := VerifyIDJAG(sign(t, mutate), jwks, idJagTestIssuer, idJagTestAudience, now); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
	})
}
