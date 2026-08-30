package handlers_http_test

// ブラウザー経路の MFA 登録 (POST /api/auth/mfa/enrollment/totp/{start,confirm}) が、
// 「登録が許可されていない」と「既に登録済み」を別の code で返すことを固定する。
// 管理 API と account API は後者を 409 mfa_already_enrolled として返しており、
// ブラウザー経路だけが両者を 403 mfa_enrollment_not_allowed へ畳んでいた。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// newEnrollmentServer は「登録待ちのセッションを持つ利用者」を組み立てる。
// enrolled が true なら、その利用者は既に TOTP factor を持っている。
func newEnrollmentServer(t *testing.T, enrolled bool) *echo.Echo {
	t.Helper()
	now := time.Now().UTC()
	deadline := now.Add(24 * time.Hour)

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: "alice",
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	factorRepo := totpmemory.NewMfaFactorRepository()
	if enrolled {
		secret := "JBSWY3DPEHPK3PXP"
		if err := factorRepo.Save(context.Background(), &totpdomain.MfaFactor{
			UserID: "user_alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:   "http://test",
		UserRepo: userRepo,
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
			UserID: "user_alice", SessionID: "sess-enroll", AuthTime: now.Unix(), AMR: []string{"pwd"},
			AuthenticationPending: true,
			PendingPurpose:        authdomain.LoginPendingEnrollment,
			EnrollmentDeadline:    &deadline,
		}},
		MfaFactorRepo: factorRepo,
	})
	return e
}

func postEnrollment(t *testing.T, e *echo.Echo, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/realms/default"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", "csrf-val")
	req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func problemOf(t *testing.T, rec *httptest.ResponseRecorder) (string, int) {
	t.Helper()
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v (body=%s)", err, rec.Body.String())
	}
	return problem.Type, problem.Status
}

func TestBrowserMfaEnrollmentSeparatesAlreadyEnrolledFromNotAllowed(t *testing.T) {
	// 既に登録済みの利用者が登録を開始しようとした場合。利用者から見て
	// 「管理者に問い合わせる」状況ではなく「何もしなくてよい」状況なので、
	// 登録が許可されていない場合とは別の code でなければ画面が案内を分けられない。
	t.Run("既に登録済みなら 409 mfa_already_enrolled", func(t *testing.T) {
		e := newEnrollmentServer(t, true)
		rec := postEnrollment(t, e, "/api/auth/mfa/enrollment/totp/start", `{}`)
		problemType, status := problemOf(t, rec)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status=%d body=%s, want 409", rec.Code, rec.Body.String())
		}
		if problemType != "urn:idmagic:error:mfa_already_enrolled" {
			t.Fatalf("type=%q, want urn:idmagic:error:mfa_already_enrolled", problemType)
		}
		if status != http.StatusConflict {
			t.Fatalf("problem status=%d, want 409", status)
		}
		// 拒否が何も通していないこと。応答は登録開始の本体を返していない。
		if strings.Contains(rec.Body.String(), `"secret"`) {
			t.Fatalf("refused start returned an enrollment secret: %s", rec.Body.String())
		}
	})

	t.Run("確認の側も同じ code を返す", func(t *testing.T) {
		e := newEnrollmentServer(t, true)
		rec := postEnrollment(t, e, "/api/auth/mfa/enrollment/totp/confirm",
			`{"secret":"JBSWY3DPEHPK3PXP","code":"000000","return_to":"/realms/default/admin"}`)
		problemType, _ := problemOf(t, rec)
		if rec.Code != http.StatusConflict || problemType != "urn:idmagic:error:mfa_already_enrolled" {
			t.Fatalf("status=%d type=%q body=%s, want 409 mfa_already_enrolled",
				rec.Code, problemType, rec.Body.String())
		}
	})

	t.Run("登録が許可されていない場合は 403 のまま", func(t *testing.T) {
		// 登録待ちでないセッション。ここは fail-closed の側なので変えない。
		e := echo.New()
		httpadapter.Register(e, httpadapter.Deps{
			Issuer:        "http://test",
			UserRepo:      usermemory.NewUserRepository(),
			MfaFactorRepo: totpmemory.NewMfaFactorRepository(),
		})
		rec := postEnrollment(t, e, "/api/auth/mfa/enrollment/totp/start", `{}`)
		problemType, _ := problemOf(t, rec)
		if rec.Code != http.StatusForbidden || problemType != "urn:idmagic:error:mfa_enrollment_not_allowed" {
			t.Fatalf("status=%d type=%q body=%s, want 403 mfa_enrollment_not_allowed",
				rec.Code, problemType, rec.Body.String())
		}
	})
}
