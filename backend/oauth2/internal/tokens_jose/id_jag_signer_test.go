package tokens_jose

import (
	"context"
	"strings"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-OAUTH2-067, IDJAG-ISSUANCE: 自身が署名した ID Token だけを受け、アクセストークン、ID-JAG、ログアウトトークン、token_use のない JWT を受けないこと
func TestVerifyIDTokenAssertion_REQ_OAUTH2_067(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	store, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.example", store)
	client := &domain.OAuth2Client{ClientID: "client"}
	user := &userdomain.User{ID: "user-1", TenantID: "t"}

	idToken, err := signer.SignIDToken(ctx, oauthports.IDTokenInput{Client: client, User: user, AuthTime: 7, AMR: []string{"pwd"}, ACR: "acr"})
	if err != nil {
		t.Fatal(err)
	}
	assertion, err := signer.VerifyIDTokenAssertion(ctx, idToken)
	if err != nil {
		t.Fatal(err)
	}
	if assertion.Subject != "user-1" || len(assertion.Audiences) != 1 || assertion.Audiences[0] != "client" || assertion.AuthTime != 7 || assertion.ACR != "acr" {
		t.Fatalf("assertion = %+v", assertion)
	}

	key, err := store.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	baseClaims := func() map[string]any {
		return map[string]any{
			"iss": tenantports.Issuer(ctx, signer.Issuer), "sub": "user-1", "aud": "client",
			"exp": time.Now().Add(time.Hour).Unix(), "token_use": "id_token",
		}
	}
	sign := func(t *testing.T, header map[string]string, claims map[string]any) string {
		t.Helper()
		token, err := SignPS256(key, header, claims)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	accessToken, _, err := signer.SignAccessToken(ctx, oauthports.AccessTokenInput{Client: client, Sub: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	idJag, _, err := signer.SignIDJAG(ctx, oauthports.IDJAGInput{Subject: "user-1", Audience: "https://other.example/", ClientID: "client"})
	if err != nil {
		t.Fatal(err)
	}
	logoutToken, err := signer.SignLogoutToken(ctx, logoutports.LogoutTokenInput{
		Issuer: tenantports.Issuer(ctx, signer.Issuer), Subject: "user-1", Audience: "client", JTI: "j", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// 対照: 拒否の表の JWT が、狙った条件以外では受理される形であることを確かめる。
	if _, err := signer.VerifyIDTokenAssertion(ctx, sign(t, nil, baseClaims())); err != nil {
		t.Fatalf("control: %v", err)
	}
	// 既存の鍵で署名した、種類の印のない JWT は ID Token として受けない。
	withoutUse := baseClaims()
	delete(withoutUse, "token_use")
	expired := baseClaims()
	expired["exp"] = time.Now().Add(-time.Second).Unix()
	otherIssuer := baseClaims()
	otherIssuer["iss"] = "https://evil.example"
	for name, token := range map[string]string{
		"access token":   accessToken,
		"ID-JAG":         idJag,
		"logout token":   logoutToken,
		"typed id token": sign(t, map[string]string{"typ": "JWT"}, baseClaims()),
		"no token_use":   sign(t, nil, withoutUse),
		"expired":        sign(t, nil, expired),
		"another issuer": sign(t, nil, otherIssuer),
		"malformed":      "a.b",
		"tampered":       idToken[:strings.LastIndex(idToken, ".")] + ".AAAA",
	} {
		if _, err := signer.VerifyIDTokenAssertion(ctx, token); err == nil {
			t.Errorf("%s was accepted as an ID Token", name)
		}
	}
}

//spec:covers REQ-SIGNINGKEYS-013, IDJAG-SIGNING-ALG: ID-JAG は専用鍵の RS256 で署名し、アクセストークンの検証を通らないこと
func TestSignIDJAGUsesDedicatedRS256Key_REQ_SIGNINGKEYS_013(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	store, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := NewJWTSigner("https://idp.example", store)
	token, _, err := signer.SignIDJAG(ctx, oauthports.IDJAGInput{Subject: "user-1", Audience: "https://other.example/", ClientID: "client"})
	if err != nil {
		t.Fatal(err)
	}
	key, err := store.GetActiveKey(signingports.WithKeyUsage(ctx, signingdomain.KeyUsageIdJagSigning))
	if err != nil {
		t.Fatal(err)
	}
	if key.Alg != signingdomain.SigAlgRS256 {
		t.Fatalf("alg = %s", key.Alg)
	}
	res, err := signer.IntrospectAccessToken(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if res.Active {
		t.Fatal("an ID-JAG passed access token introspection")
	}
}
