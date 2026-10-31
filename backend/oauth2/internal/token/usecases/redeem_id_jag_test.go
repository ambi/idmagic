package usecases

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

const (
	redeemIssuer   = "https://acme.idp.example/"
	redeemSelf     = "https://acme.chat.example/"
	redeemResource = "https://api.chat.example/"
	redeemClient   = "f53f191f9311af35"
	redeemExternal = "U019488227"
)

var (
	redeemKey, _ = rsa.GenerateKey(rand.Reader, 2048)
	otherKey, _  = rsa.GenerateKey(rand.Reader, 2048)
)

type stubIssuerTrust struct {
	jwks     []map[string]any
	subjects map[string]string
}

func (s stubIssuerTrust) ResolveIssuer(_ context.Context, _, issuer string) (*ports.IDJAGTrustedIssuer, error) {
	if issuer != redeemIssuer {
		return nil, nil //nolint:nilnil // 信頼しない発行者を表す。
	}
	return &ports.IDJAGTrustedIssuer{ProviderID: "idp", JWKS: s.jwks}, nil
}

func (s stubIssuerTrust) ProviderExists(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s stubIssuerTrust) ResolveSubject(_ context.Context, _, _, subject string) (string, error) {
	return s.subjects[subject], nil
}

type idJagRedeemFixture struct {
	deps   RedeemIDJAGDeps
	issuer *recordingIssuer
	events []spec.DomainEvent
	in     RedeemIDJAGInput
	now    time.Time
}

func redeemClaims(now time.Time) map[string]any {
	return map[string]any{
		"jti": "9e43f81b64a33f20116179", "iss": redeemIssuer, "sub": redeemExternal, "aud": redeemSelf,
		"client_id": redeemClient, "iat": now.Unix(), "exp": now.Add(300 * time.Second).Unix(),
		"resource": redeemResource, "scope": "chat.read chat.history", "auth_time": now.Unix() - 10, "amr": []string{"mfa"},
	}
}

