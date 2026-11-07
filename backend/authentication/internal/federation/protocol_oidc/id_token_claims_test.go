package protocol_oidc

import (
	"maps"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

//spec:covers REQ-AUTHENTICATION-001: ID Token の署名方式を RS256 に限り、exp を必須にし、exp、nbf、issuer、audience を秒単位の境界で判定する。
func TestVerifyUpstreamIDTokenChecksAlgorithmTimeIssuerAndAudience(t *testing.T) {
	connection, attempt, key, keySet, now := fuzzIDTokenFixture(t)
	base := map[string]any{
		"iss": connection.Issuer, "aud": connection.ClientID, "sub": "upstream-user",
		"nonce": attempt.Nonce, "exp": now.Add(time.Minute).Unix(),
	}
	sign := tokens_jose.SignRS256
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		sign   func(any, string, map[string]string, map[string]any) (string, error)
		accept bool
	}{
		{name: "valid", accept: true},
		{name: "PS256 with the same RSA key", sign: tokens_jose.SignPS256},
		{name: "missing exp", mutate: func(c map[string]any) { delete(c, "exp") }},
		{name: "exp equals now", mutate: func(c map[string]any) { c["exp"] = now.Unix() }},
		{name: "exp within the current second", mutate: func(c map[string]any) { c["exp"] = float64(now.Unix()) + 0.9 }},
		{name: "exp one second later", mutate: func(c map[string]any) { c["exp"] = now.Unix() + 1 }, accept: true},
		{name: "exp is not a number", mutate: func(c map[string]any) { c["exp"] = "2030-01-01" }},
		{name: "nbf equals now", mutate: func(c map[string]any) { c["nbf"] = now.Unix() }, accept: true},
		{name: "nbf one second later", mutate: func(c map[string]any) { c["nbf"] = now.Unix() + 1 }},
		{name: "nbf is not a number", mutate: func(c map[string]any) { c["nbf"] = "now" }},
		{name: "missing issuer", mutate: func(c map[string]any) { delete(c, "iss") }},
		{name: "non-string issuer", mutate: func(c map[string]any) { c["iss"] = 1 }},
		{name: "audience array with the client", mutate: func(c map[string]any) { c["aud"] = []any{"other", connection.ClientID} }, accept: true},
		{name: "audience array without the client", mutate: func(c map[string]any) { c["aud"] = []any{"other"} }},
		{name: "audience array with a non-string member", mutate: func(c map[string]any) { c["aud"] = []any{connection.ClientID, 1} }},
		{name: "missing audience", mutate: func(c map[string]any) { delete(c, "aud") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := maps.Clone(base)
			if tc.mutate != nil {
				tc.mutate(claims)
			}
			signer := sign
			if tc.sign != nil {
				signer = tc.sign
			}
			token, err := signer(key, "upstream-key", nil, claims)
			if err != nil {
				t.Fatal(err)
			}
			got, err := verifyUpstreamIDToken(token, keySet, connection, attempt, now)
			if tc.accept {
				if err != nil || got["sub"] != "upstream-user" {
					t.Fatalf("claims=%v err=%v, want verified upstream-user", got, err)
				}
			} else if err == nil || got != nil {
				t.Fatalf("claims=%v err=%v, want rejection without claims", got, err)
			}
		})
	}
}
