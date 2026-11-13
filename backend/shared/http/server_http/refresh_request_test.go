package server_http_test

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-OAUTH2-006, RFC6749-REFRESH-SCOPE, EX-OAUTH2-006-03: scope を絞った発行と次の交換での元の付与の保持、範囲外の拒否後の未消費を観測する。
//verify:operation Token: refresh_token の要求、成功時の scope と aud、拒否の本文と 400 を検証する。
func TestRefreshRequestScopesAreBoundedByTheOriginalGrant(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	for _, tc := range []struct {
		name, scope, want string
	}{
		{"部分集合", "profile", "profile"},
		{"全体", "openid profile offline_access", "openid profile offline_access"},
		{"順序を変えた部分集合", "profile openid", "profile openid"},
		{"空値は省略", "", "openid profile offline_access"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := f.authorizationCode(t, url.Values{"resource": {tiResource}})
			first := f.mustToken(t, f.codeExchangeForm(code, nil))
			out := f.mustToken(t, refreshRequest(first.RefreshToken, url.Values{
				"scope": {tc.scope}, "resource": {tiResource},
			}))
			assertRefreshClaims(t, out, tc.want, tiResource)
			next := f.refreshRecord(t.Context(), t, out.RefreshToken)
			if !slices.Equal(next.Scopes, []string{"openid", "profile", "offline_access"}) || next.Resource == nil || *next.Resource != tiResource {
				t.Fatalf("元の付与が保持されていない: %+v", next)
			}
			restored := f.mustToken(t, refreshRequest(out.RefreshToken, nil))
			assertRefreshClaims(t, restored, "openid profile offline_access", tiResource)
		})
	}
	for _, scope := range []string{"write", "openid write", "pro", "Profile"} {
		t.Run("範囲外_"+scope, func(t *testing.T) {
			code := f.authorizationCode(t, nil)
			first := f.mustToken(t, f.codeExchangeForm(code, nil))
			assertRefreshRefusal(t.Context(), t, f, first.RefreshToken, url.Values{"scope": {scope}}, "invalid_scope")
			retried := f.mustToken(t, refreshRequest(first.RefreshToken, nil))
			assertRefreshClaims(t, retried, "openid profile offline_access", tiClientID)
		})
	}
}

//spec:covers REQ-OAUTH2-006, REQ-OAUTH2-054, RFC8707-MCP-RESOURCE-BINDING, EX-OAUTH2-006-04: 元の有効な対象の再指定と省略、対象の追加と変更、未登録、無効、複数指定の拒否と親の未消費を観測する。
func TestRefreshRequestResourceCannotWidenTheOriginalGrant(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	ctx := testing_tenant.Default(t.Context())
	other, err := f.resourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, tiOtherResource)
	if err != nil || other == nil {
		t.Fatalf("別のリソースの準備: %v", err)
	}
	other.State = domain.McpResourceServerActive
	if err := f.resourceServers.Save(ctx, other); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		bound       bool
		resources   []string
		changeState domain.McpResourceServerState
		remove      bool
		wantError   bool
	}{
		{name: "一致", bound: true, resources: []string{tiResource}},
		{name: "省略", bound: true},
		{name: "空値は省略", bound: true, resources: []string{""}},
		{name: "未束縛で省略"},
		{name: "後付け", resources: []string{tiResource}, wantError: true},
		{name: "別の有効な対象", bound: true, resources: []string{tiOtherResource}, wantError: true},
		{name: "未登録", bound: true, resources: []string{"https://api.example/unknown"}, wantError: true},
		{name: "複数", bound: true, resources: []string{tiResource, tiOtherResource}, wantError: true},
		{name: "同じ値が二つ", bound: true, resources: []string{tiResource, tiResource}, wantError: true},
		{name: "空値と対象", bound: true, resources: []string{"", tiResource}, wantError: true},
		{name: "束縛先が無効", bound: true, resources: []string{tiResource}, changeState: domain.McpResourceServerDisabled, wantError: true},
		{name: "束縛先の登録削除", bound: true, resources: []string{tiResource}, remove: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var authorize url.Values
			wantAud := tiClientID
			if tc.bound {
				authorize = url.Values{"resource": {tiResource}}
				wantAud = tiResource
			}
			code := f.authorizationCode(t, authorize)
			first := f.mustToken(t, f.codeExchangeForm(code, nil))
			if tc.changeState != "" || tc.remove {
				r, err := f.resourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, tiResource)
				if err != nil || r == nil {
					t.Fatalf("束縛先の取得: %v", err)
				}
				if tc.remove {
					if err := f.resourceServers.Delete(ctx, tenancydomain.DefaultTenantID, r.ID); err != nil {
						t.Fatal(err)
					}
				} else {
					r.State = tc.changeState
					if err := f.resourceServers.Save(ctx, r); err != nil {
						t.Fatal(err)
					}
				}
				t.Cleanup(func() {
					r.State = domain.McpResourceServerActive
					if err := f.resourceServers.Save(ctx, r); err != nil {
						t.Error(err)
					}
				})
			}
			extra := url.Values{}
			if tc.resources != nil {
				extra["resource"] = tc.resources
			}
			if tc.wantError {
				assertRefreshRefusal(ctx, t, f, first.RefreshToken, extra, "invalid_target")
				extra = nil
			}
			out := f.mustToken(t, refreshRequest(first.RefreshToken, extra))
			assertRefreshClaims(t, out, "openid profile offline_access", wantAud)
		})
	}
}