func signRedeem(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	token, err := tokens_jose.SignRS256(key, "k1", map[string]string{"typ": "oauth-id-jag+jwt"}, claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func newIDJAGRedeemFixture(t *testing.T) *idJagRedeemFixture {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	now := time.Now().UTC().Truncate(time.Second)
	clients := oauth2memory.NewClientRepository()
	clients.Seed(&domain.OAuth2Client{ClientID: redeemClient, ClientType: spec.ClientConfidential, Scope: "chat.read chat.history", CreatedAt: now})
	clients.Seed(&domain.OAuth2Client{ClientID: "other-client", ClientType: spec.ClientConfidential, Scope: "chat.read", CreatedAt: now})
	clients.Seed(&domain.OAuth2Client{ClientID: "public-client", ClientType: spec.ClientPublic, Scope: "chat.read", CreatedAt: now})
	users := idmanagement.NewMemoryModule().UserRepo
	for id, status := range map[string]idmdomain.UserStatus{"local-1": idmdomain.UserStatusActive, "local-suspended": idmdomain.UserStatusDisabled} {
		if err := users.Save(ctx, &userdomain.User{ID: id, TenantID: kernel.DefaultTenantID, PreferredUsername: id, Lifecycle: userdomain.UserLifecycle{Status: status}}); err != nil {
			t.Fatal(err)
		}
	}
	policies := oauth2memory.NewCrossAppAccessPolicyRepository()
	if err := policies.CreateRedeemPolicy(ctx, &domain.CrossAppAccessRedeemPolicy{
		TenantID: kernel.DefaultTenantID, ID: "p1", ProviderID: "idp", ClientID: redeemClient, Scopes: []string{"chat.read", "chat.history"},
	}); err != nil {
		t.Fatal(err)
	}
	f := &idJagRedeemFixture{issuer: &recordingIssuer{}, now: now}
	f.deps = RedeemIDJAGDeps{
		ClientRepo: clients, UserRepo: users, Policies: policies,
		Trust: stubIssuerTrust{
			jwks: []map[string]any{{
				"kty": "RSA", "kid": "k1", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(redeemKey.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(redeemKey.E)).Bytes()),
			}},
			subjects: map[string]string{redeemExternal: "local-1", "suspended": "local-suspended", "ghost": "local-ghost"},
		},
		ResourceServers: newFakeMcpResourceServerRepo(
			&domain.McpResourceServer{TenantID: kernel.DefaultTenantID, ID: "r1", Resource: redeemResource, Name: "chat", Scopes: []string{"chat.read", "chat.history"}, State: domain.McpResourceServerActive},
			&domain.McpResourceServer{TenantID: kernel.DefaultTenantID, ID: "r2", Resource: "https://disabled.example/", Name: "off", Scopes: []string{"chat.read"}, State: domain.McpResourceServerDisabled},
		),
		Replay: oauth2memory.NewClientAssertionReplayStore(), TokenIssuer: f.issuer,
		Emit: func(event spec.DomainEvent) { f.events = append(f.events, event) },
	}
	f.in = RedeemIDJAGInput{
		ClientID: redeemClient, GrantType: spec.GrantJWTBearer, SelfIssuer: redeemSelf,
		Assertion: signRedeem(t, redeemKey, redeemClaims(now)),
	}
	return f
}

func (f *idJagRedeemFixture) redeem() (*RedeemIDJAGResult, error) {
	return RedeemIDJAG(testing_tenant.Default(context.Background()), f.deps, f.in, f.now)
}

//spec:covers REQ-OAUTH2-069, REQ-OAUTH2-070, IDJAG-REDEMPTION: 償還のトークンの主体、audience、スコープ、イベント
func TestRedeemIDJAG_REQ_OAUTH2_069(t *testing.T) {
	f := newIDJAGRedeemFixture(t)
	res, err := f.redeem()
	if err != nil {
		t.Fatal(err)
	}
	if res.TokenType != "Bearer" || res.Scope != "chat.read chat.history" {
		t.Fatalf("result = %+v", res)
	}
	got := f.issuer.lastInput
	if got.Sub != "local-1" || len(got.Audiences) != 1 || got.Audiences[0] != redeemResource || got.Client.ClientID != redeemClient || got.Act != nil {
		t.Fatalf("token input = %+v", got)
	}
	var redeemed *domain.IdJagRedeemed
	for _, event := range f.events {
		if e, ok := event.(*domain.IdJagRedeemed); ok {
			redeemed = e
		}
	}
	if redeemed == nil || redeemed.Issuer != redeemIssuer || redeemed.UserID != "local-1" {
		t.Fatalf("events = %+v", f.events)
	}
}

//spec:covers REQ-OAUTH2-070: スコープはポリシー、クライアント、リソースのすべてに含まれるものに限ること
func TestRedeemIDJAGNarrowsScopes_REQ_OAUTH2_070(t *testing.T) {
	f := newIDJAGRedeemFixture(t)
	claims := redeemClaims(f.now)
	claims["scope"] = "chat.read admin"
	f.in.Assertion = signRedeem(t, redeemKey, claims)
	res, err := f.redeem()
	if err != nil {
		t.Fatal(err)
	}
	if res.Scope != "chat.read" {
		t.Fatalf("scope = %q", res.Scope)
	}
}

//spec:covers REQ-OAUTH2-069, IDJAG-REDEMPTION: 偽造、受け手違い、期限切れ、自己発行、クライアント違い、行為者、未統治のリソース、許可なしの拒否と無作用
func TestRedeemIDJAGRefusals_REQ_OAUTH2_069(t *testing.T) {
	withClaim := func(t *testing.T, f *idJagRedeemFixture, mutate func(map[string]any)) {
		t.Helper()
		claims := redeemClaims(f.now)
		mutate(claims)
		f.in.Assertion = signRedeem(t, redeemKey, claims)
	}
	cases := map[string]struct {
		mutate func(t *testing.T, f *idJagRedeemFixture)
		code   string
	}{
		"forged": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			f.in.Assertion = signRedeem(t, otherKey, redeemClaims(f.now))
		}, "invalid_grant"},
		"untrusted issuer": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["iss"] = "https://evil.example/" })
		}, "invalid_grant"},
		"self issued": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			f.in.SelfIssuer = redeemIssuer
			withClaim(t, f, func(c map[string]any) { c["aud"] = redeemIssuer })
		}, "invalid_grant"},
		"another audience": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["aud"] = "https://other.example/" })
		}, "invalid_grant"},
		"expired": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) {
				c["iat"] = f.now.Add(-400 * time.Second).Unix()
				c["exp"] = f.now.Add(-100 * time.Second).Unix()
			})
		}, "invalid_grant"},
		"another client": {func(t *testing.T, f *idJagRedeemFixture) { t.Helper(); f.in.ClientID = "other-client" }, "invalid_grant"},
		"act present": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["act"] = map[string]any{"sub": "agent"} })
		}, "invalid_grant"},
		"no resource": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { delete(c, "resource") })
		}, "invalid_grant"},
		"unregistered resource": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["resource"] = "https://attacker.example/" })
		}, "invalid_grant"},
		"disabled resource": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["resource"] = "https://disabled.example/" })
		}, "invalid_grant"},
		"public client": {func(t *testing.T, f *idJagRedeemFixture) { t.Helper(); f.in.ClientID = "public-client" }, "invalid_client"},
		"no assertion":  {func(t *testing.T, f *idJagRedeemFixture) { t.Helper(); f.in.Assertion = "" }, "invalid_request"},
		"no redeem policy": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			f.deps.Policies = oauth2memory.NewCrossAppAccessPolicyRepository()
		}, "access_denied"},
		"unlinked subject": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["sub"] = "stranger" })
		}, "access_denied"},
		"suspended user": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["sub"] = "suspended" })
		}, "access_denied"},
		"no grantable scope": {func(t *testing.T, f *idJagRedeemFixture) {
			t.Helper()
			withClaim(t, f, func(c map[string]any) { c["scope"] = "admin" })
		}, "access_denied"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newIDJAGRedeemFixture(t)
			tc.mutate(t, f)
			_, err := f.redeem()
			assertOAuthError(t, err, tc.code)
			if f.issuer.calls != 0 {
				t.Fatal("a refused request must not issue a token")
			}
			if len(f.events) != 1 {
				t.Fatalf("events = %+v", f.events)
			}
			if rejected, ok := f.events[0].(*domain.IdJagRejected); !ok || rejected.Reason != tc.code || rejected.Stage != "redeem" {
				t.Fatalf("event = %+v", f.events[0])
			}
		})
	}
	t.Run("replay is refused", func(t *testing.T) {
		f := newIDJAGRedeemFixture(t)
		if _, err := f.redeem(); err != nil {
			t.Fatal(err)
		}
		_, err := f.redeem()
		assertOAuthError(t, err, "invalid_grant")
		if f.issuer.calls != 1 {
			t.Fatalf("issued %d tokens", f.issuer.calls)
		}
	})
}

