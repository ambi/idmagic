package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type fakeIntrospector struct {
	result *ports.IntrospectionResult
	err    error
}

func (f *fakeIntrospector) IntrospectAccessToken(ctx context.Context, token string) (*ports.IntrospectionResult, error) {
	return f.result, f.err
}

// active の値だけを読むテストは、失効を報告しながら claim を返し続ける応答を通してしまう。
//
//spec:covers EX-OAUTH2-011-01: 失効済みの access トークンを検査した応答は active=false だけを運び、scope や sub のような他のフィールドを 1 つも含まない。
func TestIntrospectToken(t *testing.T) {
	ctx := tenantContext()
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
			TenantID:          tenancydomain.DefaultTenantID,
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
		rec.TenantID = tenancydomain.DefaultTenantID
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
			Active:   true,
			JTI:      "jti-1",
			ClientID: "client-1",
			Sub:      "user-1",
			Scope:    "openid",
			Exp:      now.Add(10 * time.Minute).Unix(),
			Iat:      now.Add(-10 * time.Minute).Unix(),
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
			Active: true, JTI: "jti-revoked", ClientID: "client-1", Sub: "user-1",
			Scope: "openid profile", Exp: now.Add(10 * time.Minute).Unix(),
			Iat: now.Add(-10 * time.Minute).Unix(),
		}
		// Denylist に登録
		_ = denylist.Add(ctx, "jti-revoked", now.Add(10*time.Minute))

		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal, TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		assertOnlyInactive(t, resp)
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

// 参照して定める提示形式であり、トークンの区分名ではない。
//
// 読むのは 3 つである。アクセストークンは送信者制約から導いた提示形式を名乗ること、
// リフレッシュトークンは提示形式を持たないので名乗らないこと、そして無効なトークンは
// RFC 7662 §2.2 のとおり `active` 以外を返さないことである。3 つ目は、制約なしの
// 提示形式が `Bearer` であるために、無条件の代入が `active: false` の応答まで
// 汚してしまう経路を押さえる。
//
//spec:covers RFC7662-INTROSPECT: 内省の `token_type` は RFC 7662 §2.2 が RFC 6749 §5.1 を
func TestIntrospectTokenReportsTheRFC6749PresentationType(t *testing.T) {
	ctx := tenantContext()
	refreshStore := memory.NewRefreshTokenStore()
	introspector := &fakeIntrospector{}
	deps := IntrospectDeps{Introspector: introspector, RefreshStore: refreshStore}
	now := time.Now().UTC()

	t.Run("制約なしのアクセストークンは Bearer を名乗る", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-plain", ClientID: "client-1", Sub: "user-1",
			Scope: "openid", Exp: now.Add(10 * time.Minute).Unix(), Iat: now.Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{
			Token: "plain-access-token", TokenTypeHint: "access_token",
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if resp.TokenType != "Bearer" {
			t.Errorf("token_type=%q, want Bearer", resp.TokenType)
		}
	})

	t.Run("DPoP 束縛のアクセストークンは DPoP を名乗る", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-dpop", ClientID: "client-1", Sub: "user-1",
			SenderConstraint: &domain.SenderConstraint{
				Type: spec.SenderConstraintDPoP, JKT: "jkt-val",
			},
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{
			Token: "dpop-access-token", TokenTypeHint: "access_token",
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if resp.TokenType != "DPoP" {
			t.Errorf("token_type=%q, want DPoP", resp.TokenType)
		}
	})

	t.Run("リフレッシュトークンは提示形式を名乗らない", func(t *testing.T) {
		tokenVal := "refresh-token-presentation"
		if err := refreshStore.Save(ctx, &domain.RefreshTokenRecord{
			ID:                "rt-presentation",
			TenantID:          tenancydomain.DefaultTenantID,
			ClientID:          "client-1",
			UserID:            "user-1",
			Scopes:            []string{"openid"},
			IssuedAt:          now.Add(-10 * time.Minute),
			ExpiresAt:         now.Add(10 * time.Minute),
			AbsoluteExpiresAt: now.Add(24 * time.Hour),
			Hash:              domain.HashRefreshToken(tokenVal),
		}); err != nil {
			t.Fatal(err)
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: tokenVal}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Fatalf("リフレッシュトークンが active ではない: %+v", resp)
		}
		if resp.TokenType != "" {
			t.Errorf("token_type=%q, want 空 (リフレッシュトークンに提示形式は無い)", resp.TokenType)
		}
	})

	t.Run("無効なトークンは提示形式を名乗らない", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{Active: false}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{
			Token: "inactive-access-token", TokenTypeHint: "access_token",
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if resp.TokenType != "" {
			t.Errorf("token_type=%q, want 空 (active=false に付随する値は返さない)", resp.TokenType)
		}
	})
}

// assertOnlyInactive は、RFC 7662 §2.2 が定める「無効なトークンには active=false だけを返す」
// 形を、応答全体を零値と突き合わせて確かめる。individual なフィールドを列挙して読むと、
// 後から足されたフィールドが漏れても気づけない。
func assertOnlyInactive(t *testing.T, resp *IntrospectionResponse) {
	t.Helper()
	if resp == nil {
		t.Fatal("応答が nil である")
	}
	if !reflect.DeepEqual(*resp, IntrospectionResponse{}) {
		t.Fatalf("active=false の応答が他のフィールドを運んでいる: %+v", resp)
	}
}
