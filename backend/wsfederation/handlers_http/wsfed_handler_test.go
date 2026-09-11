package handlers_http_test

// 主要ユースケース追跡: REQ-WSFEDERATION-002、REQ-WSFEDERATION-004。

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"html"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/requests_wstrust"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
	dsig "github.com/russellhaering/goxmldsig"
)

// stubResolver は固定の認証コンテキストを返す AuthnResolver。
type stubResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (s stubResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return s.ctx, nil
}

func devSigner(t *testing.T) *samltoken.Signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test fed signing"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	signer, err := samltoken.NewSigner(cert, key)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

func newServer(t *testing.T, authn *authdomain.AuthenticationContext) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	e, captured, _ := newServerWithSigner(t, authn)
	return e, captured
}

// newServerWithSigner は組み立て済みサーバーと、RP が応答を検証するための IdP 署名者を返す。
func newServerWithSigner(t *testing.T, authn *authdomain.AuthenticationContext) (*echo.Echo, *[]spec.DomainEvent, *samltoken.Signer) {
	t.Helper()

	captured := &[]spec.DomainEvent{}

	rpRepo := wsfedmemory.NewWsFedRelyingPartyRepository()
	rpRepo.Seed(&feddomain.WsFedRelyingParty{
		Wtrealm: "urn:idmagic:demo-rp",
		// 2 つ目の返信先は既定 (先頭) ではないので、wreply を指定した要求がその指定どおりに
		// 届いたのか、単に先頭が使われただけなのかを区別できる。
		ReplyURLs: []string{"https://rp.example/wsfed", "https://rp.example/wsfed/alternate"},
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format:          "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
				SourceAttribute: "user_id",
			},
			Rules: []claimdomain.ClaimMappingRule{
				{ClaimType: "http://schemas.xmlsoap.org/claims/UPN", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "preferred_username", Required: true},
			},
		},
	})

	userRepo := usermemory.NewUserRepository()
	hasher := testing_passwords.NewHasher()
	passwordHash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	sentinel, err := hasher.Hash("sentinel-password")
	if err != nil {
		t.Fatalf("hash sentinel: %v", err)
	}
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice", PasswordHash: passwordHash})

	signer := devSigner(t)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:   "https://idp.example",
		Contract: spec.CurrentRuntimeContract(),

		Emit: func(ev spec.DomainEvent) { *captured = append(*captured, ev) }, WsFederation: wsfederation.Module{RPRepo: rpRepo},
		UserRepo:             userRepo,
		PasswordHasher:       hasher,
		SentinelPasswordHash: sentinel,
		OAuth2:               oauth2.Module{ClientAssertionReplayStore: memory.NewClientAssertionReplayStore()},
		FederationSigner:     signer,
		AuthnResolver:        stubResolver{ctx: authn},
	})
	return e, captured, signer
}

// hasEvent は指定 EventType の event が捕捉されたかを返す。
func hasEvent(events []spec.DomainEvent, eventType string) bool {
	for _, ev := range events {
		if ev.EventType() == eventType {
			return true
		}
	}
	return false
}

func get(e *echo.Echo, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, defaultRealmPath(target), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestWsFedSignIn_AuthenticatedIssuesPassiveForm(t *testing.T) {
	e, events, signer := newServerWithSigner(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})

	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wctx=ctx-42")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `action="https://rp.example/wsfed"`) {
		t.Fatalf("form action missing: %s", body)
	}
	if !strings.Contains(body, `value="wsignin1.0"`) {
		t.Fatal("wa hidden input missing")
	}
	if !strings.Contains(body, "RequestSecurityTokenResponse") {
		t.Fatal("RSTR not present in wresult")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	if !hasEvent(*events, "WsFedSignInIssued") {
		t.Fatal("WsFedSignInIssued not emitted")
	}

	// 最終効果は「RP が検証可能なサインイン応答を受け取る」ことなので、RP と同じ手順で
	// wresult を取り出し、IdP 証明書に対して assertion 署名を検証する。
	document := etree.NewDocument()
	if err := document.ReadFromString(wresultFromPassiveForm(t, body)); err != nil {
		t.Fatalf("parse wresult: %v", err)
	}
	assertion := document.FindElement("//Assertion")
	if assertion == nil {
		t.Fatalf("assertion missing in wresult: %s", body)
	}
	if assertion.FindElement("./Signature") == nil {
		t.Fatalf("assertion is not signed: %s", body)
	}
	store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{signer.Certificate()}}
	validation := dsig.NewDefaultValidationContext(store)
	validation.IdAttribute = idAttributeForAssertion(assertion)
	if _, err := validation.Validate(assertion); err != nil {
		t.Fatalf("assertion signature did not validate against the IdP certificate: %v", err)
	}
}

