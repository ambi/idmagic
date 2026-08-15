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
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(), TenantRepo: tenantRepo,
			Emit: func(spec.DomainEvent) {},
		},
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
	return e, sessionID
}

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

// REQ-AUTHENTICATION-033: 全種別が返り、必須の種別には mandatory が付く。
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

// REQ-AUTHENTICATION-034: 停止した種別だけが無効になり、読み戻せる。
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

// REQ-AUTHENTICATION-033: 必須の種別を含む更新は 400 で拒否し、許された分も保存しない。
func TestUpdateNotificationPreferencesRejectsMandatoryCategories(t *testing.T) {
	e, sessionID := newPreferencesServer(t)

	rec := preferencesRequest(t, e, http.MethodPut, sessionID,
		map[string]any{"disabled_categories": []string{"new_device_sign_in", "credential_change"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}
	var problem map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem["error"] != "mandatory_notification_category" {
		t.Errorf("error=%v, want mandatory_notification_category", problem["error"])
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
