package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// rootTestCSRF は /api/auth/password_reset_context (CSRF cookie 発行専用の GET) を叩いて
// CSRF token/cookie を得る。password feature の password_reset_handler_test.go の
// passwordResetCSRF と同じ実装だが、_test.go はパッケージを跨げないため複製する
// (ADR-130 Phase 2 と同方針)。
func rootTestCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/auth/password_reset_context", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("context status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 || body.CSRFToken == "" {
		t.Fatalf("csrf=%q cookies=%v", body.CSRFToken, cookies)
	}
	return body.CSRFToken, cookies[0]
}

func TestDisabledUserCannotLogIn(t *testing.T) {
	repo := usermemory.NewUserRepository()
	requestStore := memory.NewAuthorizationRequestStore()
	hasher := crypto.NewArgon2idPasswordHasher()
	hash, err := hasher.Hash("current-password-1")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	repo.Seed(&userdomain.User{
		ID: "disabled", PreferredUsername: "disabled", PasswordHash: hash,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled, StatusChangedAt: &now},
		CreatedAt: now, UpdatedAt: now,
	})
	if err := requestStore.Save(context.Background(), &domain.AuthorizationRequest{
		ID: "transaction", State: spec.AuthFlowReceived, ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{Issuer: "http://idp.test"}, UserRepo: repo,
		OAuth2:         oauth2.Module{RequestStore: requestStore},
		PasswordHasher: hasher,
	})
	csrf, csrfCookie := rootTestCSRF(t, e)
	requestBody, err := json.Marshal(map[string]string{
		"username": "disabled", "password": "current-password-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(csrfCookie)
	request.AddCookie(&http.Cookie{Name: "idmagic_transaction", Value: "transaction"})
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"invalid_credentials"`) {
		t.Fatalf("unexpected body=%s", response.Body.String())
	}
}
