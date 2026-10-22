package handlers_http_test

// docs/modules/ws-federation/features/active-sts.md の REQ-WSFEDERATION-004 と REQ-WSFEDERATION-005 が
// 宣言する具体例を、能動 STS の入口 `/trust/usernamemixed` から観測する。

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/authentication"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/internal/requests_wstrust"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	wsTrustValidTo  = "https://idp.example/realms/default/trust/usernamemixed"
	brokenMessageID = "urn:uuid:broken"
)

// brokenRST は、有効な RST から要素を 1 つだけ崩した要求である。崩すのは 1 度に 1 つだけなので、
// 通らなかった理由はその要素に限られる。envelope は REQ-WSFEDERATION-005 が名指す
// エンベロープの要素 (To、MessageID、AppliesTo、Action、RequestType、KeyType) であることを表す。
// 崩し方は、AppliesTo の値、Timestamp のずれ、本文の 1 か所の置換のどれか 1 つである。
type brokenRST struct {
	name             string
	envelope         bool
	appliesTo        string
	shift            time.Duration
	old, replacement string
}

func (b brokenRST) body(t *testing.T) string {
	t.Helper()
	out := wsTrustRST(time.Now().UTC().Add(b.shift), brokenMessageID, b.appliesTo)
	if b.old == "" {
		return out
	}
	if !strings.Contains(out, b.old) {
		t.Fatalf("崩す対象 %q が RST に無い", b.old)
	}
	return strings.Replace(out, b.old, b.replacement, 1)
}

func brokenRSTs() []brokenRST {
	const rp = "urn:idmagic:demo-rp"
	return []brokenRST{
		{
			name: "To outside the active STS endpoint", envelope: true, appliesTo: rp,
			old: "<a:To>" + wsTrustValidTo + "</a:To>", replacement: "<a:To>https://evil.example/trust/usernamemixed</a:To>",
		},
		{
			name: "MessageID missing", envelope: true, appliesTo: rp,
			old: "<a:MessageID>" + brokenMessageID + "</a:MessageID>",
		},
		{name: "AppliesTo not registered", envelope: true, appliesTo: "urn:not-registered"},
		{name: "AppliesTo missing", envelope: true, appliesTo: ""},
		{
			name: "Action other than Issue", envelope: true, appliesTo: rp,
			old: "<a:Action>" + wstrust.ActionIssue + "</a:Action>", replacement: "<a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Renew</a:Action>",
		},
		{
			name: "RequestType other than Issue", envelope: true, appliesTo: rp,
			old: "<t:RequestType>" + wstrust.RequestIssue + "</t:RequestType>", replacement: "<t:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Renew</t:RequestType>",
		},
		{
			name: "KeyType other than Bearer", envelope: true, appliesTo: rp,
			old: "<wsp:AppliesTo>", replacement: "<t:KeyType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/PublicKey</t:KeyType><wsp:AppliesTo>",
		},
		{name: "UsernameToken Username missing", appliesTo: rp, old: "<o:Username>alice</o:Username>"},
		{name: "UsernameToken Password missing", appliesTo: rp, old: "<o:Password>correct-password</o:Password>"},
		{name: "Timestamp expired", appliesTo: rp, shift: -10 * time.Minute},
		{name: "Timestamp created in the future", appliesTo: rp, shift: 10 * time.Minute},
	}
}

// issuedTokenEvent は捕捉した WsTrustTokenIssued のうち最初のものを返す。
func issuedTokenEvent(events []spec.DomainEvent) *feddomain.WsTrustTokenIssued {
	for _, event := range events {
		if issued, ok := event.(*feddomain.WsTrustTokenIssued); ok {
			return issued
		}
	}
	return nil
}

