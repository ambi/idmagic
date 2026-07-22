package tokens_jose

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

func TestManagedAccessTokenUsesRFC9068ProfileAndExplicitExpiry(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	exp := time.Now().Add(24 * time.Hour).Unix()
	token, jti, err := signer.SignAccessToken(context.Background(), ports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "idmagic-api-token"}, Sub: "user-1", Scopes: []string{"account:read"},
		Audiences: []string{"https://idp.test/realms/acme"}, ExpiresAt: exp, Managed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if claims["token_use"] != "managed_api_token" || claims["client_id"] != "idmagic-api-token" || claims["jti"] != jti {
		t.Fatalf("claims=%+v", claims)
	}
	if int64(claims["exp"].(float64)) != exp {
		t.Fatalf("exp=%v", claims["exp"])
	}
	result, err := signer.IntrospectAccessToken(context.Background(), token)
	if err != nil || !result.Active || !result.Managed || result.Sub != "user-1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
