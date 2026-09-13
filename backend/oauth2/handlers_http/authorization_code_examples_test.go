package handlers_http_test

// docs/contexts/oauth2/scenarios.feature.md の REQ-OAUTH2-005、008、021、022 が宣言する
// 具体例を、ブラウザー経由の認可からトークンの交換までを 1 本通して観測する。
//
// スタックは `testing_stack` が建てる。この具体例群が要求するのは「/authorize から
// /token まで」と「ログインできる利用者」と「リダイレクト先を登録したクライアント」の
// 3 つで、`WithBrowserFlow` がその組をまとめて配る。
//
// 通常経路の `Then` はイベントの発行で書かれている。応答だけを読むテストは、トークンは
// 出すが記録を残さない実装を通してしまうので、発行されたイベントも併せて読む。

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const codeVerifier = "authorization-code-examples-pkce-verifier-0123456789"

//spec:covers EX-OAUTH2-005-01, EX-OAUTH2-021-01: 同意済みの認可リクエストを正しい verifier で交換すると access_token・id_token・refresh_token を Bearer として返し、認証・発行・交換の 5 つのイベントが残る。
func TestAuthorizationCodeFlowIssuesTheDeclaredTokensAndRecords(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	// offline_access を要求するのは、refresh トークンを返す条件がそれだからである
	// (REQ-OAUTH2-021)。要求しない経路は EX-OAUTH2-021-02 が別に持つ。
	code, _ := browser.AuthorizationCode(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"scope": "openid profile offline_access"}))

	status, body := browser.ExchangeCode(t, code, codeVerifier)
	if status != http.StatusOK {
		t.Fatalf("認可コードの交換 status=%d body=%v", status, body)
	}
	for _, field := range []string{"access_token", "id_token", "refresh_token"} {
		if token, _ := body[field].(string); token == "" {
			t.Fatalf("応答が %s を運んでいない: %v", field, body)
		}
	}
	if body["token_type"] != "Bearer" {
		t.Fatalf("token_type=%v, want Bearer", body["token_type"])
	}
	s.Events.AssertEmitted(t,
		"UserAuthenticated", "AuthorizationCodeIssued", "AuthorizationCodeRedeemed",
		"AccessTokenIssued", "RefreshTokenIssued")
}

//spec:covers EX-OAUTH2-022-01: 認可リクエストで指定した nonce が、交換で得た ID トークンの nonce クレームへそのまま伝播する。
func TestAuthorizationRequestNoncePropagatesToTheIDToken(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	const nonce = "n-12345"

	code, _ := browser.AuthorizationCode(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"scope": "openid", "nonce": nonce}))
	status, body := browser.ExchangeCode(t, code, codeVerifier)
	if status != http.StatusOK {
		t.Fatalf("認可コードの交換 status=%d body=%v", status, body)
	}
	idToken, _ := body["id_token"].(string)
	if got := jwtClaims(t, idToken)["nonce"]; got != nonce {
		t.Fatalf("id_token の nonce=%#v, want %q", got, nonce)
	}
}

//spec:covers EX-OAUTH2-005-02: 未登録の redirect_uri を指定した認可リクエストはリダイレクトせず、IdP がエラーを表示し、認可コードも認可リクエストも作られない。
func TestAuthorizationRefusesAnUnregisteredRedirectURI(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	response := browser.Authorize(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"redirect_uri": "https://evil.example.com/callback"}))
	assertNotRedirected(t, response, "未登録の redirect_uri")
	s.Events.AssertNotEmitted(t, "AuthorizationCodeIssued")
}

//spec:covers EX-OAUTH2-005-03: 単一値の認可パラメーターが重複した、あるいは prompt が壊れた認可リクエストは認可コードを発行しない。redirect_uri を安全に確定できないときはリダイレクトもしない。
func TestAuthorizationRefusesDuplicatedParametersAndBrokenPrompt(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())

	for _, test := range []struct {
		name  string
		query url.Values
	}{
		{
			name: "client_id が重複する",
			query: func() url.Values {
				query := stack.AuthorizationQuery(codeVerifier, nil)
				query.Add("client_id", stack.BrowserClientID)
				return query
			}(),
		},
		{
			name: "redirect_uri が重複する",
			query: func() url.Values {
				query := stack.AuthorizationQuery(codeVerifier, nil)
				query.Add("redirect_uri", stack.BrowserRedirectURI)
				return query
			}(),
		},
		{
			name:  "prompt に未対応のトークンがある",
			query: stack.AuthorizationQuery(codeVerifier, map[string]string{"prompt": "select_account"}),
		},
		{
			name:  "prompt が重複する",
			query: stack.AuthorizationQuery(codeVerifier, map[string]string{"prompt": "login login"}),
		},
		{
			name:  "prompt=none をほかのトークンと併用する",
			query: stack.AuthorizationQuery(codeVerifier, map[string]string{"prompt": "none login"}),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			browser := s.Browser(t, tenancydomain.DefaultRealm)
			response := browser.Authorize(t, test.query)
			assertNoAuthorizationCode(t, response, test.name)
		})
	}
	s.Events.AssertNotEmitted(t, "AuthorizationCodeIssued")
}

