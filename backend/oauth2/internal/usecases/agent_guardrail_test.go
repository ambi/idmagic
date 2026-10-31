package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	ratememory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type guardrailSigner struct {
	ports.TokenIssuer
	signed []ports.AccessTokenInput
}

func (s *guardrailSigner) SignAccessToken(_ context.Context, in ports.AccessTokenInput) (string, string, error) {
	s.signed = append(s.signed, in)
	return "token", "jti-1", nil
}
func (s *guardrailSigner) AccessTokenTTLSeconds() int { return 600 }

type failingGuardrailPolicies struct {
	ports.AgentGuardrailPolicyRepository
}

func (failingGuardrailPolicies) ListEffective(context.Context, string, []string) ([]*domain.AgentGuardrailPolicy, error) {
	return nil, errors.New("store down")
}

type failingLimiter struct{ rlports.RateLimiter }

func (failingLimiter) AllowWithin(context.Context, string, string, rlports.RateLimitPolicyConfig, time.Time) (rlports.RateLimitResult, error) {
	return rlports.RateLimitResult{}, errors.New("store down")
}

type guardrailFixture struct {
	ctx      context.Context
	now      time.Time
	policies *oauth2memory.AgentGuardrailPolicyRepository
	signer   *guardrailSigner
	events   []*domain.AgentGuardrailEvaluated
	deps     sharedusecases.AgentGuardrailDeps
}

