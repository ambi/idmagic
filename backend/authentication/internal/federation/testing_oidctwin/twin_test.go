package testing_oidctwin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// fidelityExchanges は模倣と照合する往復である。
// jwks-encryption-and-signing-keys は先頭が暗号化用の EC 鍵であり、署名用の RSA 鍵だけを配信する模倣とは形が違うので、
// 製品のクライアントが EC 鍵を読み飛ばすことの再生だけに使う。
var (
	fidelityExchanges = []string{"discovery", "token-authorization-code", "token-authorization-code-profile-claims", "token-invalid-request", "jwks"}
	replayOnly        = []string{"jwks-encryption-and-signing-keys"}
)

// 模倣は、記録した往復の要求に記録と同じ応答を返す。
func TestTwin_MatchesOIDCRecording(t *testing.T) {
	tw := NewTwin(t, RecordedIssuer, RecordedClientID, RecordedClientSecret)
	for _, diff := range Diverge(tw, OIDCRecording(t).Select(t, fidelityExchanges...)) {
		t.Error(diff)
	}
}

// 記録のすべての往復が、照合か再生だけの一覧のどちらかに含まれる。往復を足したときに照合から漏れないためである。
func TestTwin_FidelityCoversEveryExchange(t *testing.T) {
	covered := map[string]bool{}
	for _, name := range append(append([]string{}, fidelityExchanges...), replayOnly...) {
		covered[name] = true
	}
	for _, ex := range OIDCRecording(t).Exchanges {
		if !covered[ex.Name] {
			t.Errorf("exchange %q is neither compared with the twin nor listed as replay-only", ex.Name)
		}
	}
}

// 照合が空虚でないことを確かめる。模倣の応答を記録から外すと、照合が食い違いを報告する。
// id_token と鍵の kid、n を型だけで比べても、ほかの値と形は比べ続ける。
func TestDiverge_ReportsTwinThatLeavesTheRecording(t *testing.T) {
	tests := []struct {
		name   string
		issuer string
		setup  func(tw *Twin)
		want   string
	}{
		{
			"issuer が記録と違う", "https://other.example.com", func(*Twin) {},
			"discovery: body.issuer = https://other.example.com, recorded https://server.example.com",
		},
		{"Discovery が 500 を返す", RecordedIssuer, func(tw *Twin) {
			tw.InjectFaults("/.well-known/openid-configuration", Fault{Status: 500, Body: "boom"})
		}, "discovery: status = 500, recorded 200"},
		{"トークンの応答が id_token を持たない", RecordedIssuer, func(tw *Twin) {
			tw.InjectFaults("/token", Fault{Status: 200, ContentType: "application/json", Body: `{"access_token":"a","token_type":"Bearer","refresh_token":"r","expires_in":3600}`})
		}, "token-authorization-code: body.id_token is missing"},
		{
			"Basic 認証を受け付けない", RecordedIssuer, func(tw *Twin) { tw.AuthMethods = []string{ClientSecretPost} },
			"token-authorization-code: status = 401, recorded 200",
		},
		{"JWK Set の鍵の e が違う", RecordedIssuer, func(tw *Twin) {
			tw.InjectFaults("/jwks.json", Fault{Status: 200, ContentType: "application/json", Body: `{"keys":[{"kty":"RSA","kid":"k","n":"AQAB","e":"Aw"}]}`})
		}, "jwks: body.keys[0].e = Aw, recorded AQAB"},
		{
			"JWK Set が鍵の kid を持たない", RecordedIssuer, func(tw *Twin) { tw.OmitKid = true },
			"jwks: body.keys[0].kid is missing",
		},
		{"エラー応答が Cache-Control を持たない", RecordedIssuer, func(tw *Twin) {
			tw.InjectFaults("/token", Fault{}, Fault{}, Fault{Status: 400, ContentType: "application/json", Body: `{"error":"invalid_request"}`})
		}, `token-invalid-request: header Cache-Control = "", recorded "no-store"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := NewTwin(t, tc.issuer, RecordedClientID, RecordedClientSecret)
			tc.setup(tw)
			diffs := Diverge(tw, OIDCRecording(t).Select(t, fidelityExchanges...))
			if !strings.Contains(strings.Join(diffs, "\n"), tc.want) {
				t.Errorf("Diverge() = %q, want a line %q", diffs, tc.want)
			}
		})
	}
}

// 記録にない組み合わせ：PKCE の不一致、使用済みの認可コード、受け付けない認証方式、未知のグラント。
func TestTwin_RefusesOutsideTheRecording(t *testing.T) {
	tw := NewTwin(t, RecordedIssuer, RecordedClientID, RecordedClientSecret)
	client := tw.Client(t)
	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	authorize := url.Values{
		"response_type": {"code"}, "client_id": {RecordedClientID}, "redirect_uri": {RecordedRedirectURI},
		"scope": {"openid"}, "state": {"s"}, "nonce": {"n"},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
	}
	resp, err := client.Get(RecordedIssuer + "/authorize?" + authorize.Encode())
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if resp.StatusCode != http.StatusFound || code == "" || location.Query().Get("state") != "s" {
		t.Fatalf("authorize = %d %q, want 302 to the redirect URI with code and state", resp.StatusCode, location)
	}

	post := func(t *testing.T, form url.Values, basic bool) (int, map[string]any) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, RecordedIssuer+"/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if basic {
			req.SetBasicAuth(RecordedClientID, RecordedClientSecret)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return resp.StatusCode, body
	}
	exchange := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {RecordedRedirectURI}}

	wrongVerifier := url.Values{"code_verifier": {strings.Repeat("w", 43)}}
	maps.Copy(wrongVerifier, exchange)
	if status, body := post(t, wrongVerifier, true); status != 400 || body["error"] != "invalid_grant" {
		t.Errorf("wrong code_verifier = %d %v, want 400 invalid_grant", status, body)
	}
	exchange.Set("code_verifier", verifier)
	if status, body := post(t, exchange, true); status != 200 || body["id_token"] == nil {
		t.Fatalf("code exchange = %d %v, want 200 with id_token", status, body)
	}
	if status, body := post(t, exchange, true); status != 400 || body["error"] != "invalid_grant" {
		t.Errorf("reused code = %d %v, want 400 invalid_grant", status, body)
	}

	tw.AuthMethods = []string{ClientSecretBasic}
	inBody := url.Values{"grant_type": {"authorization_code"}, "code": {"x"}, "client_id": {RecordedClientID}, "client_secret": {RecordedClientSecret}}
	if status, body := post(t, inBody, false); status != 401 || body["error"] != "invalid_client" {
		t.Errorf("client_secret_post to a basic-only OP = %d %v, want 401 invalid_client", status, body)
	}
	if status, body := post(t, url.Values{"grant_type": {"password"}}, true); status != 400 || body["error"] != "unsupported_grant_type" {
		t.Errorf("unknown grant = %d %v, want 400 unsupported_grant_type", status, body)
	}
}