// rejectedTokenReasons は捕捉した WsTrustTokenRejected の理由を発行順に返す。
func rejectedTokenReasons(events []spec.DomainEvent) []string {
	var reasons []string
	for _, event := range events {
		if rejected, ok := event.(*feddomain.WsTrustTokenRejected); ok {
			reasons = append(reasons, rejected.Reason)
		}
	}
	return reasons
}

// 1 つ目の Then は「すべて検証する」なので、成功経路だけでは観測できない。何も読まずに発行する
// 実装も成功経路は通すからである。Given が挙げる要素を 1 つずつ崩し、そのどれでもトークンが
// 出ないことを読む。2 つ目の Then は RSTR の外形だけでは足りない。この要求に答えていること
// (RelatesTo と AppliesTo)、IdP の証明書で検証できる署名済みの assertion を運ぶこと、
// UsernameToken が名乗った利用者について発行したことまで読む。
//
//spec:covers EX-WSFEDERATION-004-01: UsernameToken、MessageID、Timestamp、To、Action、RequestType、KeyType、AppliesTo のどれか 1 つを崩すとトークンが出ず、すべて有効な RST には、RelatesTo が MessageID を指し AppliesTo を返し IdP の証明書で検証できる alice の assertion を運ぶ RSTR を返して WsTrustTokenIssued を発行すること。
//verify:operation WsTrustIssue: SOAP 要求に対する 200 の SOAP 本文から RelatesTo、AppliesTo、利用者のクレームと Assertion の署名を照合する。
func TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST(t *testing.T) {
	t.Run("a valid RST receives a signed RSTR for the authenticated user", func(t *testing.T) {
		e, events, signer := newServerWithSigner(t, nil)
		const messageID = "urn:uuid:valid-rst"
		rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), messageID, "urn:idmagic:demo-rp"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Type") != "application/soap+xml; charset=utf-8" {
			t.Fatalf("Content-Type=%q, want SOAP XML UTF-8", rec.Header().Get("Content-Type"))
		}
		document := etree.NewDocument()
		if err := document.ReadFromString(rec.Body.String()); err != nil {
			t.Fatalf("parse RSTR: %v", err)
		}
		if got := document.FindElement("//a:RelatesTo"); got == nil || got.Text() != messageID {
			t.Fatalf("RSTR does not relate to the request's MessageID: %+v", got)
		}
		if got := document.FindElement("//t:RequestSecurityTokenResponse/wsp:AppliesTo/a:EndpointReference/a:Address"); got == nil || got.Text() != "urn:idmagic:demo-rp" {
			t.Fatalf("RSTR AppliesTo = %+v, want urn:idmagic:demo-rp", got)
		}
		assertion := document.FindElement("//t:RequestedSecurityToken/Assertion")
		if assertion == nil {
			t.Fatalf("RequestedSecurityToken does not carry an assertion: %s", rec.Body.String())
		}
		store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{signer.Certificate()}}
		validation := dsig.NewDefaultValidationContext(store)
		validation.IdAttribute = idAttributeForAssertion(assertion)
		if _, err := validation.Validate(assertion); err != nil {
			t.Fatalf("assertion signature did not validate against the IdP certificate: %v", err)
		}
		// RP のクレームポリシーは UPN を preferred_username から出す。UsernameToken が名乗った
		// alice の値が載っていることが、名乗った利用者について発行した証拠になる。
		carriesAlice := false
		for _, value := range assertion.FindElements(".//AttributeValue") {
			if value.Text() == "alice" {
				carriesAlice = true
			}
		}
		if !carriesAlice {
			t.Fatalf("the assertion does not carry the authenticated user's UPN: %s", rec.Body.String())
		}
		issued := issuedTokenEvent(*events)
		if issued == nil || issued.UserID != "user-1" || issued.AppliesTo != "urn:idmagic:demo-rp" {
			t.Fatalf("WsTrustTokenIssued = %+v, want user-1 for urn:idmagic:demo-rp", issued)
		}
	})

	for _, tc := range brokenRSTs() {
		t.Run(tc.name+" receives no token", func(t *testing.T) {
			e, events := newServer(t, nil)
			rec := postWsTrustSOAP(e, tc.body(t))
			if rec.Code < 400 {
				t.Fatalf("status=%d body=%s, want a refusal", rec.Code, rec.Body.String())
			}
			assertNoWsTrustTokenIssued(t, rec, *events, 0)
		})
	}
}

