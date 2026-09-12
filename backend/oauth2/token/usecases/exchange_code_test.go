package usecases

// 主要ユースケース追跡: REQ-OAUTH2-005。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	logoutmemory "github.com/ambi/idmagic/backend/oauth2/logout/db_memory"

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
	events       *[]spec.DomainEvent
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
	// 具体例が `Then` にイベントの発行を並べているので、fixture は発行されたイベントを
	// 保持する。応答だけを読むと、失効の記録は残しつつ通知を出さない実装を見分けられない。
	events := &[]spec.DomainEvent{}
	return exchangeFixture{
		deps: ExchangeCodeDeps{
			ClientRepo: clientRepo, UserRepo: userRepo, CodeStore: codeStore,
			RefreshStore: refreshStore, TokenIssuer: issuer,
			Emit: func(event spec.DomainEvent) { *events = append(*events, event) },
		},
		codeStore: codeStore, refreshStore: refreshStore, code: code, issuer: issuer,
		events: events,
	}
}

// emitted は発行されたイベント型の一覧を返す。
func (f exchangeFixture) emitted() []string {
	types := make([]string, 0, len(*f.events))
	for _, event := range *f.events {
		types = append(types, event.EventType())
	}
	return types
}

func exchangeInput(verifier string) ExchangeCodeInput {
	return ExchangeCodeInput{
		ClientID: "client", Code: "authorization-code",
		CodeVerifier: verifier, RedirectURI: "https://client.example/cb",
	}
}

// EX-OAUTH2-005-06: 認可コードを誤った code_verifier で交換すると InvalidGrantError で
// 拒否され、トークンは 1 本も発行されない。認可コードは消費されないので、正しい verifier
// なら後から交換できる。
//
// 拒否の型まで読むのは、`invalid_request` で落ちる実装 — 例えば verifier の長さ検査に
// 先に引っかかる実装 — と区別するためである。「トークンは発行されない」は応答の 3 本と
// イベントの双方で読む。エラーを返しつつ署名器を呼ぶ実装は、応答だけでは見分けられない。
func TestExchangeCodePKCEFailureDoesNotConsumeCode(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	refused, err := ExchangeCodeForToken(context.Background(), f.deps, exchangeInput("wrong-verifier"))
	if err == nil {
		t.Fatal("expected PKCE failure")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("PKCE 不一致の拒否が invalid_grant ではない: %v", err)
	}
	if refused != nil {
		t.Fatalf("拒否された交換がトークンを返した: %+v", refused)
	}
	if emitted := f.emitted(); len(emitted) != 0 {
		t.Fatalf("拒否された交換がイベントを発行した: %v", emitted)
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

// 認可コードの再交換は invalid_grant で拒否され、発行ファミリーのトークンは失効する。
//
// 通知のうち TokenRevoked はまだ出ない。RevokeFamily が失効させた token を返さないためで、
// 台帳の該当行に finding として記録し、wi-566 が引き取る。ここでは失効の記録と
// RefreshTokenReuseDetected の発行までを固定する。
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
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatalf("1 回目がトークンを返していない: %+v", out)
	}
	_, err = ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	)
	if err == nil {
		t.Fatal("expected replay rejection")
	}
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Fatalf("再交換の拒否が invalid_grant ではない: %v", err)
	}
	rec, err := f.refreshStore.FindByHash(context.Background(), domain.HashRefreshToken(out.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || !rec.Revoked {
		t.Fatal("refresh family was not revoked")
	}
	if emitted := f.emitted(); !slices.Contains(emitted, "RefreshTokenReuseDetected") {
		t.Fatalf("再交換の検出で RefreshTokenReuseDetected が発行されていない: %v", emitted)
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
	// AuthorizationCodeRecord.sid は発行 RefreshTokenRecord.sid と
	// id_token の sid claim にそのまま引き継がれる。
	f := newExchangeFixture(t, []string{"openid", "offline_access"})
	sid := "10000000-0000-4000-8000-000000000001"
	f.code.Sid = &sid
	clientSessions := logoutmemory.NewClientSessionStore()
	f.deps.ClientSessionStore = clientSessions
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
	participations, err := clientSessions.ListBySid(context.Background(), f.code.TenantID, sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(participations) != 1 || participations[0].ClientID != f.code.ClientID || participations[0].FirstIssuedAt.IsZero() || participations[0].LastIssuedAt.IsZero() {
		t.Fatalf("unexpected client session: %+v", participations)
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
	// 認可コードは一度だけ交換できる。再利用が通ると、盗まれたコードから
	// 何度でもトークンを取れてしまう。
	if _, err := ExchangeCodeForToken(
		context.Background(),
		f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
	); err == nil {
		t.Fatal("replaying the authorization code must be rejected")
	}
}
