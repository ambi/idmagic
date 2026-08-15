package usecases

import (
	"context"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// REQ-OAUTH2-049: 交換が監査へ残したモードと、発行トークンをイントロスペクトした
// モードが一致する。導出が 2 箇所に分かれると、この 2 つが食い違ったまま気付けない。
func TestDelegationModeAgreesBetweenAuditAndIntrospection(t *testing.T) {
	ctx := tenantContext()

	var events []spec.DomainEvent
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read"},
	})
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }
	if _, err := ExchangeToken(ctx, deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
	}, time.Now().UTC()); err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	var exchanged *domain.TokenExchanged
	for _, event := range events {
		if typed, ok := event.(*domain.TokenExchanged); ok {
			exchanged = typed
		}
	}
	if exchanged == nil {
		t.Fatal("a successful exchange must emit TokenExchanged")
	}

	// 発行トークンのクレームをそのまま introspection の材料にする。
	issued := issuer.lastInput
	introspectDeps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &ports.IntrospectionResult{
			Active: true, Sub: issued.Sub, ClientID: issued.Client.ClientID, Act: issued.Act,
		}},
		RefreshStore: memory.NewRefreshTokenStore(),
	}
	resp, err := IntrospectToken(ctx, introspectDeps, IntrospectInput{
		Token: "exchanged-access-token", TokenTypeHint: "access_token",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IntrospectToken: %v", err)
	}
	if resp.DelegationMode != exchanged.DelegationMode {
		t.Fatalf("introspection reports %q but the audit recorded %q", resp.DelegationMode, exchanged.DelegationMode)
	}
	if resp.DelegationMode != domain.DelegationModeOnBehalfOf {
		t.Fatalf("DelegationMode = %q, want %q", resp.DelegationMode, domain.DelegationModeOnBehalfOf)
	}
}

// REQ-OAUTH2-049: エージェント自身のトークンは自律実行として返る。
func TestIntrospectionReportsAutonomousForAgentTokens(t *testing.T) {
	ctx := tenantContext()
	deps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &ports.IntrospectionResult{
			Active: true, Sub: "agent-client", ClientID: "agent-client",
			PrincipalType: domain.PrincipalTypeAgent, AgentID: "agent-1",
		}},
		RefreshStore: memory.NewRefreshTokenStore(),
	}
	resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IntrospectToken: %v", err)
	}
	if resp.DelegationMode != domain.DelegationModeAutonomous {
		t.Fatalf("DelegationMode = %q, want %q", resp.DelegationMode, domain.DelegationModeAutonomous)
	}
}

// 失効しているトークンにはモードを載せない。active でない応答が立場を語ると、
// リソースサーバーが active の確認を飛ばす余地ができる。
func TestIntrospectionOmitsDelegationModeForInactiveTokens(t *testing.T) {
	ctx := tenantContext()
	deps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &ports.IntrospectionResult{Active: false}},
		RefreshStore: memory.NewRefreshTokenStore(),
	}
	resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IntrospectToken: %v", err)
	}
	if resp.DelegationMode != "" {
		t.Fatalf("DelegationMode = %q, want empty for an inactive token", resp.DelegationMode)
	}
}

var _ = context.Background
