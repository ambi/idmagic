package handlers_http_test

// 検証済みメールアドレスによる自動リンクの拒否について、応答と「拒否が変えなかった
// 状態」の両方を確かめる。
//
// 入口は production と同じ callback である。自動リンクの判定はプロトコル検証の後、
// LoginSession の発行の前に立つので、use case を直接呼ぶテストでは順序ごと失われる。
// 上流の IdP だけは stub に置き換える。外部との往復はこの拒否の前提であって、
// 確かめたい判定ではない。

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/federation/handlers_http"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	autoLinkProviderID = "workforce"
	autoLinkSubject    = "external-subject"
	autoLinkEmail      = "shared@example.test"
)

// claimsDriver は上流の応答を差し替える stub。State は start が採番した値を控える。
type claimsDriver struct {
	state       string
	claims      federationdomain.NormalizedClaims
	completeErr error
	completed   int
}

func (d *claimsDriver) Start(
	_ federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	d.state = attempt.State
	return "https://idp.example/authorize", nil
}

func (d *claimsDriver) Complete(
	_ context.Context,
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ string,
	_ time.Time,
) (federationdomain.NormalizedClaims, error) {
	d.completed++
	return d.claims, d.completeErr
}

// autoLinkFixture は自動リンクの拒否が変えなかったものを読み直すための保存層を持つ。
type autoLinkFixture struct {
	e        *echo.Echo
	repos    federationmemory.Repositories
	sessions *sessionmemory.SessionStore
	driver   *claimsDriver
	events   []string
	// rejections は発行された FederatedLoginRejected を、Reason を読めるように値ごと控える。
	rejections []federationdomain.FederatedLoginRejected
}

// newAutoLinkServer は 1 本の接続と、`policy` に応じた linking policy を持つ環境を組む。
// duplicateEmail が true のとき、同じ検証済みメールアドレスを持つ利用者を 2 人置く。
func newAutoLinkServer(
	t *testing.T,
	policy federationdomain.LinkingPolicy,
	duplicateEmail bool,
	claims federationdomain.NormalizedClaims,
) *autoLinkFixture {
	t.Helper()
	return newFederationServer(t, policy, duplicateEmail, &claimsDriver{claims: claims})
}

// newFederationServer は driver を呼び出し側が握る形の同じ環境である。上流の応答が
// 検証に落ちる経路は driver の戻り値でしか作れず、発行されたイベントも読み直せる
// ようにしてある。
func newFederationServer(
	t *testing.T,
	policy federationdomain.LinkingPolicy,
	duplicateEmail bool,
	driver *claimsDriver,
) *autoLinkFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	users := usermemory.NewUserRepository()
	holders := []string{"user-alice"}
	if duplicateEmail {
		holders = append(holders, "user-alice-duplicate")
	}
	for _, id := range holders {
		email := autoLinkEmail
		users.Seed(&userdomain.User{
			ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id,
			Email: &email, EmailVerified: true, PasswordHash: "unused",
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	repos := federationmemory.NewRepositories()
	if err := repos.Connections.Save(ctx, &federationdomain.IdentityProviderConnection{
		ID: autoLinkProviderID, TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: policy, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	store := sessionmemory.NewSessionStore()
	sessions := sessionusecases.NewSessionManager(store)
	fixture := &autoLinkFixture{repos: repos, sessions: store, driver: driver}
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group(""), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Users: users, Sessions: sessions,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: driver,
			},
			Emit: func(event spec.DomainEvent) {
				fixture.events = append(fixture.events, event.EventType())
				if rejected, ok := event.(*federationdomain.FederatedLoginRejected); ok {
					fixture.rejections = append(fixture.rejections, *rejected)
				}
			},
		},
		Auth: httpdeps.Deps{Deps: support.Deps{Issuer: "http://idp.test"}},
	})
	fixture.e = e
	return fixture
}

// completeLogin は start から callback までを production と同じ経路で 1 往復する。
func (f *autoLinkFixture) completeLogin(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	return f.callback(t, f.startLogin(t))
}

// startLogin は login attempt を 1 件作り、上流へ渡した state を返す。
func (f *autoLinkFixture) startLogin(t *testing.T) string {
	t.Helper()
	started := httptest.NewRecorder()
	f.e.ServeHTTP(started, httptest.NewRequest(
		http.MethodGet, "/api/auth/federation/start?provider_id="+autoLinkProviderID, http.NoBody,
	))
	if started.Code != http.StatusSeeOther || f.driver.state == "" {
		t.Fatalf("前提が壊れている: start status=%d state=%q", started.Code, f.driver.state)
	}
	return f.driver.state
}

