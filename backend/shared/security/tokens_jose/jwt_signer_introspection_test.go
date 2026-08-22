package tokens_jose

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

func TestAccessTokenAndIDTokenTTLSeconds(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	if got := signer.AccessTokenTTLSeconds(); got != accessTokenTTLSeconds {
		t.Fatalf("AccessTokenTTLSeconds() = %d, want %d", got, accessTokenTTLSeconds)
	}
	if got := signer.IDTokenTTLSeconds(); got != idTokenTTLSeconds {
		t.Fatalf("IDTokenTTLSeconds() = %d, want %d", got, idTokenTTLSeconds)
	}
}

func TestSignIDTokenIncludesAtHash(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, err := signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, User: idTokenTestUser(),
		Scopes: []string{"openid"}, AtHashFor: "the-access-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if claims["at_hash"] != atHash("the-access-token") {
		t.Fatalf("at_hash claim mismatch: %#v", claims)
	}
}

func TestSignAccessTokenIncludesAMRACRAndAgentBinding(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, _, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "agent-client",
		Scopes: []string{"account:read"}, AMR: []string{"agent"}, ACR: "urn:agent",
		AgentID: "agent-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if claims["agent_id"] != "agent-1" || claims["principal_type"] != oauthdomain.PrincipalTypeAgent {
		t.Fatalf("agent claims missing: %#v", claims)
	}
	amr, _ := claims["amr"].([]any)
	if len(amr) != 1 || amr[0] != "agent" {
		t.Fatalf("amr claim missing: %#v", claims)
	}
	if claims["acr"] != "urn:agent" {
		t.Fatalf("acr claim missing: %#v", claims)
	}
}

func TestSignAccessTokenMultiAudienceAndCnf(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, _, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "user-1", Scopes: []string{"account:read"},
		Audiences:        []string{"https://rs1.example", "https://rs2.example"},
		SenderConstraint: &oauthdomain.SenderConstraint{Type: oauthdomain.SenderConstraintDPoP, JKT: "jkt-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	aud, _ := claims["aud"].([]any)
	if len(aud) != 2 || aud[0] != "https://rs1.example" || aud[1] != "https://rs2.example" {
		t.Fatalf("multi-audience aud mismatch: %#v", claims)
	}
	cnf, _ := claims["cnf"].(map[string]any)
	if cnf["jkt"] != "jkt-1" {
		t.Fatalf("cnf.jkt missing: %#v", claims)
	}

	result, err := signer.IntrospectAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Aud) != 2 || result.Aud[0] != "https://rs1.example" {
		t.Fatalf("introspection aud mismatch: %+v", result)
	}
	if result.SenderConstraint == nil || result.SenderConstraint.Type != oauthdomain.SenderConstraintDPoP || result.SenderConstraint.JKT != "jkt-1" {
		t.Fatalf("introspection sender constraint mismatch: %+v", result.SenderConstraint)
	}
}

func TestSignAccessTokenMTLSCnf(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, _, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "user-1", Scopes: []string{"account:read"},
		SenderConstraint: &oauthdomain.SenderConstraint{Type: oauthdomain.SenderConstraintMTLS, X5TS256: "thumb-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := signer.IntrospectAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if result.SenderConstraint == nil || result.SenderConstraint.Type != oauthdomain.SenderConstraintMTLS || result.SenderConstraint.X5TS256 != "thumb-1" {
		t.Fatalf("introspection sender constraint mismatch: %+v", result.SenderConstraint)
	}
}

func TestSignAccessTokenActAndAgentIntrospection(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, _, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "agent-client", Scopes: []string{"account:read"},
		AgentID: "agent-1", Act: map[string]any{"sub": "human-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := signer.IntrospectAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if result.PrincipalType != oauthdomain.PrincipalTypeAgent || result.AgentID != "agent-1" {
		t.Fatalf("agent introspection fields missing: %+v", result)
	}
	if result.Act["sub"] != "human-1" {
		t.Fatalf("act claim missing: %+v", result.Act)
	}
}

func TestIntrospectAccessTokenInactiveCases(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)

	t.Run("malformed token is inactive, not an error", func(t *testing.T) {
		result, err := signer.IntrospectAccessToken(context.Background(), "not-a-jwt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Active {
			t.Fatal("expected inactive result for malformed token")
		}
	})

	t.Run("wrong issuer is inactive", func(t *testing.T) {
		other := NewJWTSigner("https://other.test", ks)
		token, _, err := other.SignAccessToken(context.Background(), ports.AccessTokenInput{
			Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "user-1", Scopes: []string{"account:read"},
		})
		if err != nil {
			t.Fatal(err)
		}
		result, err := signer.IntrospectAccessToken(context.Background(), token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Active {
			t.Fatal("expected inactive result for issuer mismatch")
		}
	})

	t.Run("expired token is inactive", func(t *testing.T) {
		token, _, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
			Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "user-1", Scopes: []string{"account:read"},
			ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		})
		if err != nil {
			t.Fatal(err)
		}
		result, err := signer.IntrospectAccessToken(context.Background(), token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Active {
			t.Fatal("expected inactive result for expired token")
		}
	})
}

func TestNormalizeAudience(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"single string", "aud-1", []string{"aud-1"}},
		{"empty string", "", nil},
		{"string slice", []any{"a", "b"}, []string{"a", "b"}},
		{"string slice with non-string entries", []any{"a", 5, ""}, []string{"a"}},
		{"typed string slice", []string{"a", "b"}, []string{"a", "b"}},
		{"unsupported type", 42, nil},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeAudience(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("normalizeAudience(%#v) = %#v, want %#v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("normalizeAudience(%#v) = %#v, want %#v", tc.in, got, tc.want)
				}
			}
		})
	}
}

func TestVerifyIDTokenHintReturnsAudienceArray(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	key, err := ks.GetActiveKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	token, err := SignPS256(key, nil, map[string]any{
		"iss": "https://idp.test", "sub": "user-1", "aud": []any{"aud-1", "aud-2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.VerifyIDTokenHint(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if len(claims.Audiences) != 2 || claims.Audience != "aud-1" {
		t.Fatalf("unexpected audiences: %+v", claims)
	}
}
