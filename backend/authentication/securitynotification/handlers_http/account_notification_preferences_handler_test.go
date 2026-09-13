package handlers_http_test

// /api/account/v1/notification_preferences の取得と更新 (wi-90)。ここで確かめるのは、
// 全種別が必須の印つきで返ること、必須の種別を止める要求が丸ごと拒否されること、
// そして停止した設定がそのまま読み戻せることである。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	securitynotificationmemory "github.com/ambi/idmagic/backend/authentication/securitynotification/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const preferencesPath = "/realms/default/api/account/v1/notification_preferences"

func newPreferencesServer(t *testing.T) (*echo.Echo, string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user-1", PreferredUsername: "alice", TenantID: tenancydomain.DefaultTenantID,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	})
	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(ctx, &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	store := sessionmemory.NewSessionStore()
	sm := sessionusecases.NewSessionManager(store)
	federationRepos := federationmemory.NewRepositories()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(), TenantRepo: tenantRepo,
		Emit:           func(spec.DomainEvent) {},
		UserRepo:       userRepo,
		AttrSchemaRepo: usermemory.NewTenantUserAttributeSchemaRepository(),
		SessionManager: sm, AuthnResolver: sm,
		Authentication: authentication.Module{
			FederationConnectionRepo:   federationRepos.Connections,
			FederationIdentityRepo:     federationRepos.Identities,
			FederationAttemptStore:     federationRepos.Attempts,
			FederationReplayStore:      federationRepos.Replay,
			NotificationPreferenceRepo: securitynotificationmemory.NewPreferenceRepository(),
		},
	})

	// step-up 済み (認証直後) のセッション。更新はこの直近性を要求する。
	sessionID := "sess-fresh"
	if err := store.Save(ctx, &sessiondomain.LoginSession{
		ID: sessionID, TenantID: tenancydomain.DefaultTenantID, UserID: "user-1",
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: authusecases.DeriveACR([]string{"pwd"}),
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// 直近性を満たさないセッション。認証時刻も step-up も過去にある。
	if err := store.Save(ctx, &sessiondomain.LoginSession{
		ID: staleSessionID, TenantID: tenancydomain.DefaultTenantID, UserID: "user-1",
		AuthTime: now.Add(-time.Hour).Unix(), AMR: []string{"pwd"},
		ACR: authusecases.DeriveACR([]string{"pwd"}), ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	return e, sessionID
}

// staleSessionID はステップアップの直近性を満たさないセッション。
const staleSessionID = "sess-stale"

func preferencesRequest(
	t *testing.T, e *echo.Echo, method, sessionID string, body any,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, preferencesPath, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", "csrf-token-value")
	req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf-token-value"})
	req.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: sessionID})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

type categoriesBody struct {
	Categories []struct {
		Category  string `json:"category"`
		Mandatory bool   `json:"mandatory"`
		Enabled   bool   `json:"enabled"`
	} `json:"categories"`
}

func decodeCategories(t *testing.T, rec *httptest.ResponseRecorder) categoriesBody {
	t.Helper()
	var body categoriesBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return body
}

//spec:covers REQ-AUTHENTICATION-033, EX-AUTHENTICATION-033-01: 通知設定の取得で全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別に mandatory が付くことを固定する。
func TestGetNotificationPreferencesReturnsTheWholeCatalog(t *testing.T) {
	e, sessionID := newPreferencesServer(t)

	rec := preferencesRequest(t, e, http.MethodGet, sessionID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	body := decodeCategories(t, rec)
	if len(body.Categories) != 6 {
		t.Fatalf("returned %d categories, want the whole catalog", len(body.Categories))
	}
	mandatory := map[string]bool{}
	for _, category := range body.Categories {
		mandatory[category.Category] = category.Mandatory
		if !category.Enabled {
			t.Errorf("%s is disabled before any change was made", category.Category)
		}
	}
	for _, name := range []string{"credential_change", "mfa_change", "contact_change", "impersonation"} {
		if !mandatory[name] {
			t.Errorf("%s must be reported as mandatory", name)
		}
	}
	for _, name := range []string{"new_device_sign_in", "session_revoked"} {
		if mandatory[name] {
			t.Errorf("%s must be reported as optional", name)
		}
	}
}

//spec:covers REQ-AUTHENTICATION-034: 停止した種別だけが無効になり、読み戻せる。
func TestUpdateNotificationPreferencesDisablesOnlyTheNamedCategories(t *testing.T) {
	e, sessionID := newPreferencesServer(t)

	rec := preferencesRequest(t, e, http.MethodPut, sessionID,
		map[string]any{"disabled_categories": []string{"new_device_sign_in"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	for _, category := range decodeCategories(t, rec).Categories {
		want := category.Category != "new_device_sign_in"
		if category.Enabled != want {
			t.Errorf("%s: Enabled=%v, want %v", category.Category, category.Enabled, want)
		}
	}

	reread := decodeCategories(t, preferencesRequest(t, e, http.MethodGet, sessionID, nil))
	for _, category := range reread.Categories {
		if category.Category == "new_device_sign_in" && category.Enabled {
			t.Error("the disabled category came back enabled on re-read")
		}
	}
}

//spec:covers REQ-AUTHENTICATION-033, EX-AUTHENTICATION-033-01: 必須の種別を含む更新は 400 で拒否され、同じ要求に含まれた許された種別も保存されないことを固定する。
func TestUpdateNotificationPreferencesRejectsMandatoryCategories(t *testing.T) {
	e, sessionID := newPreferencesServer(t)

	rec := preferencesRequest(t, e, http.MethodPut, sessionID,
		map[string]any{"disabled_categories": []string{"new_device_sign_in", "credential_change"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Type != "urn:idmagic:error:mandatory_notification_category" {
		t.Errorf("type=%q, want urn:idmagic:error:mandatory_notification_category", problem.Type)
	}

	for _, category := range decodeCategories(t, preferencesRequest(t, e, http.MethodGet, sessionID, nil)).Categories {
		if !category.Enabled {
			t.Errorf("%s was disabled by a request that must have been rejected whole", category.Category)
		}
	}
}

// 未知の種別は保存せず拒否する。
func TestUpdateNotificationPreferencesRejectsUnknownCategories(t *testing.T) {
	e, sessionID := newPreferencesServer(t)

	rec := preferencesRequest(t, e, http.MethodPut, sessionID,
		map[string]any{"disabled_categories": []string{"does_not_exist"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}
}

// ステップアップを成立させていないセッションからの更新は、再認証を要求される。
//
// 通知設定の停止は「気づける経路を自分で閉じる」操作なので、機微操作と同じ再認証が要る。
// 応答だけを読むテストでは、403 を書いたうえで保存も続ける実装を見分けられないので、
// 取得側から読み直す。
//
//spec:covers REQ-AUTHENTICATION-033, EX-AUTHENTICATION-033-02: ステップアップを成立させていないセッションからの通知設定の更新が step_up_required で拒否され、設定がいずれの種別についても変わらないことを固定する。
func TestUpdateNotificationPreferencesWithoutStepUpChangesNothing(t *testing.T) {
	e, fresh := newPreferencesServer(t)

	refused := preferencesRequest(t, e, http.MethodPut, staleSessionID, map[string]any{
		"disabled_categories": []string{"new_device_sign_in"},
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
	}

	// 設定は取得側からしか読めない。すべての種別が有効のままであることを確かめる。
	listed := preferencesRequest(t, e, http.MethodGet, fresh, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("取得 status=%d body=%s", listed.Code, listed.Body.String())
	}
	for _, category := range decodeCategories(t, listed).Categories {
		if !category.Enabled {
			t.Fatalf("拒否されたのに %s が無効になった", category.Category)
		}
	}

	// 対照: ステップアップを満たすセッションでは同じ更新が通る。拒否の理由が
	// 直近性であって、要求の中身でも CSRF でもないと示す。
	accepted := preferencesRequest(t, e, http.MethodPut, fresh, map[string]any{
		"disabled_categories": []string{"new_device_sign_in"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: ステップアップ済みの更新が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}
