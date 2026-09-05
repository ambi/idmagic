package handlers_http_test

// account 同意 API のスコープ境界とテナント境界が宣言する拒否について、
// 応答と「拒否が変えなかった同意」の両方を確かめる。
//
// 撤回は 204 を返す操作なので、拒否と成功は本文で見分けられない。応答だけを読む
// テストは「拒否を返し、そのうえで撤回もする」実装をそのまま通してしまう。
// そこで各テストは拒否のあとに同意を読み直し、Granted のままであることを確かめる。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	consentUserID     = "alice"
	consentOtherUser  = "bob"
	consentClientID   = "portal"
	consentOtherRealm = "acme"
)

type consentRefusalFixture struct {
	e        *echo.Echo
	consents *consentmemory.ConsentRepository
	signer   *tokensjose.JWTSigner
}

func newConsentRefusalFixture(t *testing.T) *consentRefusalFixture {
	t.Helper()
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	for _, id := range []string{consentUserID, consentOtherUser} {
		users.Seed(&userdomain.User{
			ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}
	consents := consentmemory.NewConsentRepository()

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive},
		{ID: consentOtherRealm, Realm: consentOtherRealm, Status: tenancydomain.TenantStatusActive},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	// 本物の署名器を使う。トークンのテナント束縛は iss の照合として
	// イントロスペクション側にあり、偽の introspector で置き換えるとその照合ごと
	// 消えてしまう。
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner("http://idp.test", keyStore)

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:            "http://idp.test",
		TenantRepo:        tenants,
		IdManagement:      idmanagement.Module{UserRepo: users},
		OAuth2:            oauth2.Module{ConsentRepo: consents},
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
	})
	return &consentRefusalFixture{e: e, consents: consents, signer: signer}
}

// issueToken は realm のテナント文脈で、指定したスコープのアクセストークンを署名する。
// iss がテナントごとに変わるので、これがトークンのテナント束縛そのものになる。
func (f *consentRefusalFixture) issueToken(t *testing.T, tenantID, sub, scope string) string {
	t.Helper()
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenantID, Realm: realmOf(tenantID)},
		"http://idp.test/realms/"+realmOf(tenantID),
		"/realms/"+realmOf(tenantID),
	)
	token, _, err := f.signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: "api"}, Sub: sub,
		Scopes: strings.Fields(scope), AuthTime: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func realmOf(tenantID string) string {
	if tenantID == tenancydomain.DefaultTenantID {
		return tenancydomain.DefaultRealm
	}
	return tenantID
}

func (f *consentRefusalFixture) seedConsent(t *testing.T, tenantID, userID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenantID, &consentdomain.Consent{
		UserID: userID, ClientID: consentClientID, Scopes: []string{"openid"},
		State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *consentRefusalFixture) revoke(realm, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost,
		"/realms/"+realm+"/api/account/v1/consents/"+consentClientID+"/revoke", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)
	// API アクセストークンの経路はブラウザーの二重送信 CSRF を要求しない。
	// Origin だけは同一オリジンとして通す。
	request.Header.Set("Origin", "http://idp.test")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

// consentState は alice の同意の状態を読む。拒否が動かしてはならない対象は
// どのテストでも alice の同意なので、主体は固定する。
func (f *consentRefusalFixture) consentState(t *testing.T, tenantID string) consentdomain.ConsentState {
	t.Helper()
	consent, err := f.consents.Find(context.Background(), tenantID, consentUserID, consentClientID)
	if err != nil {
		t.Fatal(err)
	}
	if consent == nil {
		t.Fatalf("同意が消えている: tenant=%s user=%s", tenantID, consentUserID)
	}
	return consent.State
}

// EX-OAUTH2-002-02: account:read だけのトークンによる同意撤回は拒否され、
// 対象の同意は Granted のまま残る。
func TestRevokeAccountConsentWithReadOnlyScopeLeavesConsentGranted(t *testing.T) {
	fixture := newConsentRefusalFixture(t)
	fixture.seedConsent(t, tenancydomain.DefaultTenantID, consentUserID)

	readOnly := fixture.issueToken(t, tenancydomain.DefaultTenantID, consentUserID, "account:read")
	refused := fixture.revoke(tenancydomain.DefaultRealm, readOnly)
	if refused.Code == http.StatusNoContent {
		t.Fatalf("account:read だけの撤回が受理された: status=%d", refused.Code)
	}
	if state := fixture.consentState(t, tenancydomain.DefaultTenantID); state != consentdomain.ConsentGranted {
		t.Fatalf("拒否されたのに同意が %s になった", state)
	}

	// 対照: 撤回スコープを持つトークンでは同じ要求が通り、同意は Revoked になる。
	// これが無いと「そもそも撤回できない構成だった」と区別できない。
	canRevoke := fixture.issueToken(t, tenancydomain.DefaultTenantID, consentUserID,
		"account:read account:consents:write")
	accepted := fixture.revoke(tenancydomain.DefaultRealm, canRevoke)
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 撤回スコープ付きで status=%d body=%s",
			accepted.Code, accepted.Body.String())
	}
	if state := fixture.consentState(t, tenancydomain.DefaultTenantID); state != consentdomain.ConsentRevoked {
		t.Fatalf("前提が壊れている: 撤回後の状態が %s", state)
	}
}