// 同じ MessageID の 2 度目が通らないことに加えて、拒否が WsTrustTokenRejected として記録され、
// その理由が MessageID の再利用であることを読む。対照として、同じサーバーへ新しい MessageID で
// 送った要求は通る。これが無いと、1 度発行したあと何も通さなくなる実装と区別できない。
//
//spec:covers EX-WSFEDERATION-004-02: 発行済みの MessageID を assertion の有効期間内に再び使った RST が、400 と MessageID の再利用を理由とする WsTrustTokenRejected で拒否され、2 通目のトークンが出ないこと。新しい MessageID なら同じサーバーで通ることを対照にする。
func TestWsTrustIssue_RejectsAReusedMessageIDWithinTheAssertionLifetime(t *testing.T) {
	e, events := newServer(t, nil)
	const messageID = "urn:uuid:reused"
	first := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), messageID, "urn:idmagic:demo-rp"))
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}

	replayed := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), messageID, "urn:idmagic:demo-rp"))
	if replayed.Code != http.StatusBadRequest {
		t.Fatalf("replay status=%d body=%s, want 400", replayed.Code, replayed.Body.String())
	}
	assertNoWsTrustTokenIssued(t, replayed, *events, 1)
	if reasons := rejectedTokenReasons(*events); len(reasons) != 1 || reasons[0] != "replayed MessageID" {
		t.Fatalf("WsTrustTokenRejected reasons = %v, want one replayed MessageID", reasons)
	}

	fresh := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), "urn:uuid:fresh", "urn:idmagic:demo-rp"))
	if fresh.Code != http.StatusOK {
		t.Fatalf("control status=%d body=%s: a new MessageID must still be accepted", fresh.Code, fresh.Body.String())
	}
}

// エンベロープの 6 要素を 1 つずつ崩す。どれについても、拒否の状態コードと、WsTrustTokenRejected が
// 出ていることと、トークンが出ていないことの 3 つを読む。どれか 1 つだけを崩すテストは、
// その要素だけを検査する実装を通してしまう。崩していない要求が通ることは
// TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST の成功経路が示す。
//
//spec:covers EX-WSFEDERATION-005-01: RST の To、MessageID、AppliesTo、Action、RequestType、KeyType のどれか 1 つが不正な要求が、400 または 401 と WsTrustTokenRejected で拒否され、応答が RSTR も assertion も運ばず WsTrustTokenIssued も出ないこと。
//verify:operation WsTrustIssue: 不正な SOAP エンベロープの 400 または 401 は空でない text/plain 本文を返し、トークンを含めない。
func TestWsTrustIssue_RejectsAMalformedEnvelope(t *testing.T) {
	for _, tc := range brokenRSTs() {
		if !tc.envelope {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			e, events := newServer(t, nil)
			rec := postWsTrustSOAP(e, tc.body(t))
			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s, want 400 or 401", rec.Code, rec.Body.String())
			}
			if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") || strings.TrimSpace(rec.Body.String()) == "" {
				t.Fatalf("Content-Type=%q body=%q, want nonempty text refusal", rec.Header().Get("Content-Type"), rec.Body.String())
			}
			if !hasEvent(*events, "WsTrustTokenRejected") {
				t.Fatal("WsTrustTokenRejected not emitted")
			}
			assertNoWsTrustTokenIssued(t, rec, *events, 0)
		})
	}
}

