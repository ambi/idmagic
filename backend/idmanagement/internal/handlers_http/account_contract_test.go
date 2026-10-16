package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func idmContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d Content-Type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Fatalf("JSON object=%v err=%v", body, err)
	}
	return body
}

func idmContractProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d Content-Type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["type"] != "urn:idmagic:error:"+code || body["status"] != float64(status) {
		t.Fatalf("Problem Details=%v err=%v", body, err)
	}
}

func idmContractEmpty(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s, want empty 204", rec.Code, rec.Body.String())
	}
}

func idmContractTime(t *testing.T, body map[string]any, key string) {
	t.Helper()
	value, ok := body[key].(string)
	if parsed, err := time.Parse(time.RFC3339Nano, value); !ok || err != nil || parsed.IsZero() {
		t.Fatalf("%s=%v, want RFC3339 timestamp", key, body[key])
	}
}

func idmContractProfile(t *testing.T, body map[string]any, userID, name string) {
	t.Helper()
	if body["id"] != userID || body["preferred_username"] != userID || body["name"] != name || body["status"] != string(idmdomain.UserStatusActive) || body["email_verified"] != true || body["mfa_enrolled"] != false {
		t.Fatalf("account profile=%v", body)
	}
	if _, ok := body["attributes"].(map[string]any); !ok {
		t.Fatalf("attributes=%v", body["attributes"])
	}
	if _, ok := body["editable_attributes"].([]any); !ok {
		t.Fatalf("editable_attributes=%v", body["editable_attributes"])
	}
	if _, present := body["roles"]; present {
		t.Fatalf("self profile exposes roles: %v", body)
	}
}

//verify:operation GetUserProfile: 本人 profile と summary の JSON 応答、更新および保存結果
//verify:operation UpdateUserProfile: 本人 profile と summary の JSON 応答、更新および保存結果
//verify:operation GetAccountSummary: 本人 profile と summary の JSON 応答、更新および保存結果
func TestAccountProfileHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	read := f.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountRead)
	write := f.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountWrite)
	profile := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/account/v1/profile", bearer: read}), http.StatusOK)
	idmContractProfile(t, profile, idmRefusalAlice, idmRefusalAlice)
	summary := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/account/v1/summary", bearer: read}), http.StatusOK)
	for _, key := range []string{"id", "preferred_username", "name", "email", "email_verified", "mfa_enrolled", "status"} {
		if summary[key] != profile[key] {
			t.Fatalf("summary %s=%v, profile=%v", key, summary[key], profile[key])
		}
	}
	if actions, ok := summary["required_actions"].([]any); !ok || len(actions) != 0 {
		t.Fatalf("required_actions=%v", summary["required_actions"])
	}
	if _, present := summary["roles"]; present {
		t.Fatalf("summary exposes roles: %v", summary)
	}
	updated := idmContractJSON(t, f.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: write,
		body: map[string]any{"name": "Alice Contract", "given_name": "Alice", "family_name": "Contract"},
	}), http.StatusOK)
	idmContractProfile(t, updated, idmRefusalAlice, "Alice Contract")
	if updated["given_name"] != "Alice" || updated["family_name"] != "Contract" {
		t.Fatalf("updated profile=%v", updated)
	}
	reloaded := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/account/v1/profile", bearer: read}), http.StatusOK)
	if !reflect.DeepEqual(reloaded, updated) {
		t.Fatalf("profile was not saved: updated=%v reloaded=%v", updated, reloaded)
	}
	idmContractProblem(t, f.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: read, body: map[string]any{"name": "Denied"},
	}), http.StatusForbidden, "insufficient_scope")
	if name := f.user(t, idmRefusalAlice).Name; name == nil || *name != "Alice Contract" {
		t.Fatalf("refused update changed name=%v", name)
	}
}

//verify:operation ExportAccountData: 本人データを含む 200 の JSON 応答
func TestAccountDataExportHTTPContract(t *testing.T) {
	f := newIdmRefusalServer(t)
	read := f.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountRead)
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: idmRefusalAlice, ClientID: idmRefusalClient, Scopes: []string{"openid", "profile"},
		State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	exported := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/account/v1/data-export", bearer: read}), http.StatusOK)
	idmContractTime(t, exported, "exported_at")
	profile, ok := exported["profile"].(map[string]any)
	if !ok {
		t.Fatalf("exported profile=%v", exported["profile"])
	}
	idmContractProfile(t, profile, idmRefusalAlice, idmRefusalAlice)
	consents, ok := exported["consents"].([]any)
	if !ok || len(consents) != 1 {
		t.Fatalf("consents=%v", exported["consents"])
	}
	consent := consents[0].(map[string]any)
	if consent["client_id"] != idmRefusalClient || consent["state"] != string(oauthdomain.ConsentGranted) || !reflect.DeepEqual(consent["scopes"], []any{"openid", "profile"}) {
		t.Fatalf("consent=%v", consent)
	}
	idmContractTime(t, consent, "granted_at")
	idmContractTime(t, consent, "expires_at")
}

