package handlers_http_test

// サインインポリシーの保存側 (REQ-APPLICATION-009 と REQ-APPLICATION-010) が宣言する
// 具体例を、管理 API の入口で観測する。保存した規則がフェデレーションの可否を実際に
// 変えることは backend/oauth2/handlers_http の同じ id を名指すテストが固定する。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func readAppPolicyRules(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, applicationID string) []map[string]any {
	t.Helper()
	view := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID+"/sign-in-policy", csrf, cookie, nil)
	if view.Code != http.StatusOK {
		t.Fatalf("get app policy status=%d body=%s", view.Code, view.Body.String())
	}
	var decoded struct {
		Policy struct {
			Rules []map[string]any `json:"rules"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(view.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode policy view: %v; body=%s", err, view.Body.String())
	}
	return decoded.Policy.Rules
}

//spec:covers REQ-APPLICATION-009, EX-APPLICATION-009-01: MFA 必須と再認証までの秒数を指定したサインインポリシーの保存が AppSignInPolicyUpdated を発行し、その規則を読み直せること。強制点での評価は TestAppSignInPolicyIsEvaluatedBeforeTheAuthorizationCodeIsIssued が固定する。
func TestUpdateAppSignInPolicyEmitsItsEvent(t *testing.T) {
	e, events, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})

	updated := adminJSON(t, e, http.MethodPut, "/api/admin/v1/applications/"+applicationID+"/sign-in-policy", csrf, cookie, map[string]any{
		"rules": []map[string]any{{
			"name": "MFA", "enabled": true,
			"required_authn": map[string]any{"strength": "Mfa"},
			"condition":      map[string]any{"reauth_max_age_seconds": 900},
		}},
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("put app policy status=%d body=%s", updated.Code, updated.Body.String())
	}
	if !events.contains("AppSignInPolicyUpdated") {
		t.Fatalf("events=%v, want AppSignInPolicyUpdated", events.types())
	}

	rules := readAppPolicyRules(t, e, csrf, cookie, applicationID)
	if len(rules) != 1 {
		t.Fatalf("rules=%+v, want the saved policy", rules)
	}
	condition, _ := rules[0]["condition"].(map[string]any)
	if condition["reauth_max_age_seconds"] != float64(900) {
		t.Fatalf("reauth_max_age_seconds=%v, want the saved 900", condition["reauth_max_age_seconds"])
	}
}

//spec:covers REQ-APPLICATION-009, EX-APPLICATION-009-02: admin ロールを持たない利用者によるサインインポリシーの更新が 403 access_denied で拒否され、保存済みの規則が書き換わらず、更新イベントも発行されないこと。
func TestUpdateAppSignInPolicyRefusesANonAdmin(t *testing.T) {
	e, events, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})
	seeded := adminJSON(t, e, http.MethodPut, "/api/admin/v1/applications/"+applicationID+"/sign-in-policy", csrf, cookie, map[string]any{
		"rules": []map[string]any{{"name": "MFA", "enabled": true, "required_authn": map[string]any{"strength": "Mfa"}}},
	})
	if seeded.Code != http.StatusOK {
		t.Fatalf("seed policy status=%d body=%s", seeded.Code, seeded.Body.String())
	}
	before := readAppPolicyRules(t, e, csrf, cookie, applicationID)
	events.reset()

	request := httptest.NewRequest(http.MethodPut,
		"/realms/default/api/admin/v1/applications/"+applicationID+"/sign-in-policy",
		strings.NewReader(`{"rules":[{"name":"Password","enabled":true,"required_authn":{"strength":"Password"}}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "regular")
	request.AddCookie(cookie)
	refused := httptest.NewRecorder()
	e.ServeHTTP(refused, request)

	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(refused.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, refused.Body.String())
	}
	if problem.Type != "urn:idmagic:error:access_denied" {
		t.Fatalf("problem type=%q, want the access_denied refusal", problem.Type)
	}

	// 拒否が防いだ効果。保存済みの規則も、更新イベントも動かない。
	after := readAppPolicyRules(t, e, csrf, cookie, applicationID)
	if len(after) != len(before) {
		t.Fatalf("rules changed after the refusal: %+v, want %+v", after, before)
	}
	strength, _ := after[0]["required_authn"].(map[string]any)
	if strength["strength"] != "Mfa" {
		t.Fatalf("required_authn=%v after the refusal, want the seeded Mfa", strength["strength"])
	}
	if events.contains("AppSignInPolicyUpdated") {
		t.Fatalf("events=%v, want no update event after the refusal", events.types())
	}
}

//spec:covers REQ-APPLICATION-010, EX-APPLICATION-010-02: 規則を空にしたテナントデフォルトの保存が TenantDefaultSignInPolicyUpdated を発行し、独自ポリシーを持たない Application の effective_rules を空のままにすること。フェデレーションに追加要件を課さない側は TestEmptyTenantDefaultSignInPolicyImposesNoExtraRequirement が固定する。
func TestEmptyTenantDefaultSignInPolicyEmitsItsEventAndImposesNothing(t *testing.T) {
	e, events, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})

	emptied := adminJSON(t, e, http.MethodPut, "/api/admin/v1/default-sign-in-policy", csrf, cookie, map[string]any{
		"rules": []map[string]any{},
	})
	if emptied.Code != http.StatusOK {
		t.Fatalf("put default status=%d body=%s", emptied.Code, emptied.Body.String())
	}
	if !events.contains("TenantDefaultSignInPolicyUpdated") {
		t.Fatalf("events=%v, want TenantDefaultSignInPolicyUpdated", events.types())
	}

	view := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID+"/sign-in-policy", csrf, cookie, nil)
	var decoded struct {
		EffectiveRules []map[string]any `json:"effective_rules"`
	}
	if err := json.Unmarshal(view.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode policy view: %v; body=%s", err, view.Body.String())
	}
	if len(decoded.EffectiveRules) != 0 {
		t.Fatalf("effective_rules=%+v, want nothing imposed", decoded.EffectiveRules)
	}
}