// wresultFromPassiveForm は自動 POST フォーム HTML から wresult の XML を取り出す。
func wresultFromPassiveForm(t *testing.T, body string) string {
	t.Helper()
	const marker = `name="wresult" value="`
	_, after, ok := strings.Cut(body, marker)
	if !ok {
		t.Fatalf("wresult hidden input missing: %s", body)
	}
	rest := after
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("wresult value is not terminated: %s", body)
	}
	return html.UnescapeString(rest[:end])
}

// idAttributeForAssertion は SAML 1.1 と 2.0 で異なる ID 属性名を返す。
func idAttributeForAssertion(assertion *etree.Element) string {
	if assertion.SelectAttrValue("AssertionID", "") != "" {
		return "AssertionID"
	}
	return "ID"
}

func TestWsFedSignIn_DefaultsToSAML11Token(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// 既定 token type は Entra 互換の SAML 1.1。
	if !strings.Contains(body, "urn:oasis:names:tc:SAML:1.0:assertion") {
		t.Fatalf("RSTR TokenType is not SAML 1.1: %s", body)
	}
	// SAML 1.1 assertion は MajorVersion/MinorVersion と AuthenticationStatement を持つ。
	if !strings.Contains(body, "MajorVersion=&#34;1&#34;") && !strings.Contains(body, `MajorVersion="1"`) {
		t.Fatalf("SAML 1.1 MajorVersion not present: %s", body)
	}
}

func TestWsFedSignIn_StaleSessionWithWfreshRedirectsToLogin(t *testing.T) {
	// 認証から十分時間が経過したセッションに wfresh=0 を要求すると再認証へ誘導される。
	staleAuthTime := time.Now().Add(-30 * time.Minute).Unix()
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: staleAuthTime, AMR: []string{"pwd"}})

	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wfresh=0")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303 (wfresh forces re-auth)", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/realms/default/login") {
		t.Fatalf("Location = %q, want /realms/default/login", loc)
	}
}

func TestWsFedSignIn_UnsupportedWauthRejected(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wauth=urn:federation:authentication:windows")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 (integrated Windows auth unsupported)", rec.Code)
	}
	if !hasEvent(*events, "WsFedSignInRejected") {
		t.Fatal("WsFedSignInRejected not emitted")
	}
}

func TestWsFedSignIn_UnauthenticatedRedirectsToLogin(t *testing.T) {
	e, _ := newServer(t, nil) // resolver returns no session

	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", rec.Code)
	}
	loc := rec.Header().Get("Location")
	// redirect 先はテナントの正規ロケーション配下の相対パス。
	if !strings.HasPrefix(loc, "/realms/default/login") || !strings.Contains(loc, "return_to=") {
		t.Fatalf("Location = %q, want /realms/default/login with return_to", loc)
	}
	// return_to は元の WS-Fed 要求へ戻ること。
	if decoded := loc[strings.Index(loc, "return_to=")+len("return_to="):]; !strings.Contains(mustUnescape(t, decoded), "/wsfed") {
		t.Fatalf("return_to does not point back to /wsfed: %q", loc)
	}
}

func TestWsFedSignIn_UnknownRelyingParty(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	if rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:unknown"); rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
	if !hasEvent(*events, "WsFedSignInRejected") {
		t.Fatal("WsFedSignInRejected not emitted")
	}
}

func TestWsFedSignIn_DisallowedWreplyRejected(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wreply=https://evil.example/steal")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 (open redirect prevention)", rec.Code)
	}
}