//verify:operation GetUserProfile: 未認証時の 401 Problem Details
//verify:operation GetAccountSummary: 未認証時の 401 Problem Details
//verify:operation ExportAccountData: 未認証時の 401 Problem Details
//verify:operation UpdateUserProfile: 未認証時の 401 Problem Details
//verify:operation RequestEmailChange: 未認証時の 401 Problem Details
func TestAccountHTTPContractsRefuseUnauthenticated(t *testing.T) {
	f := newIdmRefusalServer(t)
	for _, test := range []struct{ method, path string }{
		{http.MethodGet, "/api/account/v1/profile"},
		{http.MethodGet, "/api/account/v1/summary"},
		{http.MethodGet, "/api/account/v1/data-export"},
		{http.MethodPatch, "/api/account/v1/profile"},
		{http.MethodPost, "/api/account/v1/email/change-request"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			response := f.send(t, idmRefusalRequest{method: test.method, path: test.path, csrf: idmRefusalCSRF, body: map[string]any{}})
			idmContractProblem(t, response, http.StatusUnauthorized, "authentication_required")
		})
	}
	if len(f.emails.Sent) != 0 || len(*f.events) != 0 {
		t.Fatal("unauthenticated requests emitted changes")
	}
}

//spec:covers REQ-IDMANAGEMENT-053, EX-IDMANAGEMENT-053-05: 有効期限切れのステップアップ認証を 403 で拒否し、メールとユーザーを変えないこと
//verify:operation RequestEmailChange: 403 step_up_required の Problem Details、確認メールとアドレスに変化がないこと
func TestAccountEmailChangeHTTPContractRefusesExpiredStepUp(t *testing.T) {
	f := newIdmRefusalServer(t)
	session := f.seedSession(t, "email-contract-expired", tenancydomain.DefaultTenantID, idmRefusalAlice, func(session *sessiondomain.LoginSession) {
		expired := time.Now().Add(-10 * time.Minute).Unix()
		session.AuthTime = expired
		session.StepUpAt = expired
	})
	before := *f.user(t, idmRefusalAlice).Email
	rec := f.send(t, idmRefusalRequest{method: http.MethodPost, path: "/api/account/v1/email/change-request", sessionID: session, csrf: idmRefusalCSRF, body: map[string]any{"new_email": "alice-contract@example.test"}})
	idmContractProblem(t, rec, http.StatusForbidden, "step_up_required")
	if len(f.emails.Sent) != 0 || len(*f.events) != 0 || *f.user(t, idmRefusalAlice).Email != before {
		t.Fatal("step-up refusal sent email, emitted events, or changed the address")
	}
}

//verify:operation RequestEmailChange: メール変更の要求と確認の HTTP 応答、検証コードと保存したアドレス
//verify:operation ConfirmEmailChange: メール変更の要求と確認の HTTP 応答、検証コードと保存したアドレス
func TestAccountEmailChangeHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	session := f.seedSession(t, "email-contract", tenancydomain.DefaultTenantID, idmRefusalAlice, withFreshStepUp)
	request := func(email string) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: http.MethodPost, path: "/api/account/v1/email/change-request", sessionID: session, csrf: idmRefusalCSRF, body: map[string]any{"new_email": email}})
	}
	idmContractProblem(t, request("invalid"), http.StatusUnprocessableEntity, "invalid_email")
	idmContractProblem(t, request(idmRefusalBob+"@example.test"), http.StatusConflict, "email_taken")
	before := *f.user(t, idmRefusalAlice).Email
	idmContractEmpty(t, request("alice-contract@example.test"))
	if len(f.emails.Sent) != 1 || *f.user(t, idmRefusalAlice).Email != before {
		t.Fatal("email request did not send one confirmation while preserving the old address")
	}
	match := emailChangeTokenPattern.FindStringSubmatch(f.emails.Sent[0].Text)
	if len(match) != 2 {
		t.Fatal("confirmation token missing")
	}
	confirm := func(csrf string) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: http.MethodPost, path: "/api/account/v1/email/verify", csrf: csrf, body: map[string]any{"token": match[1]}})
	}
	idmContractProblem(t, confirm(""), http.StatusForbidden, "csrf_failed")
	if *f.user(t, idmRefusalAlice).Email != before {
		t.Fatal("CSRF refusal changed the email")
	}
	confirmed := idmContractJSON(t, confirm(idmRefusalCSRF), http.StatusOK)
	if confirmed["status"] != "ok" {
		t.Fatalf("confirmation=%v", confirmed)
	}
	user := f.user(t, idmRefusalAlice)
	if user.Email == nil || *user.Email != "alice-contract@example.test" || !user.EmailVerified {
		t.Fatalf("confirmed user=%+v", user)
	}
	idmContractProblem(t, confirm(idmRefusalCSRF), http.StatusGone, "invalid_email_change_token")
	idmContractProblem(t, request("alice-contract@example.test"), http.StatusUnprocessableEntity, "email_unchanged")
}
