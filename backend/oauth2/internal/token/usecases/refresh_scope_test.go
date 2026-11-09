package usecases

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type refreshScopeGuard struct {
	ports.TokenIssuer
	scopes []string
}

func (g *refreshScopeGuard) Precheck(_ context.Context, in ports.AccessTokenInput) error {
	g.scopes = in.Scopes
	if !slices.Equal(in.Scopes, []string{"openid"}) {
		return NewOAuthError("access_denied", "Only openid is allowed.")
	}
	return nil
}

func (g *refreshScopeGuard) Confirm(string) {}

//spec:covers REQ-OAUTH2-006, REQ-OAUTH2-075: 発行前の検査と署名、発行イベントには絞ったスコープを渡し、親の付与を保持する。
func TestRefreshUsesEffectiveScopesForGuardAndIssuanceEvents(t *testing.T) {
	resource := "https://mcp.example.com/tools"
	f := newRefreshFixtureWithResource(t, &resource)
	issuer := f.deps.TokenIssuer.(*fakeTokenIssuer)
	guard := &refreshScopeGuard{TokenIssuer: issuer}
	f.deps.TokenIssuer = guard
	var accessScopes, resourceScopes []string
	f.deps.Emit = func(event spec.DomainEvent) {
		switch e := event.(type) {
		case *domain.AccessTokenIssued:
			accessScopes = e.Scopes
		case *domain.ResourceScopedTokenIssued:
			resourceScopes = e.Scopes
		}
	}
	_, err := RefreshTokens(testing_tenant.Default(t.Context()), f.deps, RefreshInput{
		ClientID: "client", RefreshToken: f.token, Scope: "openid",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string][]string{
		"guard": guard.scopes, "署名": issuer.lastAccessTokenInput.Scopes,
		"AccessTokenIssued": accessScopes, "ResourceScopedTokenIssued": resourceScopes,
	} {
		if !slices.Equal(got, []string{"openid"}) {
			t.Errorf("%s scopes=%v, want [openid]", name, got)
		}
	}
	if !slices.Equal(f.record.Scopes, []string{"openid", "offline_access"}) {
		t.Fatalf("親のスコープが変わった: %v", f.record.Scopes)
	}
}
