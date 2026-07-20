package usecases

// ADR-055/wi-262: refresh token ローテーションを跨いで resource indicator の
// audience 束縛を保持することを検証する。

import (
	"context"
	"testing"
	"time"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newRefreshFixtureWithResource(t *testing.T, resource *string) refreshFixture {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	issuer := &fakeTokenIssuer{}
	now := time.Now().UTC()

	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs:            []string{"https://client.example/cb"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		Scope:                   "openid offline_access",
		FapiProfile:             domain.FapiNone,
		CreatedAt:               now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})

	gen, err := domain.GenerateInitialRefreshToken("client", "user", []string{"openid", "offline_access"}, nil, nil, resource, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshStore.Save(context.Background(), gen.Record); err != nil {
		t.Fatal(err)
	}

	return refreshFixture{
		deps: RefreshDeps{
			ClientRepo: clientRepo, UserRepo: userRepo,
			RefreshStore: refreshStore, TokenIssuer: issuer,
		},
		token:  gen.Token,
		record: gen.Record,
	}
}

func TestRefreshTokens_preservesResourceBindingAcrossRotation(t *testing.T) {
	resource := "https://mcp.example.com/tools"
	f := newRefreshFixtureWithResource(t, &resource)
	now := time.Now().UTC()

	res, err := RefreshTokens(context.Background(), f.deps, RefreshInput{
		ClientID: "client", RefreshToken: f.token,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected rotated tokens, got %+v", res)
	}

	issuer := f.deps.TokenIssuer.(*fakeTokenIssuer)
	if len(issuer.lastAccessTokenInput.Audiences) != 1 || issuer.lastAccessTokenInput.Audiences[0] != resource {
		t.Fatalf("expected rotated access token audience to stay bound to %q, got %v", resource, issuer.lastAccessTokenInput.Audiences)
	}

	// ローテーション後の RefreshTokenRecord も resource を保持しているはず。
	hash := domain.HashRefreshToken(res.RefreshToken)
	rotatedRec, err := f.deps.RefreshStore.FindByHash(context.Background(), hash)
	if err != nil || rotatedRec == nil {
		t.Fatalf("expected rotated refresh token record to be findable: %v", err)
	}
	if rotatedRec.Resource == nil || *rotatedRec.Resource != resource {
		t.Fatalf("expected rotated refresh token record to carry resource, got %v", rotatedRec.Resource)
	}
}

func TestRefreshTokens_noResourceBound_unaffected(t *testing.T) {
	f := newRefreshFixtureWithResource(t, nil)
	now := time.Now().UTC()

	res, err := RefreshTokens(context.Background(), f.deps, RefreshInput{
		ClientID: "client", RefreshToken: f.token,
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	issuer := f.deps.TokenIssuer.(*fakeTokenIssuer)
	if len(issuer.lastAccessTokenInput.Audiences) != 0 {
		t.Fatalf("expected no explicit audience binding, got %v", issuer.lastAccessTokenInput.Audiences)
	}
	if res.AccessToken == "" {
		t.Fatal("access token missing")
	}
}
