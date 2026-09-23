package handlers_http_test

// docs/domain/ws-federation/scenarios.feature.md の REQ-WSFEDERATION-002 と REQ-WSFEDERATION-003 が
// 宣言する具体例を、パッシブの入口 `/wsfed` から観測する。

import (
	"context"
	"crypto/x509"
	"html"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

const (
	passiveRealm        = "urn:idmagic:demo-rp"
	passiveAllowedReply = "https://rp.example/wsfed/alternate"
)

// behindAnApplication は fixture の RP を Application の WS-Fed binding に属させる。
// 割り当てのない利用者へ発行してはならないという判定は、Application が存在するときだけ効く。
// Application を持たない組み立てでは、この分岐そのものが経路に無い。
func behindAnApplication(t *testing.T, assigned bool) func(*httpadapter.Deps) {
	t.Helper()
	now := time.Now().UTC()
	applications := appmemory.NewApplicationRepository()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: "wsfed-app", Name: "WS-Fed RP",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now,
		Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolWsFed, Wtrealm: passiveRealm},
	}
	if err := applications.Save(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	assignments := appmemory.NewApplicationAssignmentRepository()
	if assigned {
		if err := assignments.Save(context.Background(), &appdomain.ApplicationAssignment{
			TenantID: tenancydomain.DefaultTenantID, ApplicationID: app.ID,
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user-1",
			Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return func(deps *httpadapter.Deps) {
		deps.Application = application.Module{Repo: applications, AssignmentRepo: assignments}
	}
}

func authenticatedAt(authTime time.Time) *authdomain.AuthenticationContext {
	return &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: authTime.Unix(), AMR: []string{"pwd"}}
}

func passiveSignInURL(wtrealm, wreply string, extra url.Values) string {
	query := url.Values{"wa": {"wsignin1.0"}, "wtrealm": {wtrealm}, "wreply": {wreply}}
	maps.Copy(query, extra)
	return "/wsfed?" + query.Encode()
}

// hiddenInputValue は自動 POST フォームから、名指した hidden input の値を属性の escape を解いて返す。
func hiddenInputValue(t *testing.T, body, name string) string {
	t.Helper()
	marker := `name="` + name + `" value="`
	_, after, ok := strings.Cut(body, marker)
	if !ok {
		t.Fatalf("hidden input %q missing: %s", name, body)
	}
	end := strings.Index(after, `"`)
	if end < 0 {
		t.Fatalf("hidden input %q is not terminated: %s", name, body)
	}
	return html.UnescapeString(after[:end])
}

// brokenPassive は、発行される要求から 1 つの次元だけを崩したパッシブサインインである。
// 崩すのは 1 度に 1 つだけなので、発行されない理由はその次元に限られる。
type brokenPassive struct {
	name string
	// target は /wsfed への要求。assigned と authTime は、その要求を受けるサーバーの状態である。
	target   string
	assigned bool
	authTime time.Time
}

func unregisteredRealm() brokenPassive {
	// wreply は登録済み RP の許可集合に属する値なので、崩れているのは wtrealm だけである。
	return brokenPassive{name: "unregistered wtrealm", target: passiveSignInURL("urn:not-registered", passiveAllowedReply, nil), assigned: true, authTime: time.Now()}
}

func disallowedReply() brokenPassive {
	return brokenPassive{name: "wreply outside the allowed set", target: passiveSignInURL(passiveRealm, "https://evil.example/steal", nil), assigned: true, authTime: time.Now()}
}

func unsupportedWauth() brokenPassive {
	return brokenPassive{
		name:     "wauth the IdP cannot satisfy",
		target:   passiveSignInURL(passiveRealm, passiveAllowedReply, url.Values{"wauth": {"urn:federation:authentication:windows"}}),
		assigned: true, authTime: time.Now(),
	}
}

func unassignedSubject() brokenPassive {
	return brokenPassive{name: "subject not assigned to the application", target: passiveSignInURL(passiveRealm, passiveAllowedReply, nil), assigned: false, authTime: time.Now()}
}

func staleAuthentication() brokenPassive {
	return brokenPassive{
		name:     "authentication older than wfresh",
		target:   passiveSignInURL(passiveRealm, passiveAllowedReply, url.Values{"wfresh": {"5"}}),
		assigned: true, authTime: time.Now().Add(-30 * time.Minute),
	}
}

func (b brokenPassive) serve(t *testing.T) (*httptest.ResponseRecorder, []spec.DomainEvent) {
	t.Helper()
	e, events, _ := newServerWithSigner(t, authenticatedAt(b.authTime), behindAnApplication(t, b.assigned))
	return get(e, b.target), *events
}

// 1 つ目の Then は「検証する」なので、成功経路だけでは観測できない。何も照合しない実装も成功経路は
// 通すからである。Given が挙げる wtrealm、wreply、wfresh、Application の割り当てを 1 つずつ崩し、
// そのどれでもトークンが出ないことを読む。2 つ目の Then は、RP が受け取る自動 POST フォームを
// RP と同じ手順で読む。宛先が指定した許可済みの wreply であること、wresult の RSTR が wtrealm に
// 答え、IdP の証明書で検証できる署名済みの assertion を運ぶこと、wctx が escape を解いたうえで
// 送った値と一致することである。wctx には escape を要する文字を入れ、値の往復を読めるようにする。
//
//spec:covers EX-WSFEDERATION-002-01: 登録済みの wtrealm、許可済みの wreply、wfresh 以内の認証、割り当て済みの利用者のどれか 1 つを崩すとトークンが出ず、すべて満たす wsignin1.0 には、指定した wreply へ送る自動 POST フォームで、wtrealm を AppliesTo に持ち IdP の証明書で検証できる署名済み assertion を運ぶ RSTR と、送ったのと同じ wctx を返すこと。
func TestWsFedPassiveSignIn_ValidatesTheRequestAndReturnsASignedRSTRForm(t *testing.T) {
	t.Run("a valid wsignin1.0 receives a signed RSTR form carrying wctx back", func(t *testing.T) {
		const wctx = `rm=0&id=passive&ru=/app?x="1"`
		e, events, signer := newServerWithSigner(t, authenticatedAt(time.Now()), behindAnApplication(t, true))
		rec := get(e, passiveSignInURL(passiveRealm, passiveAllowedReply, url.Values{"wfresh": {"5"}, "wctx": {wctx}}))
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, `action="`+passiveAllowedReply+`"`) {
			t.Fatalf("the form is not posted to the requested allowed wreply: %s", body)
		}
		if got := hiddenInputValue(t, body, "wctx"); got != wctx {
			t.Fatalf("wctx = %q, want the value sent in the request %q", got, wctx)
		}
		document := etree.NewDocument()
		if err := document.ReadFromString(hiddenInputValue(t, body, "wresult")); err != nil {
			t.Fatalf("parse wresult: %v", err)
		}
		if got := document.FindElement("//wsp:AppliesTo//wsa:Address"); got == nil || got.Text() != passiveRealm {
			t.Fatalf("RSTR AppliesTo = %+v, want %s", got, passiveRealm)
		}
		assertion := document.FindElement("//t:RequestedSecurityToken/Assertion")
		if assertion == nil {
			t.Fatalf("RSTR does not carry an assertion: %s", body)
		}
		validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{signer.Certificate()}})
		validation.IdAttribute = idAttributeForAssertion(assertion)
		if _, err := validation.Validate(assertion); err != nil {
			t.Fatalf("assertion signature did not validate against the IdP certificate: %v", err)
		}
		if !hasEvent(*events, "WsFedSignInIssued") {
			t.Fatal("WsFedSignInIssued not emitted")
		}
	})

	for _, tc := range []brokenPassive{unregisteredRealm(), disallowedReply(), staleAuthentication(), unassignedSubject()} {
		t.Run(tc.name+" receives no token", func(t *testing.T) {
			rec, events := tc.serve(t)
			assertNoPassiveTokenIssued(t, rec, events)
		})
	}
}

