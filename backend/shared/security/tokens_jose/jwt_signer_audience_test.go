package tokens_jose

import (
	"context"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

// resource を指定しない発行は、scope からデフォルトの資源を推定して aud に入れる (RFC 9068 §3)。
// レルムの IdMagic API の識別子は iss と同じレルムの発行者識別子なので、署名器のデフォルト値
// ではなくテナント文脈の発行者識別子が aud に入ることを読む。
//
//spec:covers REQ-OAUTH2-001, RFC9068-DEFAULT-AUDIENCE: resource を指定しない発行では、account スコープを含めば aud がテナント文脈の発行者識別子に、含まなければ client_id になり、resource の指定は推定より優先する。
func TestAccessTokenWithoutResourceInfersAudienceFromScope(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.test", ks)
	const realmAPI = "https://idp.test/realms/acme"
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, realmAPI, "/realms/acme")

	for _, tc := range []struct {
		name      string
		scopes    []string
		audiences []string
		want      any
	}{
		{name: "account スコープはレルムの IdMagic API", scopes: []string{"openid", "account:read"}, want: realmAPI},
		{name: "細かい account スコープも同じ", scopes: []string{"account:mfa:write"}, want: realmAPI},
		{name: "account スコープが無ければ client_id", scopes: []string{"openid", "idmagic.account"}, want: "c1"},
		{name: "resource の指定は推定より優先する", scopes: []string{"account:read"}, audiences: []string{"https://mcp.example"}, want: "https://mcp.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, _, err := signer.SignAccessToken(ctx, ports.AccessTokenInput{
				Client: &oauthdomain.OAuth2Client{ClientID: "c1"}, Sub: "user-1",
				Scopes: tc.scopes, Audiences: tc.audiences,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := idTokenClaims(t, token)["aud"]; got != tc.want {
				t.Fatalf("aud=%#v, want %#v", got, tc.want)
			}
		})
	}
}