//spec:covers EX-OAUTH2-005-04: request_uri と、併用を許可しないフロントチャネルの認可パラメーターが混在するリクエストは認可コードを発行しない。
func TestAuthorizationRefusesRequestURIMixedWithFrontChannelParameters(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	query := stack.AuthorizationQuery(codeVerifier, nil)
	query.Set("request_uri", "urn:ietf:params:oauth:request_uri:not-pushed")
	response := browser.Authorize(t, query)
	assertNoAuthorizationCode(t, response, "request_uri とフロントチャネルの併用")
	s.Events.AssertNotEmitted(t, "AuthorizationCodeIssued")
}

//spec:covers EX-OAUTH2-005-05: prompt=none は UI へリダイレクトせず、セッションが無ければ login_required、同意が無ければ consent_required を state と発行者識別子付きで登録済み redirect_uri へ返す。
func TestPromptNoneAnswersAtTheRedirectURIWithoutShowingUI(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())

	// セッションが無い browser。UI へ送られていないことは、リダイレクト先が
	// 登録済みの redirect_uri であることで読む。
	anonymous := s.Browser(t, tenancydomain.DefaultRealm)
	loginRequired := anonymous.Authorize(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"prompt": "none"}))
	assertPromptNoneError(t, loginRequired, "login_required")

	// 同じ browser でログインだけ済ませ、同意はしない。今度は login_required では
	// なく consent_required になる。2 つを分けて読まないと、prompt=none を一律に
	// login_required で返す実装が通る。
	signedIn := s.Browser(t, tenancydomain.DefaultRealm)
	signedIn.Authorize(t, stack.AuthorizationQuery(codeVerifier, nil)).Body.Close()
	signedIn.SignIn(t, "user", stack.UserPassword)
	consentRequired := signedIn.Authorize(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"prompt": "none"}))
	assertPromptNoneError(t, consentRequired, "consent_required")

	s.Events.AssertNotEmitted(t, "AuthorizationCodeIssued")
}

// 認可リクエストの状態そのものは、ブラウザー API が内部の id を出さないので外から
// 引けない。出し分けの観測点はログイン直後の遷移先で、同意済みなら登録済みの
// redirect_uri へ、同意が要るなら同意画面へ進む。
//
//spec:covers EX-OAUTH2-008-01, EX-OAUTH2-008-02: 同意済みの認可リクエストは同意 UI を出さずに発行まで進み、prompt=consent は同じ同意があっても同意画面へ戻す。
func TestConsentScreenAppearsOnlyWhenConsentIsMissingOrRequested(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())

	// 1 本目で同意を記録する。この 1 本が無いと、2 本目が同意画面を出さない理由が
	// 「同意済みだから」なのか「同意を読んでいないから」なのか区別できない。
	first := s.Browser(t, tenancydomain.DefaultRealm)
	first.AuthorizationCode(t, stack.AuthorizationQuery(codeVerifier, nil))

	// 2 本目は同じ scope なので、ログインの直後に同意を挟まず redirect_uri へ進む。
	second := s.Browser(t, tenancydomain.DefaultRealm)
	second.Authorize(t, stack.AuthorizationQuery(codeVerifier, nil)).Body.Close()
	signedIn := second.SignIn(t, "user", stack.UserPassword)
	redirectTo, _ := signedIn["redirect_to"].(string)
	if !strings.HasPrefix(redirectTo, stack.BrowserRedirectURI) {
		t.Fatalf("同意済みなのにログイン直後の遷移先が redirect_uri ではない: %v", signedIn)
	}

	// prompt=consent は、同じ同意があっても改めて同意を求める。
	third := s.Browser(t, tenancydomain.DefaultRealm)
	third.Authorize(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"prompt": "consent"})).Body.Close()
	next, _ := third.SignIn(t, "user", stack.UserPassword)["next"].(string)
	if !strings.HasSuffix(next, "/consent") {
		t.Fatalf("prompt=consent で同意画面へ進まない: next=%q", next)
	}
	if kind := third.Transaction(t)["kind"]; kind != "consent" {
		t.Fatalf("prompt=consent のトランザクションが consent ではない: kind=%v", kind)
	}
}