// 誤ったパスワードと未知の username の 2 通りを通す。どちらも、拒否の状態コードに加えて、応答が
// RSTR も assertion も運ばず WsTrustTokenIssued も出ていないことと、拒否が WsTrustTokenRejected として
// 記録されたことを読む。401 だけを見ると、401 を書いてから発行する実装を見分けられない。対照として、
// 同じ組み立ての正しい資格情報は発行される。これが無いと、何も発行しないスタックでも上の 2 通りは緑になる。
//
//spec:covers EX-WSFEDERATION-004-03: UsernameToken のパスワードが誤っているか username が未知の RST が、401 と WsTrustTokenRejected で拒否され、応答が RSTR も assertion も運ばず WsTrustTokenIssued も出ないこと。正しい資格情報なら同じ入口で発行されることを対照にする。
//verify:operation WsTrustIssue: 不正な資格情報の 401 は空でない text/plain 本文を返し、RSTR も Assertion も含めない。
func TestWsTrustIssue_RefusesAWrongCredentialWith401AndNoToken(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"wrong password", "<o:Password>correct-password</o:Password>", "<o:Password>wrong-password</o:Password>"},
		{"unknown username", "<o:Username>alice</o:Username>", "<o:Username>mallory</o:Username>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, events := newServer(t, nil)
			refused := brokenRST{appliesTo: "urn:idmagic:demo-rp", old: tc.old, replacement: tc.replacement}
			rec := postWsTrustSOAP(e, refused.body(t))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s, want 401", rec.Code, rec.Body.String())
			}
			if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") || strings.TrimSpace(rec.Body.String()) == "" {
				t.Fatalf("Content-Type=%q body=%q, want nonempty credential refusal", rec.Header().Get("Content-Type"), rec.Body.String())
			}
			assertNoWsTrustTokenIssued(t, rec, *events, 0)
			if reasons := rejectedTokenReasons(*events); len(reasons) != 1 {
				t.Fatalf("WsTrustTokenRejected reasons = %v, want exactly one", reasons)
			}
		})
	}

	t.Run("control: the correct credential is issued", func(t *testing.T) {
		e, events := newServer(t, nil)
		rec := postWsTrustSOAP(e, brokenRST{appliesTo: "urn:idmagic:demo-rp"}.body(t))
		if rec.Code != http.StatusOK || issuedTokenEvent(*events) == nil {
			t.Fatalf("control status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

// requiringMFA は behindAnApplication の Application に、MFA を求めるサインインポリシーを足す。
// behindAnApplication の後に適用する。
func requiringMFA(t *testing.T) func(*httpadapter.Deps) {
	t.Helper()
	policies := application.NewMemoryModule().SignInPolicyRepo
	if err := policies.Save(context.Background(), &appdomain.AppSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "wsfed-app", UpdatedAt: time.Now().UTC(),
		Rules: []appdomain.SignInRule{{RuleID: "rule-1", Name: "MFA", Enabled: true, RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa}}},
	}); err != nil {
		t.Fatal(err)
	}
	return func(deps *httpadapter.Deps) {
		deps.Application.SignInPolicyRepo = policies
	}
}

// assertRefusedByTheApplicationGate は、関門の拒否が 403 の text/plain で、トークンを運ばず、
// 拒否の理由を WsTrustTokenRejected とアプリケーションのイベントに残したことを読む。
func assertRefusedByTheApplicationGate(t *testing.T, rec *httptest.ResponseRecorder, events []spec.DomainEvent) {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") || strings.TrimSpace(rec.Body.String()) == "" {
		t.Fatalf("Content-Type=%q body=%q, want nonempty text/plain refusal", rec.Header().Get("Content-Type"), rec.Body.String())
	}
	assertNoWsTrustTokenIssued(t, rec, events, 0)
	if reasons := rejectedTokenReasons(events); len(reasons) != 1 {
		t.Fatalf("WsTrustTokenRejected reasons = %v, want exactly one", reasons)
	}
	if !hasEvent(events, "AppAccessDeniedByPolicy") {
		t.Fatal("AppAccessDeniedByPolicy was not emitted")
	}
}

// 割り当てのない利用者だけを拒否することは、拒否の側だけでは観測できない。何も発行しない実装も
// 拒否の側は通すからである。同じ RP に割り当てのある利用者が発行されることを対照に置く。
//
//spec:covers REQ-APPLICATION-011: アプリケーションに関連付けた RP へ、割り当てのない利用者が正しい UsernameToken で RST を送ると、403 で拒否され、トークンも WsTrustTokenIssued も出ず、拒否の理由が残ること。
//verify:operation WsTrustIssue: アプリケーションの関門が拒否したときの 403 と、空でない text/plain の本文を照合する。
func TestWsTrustIssue_RefusesAUserTheApplicationDoesNotAssign(t *testing.T) {
	e, events, _ := newServerWithSigner(t, nil, behindAnApplication(t, false))
	rec := postWsTrustSOAP(e, brokenRST{appliesTo: passiveRealm}.body(t))
	assertRefusedByTheApplicationGate(t, rec, *events)

	t.Run("control: the assigned user is issued", func(t *testing.T) {
		e, events, _ := newServerWithSigner(t, nil, behindAnApplication(t, true))
		rec := postWsTrustSOAP(e, brokenRST{appliesTo: passiveRealm}.body(t))
		if rec.Code != http.StatusOK || issuedTokenEvent(*events) == nil {
			t.Fatalf("control status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

//spec:covers REQ-WSFEDERATION-004: 割り当てのある利用者でも、アプリケーションのサインインポリシーが MFA を求めると、パスワードだけの UsernameToken は 403 で拒否され、トークンが出ず、AppStepUpRequired が残ること。
func TestWsTrustIssue_RefusesWhenTheSignInPolicyRequiresMFA(t *testing.T) {
	e, events, _ := newServerWithSigner(t, nil, behindAnApplication(t, true), requiringMFA(t))
	rec := postWsTrustSOAP(e, brokenRST{appliesTo: passiveRealm}.body(t))
	assertRefusedByTheApplicationGate(t, rec, *events)
	if !hasEvent(*events, "AppStepUpRequired") {
		t.Fatal("AppStepUpRequired was not emitted")
	}
}

// usernameTokenRST は、MessageID と UsernameToken を指定した有効な RST を作る。同じサーバーへ
// 何度も送るので、MessageID の再利用の拒否に当たらないよう要求ごとに変える。
func usernameTokenRST(t *testing.T, messageID, username, password string) string {
	t.Helper()
	out := wsTrustRST(time.Now().UTC(), messageID, passiveRealm)
	for old, replacement := range map[string]string{
		"<o:Username>alice</o:Username>":            "<o:Username>" + username + "</o:Username>",
		"<o:Password>correct-password</o:Password>": "<o:Password>" + password + "</o:Password>",
	} {
		if !strings.Contains(out, old) {
			t.Fatalf("RST に %q が無い", old)
		}
		out = strings.Replace(out, old, replacement, 1)
	}
	return out
}

// postWsTrustSOAPFrom は、信頼する転送ホップの外側の発信元を X-Forwarded-For で名乗って RST を送る。
func postWsTrustSOAPFrom(e *echo.Echo, body, clientIP string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/realms/default/trust/usernamemixed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/soap+xml")
	req.Header.Set("X-Forwarded-For", clientIP+", 10.0.0.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// withLoginThrottle は、ログインの API と同じ試行の制限を、IP 単位の閾値を小さくして組み立てる。
func withLoginThrottle(ipMaxFailures int) func(*httpadapter.Deps) {
	return func(deps *httpadapter.Deps) {
		deps.TrustedForwardedHops = 1
		deps.Authentication.LoginAttemptThrottle = authentication.NewMemoryModule(nil).NewLoginAttemptThrottle(sessionports.LoginThrottleConfigs{
			Account: sessionports.LoginThrottleConfig{MaxFailures: 100, WindowSeconds: 900, LockoutSeconds: 900},
			IP:      sessionports.LoginThrottleConfig{MaxFailures: ipMaxFailures, WindowSeconds: 900, LockoutSeconds: 900},
		})
	}
}

// IP 単位の制限は、一つのアカウントへの失敗では観測できない。アカウント単位の制限でも同じ拒否に
// なるからである。別々のユーザー名で同じ発信元から失敗を重ね、まだ失敗していない alice の正しい
// 資格情報が拒否されることを読む。対照として、別の発信元の alice は発行される。
//
//spec:covers REQ-WSFEDERATION-004: UsernameToken の照合がログインの API と同じ IP 単位の試行の制限と失敗の監査を使い、同じ発信元の失敗が閾値に達すると、別のアカウントの正しい資格情報も 401 で拒否されてトークンが出ず、失敗ごとに AuthenticationFailed が残ること。
func TestWsTrustIssue_SharesTheLoginThrottleAndFailureAudit(t *testing.T) {
	const attacker = "203.0.113.9"
	e, events, _ := newServerWithSigner(t, nil, withLoginThrottle(3))
	for i, username := range []string{"mallory", "eve", "trudy"} {
		rec := postWsTrustSOAPFrom(e, usernameTokenRST(t, "urn:uuid:spray-"+username, username, "guess"), attacker)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("spray #%d status=%d body=%s, want 401", i, rec.Code, rec.Body.String())
		}
	}
	if got := countEvents(*events, "AuthenticationFailed"); got != 3 {
		t.Fatalf("AuthenticationFailed count = %d, want 3 (one per refused UsernameToken)", got)
	}

	rec := postWsTrustSOAPFrom(e, usernameTokenRST(t, "urn:uuid:alice-throttled", "alice", "correct-password"), attacker)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401 for a throttled source", rec.Code, rec.Body.String())
	}
	assertNoWsTrustTokenIssued(t, rec, *events, 0)

	t.Run("control: another source is issued", func(t *testing.T) {
		rec := postWsTrustSOAPFrom(e, usernameTokenRST(t, "urn:uuid:alice-elsewhere", "alice", "correct-password"), "198.51.100.7")
		if rec.Code != http.StatusOK || issuedTokenEvent(*events) == nil {
			t.Fatalf("control status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}

// requiringPasswordChange は、fixture の alice に必須操作 update_password を付ける。
func requiringPasswordChange(t *testing.T) func(*httpadapter.Deps) {
	t.Helper()
	return func(deps *httpadapter.Deps) {
		ctx := context.Background()
		user, err := deps.UserRepo.FindBySub(ctx, "user-1")
		if err != nil || user == nil {
			t.Fatalf("fixture user: %v", err)
		}
		user.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}
		if err := deps.UserRepo.Save(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
}

//spec:covers REQ-WSFEDERATION-004: パスワードの変更を求められている User の正しい UsernameToken は 403 で拒否され、トークンが出ず、WsTrustTokenRejected が残ること。
func TestWsTrustIssue_RefusesAUserWhosePasswordMustChange(t *testing.T) {
	e, events, _ := newServerWithSigner(t, nil, requiringPasswordChange(t))
	rec := postWsTrustSOAP(e, brokenRST{appliesTo: passiveRealm}.body(t))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") || strings.TrimSpace(rec.Body.String()) == "" {
		t.Fatalf("Content-Type=%q body=%q, want nonempty text/plain refusal", rec.Header().Get("Content-Type"), rec.Body.String())
	}
	assertNoWsTrustTokenIssued(t, rec, *events, 0)
	if reasons := rejectedTokenReasons(*events); len(reasons) != 1 {
		t.Fatalf("WsTrustTokenRejected reasons = %v, want exactly one", reasons)
	}
}
