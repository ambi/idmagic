package tokens_jose

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/signingkeys"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-OAUTH2-096: 一覧取得用のトークンは印を持ち、署名と期限が正しくても IdMagic の検証では無効になる。
func TestToolDiscoveryTokenIsInactiveAtIdMagic(t *testing.T) {
	ks, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	ctx := testing_tenant.Default(context.Background())
	exp := time.Now().Add(time.Minute).Unix()
	token, _, err := signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: oauthdomain.McpToolDiscoveryClientID}, Sub: oauthdomain.McpToolDiscoveryClientID,
		Audiences: []string{"https://mcp.example.com/mcp"}, ExpiresAt: exp, ToolDiscovery: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := idTokenClaims(t, token)
	if claims["token_use"] != oauthdomain.McpToolDiscoveryTokenUse || claims["aud"] != "https://mcp.example.com/mcp" || claims["scope"] != "" || int64(claims["exp"].(float64)) != exp {
		t.Fatalf("claims=%+v", claims)
	}
	result, err := signer.IntrospectAccessToken(ctx, token)
	if err != nil || result.Active {
		t.Fatalf("result=%+v err=%v; want inactive", result, err)
	}

	ordinary, _, err := signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "client-1"}, Sub: "client-1", Audiences: []string{"https://mcp.example.com/mcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := signer.IntrospectAccessToken(ctx, ordinary); err != nil || !result.Active {
		t.Fatalf("ordinary token: result=%+v err=%v", result, err)
	}
}
