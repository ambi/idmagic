package server_http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func browserContractProblem(t *testing.T, response *http.Response, status int, code string) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != status || response.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d content-type=%q, want %d Problem Details", response.StatusCode, response.Header.Get("Content-Type"), status)
	}
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
		t.Fatal(err)
	}
	if problem.Type != "urn:idmagic:error:"+code || problem.Status != status || problem.Detail == "" {
		t.Fatalf("problem=%+v, want %s status=%d", problem, code, status)
	}
}

//verify:operation GetBrowserTransaction: 外部 return_to への 400 invalid_request と transaction cookie を発行しないこと
func TestBrowserTransactionHTTPContractRefusesExternalReturnTo(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	response, err := browserClient(t).Get(srv.URL + "/realms/default/api/auth/transaction?return_to=https%3A%2F%2Fevil.example%2F")
	if err != nil {
		t.Fatal(err)
	}
	browserContractProblem(t, response, http.StatusBadRequest, "invalid_request")
	if findCookie(response.Cookies(), "idmagic_transaction") != nil {
		t.Fatal("refused transaction issued a transaction cookie")
	}
}

//verify:operation SubmitBrowserLogin: CSRF 拒否の 403 Problem Details
//verify:operation SubmitBrowserConsent: CSRF 拒否の 403 Problem Details
//verify:operation SubmitBrowserTotp: CSRF 拒否の 403 Problem Details
//verify:operation StartBrowserMfaEnrollment: CSRF 拒否の 403 Problem Details
//verify:operation ConfirmBrowserMfaEnrollment: CSRF 拒否の 403 Problem Details
func TestBrowserWriteHTTPContractsRefuseCSRF(t *testing.T) {
	srv := newServerWithTOTPPolicy(t, totpTestSecret, true)
	defer srv.Close()
	client := browserClient(t)
	for _, path := range []string{
		"/api/auth/login", "/api/auth/consent", "/api/auth/totp",
		"/api/auth/mfa/enrollment/totp/start", "/api/auth/mfa/enrollment/totp/confirm",
	} {
		t.Run(path, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, srv.URL+"/realms/default"+path, strings.NewReader(`{}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", "http://test")
			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			browserContractProblem(t, response, http.StatusForbidden, "csrf_failed")
			if findCookie(response.Cookies(), "idmagic_session") != nil {
				t.Fatal("refused browser write issued a session cookie")
			}
		})
	}
}

//verify:operation ResumeFederatedAuthorization: 未認証時の 401 とログイン後の 303 Location、transaction の遷移
func TestFederatedAuthorizationResumeHTTPContractRequiresCompletedSession(t *testing.T) {
	srv := newServer(t)
	defer srv.Close()
	client := browserClient(t)
	started := startAuthorization(t, client, srv.URL+"/realms/default", "federated-resume-contract-verifier-0123456789", "resume-state")
	started.Body.Close()
	response, err := client.Get(srv.URL + "/realms/default/authorize/resume")
	if err != nil {
		t.Fatal(err)
	}
	browserContractProblem(t, response, http.StatusUnauthorized, "authentication_required")
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	if transaction.Kind != "login" {
		t.Fatalf("refused resume changed transaction=%+v", transaction)
	}
	postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": demoUsername, "password": demoPassword})
	resumed, err := client.Get(srv.URL + "/realms/default/authorize/resume")
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Body.Close()
	if resumed.StatusCode != http.StatusSeeOther || resumed.Header.Get("Location") != "/realms/default/consent" {
		t.Fatalf("resume status=%d location=%q", resumed.StatusCode, resumed.Header.Get("Location"))
	}
	consent := getJSON[struct {
		Kind string `json:"kind"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	if consent.Kind != "consent" {
		t.Fatalf("resumed transaction=%+v", consent)
	}
}