// assertNoPassiveTokenIssued は passive の応答が「トークンを返していない」ことを確かめる。
//
// 拒否の状態符号だけを見ると、400 を返しつつ本文に wresult を載せる実装や、拒否したのに
// 発行 event を出す実装と区別できない。応答本文と event の両方を見る。
func assertNoPassiveTokenIssued(t *testing.T, rec *httptest.ResponseRecorder, events []spec.DomainEvent) {
	t.Helper()
	body := rec.Body.String()
	for _, marker := range []string{"wresult", "RequestSecurityTokenResponse", "Assertion"} {
		if strings.Contains(body, marker) {
			t.Fatalf("refused sign-in still carries %q in its body: %s", marker, body)
		}
	}
	if hasEvent(events, "WsFedSignInIssued") {
		t.Fatal("WsFedSignInIssued emitted for a refused sign-in")
	}
}

// WSFed-PassiveSignIn: wsignin1.0 が、登録済み wtrealm と許可済み wreply の組にだけトークンを返すことを
// 固定する。
//
// 成功経路だけを観測すると、wtrealm も wreply も照合しない実装と区別できない。したがって観測は 2 つ要る。
// 登録済みの組では指定した wreply へトークンが届くこと (既定の先頭 URL で代用されないこと)、および
// 未登録の wtrealm と、登録済み wtrealm に対する許可外の wreply のそれぞれでトークンが出ないことである。
func TestWsFedPassiveSignIn_IssuesOnlyToTheRegisteredRealmAndAllowedReply(t *testing.T) {
	const allowedReply = "https://rp.example/wsfed/alternate"

	t.Run("registered wtrealm with an allowed wreply receives the token", func(t *testing.T) {
		e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})
		rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wreply="+url.QueryEscape(allowedReply))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		// 既定は先頭の返信先なので、ここが alternate であることが「指定した許可済み wreply へ返した」証拠になる。
		if !strings.Contains(body, `action="`+allowedReply+`"`) {
			t.Fatalf("token was not posted to the requested allowed reply URL: %s", body)
		}
		if !strings.Contains(wresultFromPassiveForm(t, body), "RequestSecurityTokenResponse") {
			t.Fatalf("wresult does not carry an RSTR: %s", body)
		}
		if !hasEvent(*events, "WsFedSignInIssued") {
			t.Fatal("WsFedSignInIssued not emitted for an allowed wtrealm/wreply pair")
		}
	})

	t.Run("unregistered wtrealm receives no token", func(t *testing.T) {
		e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})
		// wreply は登録済み RP の許可集合に属する値なので、拒否の理由は wtrealm が未登録であることに限られる。
		rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:not-registered&wreply="+url.QueryEscape(allowedReply))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400 for an unregistered wtrealm", rec.Code)
		}
		assertNoPassiveTokenIssued(t, rec, *events)
	})

	t.Run("wreply outside the allowed set receives no token", func(t *testing.T) {
		e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})
		rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wreply="+url.QueryEscape("https://evil.example/steal"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400 for a wreply outside the allowed set", rec.Code)
		}
		assertNoPassiveTokenIssued(t, rec, *events)
	})
}

// WSFed-SilentSignIn: 無音サインイン (prompt=none 相当) を提供しないことを固定する。
//
// excluded の行なので、観測は行の Statement を満たすことではなく満たさないことになる。無音を求める入力が
// 正式な入口へ届いたとき、(1) 無音でトークンが出ないこと、(2) 利用者に何も見せない失敗応答にも化けず、
// 対話的なログインへ誘導するか明示的に拒否することの 2 つを観測する。この 2 つを分けないと、「無音の
// 発行はしないが無音で静かに失敗する」実装を通してしまう。
//
// 本項目はこの行で excluded の観測の型を決め、WSTrust13-WindowsTransport へ広げた。
func TestWsFedSilentSignIn_NotProvided(t *testing.T) {
	t.Run("a silent-auth hint on an unauthenticated request still requires interactive login", func(t *testing.T) {
		e, events := newServer(t, nil) // セッション無し。
		rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&prompt=none")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303: prompt=none must not turn into a silent response", rec.Code)
		}
		if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/realms/default/login") {
			t.Fatalf("Location = %q, want the interactive login screen", loc)
		}
		assertNoPassiveTokenIssued(t, rec, *events)
	})

	t.Run("the integrated Windows wauth that would authenticate silently is refused", func(t *testing.T) {
		// セッションはあるので、拒否の理由は「要求された無音の方式を提供しない」ことに限られる。
		e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})
		rec := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wauth=urn:federation:authentication:windows")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400: the silent integrated-Windows method is not provided", rec.Code)
		}
		// 実施済みのパスワード認証で黙って代用しない。要求された方式を満たさないまま発行するのは、
		// RP から見れば無音認証が成立したのと同じ意味になる。
		assertNoPassiveTokenIssued(t, rec, *events)
		if !hasEvent(*events, "WsFedSignInRejected") {
			t.Fatal("WsFedSignInRejected not emitted")
		}
	})
}

