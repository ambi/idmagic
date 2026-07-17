package usecases

// SCL シナリオ "absolute_expires_at を超えた refresh token はローテーション不可" と
// sender constraint 不一致 (DPoP / mTLS) で invalid_grant になることを担保する。

import (
	"context"
	"errors"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

type refreshFixture struct {
	deps   RefreshDeps
	token  string
	record *domain.RefreshTokenRecord
}

func newRefreshFixture(t *testing.T, sc *domain.SenderConstraint, now time.Time, ttl time.Duration) refreshFixture {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := idmmemory.NewUserRepository()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	issuer := &fakeTokenIssuer{}

	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs:             []string{"https://client.example/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	userRepo.Seed(&idmdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})

	gen, err := domain.GenerateInitialRefreshToken("client", "user", []string{"openid", "offline_access"}, sc, now)
	if err != nil {
		t.Fatal(err)
	}
	// 期限を上書きして AbsoluteExpiresAt の境界を意図した値に揃える。
	gen.Record.AbsoluteExpiresAt = now.Add(ttl)
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

func TestRefreshTokensRejectsAbsoluteTTLExpired(t *testing.T) {
	now := time.Now().UTC()
	// AbsoluteExpiresAt を過去にしてローテーション不可を観測する (ADR-004)。
	f := newRefreshFixture(t, nil, now, -time.Minute)
	_, err := RefreshTokens(context.Background(), f.deps, RefreshInput{
		ClientID: "client", RefreshToken: f.token,
	}, now)
	if err == nil {
		t.Fatal("expected absolute_expires_at rejection")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("expected invalid_grant, got %v", err)
	}
}

func TestRefreshTokensRejectsDPoPSenderConstraintMismatch(t *testing.T) {
	now := time.Now().UTC()
	sc := &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: "expected-jkt"}
	f := newRefreshFixture(t, sc, now, time.Hour)
	_, err := RefreshTokens(context.Background(), f.deps, RefreshInput{
		ClientID:     "client",
		RefreshToken: f.token,
		ProofJKT:     "different-jkt",
	}, now)
	if err == nil {
		t.Fatal("expected DPoP sender constraint rejection")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("expected invalid_grant, got %v", err)
	}
}

func TestRefreshTokensRejectsMTLSSenderConstraintMismatch(t *testing.T) {
	now := time.Now().UTC()
	sc := &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: "expected-thumbprint"}
	f := newRefreshFixture(t, sc, now, time.Hour)
	_, err := RefreshTokens(context.Background(), f.deps, RefreshInput{
		ClientID:     "client",
		RefreshToken: f.token,
		ProofX5TS256: "attacker-thumbprint",
	}, now)
	if err == nil {
		t.Fatal("expected mTLS sender constraint rejection")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("expected invalid_grant, got %v", err)
	}
}

func TestRefreshTokensAcceptsMatchingDPoPProof(t *testing.T) {
	now := time.Now().UTC()
	sc := &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: "matching-jkt"}
	f := newRefreshFixture(t, sc, now, time.Hour)
	// tenant context が無いと FindByID は default を期待するが、Seed では明示せず
	// oauth2memory.OAuth2ClientRepository が空 tenant_id でマッチするため通る。
	res, err := RefreshTokens(
		tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: f.record.TenantID, Status: tenancydomain.TenantStatusActive}, "", ""),
		f.deps,
		RefreshInput{ClientID: "client", RefreshToken: f.token, ProofJKT: "matching-jkt"},
		now,
	)
	if err != nil {
		t.Fatalf("matching proof rejected: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected rotated tokens, got %+v", res)
	}
}
