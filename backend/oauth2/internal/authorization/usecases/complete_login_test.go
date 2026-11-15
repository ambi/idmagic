package usecases

// 主要ユースケース追跡: REQ-AUTHENTICATION-007。

import (
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/internal/authorization/db_memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-OAUTH2-005: 認証済みと同意済みから発行でき、認証前、同意待ち、処理済みの要求では状態とコードを変更しない。
func TestCompleteLoginRequiresCompletedAuthenticationAndConsent(t *testing.T) {
	for _, state := range []spec.AuthorizationCodeFlowState{
		spec.AuthFlowReceived, spec.AuthFlowAuthenticationPending, spec.AuthFlowAuthenticated,
		spec.AuthFlowConsentPending, spec.AuthFlowConsented, spec.AuthFlowCodeIssued,
		spec.AuthFlowExchanged, spec.AuthFlowRejected, spec.AuthFlowExpired,
	} {
		t.Run(string(state), func(t *testing.T) {
			ctx := tenantContext()
			requests := memory.NewAuthorizationRequestStore()
			codes := memory.NewAuthorizationCodeStore()
			now := time.Now().UTC()
			req := &domain.AuthorizationRequest{
				ID: "11111111-2222-4333-8444-555555555555", TenantID: tenancydomain.DefaultTenantID,
				ClientID: "client-1", Scope: "openid", RedirectURI: "https://example.com/cb",
				State: state, ExpiresAt: now.Add(10 * time.Minute),
			}
			if state == spec.AuthFlowConsented {
				sub := "user-1"
				req.UserID = &sub
			}
			if err := requests.Save(ctx, req); err != nil {
				t.Fatal(err)
			}
			out, err := CompleteLogin(ctx, CompleteLoginDeps{RequestStore: requests, CodeStore: codes}, CompleteLoginInput{
				RequestID: req.ID, Sub: "user-1", AuthTime: now, AMR: []string{"pwd"}, ACR: "urn:pwd",
			})
			if state == spec.AuthFlowAuthenticated || state == spec.AuthFlowConsented {
				if err != nil {
					t.Fatal(err)
				}
				stored, findErr := codes.Find(ctx, out.Code.Code)
				if findErr != nil || stored == nil || req.State != spec.AuthFlowCodeIssued {
					t.Fatalf("発行結果=%v state=%s err=%v", stored, req.State, findErr)
				}
				return
			}
			assertOAuthError(t, err, "invalid_request")
			if out != nil || req.State != state || req.UserID != nil {
				t.Fatalf("拒否後に変更: out=%+v req=%+v", out, req)
			}
		})
	}
}

func TestCompleteLogin(t *testing.T) {
	ctx := tenantContext()
	requestStore := memory.NewAuthorizationRequestStore()
	codeStore := memory.NewAuthorizationCodeStore()

	deps := CompleteLoginDeps{
		RequestStore: requestStore,
		CodeStore:    codeStore,
	}

	now := time.Now().UTC()
	reqID := "11111111-2222-4333-8444-555555555555"

	t.Run("Succeeds", func(t *testing.T) {
		req := &domain.AuthorizationRequest{
			ID:                  reqID,
			TenantID:            tenancydomain.DefaultTenantID,
			ClientID:            "client-1",
			Scope:               "openid profile",
			RedirectURI:         "https://example.com/cb",
			State:               spec.AuthFlowAuthenticated,
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
		// 認可要求そのものが認証済み主体に結び付いていなければ、以後の再開も
		// トークン発行も別人の文脈で進んでしまう。
		if updated.UserID == nil || *updated.UserID != "user-1" {
			t.Errorf("authenticated subject was not attached to the request: %v", updated.UserID)
		}
		if updated.AuthTime == nil || *updated.AuthTime != now.UTC().Unix() {
			t.Errorf("auth_time was not attached to the request: %v", updated.AuthTime)
		}
		if len(updated.AMR) != 1 || updated.AMR[0] != "pwd" {
			t.Errorf("amr was not attached to the request: %v", updated.AMR)
		}
		if updated.ACR == nil || *updated.ACR != "urn:pwd" {
			t.Errorf("acr was not attached to the request: %v", updated.ACR)
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
			TenantID:  tenancydomain.DefaultTenantID,
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
			TenantID:  tenancydomain.DefaultTenantID,
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
