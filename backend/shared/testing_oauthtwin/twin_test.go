package testing_oauthtwin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

var (
	bearerExchanges  = []string{"authorization-code-bearer", "client-credentials-bearer", "refresh-bearer", "refresh-invalid-request"}
	exampleExchanges = []string{"authorization-code-example-type", "client-credentials-example-type"}
)

// 模倣は、記録した往復の要求に記録と同じ応答を返す。
// RFC 6749 の例の応答は token_type が example で拡張の欄を持つので、その設定の模倣で照合する。
func TestTwin_MatchesRFC6749Recording(t *testing.T) {
	rec := RFC6749Recording(t)
	t.Run("bearer", func(t *testing.T) {
		tw := NewTwin(RecordedClientID, RecordedClientSecret)
		for _, diff := range Diverge(tw, rec.Select(t, bearerExchanges...)) {
			t.Error(diff)
		}
	})
	t.Run("example", func(t *testing.T) {
		tw := NewTwin(RecordedClientID, RecordedClientSecret)
		tw.TokenType = "example"
		tw.Extra = map[string]any{"example_parameter": "example_value"}
		for _, diff := range Diverge(tw, rec.Select(t, exampleExchanges...)) {
			t.Error(diff)
		}
	})
}

// 記録のすべての往復が、どちらかの設定の照合に含まれる。往復を足したときに照合から漏れないためである。
func TestTwin_FidelityCoversEveryExchange(t *testing.T) {
	covered := map[string]bool{}
	for _, name := range append(append([]string{}, bearerExchanges...), exampleExchanges...) {
		covered[name] = true
	}
	for _, ex := range RFC6749Recording(t).Exchanges {
		if !covered[ex.Name] {
			t.Errorf("exchange %q is not compared with the twin", ex.Name)
		}
	}
}

// 照合が空虚でないことを確かめる。模倣の応答を記録から外すと、照合が食い違いを報告する。
func TestDiverge_ReportsTwinThatLeavesTheRecording(t *testing.T) {
	tests := []struct {
		name  string
		setup func(tw *Twin)
		want  string
	}{
		{"更新トークンを回転しない", func(tw *Twin) { tw.RotateRefreshTokens = false }, "refresh-bearer: body.refresh_token is missing"},
		{"expires_in を返さない", func(tw *Twin) { tw.OmitExpiresIn = true }, "client-credentials-bearer: body.expires_in is missing"},
		{"token_type の綴りが記録と違う", func(tw *Twin) { tw.TokenType = "bearer" }, "authorization-code-bearer: body.token_type = bearer, recorded Bearer"},
		{"発行が 500 を返す", func(tw *Twin) {
			tw.InjectFaults(Fault{}, Fault{Status: 500, Body: "boom"})
		}, "client-credentials-bearer: status = 500, recorded 200"},
		{"Cache-Control を返さない", func(tw *Twin) {
			tw.InjectFaults(Fault{Status: 400, ContentType: "application/json", Body: `{"error":"invalid_request"}`})
		}, `authorization-code-bearer: header Cache-Control = "", recorded "no-store"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := NewTwin(RecordedClientID, RecordedClientSecret)
			tc.setup(tw)
			diffs := Diverge(tw, RFC6749Recording(t).Select(t, bearerExchanges...))
			if !strings.Contains(strings.Join(diffs, "\n"), tc.want) {
				t.Errorf("Diverge() = %q, want a line %q", diffs, tc.want)
			}
		})
	}
}

// 記録にない組み合わせ：使用済みの認可コード、回転した後の旧い更新トークン、資格情報の誤り、未知のグラント。
func TestTwin_RefusesOutsideTheRecording(t *testing.T) {
	tw := NewTwin(RecordedClientID, RecordedClientSecret)
	server := tw.Start(t)
	tw.AddAuthorizationCode(RecordedCode, RecordedRedirectURI)
	tw.AddRefreshToken(RecordedRefreshToken)
	post := func(t *testing.T, clientSecret string, form url.Values) (int, map[string]any, http.Header) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(RecordedClientID, clientSecret)
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body, resp.Header
	}
	codeForm := url.Values{"grant_type": {"authorization_code"}, "code": {RecordedCode}, "redirect_uri": {RecordedRedirectURI}}
	refreshForm := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {RecordedRefreshToken}}

	if status, _, _ := post(t, RecordedClientSecret, codeForm); status != 200 {
		t.Fatalf("first code exchange status = %d", status)
	}
	if status, body, _ := post(t, RecordedClientSecret, codeForm); status != 400 || body["error"] != "invalid_grant" {
		t.Errorf("reused code = %d %v, want 400 invalid_grant", status, body)
	}
	status, rotated, _ := post(t, RecordedClientSecret, refreshForm)
	if status != 200 || rotated["refresh_token"] == RecordedRefreshToken || rotated["refresh_token"] == nil {
		t.Fatalf("refresh = %d %v, want a rotated refresh token", status, rotated)
	}
	if status, body, _ := post(t, RecordedClientSecret, refreshForm); status != 400 || body["error"] != "invalid_grant" {
		t.Errorf("old refresh token after rotation = %d %v, want 400 invalid_grant", status, body)
	}
	if status, body, header := post(t, "wrong", url.Values{"grant_type": {"client_credentials"}}); status != 401 || body["error"] != "invalid_client" || header.Get("WWW-Authenticate") == "" {
		t.Errorf("wrong secret = %d %v %v, want 401 invalid_client with WWW-Authenticate", status, body, header)
	}
	if status, body, _ := post(t, RecordedClientSecret, url.Values{"grant_type": {"password"}}); status != 400 || body["error"] != "unsupported_grant_type" {
		t.Errorf("unknown grant = %d %v, want 400 unsupported_grant_type", status, body)
	}
	if status, body, _ := post(t, RecordedClientSecret, url.Values{"grant_type": {"client_credentials"}}); status != 200 || body["refresh_token"] != nil {
		t.Errorf("client credentials = %d %v, want 200 without refresh_token", status, body)
	}
}
