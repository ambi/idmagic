package handlers_http_test

// REQ-AUTHENTICATION-036: 復旧コードで成立した第二要素が MFA の要求を満たすことを、製品の
// 正式な入口から通しで観測する。TOTP 認証器を失った利用者にとって復旧コードが唯一の経路
// なので、ここが通らないと MFA 必須のアプリケーションから締め出される。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	recoveryCodePlaintext = "abcd-efgh-ij"
	recoveryReturnTo      = "/realms/default/admin"
)

// mfaEveryTimePolicy は「毎回 MFA、記憶済みデバイスは認めない」という実効ポリシー。
// allow_trusted_device=false にしてあるので、充足の判定は amr の第二要素そのものに掛かる。
func mfaEveryTimePolicy(t *testing.T) *appmemory.DefaultSignInPolicyRepository {
	t.Helper()
	repo := appmemory.NewDefaultSignInPolicyRepository()
	allow := false
	if err := repo.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID,
		Rules: []appdomain.SignInRule{{
			RuleID: "mfa-every-time", Name: "MFA every time", Enabled: true,
			RequiredAuthn:      appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
			AllowTrustedDevice: &allow,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// newRecoveryCodeServer は「TOTP を登録済みだが認証器を失い、復旧コードだけが手元にある
// 利用者」を、実物の SessionManager の上に組み立てる。AuthnResolver を差し替えないので、
// セッションの解決は cookie と保存済みの LoginSession を実際に通る。
func newRecoveryCodeServer(t *testing.T) (*echo.Echo, *sessionmemory.SessionStore, *sessionusecases.SessionManager) {
	t.Helper()
	now := time.Now().UTC()
	store := sessionmemory.NewSessionStore()
	manager := sessionusecases.NewSessionManager(store)

	codes := recoverymemory.NewRecoveryCodeRepository()
	sum := sha256.Sum256([]byte(strings.ReplaceAll(recoveryCodePlaintext, "-", "")))
	if err := codes.ReplaceAll(context.Background(), "user_alice", []*recoverydomain.RecoveryCode{{
		UserID: "user_alice", CodeHash: hex.EncodeToString(sum[:]), GeneratedAt: now,
	}}); err != nil {
		t.Fatal(err)
	}
	// TOTP 認証要素は登録済みである。第二要素の手段が 1 つも無いと、ポリシーの分岐が
	// 「登録を促す」側へ抜けてしまい、復旧コードの充足を観測できない。
	factors := totpmemory.NewMfaFactorRepository()
	secret := "JBSWY3DPEHPK3PXP"
	if err := factors.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: "user_alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	e, _ := newAuthorizeTestServer(t,
		&authdomain.AuthenticationContext{UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"}},
		nil,
		func(deps *httpadapter.Deps) {
			// 実物のセッション解決を使う。fake の resolver では CompleteFactor が書いた
			// 内容が次の要求へ届かず、観測したいことが観測できない。
			deps.AuthnResolver = nil
			deps.SessionManager = manager
			deps.RecoveryCodeRepo = codes
			deps.MfaFactorRepo = factors
			deps.Application.DefaultSignInPolicyRepo = mfaEveryTimePolicy(t)
		},
	)
	return e, store, manager
}

// pendingSession は第二要素を待っているセッションを作り、その cookie を返す。
func pendingSession(t *testing.T, manager *sessionusecases.SessionManager) (string, string) {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	authn, err := manager.CreateWithPending(ctx, "user_alice", []string{authdomain.AMRPassword}, time.Now().UTC(), true)
	if err != nil {
		t.Fatal(err)
	}
	return authn.SessionID, sessionusecases.SessionCookie + "=" + authn.SessionID
}

func postRecoveryCode(t *testing.T, e *echo.Echo, cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/recovery-code", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://test")
	request.Header.Set("X-Csrf-Token", "csrf-val")
	request.Header.Set("Cookie", cookie+"; idmagic_csrf=csrf-val")
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

// REQ-AUTHENTICATION-036 / EX-AUTHENTICATION-036-01: 正しい復旧コードで第二要素が成立し、
// 同じセッションによる次の認可要求が第二要素の画面へ戻されずに認可コードの発行まで進む。
//
// 観測を `/authorize` の応答まで伸ばしているのは、`amr` と `acr` を読むだけでは足りない
// からである。値が正しくても、ポリシーがそれを充足として読まなければ利用者は同じ画面へ
// 戻され続ける。復旧コードは要素を失ったときの唯一の経路なので、そこが締め出しになる。
func TestRecoveryCodeSecondFactorSatisfiesMfaPolicy_REQ_AUTHENTICATION_036(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	e, store, manager := newRecoveryCodeServer(t)
	sessionID, cookie := pendingSession(t, manager)

	recorder := postRecoveryCode(t, e, cookie, `{"code":"`+recoveryCodePlaintext+`","return_to":"`+recoveryReturnTo+`"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("recovery code status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}

	// 保存された LoginSession が rc を持ち、acr が mfa へ上がっている。
	stored, err := store.Find(ctx, sessionID)
	if err != nil || stored == nil {
		t.Fatalf("session=%v err=%v", stored, err)
	}
	if !strings.Contains(strings.Join(stored.AMR, " "), authdomain.AMRRecoveryCode) {
		t.Fatalf("stored amr=%v, want it to carry %q", stored.AMR, authdomain.AMRRecoveryCode)
	}
	if stored.ACR != "urn:idmagic:acr:mfa" {
		t.Fatalf("stored acr=%q, want urn:idmagic:acr:mfa", stored.ACR)
	}
	if stored.AuthenticationPending {
		t.Fatal("the session is still waiting on another factor")
	}

	// 次の認可要求。MFA を毎回要求する first-party クライアントでも、第二要素の画面
	// (/totp) へは戻らず、認可コードがリダイレクト先へ発行される。
	query := authorizeQuery(url.Values{"client_id": {authFirstPartyClientID}, "scope": {"openid profile"}})
	request := httptest.NewRequest(http.MethodGet, "/realms/default/authorize?"+query.Encode(), http.NoBody)
	request.Header.Set("Cookie", cookie)
	authorized := httptest.NewRecorder()
	e.ServeHTTP(authorized, request)

	location := authorized.Header().Get("Location")
	if strings.HasSuffix(location, "/totp") || strings.HasSuffix(location, "/mfa-enrollment") {
		t.Fatalf("the authorization was sent back to a second factor screen: %q", location)
	}
	if authorized.Code < 300 || authorized.Code >= 400 ||
		!strings.HasPrefix(location, authRedirectURI) || !strings.Contains(location, "code=") {
		t.Fatalf("status=%d location=%q body=%s, want an authorization code redirect",
			authorized.Code, location, authorized.Body.String())
	}
}

// REQ-AUTHENTICATION-036 / EX-AUTHENTICATION-036-02: 誤った復旧コードは拒否され、その
// 拒否が何も動かしていない。戻り値の 401 だけを観測すると、amr を先に書いてから拒否を
// 返す実装と区別が付かない。
func TestWrongRecoveryCodeLeavesTheSessionPending_REQ_AUTHENTICATION_036(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	e, store, manager := newRecoveryCodeServer(t)
	sessionID, cookie := pendingSession(t, manager)

	recorder := postRecoveryCode(t, e, cookie, `{"code":"zzzz-zzzz-zz","return_to":"`+recoveryReturnTo+`"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", recorder.Code, recorder.Body.String())
	}

	stored, err := store.Find(ctx, sessionID)
	if err != nil || stored == nil {
		t.Fatalf("session=%v err=%v", stored, err)
	}
	if strings.Contains(strings.Join(stored.AMR, " "), authdomain.AMRRecoveryCode) {
		t.Fatalf("stored amr=%v; the refused code was recorded anyway", stored.AMR)
	}
	if stored.ACR == "urn:idmagic:acr:mfa" {
		t.Fatal("the refused code still raised acr to mfa")
	}
	if !stored.AuthenticationPending {
		t.Fatal("the refused code still cleared authentication_pending")
	}
}