func newGuardrailFixture(t *testing.T) *guardrailFixture {
	t.Helper()
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	ctx := testing_tenant.Default(context.Background())
	module := idmanagement.NewMemoryModule()
	for _, id := range []string{"alice", "bob"} {
		if err := module.UserRepo.Save(ctx, &userdomain.User{ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id, PasswordHash: "x", CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"a1", "a2"} {
		if err := module.AgentRepo.Save(ctx, &agentdomain.Agent{ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "alice", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	f := &guardrailFixture{ctx: ctx, now: now, policies: oauth2memory.NewAgentGuardrailPolicyRepository(), signer: &guardrailSigner{}}
	f.deps = sharedusecases.AgentGuardrailDeps{
		Policies: f.policies, Limiter: ratememory.NewRateLimiter(rlports.RateLimitConfigs{}), AgentRepo: module.AgentRepo, UserRepo: module.UserRepo,
		ClientID: "agent-app", GrantType: "client_credentials", RequestID: "req-1", RequestingAgentID: "a1", Now: now,
		Emit: func(event spec.DomainEvent) {
			if evaluated, ok := event.(*domain.AgentGuardrailEvaluated); ok {
				f.events = append(f.events, evaluated)
			}
		},
	}
	return f
}

func (f *guardrailFixture) put(t *testing.T, agentID string, limits domain.AgentGuardrailLimits) {
	t.Helper()
	if err := f.policies.Save(f.ctx, &domain.AgentGuardrailPolicy{TenantID: tenancydomain.DefaultTenantID, AgentID: agentID, AgentGuardrailLimits: limits, UpdatedAt: f.now}, ""); err != nil {
		t.Fatal(err)
	}
}

func (f *guardrailFixture) sign(in ports.AccessTokenInput) (ports.TokenIssuer, error) {
	issuer := sharedusecases.IssuerWithAgentGuardrail(f.signer, f.deps)
	_, _, err := issuer.SignAccessToken(f.ctx, in)
	return issuer, err
}

func oauthCode(t *testing.T, err error) string {
	t.Helper()
	var oe *sharedusecases.OAuthError
	if !errors.As(err, &oe) {
		t.Fatalf("expected an OAuth error, got %v", err)
	}
	return oe.Code
}

func rateOf(maxIssuances int) *domain.AgentGuardrailIssuanceRate {
	return &domain.AgentGuardrailIssuanceRate{MaxIssuances: maxIssuances, WindowSeconds: 60}
}

//spec:covers REQ-OAUTH2-073, EX-OAUTH2-073-01: 連鎖の Agent のどれかの上限を超えれば拒否する
func TestGuardedIssuerEvaluatesEveryAgentInTheChain(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "a2", domain.AgentGuardrailLimits{AllowedResources: []string{"https://a.example"}})
	_, err := f.sign(ports.AccessTokenInput{AgentChain: []string{"a1", "a2"}, Audiences: []string{"https://b.example"}})
	if oauthCode(t, err) != "access_denied" || len(f.signer.signed) != 0 {
		t.Fatalf("err=%v signed=%d", err, len(f.signer.signed))
	}
	if len(f.events) != 1 || f.events[0].AgentID != "a2" || f.events[0].Outcome != "denied" || f.events[0].Reasons[0] != domain.AgentGuardrailReasonResource {
		t.Fatalf("events = %+v", f.events)
	}
}

//spec:covers EX-OAUTH2-073-02: Agent のない発行はポリシーを読まない
func TestGuardedIssuerSkipsIssuanceWithoutAgent(t *testing.T) {
	f := newGuardrailFixture(t)
	f.deps.RequestingAgentID = ""
	f.deps.Policies = failingGuardrailPolicies{}
	if issuer, err := f.sign(ports.AccessTokenInput{}); err != nil || issuer.AccessTokenTTLSeconds() != 600 {
		t.Fatalf("err=%v", err)
	}
}

//spec:covers EX-OAUTH2-074-01: 流量の境界の両側
func TestGuardedIssuerLimitsIssuanceRate(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "a1", domain.AgentGuardrailLimits{IssuanceRate: rateOf(2)})
	for range 2 {
		if _, err := f.sign(ports.AccessTokenInput{}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.sign(ports.AccessTokenInput{})
	if oauthCode(t, err) != "access_denied" {
		t.Fatalf("err=%v", err)
	}
	last := f.events[len(f.events)-1]
	if last.Reasons[0] != domain.AgentGuardrailReasonIssuanceRate || last.RequestID != "req-1" {
		t.Fatalf("event = %+v", last)
	}
}

//spec:covers EX-OAUTH2-074-02: 厳しい層の有効期間が exp と expires_in に効く
func TestGuardedIssuerShortensLifetimeByStrictestLayer(t *testing.T) {
	f := newGuardrailFixture(t)
	short, long := 60, 300
	f.put(t, "", domain.AgentGuardrailLimits{MaxTokenTTLSeconds: &short})
	f.put(t, "a1", domain.AgentGuardrailLimits{MaxTokenTTLSeconds: &long})
	issuer, err := f.sign(ports.AccessTokenInput{})
	if err != nil {
		t.Fatal(err)
	}
	if issuer.AccessTokenTTLSeconds() != 60 || f.signer.signed[0].ExpiresAt != f.now.Unix()+60 {
		t.Fatalf("ttl=%d exp=%d", issuer.AccessTokenTTLSeconds(), f.signer.signed[0].ExpiresAt)
	}
}

//spec:covers REQ-OAUTH2-074, EX-OAUTH2-074-03: ポリシーか計数を読めなければ発行しない
func TestGuardedIssuerDeniesWhenStoresFail(t *testing.T) {
	t.Run("policy", func(t *testing.T) {
		f := newGuardrailFixture(t)
		f.deps.Policies = failingGuardrailPolicies{}
		if _, err := f.sign(ports.AccessTokenInput{}); oauthCode(t, err) != "access_denied" || len(f.signer.signed) != 0 {
			t.Fatalf("err=%v", err)
		}
		if f.events[0].Reasons[0] != domain.AgentGuardrailReasonUnavailable {
			t.Fatalf("event = %+v", f.events[0])
		}
	})
	t.Run("counter", func(t *testing.T) {
		f := newGuardrailFixture(t)
		f.put(t, "a1", domain.AgentGuardrailLimits{IssuanceRate: rateOf(5)})
		f.deps.Limiter = failingLimiter{}
		if _, err := f.sign(ports.AccessTokenInput{}); oauthCode(t, err) != "access_denied" || len(f.signer.signed) != 0 {
			t.Fatalf("err=%v", err)
		}
		if f.events[0].Reasons[0] != domain.AgentGuardrailReasonUnavailable {
			t.Fatalf("event = %+v", f.events[0])
		}
	})
}

//spec:covers EX-OAUTH2-074-04: 他の理由で拒否する要求は流量を消費しない
func TestGuardedIssuerDoesNotCountDeniedRequests(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "a1", domain.AgentGuardrailLimits{IssuanceRate: rateOf(1), AllowedResources: []string{"https://a.example"}})
	if _, err := f.sign(ports.AccessTokenInput{Audiences: []string{"https://b.example"}}); oauthCode(t, err) != "access_denied" {
		t.Fatalf("err=%v", err)
	}
	if _, err := f.sign(ports.AccessTokenInput{Audiences: []string{"https://a.example"}}); err != nil {
		t.Fatal(err)
	}
}

//spec:covers REQ-OAUTH2-076, EX-OAUTH2-076-01, EX-OAUTH2-076-02: 所有者の承認だけが降格した上限を超えられる
func TestGuardedIssuerWaivesOnlyForOwnerApproval(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "a1", domain.AgentGuardrailLimits{
		IssuanceRate: rateOf(1),
		OnViolation:  domain.AgentGuardrailOnViolation{IssuanceRate: domain.AgentGuardrailRequireApproval},
	})
	if _, err := f.sign(ports.AccessTokenInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.sign(ports.AccessTokenInput{}); oauthCode(t, err) != "unauthorized_client" {
		t.Fatalf("err=%v", err)
	}
	if got := f.events[len(f.events)-1].Outcome; got != "approval_required" {
		t.Fatalf("outcome = %s", got)
	}
	if _, err := f.sign(ports.AccessTokenInput{Sub: "bob", ApprovalRequestID: "ar"}); oauthCode(t, err) != "access_denied" {
		t.Fatalf("non-owner err=%v", err)
	}
	if _, err := f.sign(ports.AccessTokenInput{Sub: "alice", ApprovalRequestID: "ar"}); err != nil {
		t.Fatal(err)
	}
	if got := f.events[len(f.events)-1].Outcome; got != "approved" {
		t.Fatalf("outcome = %s", got)
	}
}

// 降格は拒否に勝たない。所有者の承認でも deny の項目は超えられない (REQ-OAUTH2-074)。
//
//spec:covers REQ-OAUTH2-074
func TestGuardedIssuerApprovalDoesNotWaiveDeny(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "", domain.AgentGuardrailLimits{RequireApproval: true})
	f.put(t, "a1", domain.AgentGuardrailLimits{AllowedResources: []string{}})
	if _, err := f.sign(ports.AccessTokenInput{Sub: "alice", ApprovalRequestID: "ar", Audiences: []string{"https://a.example"}}); oauthCode(t, err) != "access_denied" {
		t.Fatalf("err=%v", err)
	}
}

//spec:covers EX-OAUTH2-078-01: 許可の評価に版、相関、トークンが残り、詳細の値は残らない
func TestGuardedIssuerRecordsAllowedEvaluation(t *testing.T) {
	f := newGuardrailFixture(t)
	limits := domain.AgentGuardrailLimits{AllowedResources: []string{"https://a.example"}}
	f.put(t, "a1", limits)
	if _, err := f.sign(ports.AccessTokenInput{Audiences: []string{"https://a.example"}}); err != nil {
		t.Fatal(err)
	}
	event := f.events[0]
	if event.Outcome != "allowed" || event.TokenID != "jti-1" || event.RequestID != "req-1" || event.PolicyVersions[0] != limits.Version() {
		t.Fatalf("event = %+v", event)
	}
}

// 前判定と署名で二重に数えない (REQ-OAUTH2-075)。
//
//spec:covers REQ-OAUTH2-075
func TestGuardedIssuerPrecheckCountsOnce(t *testing.T) {
	f := newGuardrailFixture(t)
	f.put(t, "a1", domain.AgentGuardrailLimits{IssuanceRate: rateOf(1)})
	issuer := sharedusecases.IssuerWithAgentGuardrail(f.signer, f.deps)
	guard := issuer.(ports.IssuanceGuard)
	if err := guard.Precheck(f.ctx, ports.AccessTokenInput{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := issuer.SignAccessToken(f.ctx, ports.AccessTokenInput{}); err != nil {
		t.Fatal(err)
	}
}
