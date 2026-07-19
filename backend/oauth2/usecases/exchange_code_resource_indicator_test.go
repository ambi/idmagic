package usecases

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestExchangeCodeForToken_noResourceBound_audienceUnaffected(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	out, err := ExchangeCodeForToken(context.Background(), f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("access token missing")
	}
	if len(f.issuer.lastAccessTokenInput.Audiences) != 0 {
		t.Fatalf("expected no explicit audience binding, got %v", f.issuer.lastAccessTokenInput.Audiences)
	}
}

func TestExchangeCodeForToken_resourceBoundAtAuthorize_audienceBoundAtRedemption(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	resource := "https://mcp.example.com/tools"
	f.code.Resource = &resource
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}

	var emitted []spec.DomainEvent
	f.deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }

	out, err := ExchangeCodeForToken(context.Background(), f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("access token missing")
	}
	if len(f.issuer.lastAccessTokenInput.Audiences) != 1 || f.issuer.lastAccessTokenInput.Audiences[0] != resource {
		t.Fatalf("expected audience to be bound to %q, got %v", resource, f.issuer.lastAccessTokenInput.Audiences)
	}

	found := false
	for _, e := range emitted {
		if e.EventType() == "ResourceScopedTokenIssued" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ResourceScopedTokenIssued to be emitted")
	}
}

func TestExchangeCodeForToken_tokenRequestResourceMismatch_rejectedAsInvalidTarget(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	resource := "https://mcp.example.com/tools"
	f.code.Resource = &resource
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}

	in := exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	in.Resource = []string{"https://mcp.example.com/different"}
	_, err := ExchangeCodeForToken(context.Background(), f.deps, in)
	assertOAuthError(t, err, "invalid_target")
}

func TestExchangeCodeForToken_tokenRequestResourceWithoutAuthorizeBinding_rejectedAsInvalidTarget(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	// f.code.Resource は nil (authorize 時に resource 指定なし)。
	in := exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	in.Resource = []string{"https://mcp.example.com/tools"}
	_, err := ExchangeCodeForToken(context.Background(), f.deps, in)
	assertOAuthError(t, err, "invalid_target")
}

// ADR-055/wi-262: authorize 時に束縛された resource は発行される RefreshTokenRecord にも
// 伝播し、後続のローテーションで audience 限定を保持できるようにする。
func TestExchangeCodeForToken_resourceBoundAtAuthorize_propagatesToRefreshTokenRecord(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid", "offline_access"})
	resource := "https://mcp.example.com/tools"
	f.code.Resource = &resource
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}
	out, err := ExchangeCodeForToken(context.Background(), f.deps,
		exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RefreshToken == "" {
		t.Fatal("refresh token missing")
	}
	hash := domain.HashRefreshToken(out.RefreshToken)
	rec, err := f.refreshStore.FindByHash(context.Background(), hash)
	if err != nil || rec == nil {
		t.Fatalf("expected refresh token record to be findable: %v", err)
	}
	if rec.Resource == nil || *rec.Resource != resource {
		t.Fatalf("expected refresh token record to carry resource, got %v", rec.Resource)
	}
}

func TestExchangeCodeForToken_tokenRequestResourceMatching_accepted(t *testing.T) {
	f := newExchangeFixture(t, []string{"openid"})
	resource := "https://mcp.example.com/tools"
	f.code.Resource = &resource
	if err := f.codeStore.Save(context.Background(), f.code); err != nil {
		t.Fatal(err)
	}
	in := exchangeInput("verifier-of-sufficient-length-ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	in.Resource = []string{resource}
	out, err := ExchangeCodeForToken(context.Background(), f.deps, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("access token missing")
	}
}
