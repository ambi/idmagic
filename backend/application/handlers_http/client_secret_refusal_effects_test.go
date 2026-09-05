package handlers_http_test

// クライアントシークレットの追加発行とローテーションが宣言する拒否について、
// 応答と「拒否が変えなかった資格情報」の両方を確かめる。
//
// これらの拒否は 4xx を書く分岐と資格情報を書き換える分岐が別なので、ステータスと
// エラー種別だけを読むテストは「拒否を書き、そのうえで発行もする」実装を通してしまう。
// そこで各テストは拒否の前後で資格情報一覧を読み直し、件数、`credential_id`、
// 期限、状態のどれも動いていないことを確かめる。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
)

// readSecretCredentials は Application 詳細から資格情報の一覧を読む。
// 拒否が「変えなかったもの」を、書き込み側ではなく読み出し側の入口から読む。
func readSecretCredentials(
	t *testing.T,
	e *echo.Echo,
	csrf string,
	cookie *http.Cookie,
	applicationID string,
) []applicationSecretCredential {
	t.Helper()
	response := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		OIDC struct {
			Credentials []applicationSecretCredential `json:"secret_credentials"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.OIDC.Credentials
}

func assertCredentialsUnchanged(t *testing.T, before, after []applicationSecretCredential) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("拒否が資格情報の件数を変えた: before=%d after=%d", len(before), len(after))
	}
	for index := range before {
		if !sameCredential(before[index], after[index]) {
			t.Fatalf("拒否が資格情報を変えた: before=%s after=%s",
				describeCredential(before[index]), describeCredential(after[index]))
		}
	}
}

// applicationSecretCredential はポインターの任意項目を持つので、値で突き合わせる。
func sameCredential(left, right applicationSecretCredential) bool {
	return left.CredentialID == right.CredentialID &&
		left.CreatedAt == right.CreatedAt &&
		left.Status == right.Status &&
		optionalTime(left.ExpiresAt) == optionalTime(right.ExpiresAt) &&
		optionalTime(left.RevokedAt) == optionalTime(right.RevokedAt)
}

func optionalTime(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func describeCredential(credential applicationSecretCredential) string {
	return credential.CredentialID + " status=" + credential.Status +
		" expires_at=" + optionalTime(credential.ExpiresAt) +
		" revoked_at=" + optionalTime(credential.RevokedAt)
}

// createPublicClientApplication は公開クライアントの Application を作る。
// 公開クライアントは client_secret を持たないので、追加発行もローテーションも
// 受け付けてはならない対象の 1 つである。
func createPublicClientApplication(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie) string {
	t.Helper()
	response := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, map[string]any{
		"name": "Public App", "type": "oidc",
		"redirect_uris":              []string{"https://public.example/callback"},
		"client_type":                "public",
		"token_endpoint_auth_method": "none",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Application.ID
}

func issueSecret(
	t *testing.T,
	e *echo.Echo,
	csrf string,
	cookie *http.Cookie,
	applicationID string,
	body map[string]any,
) int {
	t.Helper()
	response := adminJSON(t, e, http.MethodPost,
		"/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets", csrf, cookie, body)
	return response.Code
}

// EX-OAUTH2-036-02: 1..730 の範囲外の expires_in_days は拒否され、資格情報は増えない。
func TestIssueClientSecretOutOfRangeExpiryAddsNoCredential(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createSecretApplication(t, e, csrf, cookie)
	before := readSecretCredentials(t, e, csrf, cookie, applicationID)

	for _, days := range []int{0, -1, 731} {
		if status := issueSecret(t, e, csrf, cookie, applicationID, map[string]any{
			"expires_in_days": days,
		}); status == http.StatusCreated {
			t.Fatalf("expires_in_days=%d が受理された", days)
		}
	}
	assertCredentialsUnchanged(t, before, readSecretCredentials(t, e, csrf, cookie, applicationID))
}

// EX-OAUTH2-036-03: Active の資格情報が上限に達している状態の追加発行は拒否され、
// 既存の資格情報は期限も状態も変わらない。
func TestIssueClientSecretBeyondActiveLimitLeavesExistingCredentials(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createSecretApplication(t, e, csrf, cookie)

	if status := issueSecret(t, e, csrf, cookie, applicationID, map[string]any{
		"expires_in_days": 90,
	}); status != http.StatusCreated {
		t.Fatalf("前提が壊れている: 1 件目の追加発行が status=%d", status)
	}
	before := readSecretCredentials(t, e, csrf, cookie, applicationID)
	if len(before) != 2 {
		t.Fatalf("前提が壊れている: 上限到達前の資格情報が %d 件", len(before))
	}

	if status := issueSecret(t, e, csrf, cookie, applicationID, map[string]any{
		"expires_in_days": 90,
	}); status != http.StatusUnprocessableEntity {
		t.Fatalf("上限超過が status=%d、期待は 422", status)
	}
	assertCredentialsUnchanged(t, before, readSecretCredentials(t, e, csrf, cookie, applicationID))
}

// EX-OAUTH2-036-04: シークレットを持てないクライアント (公開クライアント、
// private_key_jwt、mTLS) への追加発行は拒否され、
// 資格情報は増えない。
func TestIssueClientSecretForUnsupportedAuthMethodAddsNoCredential(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createPublicClientApplication(t, e, csrf, cookie)
	before := readSecretCredentials(t, e, csrf, cookie, applicationID)

	if status := issueSecret(t, e, csrf, cookie, applicationID, map[string]any{
		"expires_in_days": 90,
	}); status == http.StatusCreated {
		t.Fatal("シークレットを持てないクライアントへ追加発行が通った")
	}
	assertCredentialsUnchanged(t, before, readSecretCredentials(t, e, csrf, cookie, applicationID))
}

// EX-OAUTH2-036-05: 別クライアントまたは存在しない credential_id の失効は拒否され、
// その資格情報は Active のまま残る。
func TestRevokeClientSecretWithForeignCredentialLeavesItActive(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	target := createSecretApplication(t, e, csrf, cookie)
	other := createSecretApplication(t, e, csrf, cookie)

	otherCredentials := readSecretCredentials(t, e, csrf, cookie, other)
	if len(otherCredentials) != 1 {
		t.Fatalf("前提が壊れている: 別クライアントの資格情報が %d 件", len(otherCredentials))
	}
	foreignID := otherCredentials[0].CredentialID
	targetBefore := readSecretCredentials(t, e, csrf, cookie, target)

	for _, credentialID := range []string{foreignID, "00000000-0000-4000-8000-000000000000"} {
		response := adminJSON(t, e, http.MethodDelete,
			"/api/admin/v1/applications/"+target+"/oidc/client-secrets/"+credentialID, csrf, cookie, nil)
		if response.Code == http.StatusOK {
			t.Fatalf("credential_id=%s の失効が通った: body=%s", credentialID, response.Body.String())
		}
	}

	assertCredentialsUnchanged(t, targetBefore, readSecretCredentials(t, e, csrf, cookie, target))
	after := readSecretCredentials(t, e, csrf, cookie, other)
	assertCredentialsUnchanged(t, otherCredentials, after)
	if after[0].Status != "Active" {
		t.Fatalf("別クライアントの資格情報が %s になった", after[0].Status)
	}
}

// EX-OAUTH2-037-02: 1..30 の範囲外の grace_days はローテーションを拒否され、
// シークレットはローテーションされない。
//
// 「既存のシークレットで認証が引き続き成功する」は、この入口からはトークン
// エンドポイントへ届かないので、同じことを資格情報側から読む。`credential_id` が
// 変わらず、Active のまま、新しい資格情報も増えていなければ、認証に使う資格情報は
// 拒否の前後で同一である。
func TestRotateClientSecretOutOfRangeGraceDaysDoesNotRotate(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createSecretApplication(t, e, csrf, cookie)
	before := readSecretCredentials(t, e, csrf, cookie, applicationID)

	for _, days := range []int{-1, 31} {
		response := adminJSON(t, e, http.MethodPost,
			"/api/admin/v1/applications/"+applicationID+"/oidc/rotate-secret", csrf, cookie,
			map[string]any{"grace_days": days})
		if response.Code == http.StatusOK {
			t.Fatalf("grace_days=%d が受理された: body=%s", days, response.Body.String())
		}
		if response.Body.Len() > 0 && bodyCarriesClientSecret(t, response.Body.Bytes()) {
			t.Fatalf("拒否された応答が client_secret を運んでいる: %s", response.Body.String())
		}
	}
	assertCredentialsUnchanged(t, before, readSecretCredentials(t, e, csrf, cookie, applicationID))
}

// EX-OAUTH2-037-03: シークレットを持てないクライアント (公開クライアント、
// private_key_jwt、mTLS) のローテーションは拒否され、
// シークレットはローテーションされない。
func TestRotateClientSecretForUnsupportedAuthMethodDoesNotRotate(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createPublicClientApplication(t, e, csrf, cookie)
	before := readSecretCredentials(t, e, csrf, cookie, applicationID)

	response := adminJSON(t, e, http.MethodPost,
		"/api/admin/v1/applications/"+applicationID+"/oidc/rotate-secret", csrf, cookie,
		map[string]any{"grace_days": 7})
	if response.Code == http.StatusOK {
		t.Fatalf("シークレットを持てないクライアントのローテーションが通った: body=%s", response.Body.String())
	}
	if response.Body.Len() > 0 && bodyCarriesClientSecret(t, response.Body.Bytes()) {
		t.Fatalf("拒否された応答が client_secret を運んでいる: %s", response.Body.String())
	}
	assertCredentialsUnchanged(t, before, readSecretCredentials(t, e, csrf, cookie, applicationID))
}

func bodyCarriesClientSecret(t *testing.T, body []byte) bool {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	secret, present := parsed["client_secret"]
	return present && secret != nil && secret != ""
}
