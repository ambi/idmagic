package server_http_test

import (
	"net/http"
	"net/url"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// RFC 7521 §4.2 は、クライアントアサーションによる認証の `client_id` を OPTIONAL とする。
// トークンエンドポイントでの省略は TestClientAssertionIdentifiesTheClientWithoutClientID が
// 観測する。ここでは、同じクライアント認証を使うほかの五つの入口へ、`client_id` を省いた要求を
// 送る。入口ごとに `client_id` を別に読む実装は、トークンエンドポイントの試験だけでは
// 見つからない。
//
// 認証が通ったことは、入口ごとの成功応答で読む。デバイス認可と CIBA は記録を作るので、
// 記録がアサーションの `sub` のクライアントに属することまで読む。対照として、同じ要求に
// 異なる `client_id` を付けると `invalid_client` になることを確かめ、成功が認証を素通りした
// 結果でないことを示す。
//
// audience には発行者の識別子を使う。CIBA Core §7.1 が推奨する値で、製品はどの入口でも
// 受け入れる。
//
//spec:covers RFC7521-CLIENT-ID-OPTIONAL (optional): PAR、イントロスペクション、失効、デバイス認可、CIBA が client_id を省いたアサーションをアサーションの sub のクライアントとして認証し、異なる client_id は invalid_client で拒否する。
func TestClientAssertionWithoutClientIDAuthenticatesAtEveryClientEndpoint(t *testing.T) {
	fixture := newNonInteractiveGrantFixture(t)
	audience := nigIssuer + "/realms/default"
	post := func(t *testing.T, path string, form url.Values, clientID string) (int, map[string]any) {
		t.Helper()
		form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
		form.Set("client_assertion", fixture.assertionKey.Assertion(t, audience))
		if clientID != "" {
			form.Set("client_id", clientID)
		}
		return fixture.postFormWith(t, path, form, func(*http.Request) {})
	}

	for _, tc := range []struct {
		name   string
		path   string
		form   func() url.Values
		status int
		owned  func(t *testing.T, body map[string]any) string
	}{
		{
			name: "PAR", path: "/par", status: http.StatusCreated,
			form: func() url.Values {
				return url.Values{
					"response_type": {"code"}, "redirect_uri": {nigRedirectURI}, "scope": {"openid"},
					"state": {"opaque-state"}, "code_challenge": {nigChallenge()}, "code_challenge_method": {"S256"},
				}
			},
		},
		{
			name: "イントロスペクション", path: "/introspect", status: http.StatusOK,
			form: func() url.Values { return url.Values{"token": {"not-a-token"}} },
		},
		{
			name: "失効", path: "/revoke", status: http.StatusOK,
			form: func() url.Values { return url.Values{"token": {"not-a-token"}} },
		},
		{
			name: "デバイス認可", path: "/device_authorization", status: http.StatusOK,
			form: func() url.Values { return url.Values{"scope": {"openid"}} },
			owned: func(t *testing.T, body map[string]any) string {
				t.Helper()
				userCode, _ := body["user_code"].(string)
				record, err := fixture.devices.FindByUserCode(fixture.tenantContext(), oauthdomain.NormalizeUserCode(userCode))
				if err != nil || record == nil {
					t.Fatalf("デバイス認可の記録が見つからない: record=%v err=%v", record, err)
				}
				return record.ClientID
			},
		},
		{
			name: "CIBA", path: "/bc-authorize", status: http.StatusOK,
			form: func() url.Values { return url.Values{"scope": {"openid"}, "login_hint": {nigUsername}} },
			owned: func(t *testing.T, body map[string]any) string {
				t.Helper()
				authReqID, _ := body["auth_req_id"].(string)
				record, err := fixture.approvals.FindByAuthReqIDHash(fixture.tenantContext(), oauthdomain.HashAuthReqID(authReqID))
				if err != nil || record == nil {
					t.Fatalf("承認要求の記録が見つからない: record=%v err=%v", record, err)
				}
				return record.ClientID
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := post(t, tc.path, tc.form(), "")
			if status != tc.status {
				t.Fatalf("client_id を省いた要求 status=%d body=%v, want %d", status, body, tc.status)
			}
			if tc.owned != nil {
				if owner := tc.owned(t, body); owner != nigAssertionClientID {
					t.Fatalf("記録のクライアント = %q, want %q", owner, nigAssertionClientID)
				}
			}

			status, body = post(t, tc.path, tc.form(), nigClientID)
			if status != http.StatusUnauthorized || body["error"] != "invalid_client" {
				t.Fatalf("異なる client_id の要求 status=%d body=%v, want 401 invalid_client", status, body)
			}
		})
	}
}
