package handlers_http_test

// docs/contexts/saml/standards.md が宣言する行を、製品の正式な入口から観測する。
//
// 入口は `httpadapter.Register` が組み立てた `/saml/*` である。ハーネスの配線は
// `backend/cmd/idmagic/server.go`（`Saml: deps.Saml`、`FederationSigner:
// samltoken.KeyStoreSignerProvider{...}`）と同じ形にしてある。組み立てが本番と違うと、
// 入口を通したという事実そのものが根拠にならない。

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
)

// authnRequestOptions は AuthnRequest の任意属性。行ごとに 1 つだけ変えた要求を作るために使う。
type authnRequestOptions struct {
	acsURL          string
	acsIndex        string
	nameIDFormat    string
	protocolBinding string
	isPassive       bool
	destination     string
	// version と issueInstant は、空なら受理される既定値を使う。未対応の値を 1 つだけ
	// 差し替えるための入口である。
	version      string
	issueInstant string
}

// authnRequestSequence は AuthnRequest の ID を要求ごとに変える。ID を使い回すと、
// 2 件目以降がリプレイ検査で拒否され、拒否の理由が確かめたい検査でなくなる。
var authnRequestSequence atomic.Int64

// buildAuthnRequestXML は 1 つの属性だけが違う AuthnRequest を組み立てる。まとめて変えると、
// 拒否がどの検査によるものか区別できない。
func buildAuthnRequestXML(options authnRequestOptions) string {
	destination := options.destination
	if destination == "" {
		destination = "https://idp.example/realms/default/saml/sso"
	}
	version := options.version
	if version == "" {
		version = "2.0"
	}
	issueInstant := options.issueInstant
	if issueInstant == "" {
		issueInstant = time.Now().UTC().Format(time.RFC3339)
	}
	request := `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ` +
		`ID="_req-std-` + strconv.FormatInt(authnRequestSequence.Add(1), 10) + `" Version="` + version + `" ` +
		`IssueInstant="` + issueInstant + `" ` +
		`Destination="` + destination + `"`
	if options.acsURL != "" {
		request += ` AssertionConsumerServiceURL="` + options.acsURL + `"`
	}
	if options.acsIndex != "" {
		request += ` AssertionConsumerServiceIndex="` + options.acsIndex + `"`
	}
	if options.protocolBinding != "" {
		request += ` ProtocolBinding="` + options.protocolBinding + `"`
	}
	if options.isPassive {
		request += ` IsPassive="true"`
	}
	request += `><saml:Issuer>https://sp.example.com</saml:Issuer>`
	if options.nameIDFormat != "" {
		request += `<samlp:NameIDPolicy Format="` + options.nameIDFormat + `"/>`
	}
	request += `</samlp:AuthnRequest>`
	return request
}

func redirectSSO(t *testing.T, e *echo.Echo, options authnRequestOptions) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := samldomain.EncodeRedirect([]byte(buildAuthnRequestXML(options)))
	if err != nil {
		t.Fatalf("encode redirect: %v", err)
	}
	return get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(encoded))
}

