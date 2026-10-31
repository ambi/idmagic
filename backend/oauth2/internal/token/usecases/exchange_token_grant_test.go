package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type grantCase struct {
	deps  ExchangeTokenDeps
	store *oauth2memory.TokenExchangeGrantStore
	now   time.Time
	input ExchangeTokenInput
}

func newGrantCase(t *testing.T, subject *ports.IntrospectionResult) *grantCase {
	t.Helper()
	now := time.Now().UTC()
	subject.Active, subject.JTI = true, "subject-jti"
	if subject.Exp == 0 {
		subject.Exp = now.Add(time.Hour).Unix()
	}
	store := oauth2memory.NewTokenExchangeGrantStore()
	deps := newExchangeTokenDeps(t, &recordingIssuer{}, map[string]*ports.IntrospectionResult{"subj": subject})
	deps.Grants = store
	c := &grantCase{deps: deps, store: store, now: now, input: ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"}, ProofJKT: "gateway-key", TokenExchangeGrant: "g",
	}}
	c.save(t, now.Add(domain.TokenExchangeGrantTTL))
	return c
}

func (c *grantCase) save(t *testing.T, expiresAt time.Time) {
	t.Helper()
	if err := c.store.Save(testing_tenant.Default(context.Background()), domain.HashTokenExchangeGrant("g"), domain.TokenExchangeGrant{
		ClientID: "client", SubjectTokenID: "subject-jti", Resource: "https://api.example", ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatal(err)
	}
}

func (c *grantCase) exchange() (*ExchangeTokenResult, error) {
	return ExchangeToken(testing_tenant.Default(context.Background()), c.deps, c.input, c.now)
}

func requireOAuthCode(t *testing.T, err error, code string) {
	t.Helper()
	if oauthErr, ok := errors.AsType[*OAuthError](err); !ok || oauthErr.Code != code {
		t.Fatalf("err=%v, want %s", err, code)
	}
}

func dpopSubject() *ports.IntrospectionResult {
	return &ports.IntrospectionResult{Sub: "user-1", ClientID: "agent-client", Scope: "read", SenderConstraint: &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: "agent-key"}}
}

//spec:covers REQ-OAUTH2-056, REQ-OAUTH2-048: 許可を伴う交換は示した鍵へ束縛し、元のクライアントを act の内側に残し、期限切れと再利用を拒否する
func TestGrantRebindsOnceToTheProvedKey(t *testing.T) {
	c := newGrantCase(t, dpopSubject())
	issuer := c.deps.TokenIssuer.(*recordingIssuer)
	result, err := c.exchange()
	if err != nil {
		t.Fatal(err)
	}
	if issuer.lastInput.SenderConstraint == nil || issuer.lastInput.SenderConstraint.JKT != "gateway-key" {
		t.Fatalf("constraint = %+v", issuer.lastInput.SenderConstraint)
	}
	if inner, _ := issuer.lastInput.Act["act"].(map[string]any); issuer.lastInput.Act["sub"] != "client" || inner["sub"] != "agent-client" {
		t.Fatalf("act = %v", issuer.lastInput.Act)
	}
	if result.ExpiresIn != 60 {
		t.Fatalf("expires_in = %d", result.ExpiresIn)
	}
	_, err = c.exchange()
	requireOAuthCode(t, err, "invalid_grant")
}

//spec:covers REQ-OAUTH2-048: 有効期間は subject_token の残りを超えない
func TestGrantDoesNotOutliveTheSubject(t *testing.T) {
	subject := dpopSubject()
	subject.Exp = time.Now().UTC().Add(10 * time.Second).Unix()
	c := newGrantCase(t, subject)
	result, err := c.exchange()
	if err != nil {
		t.Fatal(err)
	}
	if result.ExpiresIn > 10 || result.ExpiresIn <= 0 {
		t.Fatalf("expires_in = %d", result.ExpiresIn)
	}
}

