package handlers_http_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//verify:operation StartTotpEnrollment: TOTP 登録、確認、解除の HTTP 応答と資格情報の保存結果
//verify:operation ConfirmTotpEnrollment: TOTP 登録、確認、解除の HTTP 応答と資格情報の保存結果
//verify:operation RemoveTotpFactor: TOTP 登録、確認、解除の HTTP 応答と資格情報の保存結果
func TestTotpEnrollmentHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "totp-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	send := func(path string, body any) *httptest.ResponseRecorder {
		return f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session, csrf: authRefusalCSRF, body: body})
	}
	start := authContractJSON(t, send("/api/account/v1/mfa/totp/enroll/start", nil), http.StatusOK)
	secret, ok := start["secret"].(string)
	if !ok || secret == "" || start["account_name"] != authRefusalAlice || start["issuer"] != "idp.test" {
		t.Fatalf("start=%v", start)
	}
	uri, err := url.Parse(start["otpauth_uri"].(string))
	if err != nil || uri.Scheme != "otpauth" || uri.Host != "totp" || uri.Query().Get("secret") != secret || uri.Query().Get("issuer") != start["issuer"] {
		t.Fatalf("uri=%v err=%v", uri, err)
	}
	code, err := totpusecases.GenerateTOTP(secret, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	authContractEmpty(t, send("/api/account/v1/mfa/totp/enroll/confirm", map[string]string{"secret": secret, "code": code}))
	ctx := testing_tenant.Default(t.Context())
	factor, err := f.factors.Find(ctx, authRefusalAlice, spec.MfaFactorTOTP)
	if err != nil || factor == nil || factor.Secret == nil || *factor.Secret != secret {
		t.Fatalf("factor=%v err=%v", factor, err)
	}
	authContractEmpty(t, send("/api/account/v1/mfa/totp/remove", map[string]string{"code": code}))
	factor, err = f.factors.Find(ctx, authRefusalAlice, spec.MfaFactorTOTP)
	if err != nil || factor != nil {
		t.Fatalf("removed factor=%v err=%v", factor, err)
	}
}

//verify:operation GenerateRecoveryCodes: コード生成の JSON 応答と生成、失効の保存結果
//verify:operation RevokeRecoveryCodes: コード生成の JSON 応答と生成、失効の保存結果
func TestRecoveryCodesHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "recovery-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	generated := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/mfa/recovery-codes/generate", sessionID: session, csrf: authRefusalCSRF}), http.StatusOK)
	authContractTime(t, generated, "generated_at")
	codes := generated["codes"].([]any)
	if len(codes) != 10 {
		t.Fatalf("codes=%v", codes)
	}
	seen := map[string]bool{}
	for _, raw := range codes {
		code := raw.(string)
		if len(code) != 10 || seen[code] {
			t.Fatal("recovery codes have the wrong length or duplicates")
		}
		seen[code] = true
	}
	ctx := testing_tenant.Default(t.Context())
	stored, err := f.recovery.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(stored) != 10 {
		t.Fatalf("stored codes=%d err=%v", len(stored), err)
	}
	for _, code := range stored {
		if code.CodeHash == "" || seen[code.CodeHash] || code.ConsumedAt != nil {
			t.Fatal("stored recovery code is plaintext, empty or consumed")
		}
	}
	authContractEmpty(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: "/api/account/v1/mfa/recovery-codes/revoke", sessionID: session, csrf: authRefusalCSRF}))
	stored, err = f.recovery.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(stored) != 0 {
		t.Fatalf("revoked codes=%d err=%v", len(stored), err)
	}
}

//verify:operation StartStepUpAuthentication: step-up の開始、完了、WebAuthn challenge の JSON 応答と認証状態
//verify:operation CompleteStepUpAuthentication: step-up の開始、完了、WebAuthn challenge の JSON 応答と認証状態
//verify:operation StartStepUpWebAuthnChallenge: step-up の開始、完了、WebAuthn challenge の JSON 応答と認証状態
func TestStepUpHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	f.seedTotpFactor(t, "JBSWY3DPEHPK3PXP")
	f.seedRecoveryCodes(t)
	f.seedWebAuthnCredential(t)
	session := f.seedSession(t, "step-up-contract", tenancydomain.DefaultTenantID, authRefusalAlice)
	send := func(path string, body any) *httptest.ResponseRecorder {
		return f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session, csrf: authRefusalCSRF, body: body})
	}
	start := authContractJSON(t, send("/api/account/v1/step-up/start", nil), http.StatusOK)
	if !reflect.DeepEqual(start["methods"], []any{"password", "totp", "webauthn", "recovery_code"}) {
		t.Fatalf("methods=%v", start)
	}
	challenge := authContractJSON(t, send("/api/account/v1/step-up/webauthn/challenge", nil), http.StatusOK)["publicKey"].(map[string]any)
	value, ok := challenge["challenge"].(string)
	if decoded, err := base64.RawURLEncoding.DecodeString(value); !ok || err != nil || len(decoded) == 0 || challenge["rpId"] != "idp.test" {
		t.Fatalf("challenge=%v err=%v", challenge, err)
	}
	allowed := challenge["allowCredentials"].([]any)
	if len(allowed) != 1 || allowed[0].(map[string]any)["id"] != base64.RawURLEncoding.EncodeToString([]byte("credential-"+authRefusalAlice)) || allowed[0].(map[string]any)["type"] != "public-key" {
		t.Fatalf("allowed credentials=%v", allowed)
	}
	authContractEmpty(t, send("/api/account/v1/step-up/complete", map[string]string{"method": "password", "password": authRefusalPassword}))
	stored, err := f.sessions.Find(testing_tenant.Default(t.Context()), session)
	if err != nil || stored == nil || stored.StepUpAt == 0 {
		t.Fatalf("session=%v err=%v", stored, err)
	}
}