func TestWsFedSignOut_RedirectsToAllowedWreply(t *testing.T) {
	e, events := newServer(t, nil)
	rec := get(e, "/wsfed?wa=wsignout1.0&wtrealm=urn:idmagic:demo-rp&wreply=https://rp.example/wsfed")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://rp.example/wsfed" {
		t.Fatalf("Location = %q, want allowed wreply", loc)
	}
	// セッション cookie が失効される。
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("session cookie not cleared: %q", rec.Header().Get("Set-Cookie"))
	}
	if !hasEvent(*events, "WsFedSignOut") {
		t.Fatal("WsFedSignOut not emitted")
	}
}

func TestWsFedSignOut_DisallowedWreplyNoRedirect(t *testing.T) {
	e, _ := newServer(t, nil)
	rec := get(e, "/wsfed?wa=wsignout1.0&wtrealm=urn:idmagic:demo-rp&wreply=https://evil.example/x")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200 (no open redirect)", rec.Code)
	}
}

func TestWsFedSignOutCleanup_ClearsAndReturns200(t *testing.T) {
	e, _ := newServer(t, nil)
	rec := get(e, "/wsfed?wa=wsignoutcleanup1.0")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestFederationMetadata_Published(t *testing.T) {
	e, _ := newServer(t, nil)
	rec := get(e, "/federationmetadata/2007-06/federationmetadata.xml")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`entityID="https://idp.example/realms/default"`,
		"fed:PassiveRequestorEndpoint",
		"https://idp.example/realms/default/wsfed",
		"https://idp.example/realms/default/trust/usernamemixed",
		"ds:X509Certificate",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata missing %q:\n%s", want, body)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("Content-Type=%q, want application/xml", ct)
	}
}

func TestTrustMEX_Published(t *testing.T) {
	e, _ := newServer(t, nil)
	rec := get(e, "/trust/mex")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"mex:Metadata",
		"UserNameWSTrustBinding_IWSTrust13Sync",
		"https://idp.example/realms/default/trust/usernamemixed",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("MEX missing %q:\n%s", want, body)
		}
	}
}

func TestWsTrustUsernameMixed_IssuesRSTR(t *testing.T) {
	e, events := newServer(t, nil)
	rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:issue-1", "urn:idmagic:demo-rp"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"RequestSecurityTokenResponseCollection",
		"RequestedSecurityToken",
		"urn:oasis:names:tc:SAML:1.0:assertion",
		"urn:idmagic:demo-rp",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("RSTR missing %q:\n%s", want, body)
		}
	}
	if !hasEvent(*events, "WsTrustTokenIssued") {
		t.Fatal("WsTrustTokenIssued not emitted")
	}
}

func TestWsTrustUsernameMixed_RejectsUnknownAppliesTo(t *testing.T) {
	e, events := newServer(t, nil)
	rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:issue-2", "urn:unknown"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 body=%s", rec.Code, rec.Body.String())
	}
	if !hasEvent(*events, "WsTrustTokenRejected") {
		t.Fatal("WsTrustTokenRejected not emitted")
	}
}

func TestWsTrustUsernameMixed_RejectsExpiredTimestamp(t *testing.T) {
	e, _ := newServer(t, nil)
	expired := time.Now().UTC().Add(-10 * time.Minute)
	if rec := postWsTrustSOAP(e, wsTrustRST(expired, "urn:uuid:expired", "urn:idmagic:demo-rp")); rec.Code != http.StatusBadRequest {
		t.Fatalf("expired status=%d, want 400", rec.Code)
	}
}