// callback は上流から戻る callback を、指定した state で 1 回送る。
func (f *autoLinkFixture) callback(t *testing.T, state string) *httptest.ResponseRecorder {
	t.Helper()
	completed := httptest.NewRecorder()
	f.e.ServeHTTP(completed, httptest.NewRequest(http.MethodGet,
		"/api/auth/federation/oidc/callback?state="+url.QueryEscape(state)+"&code=verified",
		http.NoBody,
	))
	return completed
}

// issuedSessions は seed 済みの利用者に対して発行されたセッションの数を返す。
func (f *autoLinkFixture) issuedSessions(t *testing.T, userID string) int {
	t.Helper()
	issued, err := f.sessions.ListBySub(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return len(issued)
}

// linkedIdentity は外部 subject に対する関連付けを読み直す。
func (f *autoLinkFixture) linkedIdentity(t *testing.T) *federationdomain.FederatedIdentity {
	t.Helper()
	identity, err := f.repos.Identities.FindBySubject(
		context.Background(), tenancydomain.DefaultTenantID, autoLinkProviderID, autoLinkSubject,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

// 一意でないとき、自動リンクと LoginSession の発行を拒否する。
//
// この拒否が漏れると、上流で作れるだけのメールアドレスが既存アカウントの鍵になる。
// 応答だけを読むテストでは、拒否を書いたうえでリンクとセッションを作る実装を
// 通してしまうので、関連付けとセッション Cookie の両方を読み直す。
//
//spec:covers REQ-AUTHENTICATION-002, EX-AUTHENTICATION-002-02: ポリシーが `None`、メールアドレスが未検証、または一致が
func TestAutoLinkRefusalCreatesNoIdentityAndNoSession(t *testing.T) {
	verified := federationdomain.NormalizedClaims{
		Subject: autoLinkSubject, Username: autoLinkEmail, Email: autoLinkEmail, EmailVerified: true,
	}

	for _, refusal := range []struct {
		name           string
		policy         federationdomain.LinkingPolicy
		duplicateEmail bool
		claims         federationdomain.NormalizedClaims
	}{
		{
			name: "ポリシーが None", policy: federationdomain.LinkingNone, claims: verified,
		},
		{
			name: "メールアドレスが未検証", policy: federationdomain.LinkingVerifiedEmail,
			claims: federationdomain.NormalizedClaims{
				Subject: autoLinkSubject, Username: autoLinkEmail, Email: autoLinkEmail, EmailVerified: false,
			},
		},
		{
			name: "一致が一意でない", policy: federationdomain.LinkingVerifiedEmail,
			duplicateEmail: true, claims: verified,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			fixture := newAutoLinkServer(t, refusal.policy, refusal.duplicateEmail, refusal.claims)

			refused := fixture.completeLogin(t)
			if refused.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s、期待は 401", refused.Code, refused.Body.String())
			}
			if identity := fixture.linkedIdentity(t); identity != nil {
				t.Fatalf("拒否されたのに関連付けが作られた: %+v", identity)
			}
			if cookies := refused.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("拒否されたのにセッション Cookie が発行された: %+v", cookies)
			}
			if location := refused.Header().Get("Location"); location != "" {
				t.Fatalf("拒否されたのに認可の継続へ遷移した: %q", location)
			}
		})
	}

	// 対照: ポリシーが VerifiedEmail で、検証済みメールアドレスが一意に一致するときは
	// 同じ往復でリンクが作られ、セッション Cookie が発行される。これが無いと
	// 「そもそもリンクできない構成だった」と区別できない。
	fixture := newAutoLinkServer(t, federationdomain.LinkingVerifiedEmail, false, verified)
	accepted := fixture.completeLogin(t)
	if accepted.Code != http.StatusSeeOther {
		t.Fatalf("前提が壊れている: 自動リンクが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	identity := fixture.linkedIdentity(t)
	if identity == nil || identity.LocalUserID != "user-alice" {
		t.Fatalf("前提が壊れている: 自動リンクの結果=%+v", identity)
	}
	if cookies := accepted.Result().Cookies(); len(cookies) != 1 || cookies[0].Value == "" {
		t.Fatalf("前提が壊れている: 自動リンク後の Cookie=%+v", cookies)
	}
}

// `state` の照合または上流の応答の検証に落ちた callback。
//
// 拒否の応答だけを読むと、セッションを発行してから error を返す実装と区別が付かない。
// 関連付けとセッションの双方を保存層から読み直し、そのうえで記録が残ることを確かめる。
// 記録が無いと、同じ IdP に対する総当たりが監査から見えない。発行していない `state` を
// 送りつける攻撃は上流の応答を用意する必要がなく、最も総当たりしやすい。
//
// `state` 不一致の driver は検証を通る claims を返す。照合が働かなければログインが
// 成立してしまう構成で拒否を観測するためである。
//
//spec:covers REQ-AUTHENTICATION-001, EX-AUTHENTICATION-001-03: `state` の不一致または上流の応答の検証失敗で拒否した callback は、LoginSession も関連付けも作らず、拒否の種類を区別できる Reason の FederatedLoginRejected を残すことを固定する。
func TestMismatchedCallbackCreatesNothingAndRecordsTheRejection(t *testing.T) {
	verified := federationdomain.NormalizedClaims{
		Subject: autoLinkSubject, Username: autoLinkEmail, Email: autoLinkEmail, EmailVerified: true,
	}

	for _, refusal := range []struct {
		name       string
		driver     *claimsDriver
		state      func(issued string) string
		reason     string
		providerID string
		validated  int
	}{
		{
			name: "上流の応答が検証に落ちる",
			driver: &claimsDriver{
				claims:      verified,
				completeErr: errors.New("id token signature does not verify"),
			},
			state:  func(issued string) string { return issued },
			reason: federationdomain.RejectionProtocolValidationFailed, providerID: autoLinkProviderID,
			validated: 1,
		},
		{
			name:   "発行していない state",
			driver: &claimsDriver{claims: verified},
			state:  func(string) string { return "state-the-broker-never-issued" },
			// attempt が見つからない以上、callback から接続は特定できない。
			reason: federationdomain.RejectionStateMismatch, providerID: "",
			validated: 0,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			fixture := newFederationServer(t, federationdomain.LinkingVerifiedEmail, false, refusal.driver)

			refused := fixture.callback(t, refusal.state(fixture.startLogin(t)))
			if refused.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s、期待は 401", refused.Code, refused.Body.String())
			}
			if identity := fixture.linkedIdentity(t); identity != nil {
				t.Fatalf("拒否されたのに関連付けが作られた: %+v", identity)
			}
			if got := fixture.issuedSessions(t, "user-alice"); got != 0 {
				t.Fatalf("拒否されたのに LoginSession が %d 件発行された", got)
			}
			if cookies := refused.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("拒否されたのにセッション Cookie が発行された: %+v", cookies)
			}
			if got := refusal.driver.completed; got != refusal.validated {
				t.Fatalf("上流の応答の検証が %d 回、期待は %d 回", got, refusal.validated)
			}
			if len(fixture.rejections) != 1 {
				t.Fatalf("FederatedLoginRejected が %d 件、期待は 1 件: events=%v", len(fixture.rejections), fixture.events)
			}
			rejected := fixture.rejections[0]
			if rejected.Reason != refusal.reason || rejected.ProviderID != refusal.providerID ||
				rejected.TenantID != tenancydomain.DefaultTenantID {
				t.Fatalf("FederatedLoginRejected=%+v、期待は Reason=%q ProviderID=%q TenantID=%q",
					rejected, refusal.reason, refusal.providerID, tenancydomain.DefaultTenantID)
			}
		})
	}

	// 対照: 同じ往復が、発行した state と検証を通る応答では成立し、拒否を記録しない。
	// これが無いと、拒否が照合や検証ではなく環境の不足で起きていた場合を見分けられない。
	accepted := newFederationServer(t, federationdomain.LinkingVerifiedEmail, false, &claimsDriver{claims: verified})
	if response := accepted.completeLogin(t); response.Code != http.StatusSeeOther {
		t.Fatalf("前提が壊れている: 検証を通る応答が status=%d", response.Code)
	}
	if len(accepted.rejections) != 0 {
		t.Fatalf("成立したログインで FederatedLoginRejected が発行された: %+v", accepted.rejections)
	}
}