// 再認証へ誘導したことは、ログイン画面への 303 と、戻り先が元の要求であることで読む。トークンを
// 出していないことは、応答本文と WsFedSignInIssued の双方で読む。対照として、同じ wfresh でも
// 認証が新しければ同じ入口から発行される。これが無いと、wfresh を付けた要求をすべて再認証へ
// 回す実装と区別できない。
//
//spec:covers EX-WSFEDERATION-002-02: 30 分前の認証に wfresh=5 を付けた wsignin1.0 が、元の要求を return_to に持つログイン画面への 303 になり、応答が wresult も assertion も運ばず WsFedSignInIssued も出ないこと。同じ wfresh で認証が新しければ発行されることを対照にする。
func TestWsFedPassiveSignIn_StaleAuthenticationIsSentToReauthenticateWithoutAToken(t *testing.T) {
	stale := staleAuthentication()
	rec, events := stale.serve(t)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s, want 303", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/realms/default/login") {
		t.Fatalf("Location = %q, want the login screen", location)
	}
	returnTo, err := url.Parse(location)
	if err != nil || !strings.Contains(returnTo.Query().Get("return_to"), "wfresh=5") {
		t.Fatalf("return_to does not point back to the original request: %q", location)
	}
	assertNoPassiveTokenIssued(t, rec, events)

	fresh := stale
	fresh.authTime = time.Now()
	control, controlEvents := fresh.serve(t)
	if control.Code != http.StatusOK || !hasEvent(controlEvents, "WsFedSignInIssued") {
		t.Fatalf("control status=%d: a fresh authentication with the same wfresh must be issued", control.Code)
	}
}