// countEvents は指定 EventType の event が捕捉された件数を返す。
//
// 同じサーバーで先に成功した発行があるときは「出ていないこと」を hasEvent では確かめられないので、
// 基準値との差を見る。
func countEvents(events []spec.DomainEvent, eventType string) int {
	count := 0
	for _, ev := range events {
		if ev.EventType() == eventType {
			count++
		}
	}
	return count
}

// assertNoWsTrustTokenIssued は、この要求に対してトークンが出ていないことを確かめる。
//
// 拒否の状態符号だけを見ると、400 を返しつつ本文に RSTR を載せる実装と区別できない。issuedBefore は
// 同じサーバーで先に成功した発行を除くための基準値。
func assertNoWsTrustTokenIssued(t *testing.T, rec *httptest.ResponseRecorder, events []spec.DomainEvent, issuedBefore int) {
	t.Helper()
	body := rec.Body.String()
	for _, marker := range []string{"RequestSecurityTokenResponse", "RequestedSecurityToken", "Assertion"} {
		if strings.Contains(body, marker) {
			t.Fatalf("refused RST still carries %q in its response: %s", marker, body)
		}
	}
	if got := countEvents(events, "WsTrustTokenIssued"); got != issuedBefore {
		t.Fatalf("WsTrustTokenIssued count = %d, want %d: a refused RST issued a token", got, issuedBefore)
	}
}

// WSTrust13-IssueBearer: Issue 要求に対して Bearer の SAML assertion を RSTR で返すことを固定する。
//
// RSTR の外形だけでは、保持者証明 (holder-of-key) の assertion を包んだ応答と区別できない。Bearer で
// あるかを決めるのは assertion の SubjectConfirmation なので、RP と同じ手順で RSTR から assertion を
// 取り出し、確認方法まで読む。SAML 1.1 は Subject を 2 箇所 (認証文と属性文) に置くので、そのすべてが
// Bearer であることを見る。1 箇所だけを見ると、片方を保持者証明にした実装を通してしまう。
func TestWsTrustIssueBearer_ReturnsABearerSAMLAssertionInTheRSTR(t *testing.T) {
	e, events := newServer(t, nil)
	rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:issue-bearer", "urn:idmagic:demo-rp"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	document := etree.NewDocument()
	if err := document.ReadFromString(rec.Body.String()); err != nil {
		t.Fatalf("parse RSTR: %v", err)
	}
	// Issue に対する応答であること。
	if got := document.FindElement("//t:RequestType"); got == nil || got.Text() != wstrust.RequestIssue {
		t.Fatalf("RSTR does not answer an Issue request: %+v", got)
	}
	assertion := document.FindElement("//t:RequestedSecurityToken/Assertion")
	if assertion == nil {
		t.Fatalf("RequestedSecurityToken does not carry a SAML assertion: %s", rec.Body.String())
	}
	methods := assertion.FindElements(".//SubjectConfirmation/ConfirmationMethod")
	if len(methods) == 0 {
		t.Fatalf("the assertion states no subject confirmation method: %s", rec.Body.String())
	}
	for _, method := range methods {
		if method.Text() != "urn:oasis:names:tc:SAML:1.0:cm:bearer" {
			t.Fatalf("subject confirmation method = %q, want bearer", method.Text())
		}
	}
	// Bearer の assertion は提示するだけで使えるので、署名が唯一の真正性の根拠になる。
	if assertion.FindElement("./Signature") == nil {
		t.Fatalf("the bearer assertion is not signed: %s", rec.Body.String())
	}
	if !hasEvent(*events, "WsTrustTokenIssued") {
		t.Fatal("WsTrustTokenIssued not emitted")
	}
}