//spec:covers REQ-OAUTH2-071, IDJAG-SENDER-CONSTRAINT: cnf と grant_type と証明の組み合わせの表、proof の失敗で jti を消費しないこと
func TestRedeemIDJAGSenderConstraint_REQ_OAUTH2_071(t *testing.T) {
	jwtDPoP := spec.GrantType("urn:ietf:params:oauth:grant-type:jwt-dpop")
	jwtBearer := spec.GrantType("urn:ietf:params:oauth:grant-type:jwt-bearer")
	cases := map[string]struct {
		cnf      map[string]any
		grant    spec.GrantType
		jkt, x5t string
		code     string
		wantJKT  string
		wantX5T  string
	}{
		"bound dpop same key":           {cnf: map[string]any{"jkt": "k"}, grant: jwtDPoP, jkt: "k", wantJKT: "k"},
		"bound dpop another key":        {cnf: map[string]any{"jkt": "k"}, grant: jwtDPoP, jkt: "z", code: "invalid_grant"},
		"bound dpop no proof":           {cnf: map[string]any{"jkt": "k"}, grant: jwtDPoP, code: "invalid_dpop_proof"},
		"bound dpop downgraded":         {cnf: map[string]any{"jkt": "k"}, grant: jwtBearer, code: "invalid_grant"},
		"bound dpop downgraded w/proof": {cnf: map[string]any{"jkt": "k"}, grant: jwtBearer, jkt: "k", code: "invalid_grant"},
		"bound mtls same cert":          {cnf: map[string]any{"x5t#S256": "c"}, grant: jwtBearer, x5t: "c", wantX5T: "c"},
		"bound mtls another cert":       {cnf: map[string]any{"x5t#S256": "c"}, grant: jwtBearer, x5t: "z", code: "invalid_grant"},
		"bound mtls no cert":            {cnf: map[string]any{"x5t#S256": "c"}, grant: jwtBearer, code: "invalid_grant"},
		"bound mtls by jwt-dpop":        {cnf: map[string]any{"x5t#S256": "c"}, grant: jwtDPoP, jkt: "k", x5t: "c", code: "invalid_grant"},
		"unsupported method":            {cnf: map[string]any{"jwk": "x"}, grant: jwtBearer, code: "invalid_grant"},
		"unbound with dpop":             {grant: jwtBearer, jkt: "k", wantJKT: "k"},
		"unbound with mtls":             {grant: jwtBearer, x5t: "c", wantX5T: "c"},
		"unbound bearer":                {grant: jwtBearer},
		"unbound by jwt-dpop":           {grant: jwtDPoP, jkt: "k", code: "invalid_grant"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newIDJAGRedeemFixture(t)
			claims := redeemClaims(f.now)
			if tc.cnf != nil {
				claims["cnf"] = tc.cnf
			}
			f.in.Assertion = signRedeem(t, redeemKey, claims)
			f.in.GrantType, f.in.ProofJKT, f.in.ProofX5TS256 = tc.grant, tc.jkt, tc.x5t
			_, err := f.redeem()
			if tc.code != "" {
				assertOAuthError(t, err, tc.code)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			sc := f.issuer.lastInput.SenderConstraint
			if tc.wantJKT == "" && tc.wantX5T == "" {
				if sc != nil {
					t.Fatalf("constraint = %+v", sc)
				}
				return
			}
			if sc == nil || sc.JKT != tc.wantJKT || sc.X5TS256 != tc.wantX5T {
				t.Fatalf("constraint = %+v", sc)
			}
		})
	}
	t.Run("a failed proof does not consume the jti", func(t *testing.T) {
		f := newIDJAGRedeemFixture(t)
		claims := redeemClaims(f.now)
		claims["cnf"] = map[string]any{"jkt": "k"}
		f.in.Assertion = signRedeem(t, redeemKey, claims)
		f.in.GrantType = jwtDPoP
		_, err := f.redeem()
		assertOAuthError(t, err, "invalid_dpop_proof")
		f.in.ProofJKT = "k"
		if _, err := f.redeem(); err != nil {
			t.Fatalf("retry after a proof failure: %v", err)
		}
	})
}
