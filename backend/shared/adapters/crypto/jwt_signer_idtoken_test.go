package crypto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/adapters/crypto"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/oauth2/ports"
)

func idTokenClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("malformed jwt: %q", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

func idTokenTestUser() *idmdomain.User {
	name := "Carol Q"
	nick := "cici"
	phone := "+819012345678"
	return &idmdomain.User{
		ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "carol", Name: &name,
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]idmdomain.AttributeValue{
			"nickname":     {Type: idmdomain.AttributeTypeString, String: &nick},
			"phone_number": {Type: idmdomain.AttributeTypeString, String: &phone},
		},
	}
}

func TestSignIDTokenIncludesAttributeClaimsByScope(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	resolve := func(_ context.Context, _ string) ([]idmdomain.UserAttributeDef, error) {
		return idmdomain.BuiltinUserAttributeDefs(), nil
	}

	token, err := signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, User: idTokenTestUser(),
		Scopes: []string{"openid", "profile", "phone"}, ResolveAttributeDefs: resolve,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if claims["name"] != "Carol Q" {
		t.Fatalf("standard profile claim missing: %#v", claims)
	}
	if claims["nickname"] != "cici" {
		t.Fatalf("nickname attribute claim missing: %#v", claims)
	}
	if claims["phone_number"] != "+819012345678" {
		t.Fatalf("phone_number attribute claim missing: %#v", claims)
	}
}

func TestSignIDTokenOmitsAttributeClaimsWithoutScope(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	resolve := func(_ context.Context, _ string) ([]idmdomain.UserAttributeDef, error) {
		return idmdomain.BuiltinUserAttributeDefs(), nil
	}

	token, err := signer.SignIDToken(context.Background(), ports.IDTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, User: idTokenTestUser(),
		Scopes: []string{"openid"}, ResolveAttributeDefs: resolve,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if _, ok := claims["nickname"]; ok {
		t.Fatalf("nickname leaked without profile scope: %#v", claims)
	}
	if _, ok := claims["phone_number"]; ok {
		t.Fatalf("phone_number leaked without phone scope: %#v", claims)
	}
}