//spec:covers REQ-OAUTH2-079: 期限の直前は受け、その時刻以降は拒否する
func TestGrantExpiryBoundary(t *testing.T) {
	c := newGrantCase(t, dpopSubject())
	c.save(t, c.now)
	_, err := c.exchange()
	requireOAuthCode(t, err, "invalid_grant")
	c.save(t, c.now.Add(time.Nanosecond))
	if _, err := c.exchange(); err != nil {
		t.Fatal(err)
	}
}

//spec:covers REQ-OAUTH2-056: mTLS に束縛した subject_token も許可があれば示した証明書へ束縛する
func TestGrantRebindsAnMTLSSubject(t *testing.T) {
	c := newGrantCase(t, &ports.IntrospectionResult{Sub: "user-1", Scope: "read", SenderConstraint: &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: "agent-cert"}})
	c.input.ProofJKT, c.input.ProofX5TS256 = "", "gateway-cert"
	if _, err := c.exchange(); err != nil {
		t.Fatal(err)
	}
	constraint := c.deps.TokenIssuer.(*recordingIssuer).lastInput.SenderConstraint
	if constraint == nil || constraint.Type != spec.SenderConstraintMTLS || constraint.X5TS256 != "gateway-cert" {
		t.Fatalf("constraint = %+v", constraint)
	}
}

//spec:covers REQ-OAUTH2-079: actor_token と外部ワークロードの subject_token との併用を拒否し、許可を消費しない
func TestGrantRefusesOtherActorSources(t *testing.T) {
	c := newGrantCase(t, dpopSubject())
	c.input.ActorToken = "subj"
	_, err := c.exchange()
	requireOAuthCode(t, err, "invalid_request")
	c.input.ActorToken, c.input.SubjectTokenType = "", tokenTypeJWTURN
	_, err = c.exchange()
	requireOAuthCode(t, err, "invalid_request")
	c.input.SubjectTokenType = ""
	if _, err := c.exchange(); err != nil {
		t.Fatalf("the refused requests consumed the grant: %v", err)
	}
}

//spec:covers REQ-OAUTH2-048, REQ-OAUTH2-079: 深さの上限を超える連鎖では、許可があっても発行せず、許可を消費しない
func TestGrantDoesNotBypassTheDepthLimit(t *testing.T) {
	subject := dpopSubject()
	subject.Act = map[string]any{"sub": "a", "act": map[string]any{"sub": "b", "act": map[string]any{"sub": "c"}}}
	c := newGrantCase(t, subject)
	_, err := c.exchange()
	requireOAuthCode(t, err, "invalid_request")
	subject.Act = map[string]any{"sub": "a"}
	if _, err := c.exchange(); err != nil {
		t.Fatalf("the refused request consumed the grant: %v", err)
	}
}

//spec:covers REQ-OAUTH2-079: 未配線の保存先で許可を受けない
func TestGrantFailsClosedWithoutAStore(t *testing.T) {
	c := newGrantCase(t, dpopSubject())
	c.deps.Grants = nil
	_, err := c.exchange()
	requireOAuthCode(t, err, "invalid_grant")
}

//spec:covers REQ-OAUTH2-060: 本人の付与がない Agent のトークンは許可があっても交換できず、許可を消費しない
func TestGrantRequiresTheUsersDelegation(t *testing.T) {
	subject := dpopSubject()
	subject.AgentID = "A1"
	c := newGrantCase(t, subject)
	c.deps.AgentRepo = nil
	c.deps.Consents = oauth2memory.NewConsentRepository()
	_, err := c.exchange()
	requireOAuthCode(t, err, "invalid_grant")
	claimed, err := c.store.Claim(testing_tenant.Default(context.Background()), domain.HashTokenExchangeGrant("g"), domain.TokenExchangeGrant{
		ClientID: "client", SubjectTokenID: "subject-jti", Resource: "https://api.example",
	}, c.now)
	if err != nil || !claimed {
		t.Fatalf("the refused request consumed the grant: claimed=%v err=%v", claimed, err)
	}
}