// EX-OAUTH2-002-03: トークンのテナントまたは user_id が操作対象と一致しない撤回は
// 拒否され、対象の同意は Granted のまま残る。
//
// user_id の側は、ハンドラが actor も target もトークンの sub に固定するため、
// 別ユーザーの同意を名指しする経路が URL に存在しない。ここではその固定が
// 実際に効いていること — 別ユーザーのトークンでは他人の同意が動かないこと — を読む。
// テナントの側は、レルムをまたいだ提示で確かめる。
func TestRevokeAccountConsentAcrossUserAndTenantLeavesConsentGranted(t *testing.T) {
	t.Run("別ユーザーのトークン", func(t *testing.T) {
		fixture := newConsentRefusalFixture(t)
		fixture.seedConsent(t, tenancydomain.DefaultTenantID, consentUserID)

		// bob のトークンで alice の同意 (同じ client_id) の撤回を試みる。
		otherUser := fixture.issueToken(t, tenancydomain.DefaultTenantID, consentOtherUser,
			"account:read account:consents:write")
		response := fixture.revoke(tenancydomain.DefaultRealm, otherUser)
		if response.Code == http.StatusNoContent {
			t.Log("bob 自身の同意として処理された。alice の同意が動いていないことを次に確かめる。")
		}
		if state := fixture.consentState(t, tenancydomain.DefaultTenantID); state != consentdomain.ConsentGranted {
			t.Fatalf("別ユーザーのトークンで alice の同意が %s になった", state)
		}
	})

	t.Run("別テナントのレルム", func(t *testing.T) {
		fixture := newConsentRefusalFixture(t)
		fixture.seedConsent(t, consentOtherRealm, consentUserID)

		// default テナントで発行したトークンを acme のレルムへ提示する。
		// トークンのテナント束縛は iss の照合なので、拒否の種別が invalid_token で
		// あることまで確かめる。単に「拒否された」だけでは、主体が acme に居ない
		// という後段の理由 (authentication_required) と区別できない。
		defaultTenantToken := fixture.issueToken(t, tenancydomain.DefaultTenantID, consentUserID,
			"account:read account:consents:write")
		response := fixture.revoke(consentOtherRealm, defaultTenantToken)
		if response.Code == http.StatusNoContent {
			t.Fatal("default テナントのトークンで acme の同意が撤回された")
		}
		if !strings.Contains(response.Body.String(), "invalid_token") {
			t.Fatalf("テナント束縛ではない理由で拒否された: status=%d body=%s",
				response.Code, response.Body.String())
		}
		if state := fixture.consentState(t, consentOtherRealm); state != consentdomain.ConsentGranted {
			t.Fatalf("拒否されたのに別テナントの同意が %s になった", state)
		}

		// 対照: 同じ acme のレルムで発行したトークンは、少なくとも
		// トークンのテナント判定は通過する (その先の主体解決で止まる)。
		sameTenantToken := fixture.issueToken(t, consentOtherRealm, consentUserID,
			"account:read account:consents:write")
		if body := fixture.revoke(consentOtherRealm, sameTenantToken).Body.String(); strings.Contains(body, "invalid_token") {
			t.Fatalf("前提が壊れている: 同一テナントのトークンまで invalid_token になった: %s", body)
		}
	})
}