func postSSO(t *testing.T, e *echo.Echo, options authnRequestOptions) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"SAMLRequest": {base64.StdEncoding.EncodeToString([]byte(buildAuthnRequestXML(options)))}}
	request := httptest.NewRequest(http.MethodPost, defaultRealmPath("/saml/sso"), strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

// issuedAnAssertion は、応答が実際に Assertion を載せた自動 POST フォームかを返す。
// 状態コードだけでは、拒否しながら Assertion を書く実装を見分けられない。
func issuedAnAssertion(t *testing.T, recorder *httptest.ResponseRecorder) bool {
	t.Helper()
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `name="SAMLResponse"`) {
		return false
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(samlResponseFromPostForm(t, recorder.Body.String())); err != nil {
		t.Fatalf("parse SAMLResponse: %v", err)
	}
	return document.FindElement("//Assertion") != nil
}

// =====================================================================
// SAML2Profile — Web Browser SSO と ECP
// =====================================================================

// 拒否される。1 属性だけを差し替えた要求を送り、対照として無傷の要求が Assertion を発行する
// ことを先に確かめる。
//
// 形式が未対応または矛盾する要求には Assertion を発行せず、検証済みの ACS が確定している
// 場合だけ HTTP-POST のプロトコルエラーを返し、それ以外は SamlSignInRejected を発行して
// フェイルクローズで拒否する。
//
// 5 つの形式をすべて通すのは、1 つだけを読むテストが「その 1 つだけ検査する実装」を
// 通してしまうためである。どの形式でも検証は ACS を確定させる前に終わるので、確定した
// ACS は無く、答えは常にフェイルクローズの側になる。プロトコルエラーを返す側の分岐は
// NoPassive で成立し、それは EX-SAML-006-05 が持つ。ここでは、確定した ACS が無いのに
// 拒否をプロトコルエラーとして ACS へ POST してしまう実装を落とすため、応答が自動 POST
// フォームでないことまで読む。
//
//spec:covers SAML2Profile-WebBrowserSSO: 未対応の ACS インデックスと NameID 形式は、フェイルクローズで
//spec:covers EX-SAML-006-04: Version、IssueInstant、ProtocolBinding、ACS インデックス、NameIDPolicy の
func TestSamlWebBrowserSSOFailsClosedOnUnsupportedRequestParameters(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	intact := authnRequestOptions{acsURL: "https://sp.example.com/acs"}
	if !issuedAnAssertion(t, redirectSSO(t, e, intact)) {
		t.Fatal("前提が壊れている: 無傷の AuthnRequest で Assertion が発行されない")
	}

	for _, tc := range []struct {
		name    string
		options authnRequestOptions
		version string
		issued  string
	}{
		{
			// 契約は ACS の閉集合を URL で持つ。索引で指されると、どの URL を指しているかを
			// IdP は決められない。決め打ちで 0 番へ寄せる実装をここで落とす。
			name:    "AssertionConsumerServiceIndex",
			options: authnRequestOptions{acsIndex: "0"},
		},
		{
			name:    "AssertionConsumerServiceURL and Index together",
			options: authnRequestOptions{acsURL: "https://sp.example.com/acs", acsIndex: "0"},
		},
		{
			name:    "unsupported NameIDPolicy format",
			options: authnRequestOptions{acsURL: "https://sp.example.com/acs", nameIDFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos"},
		},
		{
			// SAML 1.1 の要求を 2.0 として扱うと、2.0 でしか定義されていない検査が
			// 素通りする。
			name:    "unsupported Version",
			options: authnRequestOptions{acsURL: "https://sp.example.com/acs"},
			version: "1.1",
		},
		{
			// 受理窓の外にある IssueInstant。窓を見ない実装は、いつ作られた要求でも
			// 受け取ることになる。
			name:    "IssueInstant outside the accepted window",
			options: authnRequestOptions{acsURL: "https://sp.example.com/acs"},
			issued:  time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		},
		{
			// HTTP-POST 以外の応答バインディングは提供していない。
			name:    "unsupported response ProtocolBinding",
			options: authnRequestOptions{acsURL: "https://sp.example.com/acs", protocolBinding: samldomain.SamlBindingHTTPRedirect},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := tc.options
			options.version, options.issueInstant = tc.version, tc.issued
			before := len(*events)
			recorder := redirectSSO(t, e, options)
			if issuedAnAssertion(t, recorder) {
				t.Fatalf("未対応の要求で Assertion が発行された: body=%s", recorder.Body.String())
			}
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", recorder.Code, recorder.Body.String())
			}
			// 検証済みの ACS は確定していないので、答えは ACS 宛の自動 POST ではない。
			if strings.Contains(recorder.Body.String(), `name="SAMLResponse"`) {
				t.Fatalf("確定した ACS が無いのにプロトコルエラーを POST している: %s", recorder.Body.String())
			}
			if !hasEvent((*events)[before:], "SamlSignInRejected") {
				t.Fatal("SamlSignInRejected が発行されていない")
			}
		})
	}
}

