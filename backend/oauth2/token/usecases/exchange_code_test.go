package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type fakeTokenIssuer struct {
	idTokenCalls         int
	lastIDTokenInput     ports.IDTokenInput
	lastAccessTokenInput ports.AccessTokenInput
}

func (f *fakeTokenIssuer) SignAccessToken(_ context.Context, in ports.AccessTokenInput) (string, string, error) {
	f.lastAccessTokenInput = in
	return "access-token", "jti-1", nil
}

func (f *fakeTokenIssuer) SignIDToken(_ context.Context, in ports.IDTokenInput) (string, error) {
	f.idTokenCalls++
	f.lastIDTokenInput = in
	return "id-token", nil
}

func (f *fakeTokenIssuer) AccessTokenTTLSeconds() int { return 600 }
func (f *fakeTokenIssuer) IDTokenTTLSeconds() int     { return 3600 }

type exchangeFixture struct {
	deps         ExchangeCodeDeps
	codeStore    *oauth2memory.AuthorizationCodeStore
	refreshStore *oauth2memory.RefreshTokenStore
	code         *domain.AuthorizationCodeRecord
	issuer       *fakeTokenIssuer
}

func newExchangeFixture(t *testing.T, scopes []string) exchangeFixture {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	codeStore := oauth2memory.NewAuthorizationCodeStore()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	issuer := &fakeTokenIssuer{}

	now := time.Now().UTC()
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs: []string{"https://client.example/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
		ResponseTypes: []spec.ResponseType{
			spec.ResponseTypeCode,
		},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})

	verifier := "verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sum := sha256.Sum256([]byte(verifier))
	code := &domain.AuthorizationCodeRecord{
		Code:                   "authorization-code",
		AuthorizationRequestID: "00000000-0000-4000-8000-000000000001",
		ClientID:               "client",
		UserID:                 "user",
		Scopes:                 scopes,
		RedirectURI:            "https://client.example/cb",
		CodeChallenge:          base64.RawURLEncoding.EncodeToString(sum[:]),
		CodeChallengeMethod:    spec.CodeChallengeMethodS256,
		AuthTime:               now.Unix(),
		State:                  spec.AuthCodeRecordIssued,
		IssuedAt:               now,
		ExpiresAt:              now.Add(time.Minute),
	}
	if err := codeStore.Save(context.Background(), code); err != nil {
		t.Fatal(err)
	}
	return exchangeFixture{
		deps: ExchangeCodeDeps{
			ClientRepo: clientRepo, UserRepo: userRepo, CodeStore: codeStore,
			RefreshStore: refreshStore, TokenIssuer: issuer,
		},
		codeStore: codeStore, refreshStore: refreshStore, code: code, issuer: issuer,
	}
}

func exchangeInput(verifier string) ExchangeCodeInput {
	return ExchangeCodeInput{
		ClientID: "client", Code: "authorization-code",
		CodeVerifier: verifier, RedirectURI: "https://client.example/cb",
	}
}

func TestExchangeCodePKCEFailureDoesNotConsumeCode(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	if _, err := ExchangeCodeForToken(context.Background(), f.deps, exchangeInput("wrong-verifier")); err == nil {
		t.Fatal("expected PKCE failure")
	}

	out, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err != nil {
		t.Fatalf("valid retry failed: %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("access token missing")
	}
}

func TestExchangeCodeReplayRevokesRefreshFamily(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid", "offline_access"})
	out, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if out.RefreshToken == "" {
		t.Fatal("refresh token missing")
	}
	if _, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	); err == nil {
		t.Fatal("expected replay rejection")
	}
	rec, err := f.refreshStore.FindByHash(context.Background(), domain.HashRefreshToken(out.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || !rec.Revoked {
		t.Fatal("refresh family was not revoked")
	}
}

func TestExchangeCodeRejectsExpiredCode(t *testing.T) {
	// SCL invariant AuthorizationCodeTtl (60s)。expires_at を過去にしたコードは
	// invalid_grant で拒否され、family があれば失効する (RFC 9700 §4.10)。
	f := newExchangeFixture(t, []string{"openid"})
	f.code.IssuedAt = time.Now().Add(-90 * time.Second).UTC()
	f.code.ExpiresAt = time.Now().Add(-30 * time.Second).UTC()
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}
	_, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err == nil {
		t.Fatal("expected invalid_grant for expired code")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExchangeCodePropagatesNonceToIDToken(t *testing.T) {
	// OIDC Core §3.1.2.1。認可リクエストの nonce は ID トークンに伝播する。
	f := newExchangeFixture(t, []string{"openid"})
	nonce := "n-12345"
	f.code.Nonce = &nonce
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}
	out, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.IDToken == "" {
		t.Fatal("id_token must be issued for openid scope")
	}
	if f.issuer.lastIDTokenInput.Nonce == nil || *f.issuer.lastIDTokenInput.Nonce != nonce {
		t.Fatalf("nonce not propagated to id_token: got %v", f.issuer.lastIDTokenInput.Nonce)
	}
}

func TestExchangeCodePropagatesSidToRefreshTokenAndIDToken(t *testing.T) {
	// ADR-127: AuthorizationCodeRecord.sid は発行 RefreshTokenRecord.sid と
	// id_token の sid claim にそのまま引き継がれる。
	f := newExchangeFixture(t, []string{"openid", "offline_access"})
	sid := "session-1"
	f.code.Sid = &sid
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}
	out, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if f.issuer.lastIDTokenInput.Sid != sid {
		t.Fatalf("sid not propagated to id_token: got %q", f.issuer.lastIDTokenInput.Sid)
	}
	rec, err := f.refreshStore.FindByHash(context.Background(), domain.HashRefreshToken(out.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.Sid == nil || *rec.Sid != sid {
		t.Fatalf("sid not propagated to refresh token record: got %v", rec)
	}
}

func TestExchangeCodeIssuesTokensByScope(t *testing.T) {
	f := newExchangeFixture(t, []string{"profile"})
	out, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if out.IDToken != "" || f.issuer.idTokenCalls != 0 {
		t.Fatal("id_token must require openid scope")
	}
	if out.RefreshToken != "" {
		t.Fatal("refresh_token must require offline_access scope")
	}
}
