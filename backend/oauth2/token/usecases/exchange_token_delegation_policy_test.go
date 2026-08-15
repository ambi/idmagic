package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// stubDelegationPolicy はテナントの委譲深さ上限を固定で返す。err を設定すると
// 解決失敗を模す。
type stubDelegationPolicy struct {
	depth int
	err   error
}

func (s stubDelegationPolicy) MaxDelegationDepth(context.Context, string) (int, error) {
	return s.depth, s.err
}

var errPolicyUnavailable = errors.New("tenant store unavailable")

// actChainOfDepth は深さ n の act チェーンを組み立てる。
func actChainOfDepth(n int) map[string]any {
	if n <= 0 {
		return nil
	}
	act := map[string]any{"sub": "a1"}
	for i := 2; i <= n; i++ {
		act = map[string]any{"sub": "a" + string(rune('0'+i)), "act": act}
	}
	return act
}

func exchangeWithPolicy(
	t *testing.T,
	policy ports.DelegationPolicyResolver,
	existingAct map[string]any,
	events *[]spec.DomainEvent,
) (*recordingIssuer, error) {
	t.Helper()
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read", Act: existingAct},
	})
	deps.DelegationPolicy = policy
	if events != nil {
		deps.Emit = func(e spec.DomainEvent) { *events = append(*events, e) }
	}
	_, err := ExchangeToken(context.Background(), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
	}, time.Now().UTC())
	return issuer, err
}

// REQ-OAUTH2-048: テナントが下げた上限を超える交換は拒否され、上限内は通る。
func TestExchangeTokenHonoursTenantDelegationDepth(t *testing.T) {
	t.Run("a tightened limit rejects a chain the default would allow", func(t *testing.T) {
		// 既存 act の深さ 1 → 交換後は 2。システム既定 (3) なら通るが、上限 1 では拒否。
		issuer, err := exchangeWithPolicy(t, stubDelegationPolicy{depth: 1}, actChainOfDepth(1), nil)
		if err == nil {
			t.Fatal("a chain beyond the tenant limit must be rejected")
		}
		if issuer.calls != 0 {
			t.Fatal("a rejected exchange must not issue a token")
		}
	})

	t.Run("a chain within the tenant limit still succeeds", func(t *testing.T) {
		issuer, err := exchangeWithPolicy(t, stubDelegationPolicy{depth: 2}, actChainOfDepth(1), nil)
		if err != nil {
			t.Fatalf("ExchangeToken: %v", err)
		}
		if issuer.calls != 1 {
			t.Fatalf("issuer.calls = %d, want 1", issuer.calls)
		}
	})

	t.Run("an unresolvable policy rejects instead of falling back to the default", func(t *testing.T) {
		var events []spec.DomainEvent
		// 深さ 1 はシステム既定なら確実に通る。退避していれば成功してしまう。
		issuer, err := exchangeWithPolicy(t, stubDelegationPolicy{err: errPolicyUnavailable}, nil, &events)
		if err == nil {
			t.Fatal("an unresolvable delegation policy must reject, not fall back to the default")
		}
		if issuer.calls != 0 {
			t.Fatal("a rejected exchange must not issue a token")
		}
		var rejected *domain.TokenExchangeRejected
		for _, event := range events {
			if typed, ok := event.(*domain.TokenExchangeRejected); ok {
				rejected = typed
			}
		}
		if rejected == nil {
			t.Fatal("a fail-closed rejection must still be audited")
		}
	})

	t.Run("a resolver that reports more than the system default is clamped", func(t *testing.T) {
		// 保存側が上限を守っていても、評価側でも同じ規則を守る。
		issuer, err := exchangeWithPolicy(t, stubDelegationPolicy{depth: 99}, actChainOfDepth(3), nil)
		if err == nil {
			t.Fatal("a resolver cannot raise the limit above the system default")
		}
		if issuer.calls != 0 {
			t.Fatal("a rejected exchange must not issue a token")
		}
	})

	t.Run("no resolver keeps the product default", func(t *testing.T) {
		issuer, err := exchangeWithPolicy(t, nil, actChainOfDepth(2), nil)
		if err != nil {
			t.Fatalf("ExchangeToken: %v", err)
		}
		if issuer.calls != 1 {
			t.Fatalf("issuer.calls = %d, want 1", issuer.calls)
		}
		if _, err := exchangeWithPolicy(t, nil, actChainOfDepth(3), nil); err == nil {
			t.Fatal("the product default must still bound the chain when no resolver is wired")
		}
	})
}

// REQ-OAUTH2-048 / REQ-OAUTH2-049: 監査は深さと適用した上限、および委譲モードを残す。
func TestExchangeTokenAuditRecordsDepthLimitAndMode(t *testing.T) {
	var events []spec.DomainEvent
	if _, err := exchangeWithPolicy(t, stubDelegationPolicy{depth: 2}, nil, &events); err != nil {
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
	if exchanged.DelegationDepth != 1 {
		t.Fatalf("DelegationDepth = %d, want 1", exchanged.DelegationDepth)
	}
	if exchanged.MaxDelegationDepth != 2 {
		t.Fatalf("MaxDelegationDepth = %d, want 2 (the limit actually applied)", exchanged.MaxDelegationDepth)
	}
	// sub は user-1、act は client なので利用者の代理である。
	if exchanged.DelegationMode != domain.DelegationModeOnBehalfOf {
		t.Fatalf("DelegationMode = %q, want %q", exchanged.DelegationMode, domain.DelegationModeOnBehalfOf)
	}
}
