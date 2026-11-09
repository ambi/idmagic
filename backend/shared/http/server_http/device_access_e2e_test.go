package server_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func createDeviceApplication(t *testing.T, f *publicSetupPath) (string, string, string) {
	t.Helper()
	created := f.request(t, http.MethodPost, "/api/admin/v1/applications", map[string]any{
		"name": "Device", "type": "oidc", "redirect_uris": []string{setupRedirect}, "scope": "openid profile",
	}, f.admin, http.StatusCreated)
	appID := setupString(t, created["application"].(map[string]any), "id")
	f.request(t, http.MethodPatch, "/api/admin/v1/applications/"+appID+"/oidc", map[string]any{
		"grant_types": []string{"authorization_code", "urn:ietf:params:oauth:grant-type:device_code"},
	}, f.admin, http.StatusNoContent)
	return appID, setupString(t, created, "client_id"), setupString(t, created, "client_secret")
}

func deviceForm(t *testing.T, f *publicSetupPath, clientID, secret, path string, form url.Values, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default"+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, secret)
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || rec.Code != want {
		t.Fatalf("POST %s: status=%d body=%s err=%v, want %d", path, rec.Code, rec.Body.String(), err, want)
	}
	return result
}

//spec:covers REQ-APPLICATION-009, REQ-APPLICATION-011, REQ-OAUTH2-027: 公開設定した MFA ポリシーで承認を保留し、第二要素後に同じコードへ戻り、再評価して承認とトークンの発行を完了する。
func TestDeviceApprovalCompletesAfterSecondFactor(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithApplicationApi(), stack.WithDeviceAuthorization())
	userID := f.user(t, "device-user")
	loginClient, _ := f.keyClient(t, "openid profile", "authorization_code")
	_, cookie := f.login(t, "device-user", loginClient, "openid profile")
	appID, clientID, secret := createDeviceApplication(t, f)
	f.request(t, http.MethodPost, "/api/admin/v1/applications/"+appID+"/assignments", map[string]any{
		"subject_type": "user", "subject_id": userID,
	}, f.admin, http.StatusCreated)
	f.request(t, http.MethodPut, "/api/admin/v1/applications/"+appID+"/sign-in-policy", map[string]any{
		"rules": []map[string]any{{"rule_id": "mfa", "name": "MFA", "enabled": true, "required_authn": map[string]any{"strength": "Mfa"}}},
	}, f.admin, http.StatusOK)
	secretTOTP := totpTestSecret
	if err := f.s.MfaFactors.Save(f.s.RealmContext(t, "default"), &totpdomain.MfaFactor{
		UserID: userID, Type: spec.MfaFactorTOTP, Secret: &secretTOTP, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	issued := deviceForm(t, f, clientID, secret, "/device_authorization", url.Values{"scope": {"openid"}}, http.StatusOK)
	userCode := setupString(t, issued, "user_code")
	approval := map[string]any{"user_code": userCode, "action": "allow"}
	pending := f.request(t, http.MethodPost, "/api/auth/device", approval, cookie, http.StatusOK)
	next, err := url.Parse(setupString(t, pending, "next"))
	if err != nil || next.Path != "/realms/default/totp" {
		t.Fatalf("MFA next=%v err=%v", next, err)
	}
	returnTo := next.Query().Get("return_to")
	back, err := url.Parse(returnTo)
	if err != nil || back.Path != "/realms/default/device" || back.Query().Get("user_code") != userCode {
		t.Fatalf("MFA return_to=%v err=%v", back, err)
	}
	stored, err := f.s.DeviceCodes.FindByUserCode(f.s.RealmContext(t, "default"), oauthdomain.NormalizeUserCode(userCode))
	if err != nil || stored.State != spec.DeviceFlowIssued || stored.UserID != nil {
		t.Fatalf("MFA pending code=%+v err=%v", stored, err)
	}
	f.s.Events.AssertNotEmitted(t, "DeviceAuthorizationApproved")
	f.s.Events.AssertEmitted(t, "AppStepUpRequired", "AppAccessDeniedByPolicy", "MfaChallengeIssued")
	// 認証待ちのセッションで承認 API を直接再送しても承認できない。
	f.request(t, http.MethodPost, "/api/auth/device", approval, cookie, http.StatusForbidden)
	code, err := authentication.GenerateTOTP(secretTOTP, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range []string{"https://evil.example/device", "//evil.example/device", "/realms/acme/device", "/realms/default/device/extra", "/realms/default/../device"} {
		f.request(t, http.MethodPost, "/api/auth/totp", map[string]any{"code": code, "return_to": unsafe}, cookie, http.StatusBadRequest)
	}
	completed := f.request(t, http.MethodPost, "/api/auth/totp", map[string]any{"code": code, "return_to": returnTo}, cookie, http.StatusOK)
	if completed["redirect_to"] != returnTo {
		t.Fatalf("second factor did not return to the device: %v", completed)
	}
	page := f.request(t, http.MethodGet, "/api/auth/device?"+back.RawQuery, nil, cookie, http.StatusOK)
	if page["user_code"] != userCode {
		t.Fatalf("device context lost the user code: %v", page)
	}
	// 認証の間に割り当てが消えた場合も、復帰後の再評価で拒否する。
	f.request(t, http.MethodDelete, "/api/admin/v1/applications/"+appID+"/assignments/user/"+userID, nil, f.admin, http.StatusNoContent)
	f.request(t, http.MethodPost, "/api/auth/device", approval, cookie, http.StatusForbidden)
	f.s.Events.AssertNotEmitted(t, "DeviceAuthorizationApproved")
	f.request(t, http.MethodPost, "/api/admin/v1/applications/"+appID+"/assignments", map[string]any{
		"subject_type": "user", "subject_id": userID,
	}, f.admin, http.StatusCreated)
	approved := f.request(t, http.MethodPost, "/api/auth/device", approval, cookie, http.StatusOK)
	if approved["next"] != "/status?state=approved" {
		t.Fatalf("approval=%v", approved)
	}
	tokens := deviceForm(t, f, clientID, secret, "/token", url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {setupString(t, issued, "device_code")},
	}, http.StatusOK)
	setupString(t, tokens, "access_token")
	claims := unverifiedClaims(t, setupString(t, tokens, "id_token"))
	if claims["sub"] != userID || claims["aud"] != clientID {
		t.Fatalf("device token identity=%v", claims)
	}
	f.s.Events.AssertEmitted(t, "DeviceAuthorizationApproved", "MfaChallengeSucceeded")
}
