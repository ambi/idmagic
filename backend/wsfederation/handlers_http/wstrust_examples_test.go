package handlers_http_test

// docs/domain/ws-federation/scenarios.feature.md の REQ-WSFEDERATION-004 と REQ-WSFEDERATION-005 が
// 宣言する具体例を、能動 STS の入口 `/trust/usernamemixed` から観測する。

import (
	"crypto/x509"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/requests_wstrust"

	"github.com/beevik/etree"
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
			old: "<a:Action>" + wstrust.RequestIssue + "</a:Action>", replacement: "<a:Action>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Renew</a:Action>",
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
func TestWsTrustIssue_ValidatesEveryRequiredElementAndAnswersAValidRST(t *testing.T) {
	t.Run("a valid RST receives a signed RSTR for the authenticated user", func(t *testing.T) {
		e, events, signer := newServerWithSigner(t, nil)
		const messageID = "urn:uuid:valid-rst"
		rec := postWsTrustSOAP(e, wsTrustRST(time.Now().UTC(), messageID, "urn:idmagic:demo-rp"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
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
