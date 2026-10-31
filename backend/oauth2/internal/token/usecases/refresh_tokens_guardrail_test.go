package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type refusingGuard struct {
	ports.TokenIssuer
	refuse bool
}

func (g *refusingGuard) Precheck(context.Context, ports.AccessTokenInput) error {
	if g.refuse {
		return NewOAuthError("access_denied", "The token request was denied.")
	}
	return nil
}
func (g *refusingGuard) Confirm(string) {}

//spec:covers REQ-OAUTH2-075, EX-OAUTH2-075-01: 拒否はローテーションせず、回復後に同じトークンで発行できる
func TestRefreshDeniedByGuardrailDoesNotRotate(t *testing.T) {
	now := time.Now().UTC()
	f := newRefreshFixture(t, (*domain.SenderConstraint)(nil), now, time.Hour)
	guard := &refusingGuard{TokenIssuer: f.deps.TokenIssuer, refuse: true}
	f.deps.TokenIssuer = guard
	ctx := testing_tenant.Default(context.Background())
	input := RefreshInput{ClientID: "client", RefreshToken: f.token}
	if _, err := RefreshTokens(ctx, f.deps, input, now); err == nil {
		t.Fatal("expected the guardrail to refuse")
	}
	guard.refuse = false
	res, err := RefreshTokens(ctx, f.deps, input, now)
	if err != nil {
		t.Fatalf("the same refresh token must stay usable: %v", err)
	}
	if res.RefreshToken == "" || res.RefreshToken == f.token {
		t.Fatalf("expected rotation after recovery: %+v", res)
	}
}