// プロトコルレスポンスである。
//
// 同じ要求を認証済みの利用者で送ると Assertion が出るので、差が `IsPassive` と認証状態だけで
// あることが分かる。`IsPassive` を無視してログイン画面へ飛ばす実装は、SP から見ると
// 「利用者に見えない認証」という約束が破られる形になる。
//
// 遷移せず、検証済みの ACS へ HTTP-POST の NoPassive プロトコルレスポンスを返す。
// 「遷移しない」は Location ヘッダーが無いことで読む。状態コードだけでは、303 を返さずに
// 本文でログイン画面を描く実装と区別できない。
//
//spec:covers SAML2Profile-WebBrowserSSO: `IsPassive=true` でログインが必要なとき、返るのは NoPassive の
//spec:covers EX-SAML-006-05: `IsPassive=true` で利用可能な既存セッションが無いとき、ログイン画面へ
func TestSamlWebBrowserSSOReturnsNoPassiveWhenLoginIsRequired(t *testing.T) {
	options := authnRequestOptions{acsURL: "https://sp.example.com/acs", isPassive: true}

	unauthenticated, _ := newServer(t, nil)
	recorder := redirectSSO(t, unauthenticated, options)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want a protocol response (body=%s)", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("IsPassive なのにログインへ遷移している: Location=%q", location)
	}
	if issuedAnAssertion(t, recorder) {
		t.Fatal("IsPassive で未認証なのに Assertion が発行された")
	}
	// 応答は検証済みの ACS 宛の自動 POST である。
	if !strings.Contains(recorder.Body.String(), `action="https://sp.example.com/acs"`) {
		t.Fatalf("NoPassive の送信先が検証済みの ACS ではない: %s", recorder.Body.String())
	}

	document := etree.NewDocument()
	if err := document.ReadFromBytes(samlResponseFromPostForm(t, recorder.Body.String())); err != nil {
		t.Fatalf("parse SAMLResponse: %v", err)
	}
	status := document.FindElement("//StatusCode")
	if status == nil {
		t.Fatalf("StatusCode missing: %s", recorder.Body.String())
	}
	// 入れ子の第二 StatusCode が NoPassive を運ぶ。最上位だけを見る実装と区別する。
	if !strings.Contains(responseXMLString(t, document), "urn:oasis:names:tc:SAML:2.0:status:NoPassive") {
		t.Fatalf("NoPassive status missing: %s", responseXMLString(t, document))
	}

	// 対照: 同じ要求でも認証済みなら Assertion が出る。
	authenticated, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})
	if !issuedAnAssertion(t, redirectSSO(t, authenticated, options)) {
		t.Fatal("認証済みの IsPassive 要求で Assertion が発行されない")
	}
}

