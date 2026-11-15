package handlers_http_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/labstack/echo/v5"
)

type transactionFailureStore struct {
	oauthports.AuthorizationRequestStore
	failure error
}

// transactionCodeStore は列挙APIのない保存先に索引だけを足し、実際の保存結果を読み戻す。
type transactionCodeStore struct {
	oauthports.AuthorizationCodeStore
	ids []string
}

func (s *transactionCodeStore) Save(ctx context.Context, code *domain.AuthorizationCodeRecord) error {
	if err := s.AuthorizationCodeStore.Save(ctx, code); err != nil {
		return err
	}
	s.ids = append(s.ids, code.Code)
	return nil
}

func (s *transactionFailureStore) Find(ctx context.Context, id string) (*domain.AuthorizationRequest, error) {
	if s.failure != nil {
		return nil, s.failure
	}
	return s.AuthorizationRequestStore.Find(ctx, id)
}

type transactionErrorFixture struct {
	e        *echo.Echo
	requests *transactionFailureStore
	auth     authentication.Module
	request  *domain.AuthorizationRequest
	codes    *transactionCodeStore
	user     *userdomain.User
	users    func() (*userdomain.User, error)
	events   *[]spec.DomainEvent
	cookie   string
	logs     bytes.Buffer
}

func newTransactionErrorFixture(t *testing.T, enrollment bool) *transactionErrorFixture {
	t.Helper()
	ctx := testing_tenant.Default(t.Context())
	now := time.Now().UTC()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("ValidPassword123!")
	if err != nil {
		t.Fatal(err)
	}
	users := idmanagement.NewMemoryModule().UserRepo
	user := &userdomain.User{
		ID: "transaction-user", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice", PasswordHash: hash,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	auth := authentication.NewMemoryModule(users)
	auth.PasswordHasher = hasher
	manager := auth.NewSessionManager()
	session, err := manager.CreateWithPending(ctx, user.ID, []string{"pwd"}, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if enrollment {
		if _, err := manager.RequireEnrollment(ctx, session.SessionID, now.Add(time.Hour), "bypass-1"); err != nil {
			t.Fatal(err)
		}
	} else {
		secret := "JBSWY3DPEHPK3PXP"
		if err := auth.MfaFactorRepo.Save(ctx, &totpdomain.MfaFactor{UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256([]byte("abcdefghijkl"))
	if err := auth.RecoveryCodeRepo.ReplaceAll(ctx, user.ID, []*recoverydomain.RecoveryCode{{UserID: user.ID, CodeHash: hex.EncodeToString(sum[:]), GeneratedAt: now}}); err != nil {
		t.Fatal(err)
	}
	rp, err := authentication.NewWebAuthn("localhost", "Transaction", []string{"http://localhost"})
	if err != nil {
		t.Fatal(err)
	}
	auth.WebAuthnRP = rp
	f := &transactionErrorFixture{
		requests: &transactionFailureStore{AuthorizationRequestStore: memory.NewAuthorizationRequestStore()}, auth: auth,
		codes: &transactionCodeStore{AuthorizationCodeStore: memory.NewAuthorizationCodeStore()}, user: user,
		cookie: authentication.SessionCookie + "=" + session.SessionID,
		users:  func() (*userdomain.User, error) { return users.FindBySub(ctx, user.ID) },
	}
	f.request = &domain.AuthorizationRequest{
		ID: "transaction-1", TenantID: user.TenantID, ClientID: authClientID,
		State: spec.AuthFlowAuthenticationPending, RedirectURI: authRedirectURI, Scope: "openid", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := f.requests.Save(ctx, f.request); err != nil {
		t.Fatal(err)
	}
	f.e, f.events = newAuthorizeTestServer(t, nil, nil, func(d *httpadapter.Deps) {
		d.UserRepo, d.Authentication, d.SessionManager = users, auth, manager
		d.OAuth2.RequestStore, d.OAuth2.CodeStore = f.requests, f.codes
	})
	f.e.HTTPErrorHandler = support.ErrorHandler(logging.New(&f.logs, slog.LevelError, "test", ""), nil)
	return f
}

func (f *transactionErrorFixture) snapshot(t *testing.T) string {
	t.Helper()
	ctx := testing_tenant.Default(t.Context())
	sessions, err := f.auth.SessionStore.ListBySub(ctx, f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	factors, err := f.auth.MfaFactorRepo.ListBySub(ctx, f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := f.auth.RecoveryCodeRepo.ListBySub(ctx, f.user.ID)
	if err != nil {
		t.Fatal(err)
	}
	request, err := f.requests.AuthorizationRequestStore.Find(ctx, f.request.ID)
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.users()
	if err != nil {
		t.Fatal(err)
	}
	var codes []*domain.AuthorizationCodeRecord
	for _, id := range f.codes.ids {
		code, err := f.codes.Find(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
	}
	raw, err := json.Marshal([]any{sessions, factors, recovery, request, user, codes, *f.events})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func (f *transactionErrorFixture) send(t *testing.T, method, path, returnTo, body string, transactionCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	if body == "" {
		body = `{}`
	}
	if method == http.MethodPost && returnTo != "" {
		body = strings.TrimSuffix(body, "}") + `,"return_to":"` + returnTo + `"}`
	} else if returnTo != "" {
		path += "?return_to=" + url.QueryEscape(returnTo)
	}
	request := httptest.NewRequest(method, "/realms/default"+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://test")
	request.Header.Set("X-Csrf-Token", "csrf-val")
	request.Header.Set("Cookie", "idmagic_csrf=csrf-val; "+f.cookie)
	if transactionCookie {
		request.AddCookie(&http.Cookie{Name: "idmagic_transaction", Value: f.request.ID})
	}
	request = request.WithContext(logging.ContextWithRequestID(request.Context(), "transaction-request"))
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

//spec:covers REQ-SYSTEM-023, REQ-AUTHENTICATION-007: 内部障害は全入口で500と相関ID付き診断へ写し、return_toでも認証と認可の保存状態を変えない。
func TestAuthorizationTransactionInternalErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New("transaction-private-marker: SQL constraint authorization_requests_pkey"),
		fmt.Errorf("read transaction: %w", errors.New("transaction-private-marker: runtime error\n/internal/store.go:42")),
	} {
		for _, endpoint := range []struct{ method, path, body string }{
			{http.MethodGet, "/api/auth/transaction", ""},
			{http.MethodGet, "/authorize/resume", ""},
			{http.MethodPost, "/api/auth/consent", `{"action":"allow"}`},
			{http.MethodPost, "/api/auth/login", `{"username":"alice","password":"ValidPassword123!"}`},
			{http.MethodPost, "/api/auth/totp", `{"code":"123456"}`},
			{http.MethodPost, "/api/auth/recovery-code", `{"code":"abcdefghijkl"}`},
			{http.MethodPost, "/api/auth/webauthn", `{"assertion":{}}`},
			{http.MethodPost, "/api/auth/mfa/enrollment/totp/confirm", `{"secret":"JBSWY3DPEHPK3PXP","code":"123456"}`},
			{http.MethodPost, "/api/auth/negotiate", `{}`},
		} {
			for _, returnTo := range []string{"", recoveryReturnTo} {
				// Negotiateの明示的な直接ログインは認可要求を読まない契約である。
				if endpoint.path == "/api/auth/negotiate" && returnTo != "" {
					continue
				}
				t.Run(endpoint.path+"/"+returnTo+"/"+failure.Error(), func(t *testing.T) {
					f := newTransactionErrorFixture(t, strings.Contains(endpoint.path, "enrollment"))
					before := f.snapshot(t)
					f.requests.failure = failure
					response := f.send(t, endpoint.method, endpoint.path, returnTo, endpoint.body, true)
					oauthContractProblem(t, response, http.StatusInternalServerError, "internal_server_error")
					if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "transaction-private-marker") || !strings.Contains(f.logs.String(), "transaction-private-marker") || !strings.Contains(f.logs.String(), "transaction-request") {
						t.Errorf("response=%s logs=%s", response.Body.String(), f.logs.String())
					}
					if after := f.snapshot(t); before != after {
						t.Fatalf("failure changed state: before=%s after=%s", before, after)
					}
				})
			}
		}
	}
}

//spec:covers REQ-AUTHENTICATION-007: 既知の利用不能だけは401となり、許可したreturn_toがあれば通常の直接ログインが成立する。
func TestAuthorizationTransactionUnavailable(t *testing.T) {
	for _, state := range []string{"no_cookie", "missing", "expired", "foreign_tenant", "processed", "valid"} {
		for _, returnTo := range []string{"", recoveryReturnTo} {
			t.Run(state+"/"+returnTo, func(t *testing.T) {
				f := newTransactionErrorFixture(t, false)
				switch state {
				case "missing":
					f.request.ID = "missing"
				case "expired":
					f.request.ExpiresAt = time.Now().Add(-time.Hour)
				case "foreign_tenant":
					f.request.TenantID = "foreign"
				case "processed":
					f.request.State = spec.AuthFlowCodeIssued
				}
				cookie := state != "no_cookie"
				response := f.send(t, http.MethodGet, "/api/auth/transaction", returnTo, "", cookie)
				if returnTo == "" && state != "valid" {
					oauthContractProblem(t, response, http.StatusUnauthorized, "transaction_unavailable")
				} else {
					body := oauthContractJSON(t, response, http.StatusOK)
					if body["kind"] != "totp" && body["kind"] != "login" {
						t.Fatalf("initialization=%v", body)
					}
				}
				response = f.send(t, http.MethodPost, "/api/auth/login", returnTo, `{"username":"alice","password":"ValidPassword123!"}`, cookie)
				if returnTo == "" && state != "valid" {
					oauthContractProblem(t, response, http.StatusUnauthorized, "transaction_unavailable")
				} else {
					body := oauthContractJSON(t, response, http.StatusOK)
					if state == "valid" && body["next"] != "/realms/default/consent" || state != "valid" && body["redirect_to"] != returnTo {
						t.Fatalf("direct login=%v", body)
					}
				}
			})
		}
	}
}