// WSS-UsernameTokenPassword: 能動的 STS が UsernameToken の username/password を認証することを固定する。
//
// 正しい資格情報が通ることだけを観測すると、UsernameToken を読み捨てて誰にでも発行する実装と区別
// できない。誤ったパスワードと未知の username のそれぞれについて、拒否そのものと、その拒否が防いだ
// 効果 (トークンが出ていないこと) を観測する。
func TestWsTrustUsernameTokenPassword_AuthenticatesTheSuppliedCredential(t *testing.T) {
	t.Run("the registered username and password are authenticated", func(t *testing.T) {
		e, events := newServer(t, nil)
		rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:password-ok", "urn:idmagic:demo-rp"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !hasEvent(*events, "WsTrustTokenIssued") {
			t.Fatal("WsTrustTokenIssued not emitted for a valid credential")
		}
	})

	t.Run("a wrong password receives no token", func(t *testing.T) {
		e, events := newServer(t, nil)
		body := strings.Replace(
			wsTrustRST(time.Now().UTC(), "urn:uuid:password-wrong", "urn:idmagic:demo-rp"),
			"<o:Password>correct-password</o:Password>",
			"<o:Password>wrong-password</o:Password>",
			1,
		)
		rec := postWsTrustSOAP(e, body)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401 for a wrong password", rec.Code)
		}
		assertNoWsTrustTokenIssued(t, rec, *events, 0)
		if !hasEvent(*events, "WsTrustTokenRejected") {
			t.Fatal("WsTrustTokenRejected not emitted")
		}
	})

	t.Run("an unknown username receives no token", func(t *testing.T) {
		e, events := newServer(t, nil)
		body := strings.Replace(
			wsTrustRST(time.Now().UTC(), "urn:uuid:username-unknown", "urn:idmagic:demo-rp"),
			"<o:Username>alice</o:Username>",
			"<o:Username>mallory</o:Username>",
			1,
		)
		rec := postWsTrustSOAP(e, body)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401 for an unknown username", rec.Code)
		}
		assertNoWsTrustTokenIssued(t, rec, *events, 0)
	})
}