func responseXMLString(t *testing.T, document *etree.Document) string {
	t.Helper()
	out, err := document.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// `excluded` の行なので、観測は満たさないことである。ECP は PAOS バインディングを使うので、
// 入口の有無は 2 つの形で読める。メタデータが PAOS / SOAP の SingleSignOnService を広告して
// いないことと、PAOS を名乗る要求が ECP の応答（SOAP エンベロープ）を返さないことである。
// 対照として、広告している 2 つのバインディングでは SSO が成立することを併せて読む。
//
//spec:covers SAML2Profile-ECP: Enhanced Client or Proxy プロファイルは提供していない。
func TestSamlDoesNotOfferTheECPProfile(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	metadata := get(e, "/saml/metadata")
	if metadata.Code != http.StatusOK {
		t.Fatalf("metadata status=%d", metadata.Code)
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(metadata.Body.Bytes()); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	bindings := map[string]bool{}
	for _, endpoint := range document.FindElements("//SingleSignOnService") {
		bindings[endpoint.SelectAttrValue("Binding", "")] = true
	}
	for _, excluded := range []string{
		"urn:oasis:names:tc:SAML:2.0:bindings:PAOS",
		"urn:oasis:names:tc:SAML:2.0:bindings:SOAP",
	} {
		if bindings[excluded] {
			t.Fatalf("metadata advertises the ECP binding %q", excluded)
		}
	}
	// 対照: 提供している 2 つは広告されている。広告が空でも上の検査は通ってしまう。
	for _, offered := range []string{samldomain.SamlBindingHTTPRedirect, samldomain.SamlBindingHTTPPOST} {
		if !bindings[offered] {
			t.Fatalf("metadata does not advertise the supported binding %q", offered)
		}
	}

	// PAOS を名乗る ECP 要求も、ECP の応答にはならない。ECP なら SOAP エンベロープに
	// 包んだ Response が返る。
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath("/saml/sso"), http.NoBody)
	request.Header.Set("Accept", "text/html; application/vnd.paos+xml")
	request.Header.Set("Paos", `ver="urn:liberty:paos:2003-08";"urn:oasis:names:tc:SAML:2.0:profiles:SSO:ecp"`)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	if strings.Contains(recorder.Body.String(), "Envelope") || strings.Contains(recorder.Header().Get("Content-Type"), "paos") {
		t.Fatalf("PAOS を名乗る要求に ECP の応答を返した: %s", recorder.Body.String())
	}
}

// =====================================================================
// SAML2Core — 暗号化 Assertion
// =====================================================================

// `excluded` の行なので、観測は満たさないことである。発行された Response が平文の
// `<Assertion>` を持ち `<EncryptedAssertion>` を持たないこと、メタデータが暗号化鍵
// (`KeyDescriptor use="encryption"`) を広告していないこと、そして SP 登録に暗号化を
// 要求する入口が無いことを併せて読む。
//
// 平文であることだけを読むと、SP ごとに暗号化を有効にできる実装を見逃す。有効化の入口が
// 無いことまで読んで、初めて「提供していない」の観測になる。
//
//spec:covers SAML2Core-EncryptedAssertion: 暗号化 Assertion は提供していない。
func TestSamlDoesNotOfferEncryptedAssertions(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	recorder := redirectSSO(t, e, authnRequestOptions{acsURL: "https://sp.example.com/acs"})
	if !issuedAnAssertion(t, recorder) {
		t.Fatalf("前提が壊れている: Assertion が発行されない body=%s", recorder.Body.String())
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(samlResponseFromPostForm(t, recorder.Body.String())); err != nil {
		t.Fatalf("parse SAMLResponse: %v", err)
	}
	if document.FindElement("//EncryptedAssertion") != nil {
		t.Fatalf("EncryptedAssertion を発行している: %s", responseXMLString(t, document))
	}
	if document.FindElement("//Assertion/Subject/NameID") == nil {
		t.Fatalf("Assertion の NameID が平文で読めない: %s", responseXMLString(t, document))
	}

	metadata := get(e, "/saml/metadata")
	metadataDocument := etree.NewDocument()
	if err := metadataDocument.ReadFromBytes(metadata.Body.Bytes()); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	uses := map[string]bool{}
	for _, descriptor := range metadataDocument.FindElements("//KeyDescriptor") {
		uses[descriptor.SelectAttrValue("use", "")] = true
	}
	if uses["encryption"] {
		t.Fatal("metadata が暗号化鍵を広告している")
	}
	if !uses["signing"] {
		t.Fatal("前提が壊れている: metadata が署名鍵を広告していない")
	}

	// 有効化の入口が無い。暗号化を求める登録を送っても、その要求は登録へ入らない。
	admin := newAdminServer(t)
	body := `{"entity_id":"https://enc.example.com","acs_urls":["https://enc.example.com/acs"],` +
		`"encrypt_assertion":true,"want_assertions_encrypted":true,` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	created := doJSON(admin, http.MethodPost, "/api/admin/v1/saml/service-providers", body)
	if created.Code == http.StatusOK || created.Code == http.StatusCreated {
		listed := doJSON(admin, http.MethodGet, "/api/admin/v1/saml/service-providers", "")
		if strings.Contains(listed.Body.String(), "encrypt") {
			t.Fatalf("暗号化の設定が SP 登録に入っている: %s", listed.Body.String())
		}
	}
}

// =====================================================================
// SAML2Bindings — Redirect と POST
// =====================================================================

// HTTP-POST 以外の ProtocolBinding は拒否され、SAMLResponse は HTTP-POST で返る。
//
// 行は 3 つのことを言っているので観測も 3 つ要る。片方のバインディングだけを観測すると、
// もう片方を提供していない実装と区別できない。
//
//spec:covers SAML2Bindings-RedirectPost: AuthnRequest は HTTP-Redirect と HTTP-POST の両方で受理され、
func TestSamlAcceptsRedirectAndPostBindingsAndRepliesByPost(t *testing.T) {
	authn := &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}
	options := authnRequestOptions{acsURL: "https://sp.example.com/acs"}

	t.Run("HTTP-Redirect binding is accepted", func(t *testing.T) {
		e, _ := newServer(t, authn)
		if !issuedAnAssertion(t, redirectSSO(t, e, options)) {
			t.Fatal("HTTP-Redirect の AuthnRequest が受理されない")
		}
	})

	t.Run("HTTP-POST binding is accepted", func(t *testing.T) {
		e, _ := newServer(t, authn)
		if !issuedAnAssertion(t, postSSO(t, e, options)) {
			t.Fatal("HTTP-POST の AuthnRequest が受理されない")
		}
	})

	t.Run("a response ProtocolBinding other than HTTP-POST is refused", func(t *testing.T) {
		e, _ := newServer(t, authn)
		refused := options
		refused.protocolBinding = samldomain.SamlBindingHTTPRedirect
		recorder := redirectSSO(t, e, refused)
		if issuedAnAssertion(t, recorder) {
			t.Fatalf("HTTP-Redirect の ProtocolBinding で Assertion が発行された: %s", recorder.Body.String())
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", recorder.Code)
		}
		// 対照: 同じ要求で ProtocolBinding を HTTP-POST にすれば通る。
		accepted := options
		accepted.protocolBinding = samldomain.SamlBindingHTTPPOST
		if !issuedAnAssertion(t, redirectSSO(t, e, accepted)) {
			t.Fatal("HTTP-POST の ProtocolBinding が受理されない")
		}
	})

	t.Run("the SAMLResponse comes back over HTTP-POST", func(t *testing.T) {
		e, _ := newServer(t, authn)
		recorder := redirectSSO(t, e, options)
		body := recorder.Body.String()
		if !strings.Contains(body, `method="post"`) && !strings.Contains(body, `method="POST"`) {
			t.Fatalf("自動 POST フォームではない: %s", body)
		}
		if !strings.Contains(body, `action="https://sp.example.com/acs"`) {
			t.Fatalf("フォームの送信先が ACS ではない: %s", body)
		}
		// 応答は本文であって Location ヘッダーではない。Redirect で返す実装をここで落とす。
		if location := recorder.Header().Get("Location"); location != "" {
			t.Fatalf("SAMLResponse を Redirect で返している: Location=%q", location)
		}
	})

	t.Run("a replyable protocol error comes back over HTTP-POST too", func(t *testing.T) {
		e, _ := newServer(t, nil)
		passive := options
		passive.isPassive = true
		recorder := redirectSSO(t, e, passive)
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK || !strings.Contains(body, `name="SAMLResponse"`) {
			t.Fatalf("プロトコルエラーが POST フォームで返っていない: status=%d body=%s", recorder.Code, body)
		}
		if !strings.Contains(body, `action="https://sp.example.com/acs"`) {
			t.Fatalf("プロトコルエラーの送信先が ACS ではない: %s", body)
		}
	})
}

// =====================================================================
// SAML2Metadata — IDPSSODescriptor と WantAuthnRequestsSigned
// =====================================================================

// 署名証明書、NameID 形式を公開する。
//
// 4 つとも、SP が IdP を設定するために読む値である。文字列の有無ではなく、値が製品の
// 実際のエンドポイントおよび実際に受理する NameID 形式と一致することを読む。
//
//spec:covers SAML2Metadata-IDPSSODescriptor: IdP メタデータは SSO エンドポイント、SLO エンドポイント、
func TestSamlMetadataPublishesTheIDPSSODescriptorContract(t *testing.T) {
	e, _ := newServer(t, nil)
	recorder := get(e, "/saml/metadata")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(recorder.Body.Bytes()); err != nil {
		t.Fatalf("parse metadata: %v", err)
	}
	descriptor := document.FindElement("//IDPSSODescriptor")
	if descriptor == nil {
		t.Fatalf("IDPSSODescriptor missing: %s", recorder.Body.String())
	}

	locations := func(tag string) map[string]string {
		out := map[string]string{}
		for _, endpoint := range descriptor.FindElements("./" + tag) {
			out[endpoint.SelectAttrValue("Binding", "")] = endpoint.SelectAttrValue("Location", "")
		}
		return out
	}
	sso := locations("SingleSignOnService")
	slo := locations("SingleLogoutService")
	for binding, want := range map[string]string{
		samldomain.SamlBindingHTTPRedirect: "https://idp.example/realms/default/saml/sso",
		samldomain.SamlBindingHTTPPOST:     "https://idp.example/realms/default/saml/sso",
	} {
		if sso[binding] != want {
			t.Errorf("SingleSignOnService[%s] = %q, want %q", binding, sso[binding], want)
		}
	}
	for binding, want := range map[string]string{
		samldomain.SamlBindingHTTPRedirect: "https://idp.example/realms/default/saml/slo",
		samldomain.SamlBindingHTTPPOST:     "https://idp.example/realms/default/saml/slo",
	} {
		if slo[binding] != want {
			t.Errorf("SingleLogoutService[%s] = %q, want %q", binding, slo[binding], want)
		}
	}

	// 公開している SSO エンドポイントが実在する。URL を書いているだけの実装をここで落とす。
	if sso[samldomain.SamlBindingHTTPRedirect] != "" {
		live := get(e, "/saml/sso")
		if live.Code == http.StatusNotFound {
			t.Fatalf("広告した SSO エンドポイントが 404 を返す")
		}
	}

	if descriptor.FindElement("./KeyDescriptor[@use='signing']//X509Certificate") == nil {
		t.Errorf("署名証明書が公開されていない: %s", recorder.Body.String())
	}

	// 公開する NameID 形式は、製品が実際に受理する集合と一致する。
	published := map[string]bool{}
	for _, format := range descriptor.FindElements("./NameIDFormat") {
		published[strings.TrimSpace(format.Text())] = true
	}
	for _, format := range []string{
		samldomain.SamlNameIDFormatPersistent, samldomain.SamlNameIDFormatEmailAddress,
		samldomain.SamlNameIDFormatTransient, samldomain.SamlNameIDFormatUnspecified,
	} {
		if !published[format] {
			t.Errorf("NameIDFormat %q が公開されていない", format)
		}
		if !samldomain.ValidSamlNameIDFormat(format) {
			t.Errorf("公開している NameIDFormat %q を製品は受理しない", format)
		}
	}
	for format := range published {
		if !samldomain.ValidSamlNameIDFormat(format) {
			t.Errorf("受理しない NameIDFormat %q を公開している", format)
		}
	}

	// 公開したメタデータは SP が取り込める XML である。
	if err := xml.Unmarshal(recorder.Body.Bytes(), new(struct {
		XMLName xml.Name
	})); err != nil {
		t.Errorf("metadata is not well-formed XML: %v", err)
	}
}

// 要求できる。
//
// `optional` の行なので、提供しているならその振る舞いを観測する。**要求できることの観測は、
// 有効にした SP と有効にしていない SP が同じ要求に別の答えを返すことである。** 片方だけを
// 見ると、常に検証する実装とも、決して検証しない実装とも区別が付かない。
// 併せて、証明書を伴わない有効化は登録そのものが拒否されることを読む。検証する鍵を持たない
// ポリシーは、効いていないのに効いているように見える設定になる。
//
//spec:covers SAML2Metadata-WantAuthnRequestsSigned: SP ごとの信頼ポリシーとして、AuthnRequest の署名検証を
func TestSamlServiceProviderTrustPolicyCanRequireSignedAuthnRequests(t *testing.T) {
	admin := newAdminServer(t)
	certificate := strings.ReplaceAll(certPEM(t), "\n", "\\n")

	// 証明書つきの有効化は受理される。
	enabled := `{"entity_id":"https://signed.example.com","acs_urls":["https://signed.example.com/acs"],` +
		`"want_authn_requests_signed":true,"authn_request_signing_certificate_pem":"` + certificate + `",` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	if recorder := doJSON(admin, http.MethodPost, "/api/admin/v1/saml/service-providers", enabled); recorder.Code >= http.StatusBadRequest {
		t.Fatalf("証明書つきの有効化が拒否された: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	// 証明書のない有効化は拒否される。検証する鍵が無ければポリシーは効かない。
	unusable := `{"entity_id":"https://unusable.example.com","acs_urls":["https://unusable.example.com/acs"],` +
		`"want_authn_requests_signed":true,` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	if recorder := doJSON(admin, http.MethodPost, "/api/admin/v1/saml/service-providers", unusable); recorder.Code != http.StatusBadRequest {
		t.Fatalf("鍵の無い有効化が受理された: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	// ポリシーを有効にした SP は、署名の無い AuthnRequest を拒否する。
	requiring := newServerRequiringSignedAuthnRequests(t)
	options := authnRequestOptions{acsURL: "https://sp.example.com/acs"}
	refused := redirectSSO(t, requiring, options)
	if issuedAnAssertion(t, refused) {
		t.Fatalf("署名を要求する SP が署名の無い要求へ Assertion を発行した: %s", refused.Body.String())
	}
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", refused.Code)
	}

	// 対照: 同じ署名の無い要求でも、ポリシーを有効にしていない SP では通る。
	// これが無いと、上の拒否が署名ポリシーによるものだと言えない。
	permissive, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})
	if !issuedAnAssertion(t, redirectSSO(t, permissive, options)) {
		t.Fatal("ポリシーを有効にしていない SP で署名の無い要求が通らない")
	}
}

// newServerRequiringSignedAuthnRequests は、AuthnRequest の署名検証を要求する SP を 1 つだけ
// 持つスタックを組む。newServer との差は SP 登録の 2 フィールドだけなので、振る舞いの差が
// そのポリシーに由来すると言える。
func newServerRequiringSignedAuthnRequests(t *testing.T) *echo.Echo {
	t.Helper()
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID:                          "https://sp.example.com",
		ACSURLs:                           []string{"https://sp.example.com/acs"},
		SignAssertion:                     true,
		WantAuthnRequestsSigned:           true,
		AuthnRequestSigningCertificatePEM: certPEM(t),
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice"})
	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		Saml:             saml.Module{SPRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         userRepo,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		AuthnResolver: stubResolver{ctx: &authdomain.AuthenticationContext{
			UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}},
	})
	return e
}
