package handlers_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type diagnosticGate struct{ retry, records int }

func (g *diagnosticGate) Check(context.Context, string, string) (int, error) { return g.retry, nil }
func (g *diagnosticGate) Record(context.Context, string, string, bool) error { g.records++; return nil }

//spec:covers REQ-SOURCING-026 REQ-SOURCING-027 REQ-SOURCING-028: HTTP の鍵の登録、設定、配送と結果を相関し、認可、別テナント、別 Connector と相反する結果を拒否する。
//verify:operation RegisterDirectoryEncryptionCapability: PUT で別の公開鍵を登録し、200 と key_id を返す。署名鍵の共用は 400。
//verify:operation ConfigureDirectoryDelegation: PUT の enabled を保存して 200 の本文へ反映し、権限不足、CSRF と別テナントを拒否する。
//verify:operation PollDirectoryPasswordChallenge: POST の署名 JWT を認証して 200 の配送情報と暗号文を返し、要求がなければ 204。
//verify:operation CompleteDirectoryPasswordChallenge: POST の結果へ 204 を返し、別の Connector と相反する結果を 409 で拒否する。
//verify:operation TestDirectoryDelegation: POST のユーザーとパスワードから照合し、詳細結果を 200 で返す。権限不足、別テナントと試行の制限を拒否する。
func TestDelegationHTTPContracts(t *testing.T) {
	s := newServer(t)
	enrollment, key := s.enrolled("AD")
	api := func(method, path string, body any) *httptest.ResponseRecorder {
		return s.do(call{method: method, path: path, body: body, bearer: s.sign(key, enrollment, nil)})
	}
	capability := "/directory-connector/v1/encryption-capability"
	if rec := api(http.MethodPut, capability, map[string]any{"jwk": key.jwk}); rec.Code != 400 {
		t.Fatalf("shared key=%d %s", rec.Code, rec.Body.String())
	}
	encryption := newKey(t)
	rec := api(http.MethodPut, capability, map[string]any{"jwk": encryption.jwk})
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("capability=%d %s", rec.Code, rec.Body.String())
	}
	registered := decode[map[string]string](t, rec)
	if registered["key_id"] == "" || registered["key_id"] == enrollment.KeyID {
		t.Fatal("encryption key is not separate")
	}
	userID := "directory-user"
	if err := s.users.Save(t.Context(), &userdomain.User{ID: userID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "directory-user", CreatedAt: s.now, UpdatedAt: s.now}); err != nil {
		t.Fatal(err)
	}
	if err := s.correlations.SaveUserCorrelation(t.Context(), directoryports.UserCorrelation{TenantID: tenancydomain.DefaultTenantID, ConnectorID: enrollment.ConnectorID, ExternalID: "external-user", UserID: userID}); err != nil {
		t.Fatal(err)
	}
	adminPath := "/api/admin/v1/directory-connectors/" + enrollment.ConnectorID + "/delegated-authentication"
	for _, attempt := range []call{
		{method: http.MethodPut, path: adminPath, body: map[string]bool{"enabled": true}, user: "member"},
		{method: http.MethodPut, path: adminPath, body: map[string]bool{"enabled": true}, user: "admin", noCSRF: true},
	} {
		if rec := s.do(attempt); rec.Code != 403 {
			t.Fatalf("setting authorization=%d %s", rec.Code, rec.Body.String())
		}
	}
	if rec := s.do(call{method: http.MethodPut, path: adminPath, body: map[string]bool{"enabled": true}, user: "other-admin", realm: "other"}); rec.Code != 404 {
		t.Fatalf("cross-tenant setting=%d", rec.Code)
	}
	if rec := s.do(call{method: http.MethodPut, path: adminPath, body: map[string]bool{"enabled": true}, user: "admin"}); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"delegated_authentication_enabled":true`) {
		t.Fatalf("setting=%d %s", rec.Code, rec.Body.String())
	}
	diagnosticBody := map[string]string{"user_id": userID, "password": "diagnostic-secret"}
	for _, attempt := range []call{
		{method: http.MethodPost, path: adminPath + "/test", body: diagnosticBody, user: "member"},
		{method: http.MethodPost, path: adminPath + "/test", body: diagnosticBody, user: "admin", noCSRF: true},
	} {
		if rec := s.do(attempt); rec.Code != 403 {
			t.Fatalf("diagnostic authorization=%d", rec.Code)
		}
	}
	if rec := s.do(call{method: http.MethodPost, path: adminPath + "/test", body: diagnosticBody, user: "other-admin", realm: "other"}); rec.Code != 404 {
		t.Fatalf("cross-tenant diagnostic=%d", rec.Code)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- s.do(call{method: http.MethodPost, path: adminPath + "/test", body: diagnosticBody, user: "admin"})
	}()
	challengePath := "/directory-connector/v1/password-challenges"
	if rec := s.do(call{method: http.MethodPost, path: challengePath}); rec.Code != 401 {
		t.Fatalf("unsigned poll=%d", rec.Code)
	}
	rec = api(http.MethodPost, challengePath, nil)
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("poll=%d %s", rec.Code, rec.Body.String())
	}
	challenge := decode[domain.PasswordChallenge](t, rec)
	if challenge.ID == "" || challenge.TenantID != tenancydomain.DefaultTenantID || challenge.ConnectorID != enrollment.ConnectorID || challenge.KeyID != registered["key_id"] || challenge.EncryptedRequest == "" || strings.Contains(rec.Body.String(), "diagnostic-secret") || !time.Now().Before(challenge.ExpiresAt) {
		t.Fatal("invalid public challenge")
	}
	resultPath := challengePath + "/" + challenge.ID + "/result"
	otherEnrollment, otherKey := s.enrolled("other connector")
	if rec := s.do(call{method: http.MethodPost, path: resultPath, body: map[string]string{"result": "accepted"}, bearer: s.sign(otherKey, otherEnrollment, nil)}); rec.Code != 409 {
		t.Fatalf("another connector's result=%d", rec.Code)
	}
	if rec := api(http.MethodPost, resultPath, map[string]string{"result": "accepted"}); rec.Code != 204 {
		t.Fatalf("completion=%d %s", rec.Code, rec.Body.String())
	}
	select {
	case rec := <-done:
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"result":"accepted"`) || strings.Contains(rec.Body.String(), "diagnostic-secret") || len(rec.Result().Cookies()) != 0 {
			t.Fatalf("diagnostic=%d %s", rec.Code, rec.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("diagnostic did not finish")
	}
	if rec := api(http.MethodPost, resultPath, map[string]string{"result": "accepted"}); rec.Code != 204 {
		t.Fatalf("idempotent completion=%d", rec.Code)
	}
	if rec := api(http.MethodPost, resultPath, map[string]string{"result": "invalid_credentials"}); rec.Code != 409 {
		t.Fatalf("conflicting completion=%d", rec.Code)
	}
	s.diagnosticGate.retry = 60
	if rec := s.do(call{method: http.MethodPost, path: adminPath + "/test", body: diagnosticBody, user: "admin"}); rec.Code != 429 || rec.Header().Get("Retry-After") != "60" {
		t.Fatalf("diagnostic throttle=%d %s", rec.Code, rec.Body.String())
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequest(http.MethodPost, challengePath, http.NoBody).WithContext(cancelled)
	req.Header.Set("Authorization", "Bearer "+s.sign(key, enrollment, nil))
	empty := httptest.NewRecorder()
	s.e.ServeHTTP(empty, req)
	if empty.Code != 204 {
		t.Fatalf("empty poll=%d %s", empty.Code, empty.Body.String())
	}
}