// 4 つの次元を 1 本で回すのは、どれか 1 つだけを読むテストが「その次元だけ検査する実装」を
// 通してしまうためである。次元ごとに、拒否の状態コード、WsFedSignInRejected が出ていること、
// 応答がトークンを運ばず WsFedSignInIssued も出ていないことを読む。拒否の理由がその次元である
// ことは、イベントが運ぶ理由で読む。REQ-WSFEDERATION-003 の 3 つの次元 (未登録の wtrealm、許可外の
// wreply、未割り当ての利用者) はこの 4 つの部分集合である。
//
//spec:covers EX-WSFEDERATION-002-03, EX-WSFEDERATION-003-01: 未登録の wtrealm、許可外の wreply、満たせない wauth、Application に割り当てられていない利用者のどれか 1 つを崩した wsignin1.0 が、400 または 403 と、その次元を理由とする WsFedSignInRejected で拒否され、応答が wresult も assertion も運ばず WsFedSignInIssued も出ないこと。すべて満たす要求が同じ入口で発行されることを対照にする。
func TestWsFedPassiveSignIn_FailsClosedOnEveryUntrustedDimension(t *testing.T) {
	for _, tc := range []struct {
		broken     brokenPassive
		wantStatus int
		wantReason string
	}{
		{unregisteredRealm(), http.StatusBadRequest, "unknown relying party"},
		{disallowedReply(), http.StatusBadRequest, "wreply"},
		{unsupportedWauth(), http.StatusBadRequest, "requested authentication method"},
		{unassignedSubject(), http.StatusForbidden, "subject not assigned to application"},
	} {
		t.Run(tc.broken.name, func(t *testing.T) {
			rec, events := tc.broken.serve(t)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", rec.Code, rec.Body.String(), tc.wantStatus)
			}
			assertNoPassiveTokenIssued(t, rec, events)
			var reasons []string
			for _, event := range events {
				if rejected, ok := event.(*feddomain.WsFedSignInRejected); ok {
					reasons = append(reasons, rejected.Reason)
				}
			}
			if len(reasons) != 1 || !strings.Contains(reasons[0], tc.wantReason) {
				t.Fatalf("WsFedSignInRejected reasons = %v, want one mentioning %q", reasons, tc.wantReason)
			}
		})
	}

	// 対照: 同じ入口で、4 つの次元をすべて満たす要求は発行に進む。これが無いと、上の 4 件は
	// サインインを配線していないスタックでもそのまま緑になる。
	t.Run("control: an intact request is issued", func(t *testing.T) {
		intact := brokenPassive{target: passiveSignInURL(passiveRealm, passiveAllowedReply, nil), assigned: true, authTime: time.Now()}
		rec, events := intact.serve(t)
		if rec.Code != http.StatusOK || !hasEvent(events, "WsFedSignInIssued") {
			t.Fatalf("control status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
