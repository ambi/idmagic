package server_http_test

//spec:covers REQ-AUTHENTICATION-024: end-to-end coverage that a tenant's max_age_days
// reaches the login path. The use case is unit-tested in
// authentication/password/usecases; what these tests pin down is the wiring:
// the tenant override is resolved for the request, an expired password gates the
// login to change-password, and a login that is inside the window is untouched.

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
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
	server, users, _ := newPasswordExpiryServer(t, passwordExpiryOptions{
		maxAgeDays: maxAgeDays, passwordChangedDaysAgo: passwordChangedDaysAgo,
	})
	return server, users
}

// passwordExpiryOptions は有効期限の判定に効く 4 つの条件である。宣言済みの具体例は
// この 4 つのどれか 1 つだけを崩した形で書かれている。
type passwordExpiryOptions struct {
	maxAgeDays             int
	passwordChangedDaysAgo int
	// policyUpdatedDaysAgo はポリシーの更新から何日経っているか。0 のときは
	// maxAgeDays + 100 日前、つまり猶予を十分に過ぎた状態にする。
	policyUpdatedDaysAgo int
	// passwordless は利用者からパスワード資格情報を外す。
	passwordless bool
}

func newPasswordExpiryServer(
	t *testing.T, opts passwordExpiryOptions,
) (*httptest.Server, *usermemory.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	maxAgeDays, passwordChangedDaysAgo := opts.maxAgeDays, opts.passwordChangedDaysAgo
	now := time.Now().UTC()

	tenantRepo := tenancymemory.NewTenantRepository()
	policyUpdatedDaysAgo := opts.policyUpdatedDaysAgo
	if policyUpdatedDaysAgo == 0 {
		policyUpdatedDaysAgo = maxAgeDays + 100
	}
	policyUpdatedAt := now.AddDate(0, 0, -policyUpdatedDaysAgo)
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

	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash(expiryTestPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	userRepo := usermemory.NewUserRepository()
	passwordChangedAt := now.AddDate(0, 0, -passwordChangedDaysAgo)
	storedHash := hash
	if opts.passwordless {
		storedHash = ""
	}
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: expiryTestUsername, PasswordHash: storedHash,
		TenantID:  tenancydomain.DefaultTenantID,
		Lifecycle: userdomain.UserLifecycle{PasswordChangedAt: &passwordChangedAt},
		CreatedAt: passwordChangedAt, UpdatedAt: passwordChangedAt,
	})

	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)

	e := echo.New()
	events := &[]spec.DomainEvent{}
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		Emit:            func(event spec.DomainEvent) { *events = append(*events, event) },
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		TenantRepo:      tenantRepo,
		OAuth2: oauth2.Module{
			ClientRepo: oauth2memory.NewClientRepository(), ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(), CodeStore: oauth2memory.NewAuthorizationCodeStore(),
			PARStore: oauth2memory.NewPARStore(), RefreshStore: oauth2memory.NewRefreshTokenStore(),
		},
		UserRepo:            userRepo,
		PasswordHistoryRepo: passwordmemory.NewPasswordHistoryRepository(),
		PasswordHasher:      hasher, SessionManager: sessionManager, AuthnResolver: sessionManager,
	})
	return httptest.NewServer(e), userRepo, events
}

//spec:covers REQ-AUTHENTICATION-024, EX-AUTHENTICATION-024-01: 期限切れのパスワードでもログイン自体は成立し、update_password が付与されて変更画面へ誘導されること、変更を終えると必須操作が解除されて PasswordChanged が残ることを固定する。
func TestLoginWithExpiredPasswordIsGatedToChangePassword(t *testing.T) {
	srv, userRepo, events := newPasswordExpiryServer(t, passwordExpiryOptions{
		maxAgeDays: 90, passwordChangedDaysAgo: 91,
	})
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

	// ポリシーを満たす新しいパスワードへ変更すると必須操作が解ける。ここまで読まないと、
	// 立てるだけで外さない実装を通してしまい、利用者は変更後も同じ画面へ戻され続ける。
	const replacement = "fresh-expiry-password-9182"
	account := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/account")
	changed := postAuthJSON(t, client, srv.URL+"/realms/default/api/auth/change_password",
		account.CSRFToken, map[string]string{
			"current_password": expiryTestPassword, "new_password": replacement,
		})
	changedStatus := changed.StatusCode
	changed.Body.Close()
	if changedStatus != http.StatusNoContent {
		t.Fatalf("change_password status=%d, want 204", changedStatus)
	}
	stored, err = userRepo.FindBySub(t.Context(), "user_alice")
	if err != nil || stored == nil {
		t.Fatalf("load user: %+v %v", stored, err)
	}
	if len(stored.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("変更後も required actions=%v が残っている", stored.Lifecycle.RequiredActions)
	}
	assertEmitted(t, events, "PasswordChanged")
}

//spec:covers REQ-AUTHENTICATION-024, EX-AUTHENTICATION-024-04: ポリシーの更新から max_age_days が経過していない間は猶予期間として扱われ、期限切れのパスワードでも update_password が付与されないことを固定する。
func TestLoginWithinThePolicyGracePeriodIsNotGated(t *testing.T) {
	// パスワードは 91 日前、ポリシーの更新は 10 日前。パスワードの側だけを見る実装は
	// ここで必須操作を立ててしまう。
	srv, userRepo, _ := newPasswordExpiryServer(t, passwordExpiryOptions{
		maxAgeDays: 90, passwordChangedDaysAgo: 91, policyUpdatedDaysAgo: 10,
	})
	defer srv.Close()

	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	result := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login",
		transaction.CSRFToken, map[string]string{
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
		t.Fatalf("猶予期間内なのに required actions=%v", stored.Lifecycle.RequiredActions)
	}
}

//spec:covers REQ-AUTHENTICATION-024, EX-AUTHENTICATION-024-02: password_changed_at が 89 日前のログインはそのまま完了し、update_password が付与されないことを固定する。
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
//
//spec:covers REQ-AUTHENTICATION-024, EX-AUTHENTICATION-024-03: max_age_days が未設定なら、経過日数によらず update_password が付与されないことを固定する。
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
