package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestCompleteLogin(t *testing.T) {
	ctx := tenantContext(spec.DefaultTenantID)
	requestStore := memory.NewAuthorizationRequestStore()
	codeStore := memory.NewAuthorizationCodeStore()

	deps := CompleteLoginDeps{
		RequestStore: requestStore,
		CodeStore:    codeStore,
	}

	now := time.Now().UTC()
	reqID := "11111111-2222-3333-4444-555555555555"

	t.Run("Succeeds", func(t *testing.T) {
		req := &domain.AuthorizationRequest{
			ID:                  reqID,
			TenantID:            spec.DefaultTenantID,
			ClientID:            "client-1",
			Scope:               "openid profile",
			RedirectURI:         "https://example.com/cb",
			State:               spec.AuthFlowReceived,
			ExpiresAt:           now.Add(10 * time.Minute),
			CodeChallenge:       "challenge-value",
			CodeChallengeMethod: "S256",
		}
		_ = requestStore.Save(ctx, req)

		in := CompleteLoginInput{
			RequestID: reqID,
			Sub:       "user-1",
			AuthTime:  now,
			AMR:       []string{"pwd"},
			ACR:       "urn:pwd",
		}

		out, err := CompleteLogin(ctx, deps, in)
		if err != nil {
			t.Fatal(err)
		}
		if out.Code.UserID != "user-1" || out.Code.ClientID != "client-1" {
			t.Errorf("mismatched code output: %+v", out.Code)
		}

		// リクエストの状態確認
		updated, _ := requestStore.Find(ctx, reqID)
		if updated.State != spec.AuthFlowCodeIssued {
			t.Errorf("expected state AuthFlowCodeIssued, got %v", updated.State)
		}
	})

	t.Run("RequestNotFound", func(t *testing.T) {
		in := CompleteLoginInput{
			RequestID: "22222222-2222-2222-2222-222222222222",
			Sub:       "user-1",
			AuthTime:  now,
		}
		_, err := CompleteLogin(ctx, deps, in)
		var oerr *OAuthError
		if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
			t.Errorf("expected invalid_request OAuthError, got %v", err)
		}
	})

	t.Run("MismatchedTenant", func(t *testing.T) {
		req := &domain.AuthorizationRequest{
			ID:        "33333333-3333-3333-3333-333333333333",
			TenantID:  "another-tenant",
			State:     spec.AuthFlowReceived,
			ExpiresAt: now.Add(10 * time.Minute),
		}
		_ = requestStore.Save(ctx, req)

		in := CompleteLoginInput{
			RequestID: "33333333-3333-3333-3333-333333333333",
			Sub:       "user-1",
			AuthTime:  now,
		}
		_, err := CompleteLogin(ctx, deps, in)
		var oerr *OAuthError
		if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
			t.Errorf("expected invalid_request OAuthError, got %v", err)
		}
	})

	t.Run("RequestExpired", func(t *testing.T) {
		req := &domain.AuthorizationRequest{
			ID:        "44444444-4444-4444-4444-444444444444",
			TenantID:  spec.DefaultTenantID,
			State:     spec.AuthFlowReceived,
			ExpiresAt: now.Add(-1 * time.Minute),
		}
		_ = requestStore.Save(ctx, req)

		in := CompleteLoginInput{
			RequestID: "44444444-4444-4444-4444-444444444444",
			Sub:       "user-1",
			AuthTime:  now,
		}
		_, err := CompleteLogin(ctx, deps, in)
		var oerr *OAuthError
		if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
			t.Errorf("expected invalid_request OAuthError, got %v", err)
		}

		// Expiredに遷移していることの確認
		updated, _ := requestStore.Find(ctx, "44444444-4444-4444-4444-444444444444")
		if updated.State != spec.AuthFlowExpired {
			t.Errorf("expected state AuthFlowExpired, got %v", updated.State)
		}
	})

	t.Run("AlreadyProcessedState", func(t *testing.T) {
		req := &domain.AuthorizationRequest{
			ID:        "55555555-5555-5555-5555-555555555555",
			TenantID:  spec.DefaultTenantID,
			State:     spec.AuthFlowCodeIssued,
			ExpiresAt: now.Add(10 * time.Minute),
		}
		_ = requestStore.Save(ctx, req)

		in := CompleteLoginInput{
			RequestID: "55555555-5555-5555-5555-555555555555",
			Sub:       "user-1",
			AuthTime:  now,
		}
		_, err := CompleteLogin(ctx, deps, in)
		var oerr *OAuthError
		if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
			t.Errorf("expected invalid_request OAuthError, got %v", err)
		}
	})
}
