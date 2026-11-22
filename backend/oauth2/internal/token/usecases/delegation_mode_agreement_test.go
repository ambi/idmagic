package usecases

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 導出が 2 箇所に分かれると、この 2 つが食い違ったまま気付けない。
//
// リソースサーバーは応答の delegation_mode を読むだけでよく、act と principal 種別から
// 導出し直す必要がない。
//
//spec:covers REQ-OAUTH2-049, EX-OAUTH2-049-01, EX-OAUTH2-049-02: 交換が監査へ残したモードと、発行トークンをイントロスペクトしたモードが一致する。
func TestDelegationModeAgreesBetweenAuditAndIntrospection(t *testing.T) {
	ctx := tenantContext()

	var events []spec.DomainEvent
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*oauthports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read"},
	})
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }
	if _, err := ExchangeToken(ctx, deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
	}, time.Now().UTC()); err != nil {
		t.Fatalf("ExchangeToken: %v", err)
	}

	var exchanged *oauthdomain.TokenExchanged
	for _, event := range events {
		if typed, ok := event.(*oauthdomain.TokenExchanged); ok {
			exchanged = typed
		}
	}
	if exchanged == nil {
		t.Fatal("a successful exchange must emit TokenExchanged")
	}

	// 発行トークンのクレームをそのまま introspection の材料にする。
	issued := issuer.lastInput
	introspectDeps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &oauthports.IntrospectionResult{
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
	if resp.DelegationMode != oauthdomain.DelegationModeOnBehalfOf {
		t.Fatalf("DelegationMode = %q, want %q", resp.DelegationMode, oauthdomain.DelegationModeOnBehalfOf)
	}
}

//spec:covers REQ-OAUTH2-049, EX-OAUTH2-049-03: 代行が無く subject が非人間のプリンシパルなら、イントロスペクションは自律実行として返す。
func TestIntrospectionReportsAutonomousForAgentTokens(t *testing.T) {
	ctx := tenantContext()
	deps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &oauthports.IntrospectionResult{
			Active: true, Sub: "agent-client", ClientID: "agent-client",
			PrincipalType: oauthdomain.PrincipalTypeAgent, AgentID: "agent-1",
		}},
		RefreshStore: memory.NewRefreshTokenStore(),
	}
	resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IntrospectToken: %v", err)
	}
	if resp.DelegationMode != oauthdomain.DelegationModeAutonomous {
		t.Fatalf("DelegationMode = %q, want %q", resp.DelegationMode, oauthdomain.DelegationModeAutonomous)
	}
}

// 自律実行との違いは principal 種別だけなので、同じ入口で両方を読む。
//
//spec:covers EX-OAUTH2-049-04: 代行が無く subject が人間の利用者なら、イントロスペクションは直接のアクセスとして返す。
func TestIntrospectionReportsDirectAccessForUserTokens(t *testing.T) {
	ctx := tenantContext()
	deps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &oauthports.IntrospectionResult{
			Active: true, Sub: "user-alice", ClientID: "portal", Scope: "openid",
		}},
		RefreshStore: memory.NewRefreshTokenStore(),
	}
	resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("IntrospectToken: %v", err)
	}
	if resp.DelegationMode != oauthdomain.DelegationModeDirect {
		t.Fatalf("DelegationMode = %q, want %q", resp.DelegationMode, oauthdomain.DelegationModeDirect)
	}
}

// 失効しているトークンにはモードを載せない。active でない応答が立場を語ると、
// リソースサーバーが active の確認を飛ばす余地ができる。
func TestIntrospectionOmitsDelegationModeForInactiveTokens(t *testing.T) {
	ctx := tenantContext()
	deps := IntrospectDeps{
		Introspector: &fakeIntrospector{result: &oauthports.IntrospectionResult{Active: false}},
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