// assertNotRedirected は、拒否された認可リクエストがどこへもリダイレクトしていないことを
// 確かめる。未検証の宛先へ error を付けて飛ばす実装は、状態行だけでは見分けられない。
func assertNotRedirected(t *testing.T, response *http.Response, what string) {
	t.Helper()
	if response.StatusCode == http.StatusSeeOther || response.StatusCode == http.StatusFound {
		t.Fatalf("%s: リダイレクトが起きた status=%d Location=%q",
			what, response.StatusCode, response.Header.Get("Location"))
	}
}

// assertNoAuthorizationCode は、拒否された認可リクエストが認可コードを運んでいないことを
// 確かめる。リダイレクトして返すかエラーページを出すかは redirect_uri を安全に確定できたか
// で分かれるので、両方の形をこの 1 つで受ける。
func assertNoAuthorizationCode(t *testing.T, response *http.Response, what string) {
	t.Helper()
	location := response.Header.Get("Location")
	if response.StatusCode != http.StatusSeeOther && response.StatusCode != http.StatusFound {
		if response.StatusCode < http.StatusBadRequest {
			t.Fatalf("%s: 拒否されずに status=%d で先へ進んだ", what, response.StatusCode)
		}
		return
	}
	redirect, err := url.Parse(location)
	if err != nil {
		t.Fatalf("%s: Location を URL として読めない %q: %v", what, location, err)
	}
	if code := redirect.Query().Get("code"); code != "" {
		t.Fatalf("%s: 拒否されるべきリクエストが認可コードを発行した %q", what, code)
	}
	if strings.HasPrefix(location, "/") {
		t.Fatalf("%s: 拒否されるべきリクエストが UI へ進んだ Location=%q", what, location)
	}
}

// assertPromptNoneError は、prompt=none の応答が登録済み redirect_uri へ宣言どおりの
// エラーを state と発行者識別子付きで返していることを確かめる。
func assertPromptNoneError(t *testing.T, response *http.Response, want string) {
	t.Helper()
	location := response.Header.Get("Location")
	if !strings.HasPrefix(location, stack.BrowserRedirectURI) {
		t.Fatalf("%s: Location=%q が登録済みの redirect_uri を指していない", want, location)
	}
	redirect, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Location を URL として読めない %q: %v", location, err)
	}
	query := redirect.Query()
	if got := query.Get("error"); got != want {
		t.Fatalf("error=%q, want %q (Location=%q)", got, want, location)
	}
	if got := query.Get("state"); got != "opaque-state" {
		t.Fatalf("state=%q が伝播していない (Location=%q)", got, location)
	}
	if query.Get("iss") == "" {
		t.Fatalf("発行者識別子が付いていない (Location=%q)", location)
	}
}

// jwtClaims は JWT の payload を復号して返す。ID トークンの nonce は署名済みトークンの
// 中にしか現れないので、/token の応答本文を眺めるだけでは伝播を読めない。
func jwtClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT ではない: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("payload の復号: %v", err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("payload が JSON ではない: %v", err)
	}
	return claims
}

//spec:covers EX-OAUTH2-009-01: 事前送信は 600 秒以下の expires_in と request_uri を返し、その request_uri で送った認可リクエストは PAR レコードを Used にして PARStored と AuthorizationCodeIssued を残す。
func TestPushedAuthorizationRequestIsConsumedOnceAndIssuesACode(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	status, pushed := browser.PushAuthorizationRequest(t,
		stack.AuthorizationQuery(codeVerifier, nil))
	if status != http.StatusCreated {
		t.Fatalf("/par status=%d body=%v", status, pushed)
	}
	requestURI, _ := pushed["request_uri"].(string)
	if requestURI == "" {
		t.Fatalf("/par の応答が request_uri を運んでいない: %v", pushed)
	}
	expiresIn, _ := pushed["expires_in"].(float64)
	if expiresIn <= 0 || expiresIn > 600 {
		t.Fatalf("expires_in=%v, want 0 < expires_in <= 600", pushed["expires_in"])
	}
	s.Events.AssertEmitted(t, "PARStored")

	code, _ := browser.AuthorizationCode(t, url.Values{
		"request_uri": {requestURI}, "client_id": {stack.BrowserClientID},
	})
	if code == "" {
		t.Fatal("request_uri 経由の認可が認可コードを返さない")
	}
	s.Events.AssertEmitted(t, "AuthorizationCodeIssued")

	// 使い終えた記録が Used へ確定する。応答だけでは、記録を残したまま 2 回目も
	// 受け付ける実装と区別できない。
	record, err := s.PAR.Find(t.Context(), requestURI)
	if err != nil || record == nil {
		t.Fatalf("PAR レコードが見つからない: record=%v err=%v", record, err)
	}
	if !record.Used {
		t.Fatalf("消費した PAR レコードが Used になっていない: %+v", record)
	}
}
