package server_http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// Bearer で届いた要求のステップアップは、アクセストークンが示す利用者の認証の時刻で決まる。
// トークンの発行時刻は認証の時刻ではないので、リフレッシュしたばかりのトークンでも、
// 認証から 5 分を過ぎていれば RFC 9470 の 401 で再認証を求め、パスワードを変えない。
//
//spec:covers REQ-AUTHENTICATION-039: Bearer の要求を auth_time で判定し、古い認証とそのリフレッシュには 401 と insufficient_user_authentication、max_age を返してパスワードを変えず、新しい認証では変更を受け付けることを観測する。
func TestBearerStepUpFollowsTheAuthenticationTimeOfTheToken(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	scope := url.Values{"scope": {"openid profile offline_access idmagic.account"}}

	staleToken := func(t *testing.T) tiTokenResponse {
		t.Helper()
		ctx := testing_tenant.Default(t.Context())
		code := f.authorizationCode(t, scope)
		record, err := f.codes.Find(ctx, code)
		if err != nil || record == nil {
			t.Fatalf("認可コードの記録が見つからない: record=%v err=%v", record, err)
		}
		record.AuthTime = time.Now().Add(-time.Hour).Unix()
		if err := f.codes.Save(ctx, record); err != nil {
			t.Fatal(err)
		}
		return f.mustToken(t, f.codeExchangeForm(code, nil))
	}

	for _, tc := range []struct {
		name  string
		token func(t *testing.T) string
	}{
		{"認証から時間が経ったトークン", func(t *testing.T) string {
			t.Helper()
			return staleToken(t).AccessToken
		}},
		{"それをリフレッシュしたばかりのトークン", func(t *testing.T) string {
			t.Helper()
			return f.mustToken(t, refreshRequest(staleToken(t).RefreshToken, nil)).AccessToken
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, header, body := changePasswordWithBearer(t, f.base, tc.token(t), tiPassword, "stale-step-up-password-5678")
			if status != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s, want 401", status, body)
			}
			challenge := header.Get("WWW-Authenticate")
			for _, want := range []string{`Bearer`, `error="insufficient_user_authentication"`, `max_age="300"`} {
				if !strings.Contains(challenge, want) {
					t.Fatalf("WWW-Authenticate=%q に %s がない", challenge, want)
				}
			}
			var problem struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(body), &problem); err != nil || problem.Type != "urn:idmagic:error:insufficient_user_authentication" {
				t.Fatalf("本文の種類=%q err=%v body=%s", problem.Type, err, body)
			}
		})
	}

	// 拒否がパスワードを変えていなければ、元のパスワードを現在のものとして変更できる。
	fresh := f.mustToken(t, f.codeExchangeForm(f.authorizationCode(t, scope), nil))
	if status, _, body := changePasswordWithBearer(t, f.base, fresh.AccessToken, tiPassword, "fresh-step-up-password-5678"); status != http.StatusNoContent {
		t.Fatalf("新しい認証のトークンでの変更 status=%d body=%s, want 204", status, body)
	}
}

func changePasswordWithBearer(t *testing.T, base, token, current, next string) (int, http.Header, string) {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"current_password": current, "new_password": next})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, base+"/api/auth/change-password", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /api/auth/change-password: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header, string(body)
}
