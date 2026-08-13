package server_http_test

// REQ-AUTHENTICATION-024: end-to-end coverage that a tenant's max_age_days
// reaches the login path. The use case is unit-tested in
// authentication/password/usecases; what these tests pin down is the wiring:
// the tenant override is resolved for the request, an expired password gates the
// login to change-password, and a login that is inside the window is untouched.

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	passwordsArgon2id "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	expiryTestUsername = "alice"
	expiryTestPassword = "demo-password-1234"
)

// newPasswordExpiryTestServer seeds the default tenant with max_age_days=90,
// with the policy itself changed long enough ago that the grace window has
// passed, and one user whose password was last changed passwordChangedDaysAgo.
func newPasswordExpiryTestServer(t *testing.T, maxAgeDays, passwordChangedDaysAgo int) (*httptest.Server, *usermemory.UserRepository) {
	t.Helper()
	now := time.Now().UTC()

	tenantRepo := tenancymemory.NewTenantRepository()
	policyUpdatedAt := now.AddDate(0, 0, -(maxAgeDays + 100))
	tenant := &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		DisplayName: "Default", Status: tenancydomain.TenantStatusActive,
		PasswordPolicyUpdatedAt: &policyUpdatedAt,
		CreatedAt:               policyUpdatedAt, UpdatedAt: policyUpdatedAt,
	}
	if maxAgeDays > 0 {
		tenant.PasswordPolicyOverride = &tenancydomain.PasswordPolicyOverride{MaxAgeDays: &maxAgeDays}
	}
	if err := tenantRepo.Save(t.Context(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	hasher := passwordsArgon2id.NewArgon2idPasswordHasher()
	hash, err := hasher.Hash(expiryTestPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	userRepo := usermemory.NewUserRepository()
	passwordChangedAt := now.AddDate(0, 0, -passwordChangedDaysAgo)
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: expiryTestUsername, PasswordHash: hash,
		TenantID:  tenancydomain.DefaultTenantID,
		Lifecycle: userdomain.UserLifecycle{PasswordChangedAt: &passwordChangedAt},
		CreatedAt: passwordChangedAt, UpdatedAt: passwordChangedAt,
	})

	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:          "http://test",
			StartupComplete: startupComplete,
			ShuttingDown:    &atomic.Bool{},
			TenantRepo:      tenantRepo,
		},
		OAuth2: oauth2.Module{
			ClientRepo: oauth2memory.NewClientRepository(), ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(), CodeStore: oauth2memory.NewAuthorizationCodeStore(),
			PARStore: oauth2memory.NewPARStore(), RefreshStore: oauth2memory.NewRefreshTokenStore(),
		},
		UserRepo:       userRepo,
		PasswordHasher: hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
	})
	return httptest.NewServer(e), userRepo
}

func TestLoginWithExpiredPasswordIsGatedToChangePassword(t *testing.T) {
	srv, userRepo := newPasswordExpiryTestServer(t, 90, 91)
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	result := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": expiryTestUsername, "password": expiryTestPassword, "return_to": returnTo,
	})

	// The credentials themselves are accepted; only the follow-up screen differs.
	if result["next"] != "/realms/default/change_password" {
		t.Fatalf("login result=%+v, want a change_password gate", result)
	}
	stored, err := userRepo.FindBySub(t.Context(), "user_alice")
	if err != nil || stored == nil {
		t.Fatalf("load user: %+v %v", stored, err)
	}
	if len(stored.Lifecycle.RequiredActions) != 1 ||
		stored.Lifecycle.RequiredActions[0] != "update_password" {
		t.Fatalf("required actions=%v, want [update_password]", stored.Lifecycle.RequiredActions)
	}
}

func TestLoginWithinPasswordMaxAgeIsNotGated(t *testing.T) {
	srv, userRepo := newPasswordExpiryTestServer(t, 90, 89)
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	result := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": expiryTestUsername, "password": expiryTestPassword, "return_to": returnTo,
	})
	if result["next"] != "" || result["redirect_to"] != returnTo {
		t.Fatalf("login result=%+v, want the normal redirect", result)
	}
	stored, err := userRepo.FindBySub(t.Context(), "user_alice")
	if err != nil || stored == nil {
		t.Fatalf("load user: %+v %v", stored, err)
	}
	if len(stored.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("required actions=%v, want none", stored.Lifecycle.RequiredActions)
	}
}

// Without the tenant opt-in, however old the password is, nothing changes.
func TestLoginWithoutExpiryPolicyIsNeverGated(t *testing.T) {
	srv, _ := newPasswordExpiryTestServer(t, 0, 4000)
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	result := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": expiryTestUsername, "password": expiryTestPassword, "return_to": returnTo,
	})
	if result["next"] != "" || result["redirect_to"] != returnTo {
		t.Fatalf("login result=%+v, want the normal redirect", result)
	}
}