//verify:operation IssueMfaEnrollmentBypass: MFA 登録 bypass と authenticator reset の HTTP 応答および保存結果
//verify:operation ResetUserAuthenticators: MFA 登録 bypass と authenticator reset の HTTP 応答および保存結果
func TestAdminMfaHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	admin := f.seedSession(t, "admin-mfa-contract", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
	send := func(path string, body any) *httptest.ResponseRecorder {
		return f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: admin, csrf: authRefusalCSRF, body: body})
	}
	issued := authContractJSON(t, send("/api/admin/v1/users/"+authRefusalBob+"/mfa-enrollment-bypass", map[string]int{"expires_in_seconds": 900}), http.StatusCreated)["bypass"].(map[string]any)
	authContractBypass(t, issued, authRefusalBob, 15*time.Minute)
	f.seedTotpFactor(t, "JBSWY3DPEHPK3PXP")
	f.seedWebAuthnCredential(t)
	f.seedRecoveryCodes(t)
	reset := authContractJSON(t, send("/api/admin/v1/users/"+authRefusalAlice+"/authenticator-reset", map[string]any{"targets": []string{"totp", "webauthn", "recovery_code"}}), http.StatusOK)
	if reset["mfa_enrolled"] != false || reset["reenrollment_required"] != true {
		t.Fatalf("reset=%v", reset)
	}
	authContractBypass(t, reset["bypass"].(map[string]any), authRefusalAlice, time.Hour)
	ctx := testing_tenant.Default(t.Context())
	factors, err := f.factors.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(factors) != 0 {
		t.Fatalf("factors=%v err=%v", factors, err)
	}
	credentials, err := f.credentials.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(credentials) != 0 {
		t.Fatalf("credentials=%v err=%v", credentials, err)
	}
	codes, err := f.recovery.ListBySub(ctx, authRefusalAlice)
	if err != nil || len(codes) != 0 {
		t.Fatalf("codes=%v err=%v", codes, err)
	}
}

func authContractBypass(t *testing.T, bypass map[string]any, userID string, ttl time.Duration) {
	t.Helper()
	if bypass["id"] == "" || bypass["tenant_id"] != tenancydomain.DefaultTenantID || bypass["user_id"] != userID || bypass["issued_by"] != authRefusalHomeAdmin {
		t.Fatalf("bypass=%v", bypass)
	}
	authContractTime(t, bypass, "issued_at")
	authContractTime(t, bypass, "expires_at")
	issued, _ := time.Parse(time.RFC3339Nano, bypass["issued_at"].(string))
	expires, _ := time.Parse(time.RFC3339Nano, bypass["expires_at"].(string))
	if expires.Sub(issued) != ttl {
		t.Fatalf("bypass lifetime=%s", expires.Sub(issued))
	}
}

//verify:operation StartTotpEnrollment: CSRF 拒否の 403 Problem Details
//verify:operation ConfirmTotpEnrollment: CSRF 拒否の 403 Problem Details
//verify:operation RemoveTotpFactor: CSRF 拒否の 403 Problem Details
//verify:operation GenerateRecoveryCodes: CSRF 拒否の 403 Problem Details
//verify:operation RevokeRecoveryCodes: CSRF 拒否の 403 Problem Details
//verify:operation StartStepUpAuthentication: CSRF 拒否の 403 Problem Details
//verify:operation CompleteStepUpAuthentication: CSRF 拒否の 403 Problem Details
//verify:operation StartStepUpWebAuthnChallenge: CSRF 拒否の 403 Problem Details
func TestMfaHTTPContractsRefuseCSRF(t *testing.T) {
	for _, path := range []string{
		"/api/account/v1/mfa/totp/enroll/start", "/api/account/v1/mfa/totp/enroll/confirm", "/api/account/v1/mfa/totp/remove",
		"/api/account/v1/mfa/recovery-codes/generate", "/api/account/v1/mfa/recovery-codes/revoke",
		"/api/account/v1/step-up/start", "/api/account/v1/step-up/complete", "/api/account/v1/step-up/webauthn/challenge",
	} {
		t.Run(path, func(t *testing.T) {
			f := newAuthRefusalServer(t)
			session := f.seedSession(t, "mfa-refusal", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
			authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session}), http.StatusForbidden, "csrf_failed")
			if len(*f.events) != 0 || len(f.challenges.keys) != 0 {
				t.Fatal("CSRF refusal changed authentication state")
			}
		})
	}
}

//verify:operation IssueMfaEnrollmentBypass: 非管理者への 403 Problem Details
//verify:operation ResetUserAuthenticators: 非管理者への 403 Problem Details
func TestAdminMfaHTTPContractsRefuseNonAdmin(t *testing.T) {
	f := newAuthRefusalServer(t)
	session := f.seedSession(t, "mfa-non-admin", tenancydomain.DefaultTenantID, authRefusalAlice)
	for _, path := range []string{"/api/admin/v1/users/" + authRefusalBob + "/mfa-enrollment-bypass", "/api/admin/v1/users/" + authRefusalBob + "/authenticator-reset"} {
		authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session, csrf: authRefusalCSRF, body: map[string]any{}}), http.StatusForbidden, "access_denied")
	}
	if len(*f.events) != 0 {
		t.Fatal("authorization refusal emitted events")
	}
}
