package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type fakeIntrospector struct {
	result *ports.IntrospectionResult
	err    error
}

func (f *fakeIntrospector) IntrospectAccessToken(ctx context.Context, token string) (*ports.IntrospectionResult, error) {
	return f.result, f.err
}

func TestIntrospectToken(t *testing.T) {
	ctx := tenantContext(spec.DefaultTenantID)
	refreshStore := memory.NewRefreshTokenStore()
	denylist := memory.NewAccessTokenDenylist()
	introspector := &fakeIntrospector{}

	deps := IntrospectDeps{
		Introspector:        introspector,
		RefreshStore:        refreshStore,
		AccessTokenDenylist: denylist,
	}

	now := time.Now().UTC()

	t.Run("RefreshTokenSucceeds", func(t *testing.T) {
		tokenVal := "refresh-token-val"
		hash := domain.HashRefreshToken(tokenVal)
		rec := &domain.RefreshTokenRecord{
			ID:                "rt-1",
			TenantID:          spec.DefaultTenantID,
			ClientID:          "client-1",
			UserID:            "user-1",
			Scopes:            []string{"openid", "profile"},
			IssuedAt:          now.Add(-10 * time.Minute),
			ExpiresAt:         now.Add(10 * time.Minute),
			AbsoluteExpiresAt: now.Add(24 * time.Hour),
			Hash:              hash,
			SenderConstraint: &domain.SenderConstraint{
				Type:    spec.SenderConstraintDPoP,
				JKT:     "jkt-val",
				X5TS256: "x5t-val",
			},
		}
		_ = refreshStore.Save(ctx, rec)

		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Error("expected active to be true")
		}
		if resp.Sub != "user-1" || resp.ClientID != "client-1" || resp.Scope != "openid profile" {
			t.Errorf("unexpected resp: %+v", resp)
		}
		if resp.CNF == nil || resp.CNF["jkt"] != "jkt-val" || resp.CNF["x5t#S256"] != "x5t-val" {
			t.Errorf("unexpected CNF in resp: %+v", resp.CNF)
		}
	})

	t.Run("RefreshTokenInactiveCases", func(t *testing.T) {
		tokenVal := "refresh-token-inactive"
		hash := domain.HashRefreshToken(tokenVal)

		// 1. Tenant ID Mismatch
		rec := &domain.RefreshTokenRecord{
			ID:                "rt-inactive",
			TenantID:          "another-tenant",
			Hash:              hash,
			AbsoluteExpiresAt: now.Add(24 * time.Hour),
		}
		_ = refreshStore.Save(ctx, rec)
		resp, _ := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal}, now)
		if resp.Active {
			t.Error("expected inactive for tenant mismatch")
		}

		// 2. Absolute Expired
		rec.TenantID = spec.DefaultTenantID
		rec.AbsoluteExpiresAt = now.Add(-1 * time.Minute)
		_ = refreshStore.Save(ctx, rec)
		resp, _ = IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal}, now)
		if resp.Active {
			t.Error("expected inactive for absolute expired")
		}

		// 3. Revoked
		rec.AbsoluteExpiresAt = now.Add(24 * time.Hour)
		rec.Revoked = true
		_ = refreshStore.Save(ctx, rec)
		resp, _ = IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal}, now)
		if resp.Active {
			t.Error("expected inactive for revoked token")
		}
	})

	t.Run("AccessTokenSucceeds", func(t *testing.T) {
		tokenVal := "access-token-val"
		introspector.result = &ports.IntrospectionResult{
			Active:    true,
			JTI:       "jti-1",
			ClientID:  "client-1",
			Sub:       "user-1",
			Scope:     "openid",
			Exp:       now.Add(10 * time.Minute).Unix(),
			Iat:       now.Add(-10 * time.Minute).Unix(),
			TokenType: "Bearer",
			SenderConstraint: &domain.SenderConstraint{
				Type:    spec.SenderConstraintDPoP,
				JKT:     "jkt-val",
				X5TS256: "x5t-val",
			},
		}
		introspector.err = nil

		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal, TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Error("expected active to be true")
		}
		if resp.CNF == nil || resp.CNF["jkt"] != "jkt-val" || resp.CNF["x5t#S256"] != "x5t-val" {
			t.Errorf("unexpected CNF in resp: %+v", resp.CNF)
		}
	})

	t.Run("AccessTokenRevokedInDenylist", func(t *testing.T) {
		tokenVal := "access-token-revoked"
		introspector.result = &ports.IntrospectionResult{
			Active: true,
			JTI:    "jti-revoked",
		}
		// Denylist に登録
		_ = denylist.Add(ctx, "jti-revoked", now.Add(10*time.Minute))

		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal, TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Active {
			t.Error("expected inactive for revoked access token")
		}
	})

	t.Run("IntrospectorError", func(t *testing.T) {
		introspector.result = nil
		introspector.err = errors.New("introspect fail")
		_, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "some-token", TokenTypeHint: "access_token"}, now)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("ZeroTimeHandling", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{Active: false}
		introspector.err = nil
		// now = time.Time{} の場合に internal で time.Now() が使われるパスを通す
		_, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "token-zero", TokenTypeHint: "access_token"}, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
	})
}