//spec:covers REQ-OAUTH2-006: 認証から時間が経った後のリフレッシュで、アクセストークンと次の記録が元の auth_time、amr、acr を保ち、値のない記録からは載せないことを観測する。
func TestRefreshRequestKeepsTheOriginalAuthentication(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	ctx := testing_tenant.Default(t.Context())
	authenticatedAt := time.Now().Add(-time.Hour).Unix()
	amr := []string{"pwd", "otp"}
	acr := authdomain.ACRMFA

	code := f.authorizationCode(t, nil)
	record, err := f.codes.Find(ctx, code)
	if err != nil || record == nil {
		t.Fatalf("認可コードの記録が見つからない: record=%v err=%v", record, err)
	}
	record.AuthTime, record.AMR, record.ACR = authenticatedAt, amr, &acr
	if err := f.codes.Save(ctx, record); err != nil {
		t.Fatal(err)
	}
	first := f.mustToken(t, f.codeExchangeForm(code, nil))

	t.Run("ローテーションを重ねても元の認証を保つ", func(t *testing.T) {
		token := first.RefreshToken
		for range 2 {
			out := f.mustToken(t, refreshRequest(token, nil))
			_, claims := jwtParts(t, out.AccessToken)
			if got, ok := claims["auth_time"].(float64); !ok || int64(got) != authenticatedAt {
				t.Fatalf("auth_time=%v, want %d", claims["auth_time"], authenticatedAt)
			}
			if got := fmt.Sprint(claims["amr"]); got != fmt.Sprint(amr) {
				t.Fatalf("amr=%v, want %v", claims["amr"], amr)
			}
			if claims["acr"] != acr {
				t.Fatalf("acr=%v, want %q", claims["acr"], acr)
			}
			token = out.RefreshToken
		}
	})

	t.Run("値のない記録からは載せない", func(t *testing.T) {
		token := f.mustToken(t, f.codeExchangeForm(f.authorizationCode(t, nil), nil)).RefreshToken
		legacy := f.refreshRecord(ctx, t, token)
		legacy.AuthTime, legacy.AMR, legacy.ACR = nil, nil, nil
		if err := f.refreshTokens.Save(ctx, legacy); err != nil {
			t.Fatal(err)
		}
		out := f.mustToken(t, refreshRequest(token, nil))
		_, claims := jwtParts(t, out.AccessToken)
		for _, claim := range []string{"auth_time", "amr", "acr"} {
			if value, present := claims[claim]; present {
				t.Fatalf("%s=%v, want absent", claim, value)
			}
		}
	})
}

func refreshRequest(token string, extra url.Values) url.Values {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}}
	maps.Copy(form, extra)
	return form
}

func assertRefreshClaims(t *testing.T, out tiTokenResponse, wantScope, wantAud string) {
	t.Helper()
	_, claims := jwtParts(t, out.AccessToken)
	if out.Scope != wantScope || claims["scope"] != wantScope {
		t.Fatalf("scope: response=%q JWT=%v, want %q", out.Scope, claims["scope"], wantScope)
	}
	switch aud := claims["aud"].(type) {
	case string:
		if aud != wantAud {
			t.Fatalf("aud=%v, want %q", aud, wantAud)
		}
	case []any:
		if len(aud) != 1 || aud[0] != wantAud {
			t.Fatalf("aud=%v, want [%q]", aud, wantAud)
		}
	default:
		t.Fatalf("aud=%v, want %q", aud, wantAud)
	}
	if out.RefreshToken == "" {
		t.Fatal("次のリフレッシュトークンがない")
	}
}

func assertRefreshRefusal(ctx context.Context, t *testing.T, f *tiFixture, token string, extra url.Values, wantError string) {
	t.Helper()
	before := len(f.events.emitted())
	status, body := f.postToken(t, refreshRequest(token, extra))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", status, body)
	}
	assertOAuthErrorCode(t, "リフレッシュの拒否", body, wantError)
	assertNoTokenInBody(t, "リフレッシュの拒否", body)
	parent := f.refreshRecord(ctx, t, token)
	if parent.Rotated || parent.Revoked {
		t.Fatalf("拒否が親を消費した: %+v", parent)
	}
	events := f.events.emitted()[before:]
	if wantError == "invalid_target" && !slices.Contains(events, "ResourceAudienceRejected") {
		t.Fatalf("対象の拒否イベントがない: %v", events)
	}
	for _, event := range events {
		if event == "RefreshTokenRotated" || event == "RefreshTokenIssued" || event == "AccessTokenIssued" || event == "ResourceScopedTokenIssued" {
			t.Fatalf("拒否後に発行イベント: %s", event)
		}
	}
}