// WSAddressing-MessageIDToAction: MessageID をリプレイ防止のために検証し、To を能動的 STS の
// エンドポイントとして、Action を Issue として検証することを固定する。
//
// 1 行が 3 つの検証を束ねているので、3 つを 1 つの入力で崩すと手前の検証で落ちて後段が確かめられない。
// 崩すのは 1 度に 1 つだけで、崩していない要素が有効であることは、同じ組み立てから作った正しい要求が
// 通ることで先に確認する。MessageID はリプレイ防止のための検証なので、観測は「値が読めること」では
// なく「同じ MessageID の 2 度目が通らないこと」である。
func TestWsTrustAddressing_ValidatesMessageIDToAndAction(t *testing.T) {
	const validTo = "https://idp.example/realms/default/trust/usernamemixed"

	t.Run("the unmodified request is accepted", func(t *testing.T) {
		e, _ := newServer(t, nil)
		rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:addressing-baseline", "urn:idmagic:demo-rp"))
		if rec.Code != http.StatusOK {
			t.Fatalf("baseline status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("a replayed MessageID receives no second token", func(t *testing.T) {
		e, events := newServer(t, nil)
		first := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:addressing-replay", "urn:idmagic:demo-rp"))
		if first.Code != http.StatusOK {
			t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
		}
		issued := countEvents(*events, "WsTrustTokenIssued")
		// MessageID 以外はすべて新しく有効な要求。通らない理由は MessageID の再利用に限られる。
		second := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:addressing-replay", "urn:idmagic:demo-rp"))
		if second.Code != http.StatusBadRequest {
			t.Fatalf("replay status=%d, want 400", second.Code)
		}
		assertNoWsTrustTokenIssued(t, second, *events, issued)
	})

	t.Run("a To outside the active STS endpoint receives no token", func(t *testing.T) {
		e, events := newServer(t, nil)
		body := strings.Replace(
			wsTrustRST(time.Now().UTC(), "urn:uuid:addressing-to", "urn:idmagic:demo-rp"),
			"<a:To>"+validTo+"</a:To>",
			"<a:To>https://evil.example/trust/usernamemixed</a:To>",
			1,
		)
		rec := postWsTrustSOAP(e, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400 for a To that is not the active STS endpoint", rec.Code)
		}
		assertNoWsTrustTokenIssued(t, rec, *events, 0)
	})

	t.Run("an Action other than Issue receives no token", func(t *testing.T) {
		e, events := newServer(t, nil)
		// RequestType は Issue のまま残す。崩すのは WS-Addressing の Action だけ。
		body := strings.Replace(
			wsTrustRST(time.Now().UTC(), "urn:uuid:addressing-action", "urn:idmagic:demo-rp"),
			"<a:Action>"+wstrust.RequestIssue+"</a:Action>",
			"<a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Renew</a:Action>",
			1,
		)
		rec := postWsTrustSOAP(e, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400 for an Action other than Issue", rec.Code)
		}
		assertNoWsTrustTokenIssued(t, rec, *events, 0)
	})
}

// WSTrust13-WindowsTransport: WindowsTransport / Kerberos の能動的プロファイルを提供しないことを固定する。
//
// excluded の観測の型は WSFed-SilentSignIn で決めたものに従う。ただしこの行は入口そのものを持たないので、
// 「届いた要求の拒否」ではなく「要求の宛先が存在しないこと」で観測する。広告まで見るのは、入口が無くても
// metadata がその binding を広告していれば、RP は提供されていると読んで能動的プロファイルを組み立てて
// しまうからである。
func TestWsTrustWindowsTransport_NotProvided(t *testing.T) {
	e, events := newServer(t, nil)

	// AD FS が WindowsTransport / Kerberos の能動的プロファイルに用いる入口。どれも存在しない。
	for _, path := range []string{
		"/realms/default/trust/13/windowstransport",
		"/realms/default/trust/13/kerberosmixed",
		"/realms/default/trust/windowstransport",
	} {
		rec := postWsTrustSOAPTo(e, path, wsTrustRST(time.Now().UTC(), "urn:uuid:windows-"+path, "urn:idmagic:demo-rp"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404: no active Windows/Kerberos endpoint is provided", path, rec.Code)
		}
		assertNoWsTrustTokenIssued(t, rec, *events, 0)
	}

	// 広告も無い。MEX は UsernameMixed の binding だけを載せる。
	for _, target := range []string{"/trust/mex", "/federationmetadata/2007-06/federationmetadata.xml"} {
		rec := get(e, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d", target, rec.Code)
		}
		for _, forbidden := range []string{"WindowsTransport", "Kerberos", "windowstransport", "kerberosmixed"} {
			if strings.Contains(rec.Body.String(), forbidden) {
				t.Fatalf("%s advertises %q although the active Windows/Kerberos profile is not provided:\n%s",
					target, forbidden, rec.Body.String())
			}
		}
	}
}

func TestWsTrustUsernameMixed_RejectsNonBearerKeyType(t *testing.T) {
	e, _ := newServer(t, nil)
	body := strings.Replace(
		wsTrustRST(time.Now().UTC(), "urn:uuid:public-key", "urn:idmagic:demo-rp"),
		"<wsp:AppliesTo>",
		"<t:KeyType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/PublicKey</t:KeyType><wsp:AppliesTo>",
		1,
	)
	if rec := postWsTrustSOAP(e, body); rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func postWsTrustSOAP(e *echo.Echo, body string) *httptest.ResponseRecorder {
	return postWsTrustSOAPTo(e, "/realms/default/trust/usernamemixed", body)
}

// postWsTrustSOAPTo は宛先を明示して RST を POST する。提供しない能動的プロファイルの入口が存在
// しないことを確かめるために、usernamemixed 以外へも送れる必要がある。
func postWsTrustSOAPTo(e *echo.Echo, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/soap+xml")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func wsTrustRST(now time.Time, messageID, appliesTo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"
  xmlns:a="http://www.w3.org/2005/08/addressing"
  xmlns:o="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
  xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
  xmlns:t="http://docs.oasis-open.org/ws-sx/ws-trust/200512"
  xmlns:wsp="http://schemas.xmlsoap.org/ws/2004/09/policy">
  <s:Header>
    <a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</a:Action>
    <a:MessageID>` + messageID + `</a:MessageID>
    <a:To>https://idp.example/realms/default/trust/usernamemixed</a:To>
    <o:Security>
      <u:Timestamp>
        <u:Created>` + now.Format(time.RFC3339) + `</u:Created>
        <u:Expires>` + now.Add(5*time.Minute).Format(time.RFC3339) + `</u:Expires>
      </u:Timestamp>
      <o:UsernameToken>
        <o:Username>alice</o:Username>
        <o:Password>correct-password</o:Password>
      </o:UsernameToken>
    </o:Security>
  </s:Header>
  <s:Body>
    <t:RequestSecurityToken>
      <t:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</t:RequestType>
      <wsp:AppliesTo>
        <a:EndpointReference>
          <a:Address>` + appliesTo + `</a:Address>
        </a:EndpointReference>
      </wsp:AppliesTo>
    </t:RequestSecurityToken>
  </s:Body>
</s:Envelope>`
}

func newAdminServer(t *testing.T) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	objectGUID := "6f9619ff-8b86-d011-b42d-00c04fc964ff"
	userRepo.Seed(&userdomain.User{
		ID:                "admin-1",
		TenantID:          tenancydomain.DefaultTenantID,
		PreferredUsername: "admin@contoso.com",
		Roles:             []string{"admin"},
		Attributes: map[string]userdomain.AttributeValue{
			"object_guid": {Type: idmdomain.AttributeTypeString, String: &objectGUID},
		},
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:   "https://idp.example",
		Contract: spec.CurrentRuntimeContract(), WsFederation: wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		UserRepo:      userRepo,
		AuthnResolver: stubResolver{ctx: &authdomain.AuthenticationContext{UserID: "admin-1"}},
	})
	return e
}

func doJSON(e *echo.Echo, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, defaultRealmPath(target), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAdminRelyingParty_CRUD(t *testing.T) {
	e := newAdminServer(t)
	const path = "/api/admin/v1/wsfed/relying-parties"
	body := `{"wtrealm":"urn:rp:a","reply_urls":["https://a.example/acs"],"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`

	if rec := doJSON(e, http.MethodPost, path, body); rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(e, http.MethodPost, path, body); rec.Code != http.StatusOK {
		t.Fatalf("update status=%d, want 200", rec.Code)
	}
	if rec := get(e, path); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "urn:rp:a") {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(e, http.MethodDelete, path+"?wtrealm=urn:rp:a", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d, want 204", rec.Code)
	}
	if rec := get(e, path); strings.Contains(rec.Body.String(), "urn:rp:a") {
		t.Fatalf("RP still present after delete: %s", rec.Body.String())
	}
}

func TestAdminRelyingParty_RejectsInvalid(t *testing.T) {
	e := newAdminServer(t)
	// reply_urls 欠落。
	body := `{"wtrealm":"urn:rp:b","claim_policy":{"name_id":{"format":"f","source_attribute":"user_id"}}}`
	rec := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/relying-parties", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != support.ProblemContentType {
		t.Fatalf("Content-Type=%q, want %q", contentType, support.ProblemContentType)
	}
	if !strings.Contains(rec.Body.String(), "urn:idmagic:error:invalid_request") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
}

func TestAdminRelyingParty_ForbiddenForNonAdmin(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"}) // 非 admin
	if rec := get(e, "/api/admin/v1/wsfed/relying-parties"); rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestAdminConfigureEntraFederation_CreatesPresetRelyingParty(t *testing.T) {
	e := newAdminServer(t)
	body := `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`
	rec := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/entra-federation", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, want := range []string{
		`"issuer_uri":"urn:idmagic:entra:contoso.com"`,
		`"source_anchor_attribute":"object_guid"`,
		`"claim_type":"http://schemas.xmlsoap.org/claims/UPN"`,
		`"claim_type":"http://schemas.xmlsoap.org/claims/nameidentifier"`,
		`"ActiveLogOnUri":"https://idp.example/realms/default/trust/usernamemixed"`,
		"Hybrid Azure AD Join device registration is not provided",
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("response missing %q:\n%s", want, rec.Body.String())
		}
	}
	list := get(e, "/api/admin/v1/wsfed/relying-parties")
	if !strings.Contains(list.Body.String(), `"entra_profile"`) {
		t.Fatalf("configured RP missing entra_profile: %s", list.Body.String())
	}
}

func TestAdminConfigureEntraFederation_RejectsMissingSourceAnchor(t *testing.T) {
	e := newAdminServer(t)
	body := `{"domain":"contoso.com","source_anchor_attribute":"missing_anchor"}`
	rec := doJSON(e, http.MethodPost, "/api/admin/v1/wsfed/entra-federation", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sourceAnchor validation failed") {
		t.Fatalf("missing sourceAnchor error not returned: %s", rec.Body.String())
	}
}

func mustUnescape(t *testing.T, s string) string {
	t.Helper()
	out, err := url.QueryUnescape(s)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}
	return out
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す。
// bare path はどのテナントの正規ロケーションでもなくなったため、
// テストのリクエスト先も /realms/default 配下でなければ 404 になる。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}
