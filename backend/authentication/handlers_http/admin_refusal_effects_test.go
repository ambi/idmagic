package handlers_http_test

// 管理操作と外部アイデンティティの拒否について、応答と「拒否が変えなかった状態」の
// 両方を確かめる。組み立てとヘルパーは refusal_effects_test.go が持つ。
//
// ここで扱う 4 つはいずれも 204 か本文の無い応答で終わる操作で、呼び出し元からは
// 拒否と成功を本文で見分けられない。応答だけを読むテストは「拒否を書き、そのうえで
// 実行も続ける」実装をそのまま通してしまう。

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 拒否され、対象ユーザーのセッションは失効しない。
//
// 失効は 204 を返す操作なので、拒否と成功は本文で見分けられない。各操作のあとに
// 対象のセッションを読み直して、tombstone が付いていないことを確かめる。
//
//spec:covers EX-AUTHENTICATION-021-02: 他テナントの管理者による対象ユーザーのセッション操作は
func TestAdminSessionOperationsAcrossTenantsRevokeNothing(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	target := fixture.seedSession(t, "sess-target", tenancydomain.DefaultTenantID, authRefusalAlice)
	foreignAdmin := fixture.seedSession(t, "sess-foreign-admin", authRefusalOtherTenant, authRefusalForeignAdmin)

	listed := fixture.send(t, authRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/" + authRefusalAlice + "/sessions",
		sessionID: foreignAdmin,
	})
	if listed.Code == http.StatusOK {
		t.Fatalf("他テナントの管理者にセッション一覧が返った: body=%s", listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), target) {
		t.Fatalf("拒否した応答が対象のセッション id を含む: %s", listed.Body.String())
	}

	for _, operation := range []struct{ name, path string }{
		{"個別失効", "/api/admin/v1/users/" + authRefusalAlice + "/sessions/" + target + "/revoke"},
		{"全失効", "/api/admin/v1/users/" + authRefusalAlice + "/sessions/revoke_all"},
	} {
		refused := fixture.send(t, authRefusalRequest{
			method: http.MethodPost, path: operation.path, sessionID: foreignAdmin,
			csrf: authRefusalCSRF, body: map[string]any{},
		})
		if refused.Code == http.StatusNoContent {
			t.Fatalf("%s: 他テナントの管理者の要求が受理された", operation.name)
		}
		if fixture.sessionRevoked(t, target, authRefusalAlice) {
			t.Fatalf("%s: 拒否されたのに対象のセッションが失効している", operation.name)
		}
	}

	// 対照: 同じテナントの管理者では同じ要求が通り、対象のセッションは失効する。
	// これが無いと「そもそも失効できない構成だった」と区別できない。
	homeAdmin := fixture.seedSession(t, "sess-home-admin", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, csrf: authRefusalCSRF, sessionID: homeAdmin, body: map[string]any{},
		path: "/api/admin/v1/users/" + authRefusalAlice + "/sessions/" + target + "/revoke",
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 同一テナントの失効が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if !fixture.sessionRevoked(t, target, authRefusalAlice) {
		t.Fatal("前提が壊れている: 同一テナントの失効でセッションが失効しなかった")
	}
}

// EX-AUTHENTICATION-021-02 (続き): 別テナントの管理者が自分のレルムから他テナントの
// 利用者を名指しても、そのセッションには届かない。
//
// 前のテストは越境した主体が自分のセッションを他テナントのレルムへ提示する形を見る。
// こちらは主体が自テナントでは正当な管理者である形で、防護はセッションの保存層の
// テナント境界そのものになる。
func TestAdminSessionRevokeFromOwnRealmDoesNotReachAnotherTenant(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	target := fixture.seedSession(t, "sess-cross-target", tenancydomain.DefaultTenantID, authRefusalAlice)
	foreignAdmin := fixture.seedSession(t, "sess-cross-admin", authRefusalOtherTenant, authRefusalForeignAdmin)

	refused := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, tenantID: authRefusalOtherTenant, csrf: authRefusalCSRF,
		sessionID: foreignAdmin, body: map[string]any{},
		path: "/api/admin/v1/users/" + authRefusalAlice + "/sessions/revoke_all",
	})
	if fixture.sessionRevoked(t, target, authRefusalAlice) {
		t.Fatalf("acme の管理者が default の利用者のセッションを失効させた: status=%d body=%s",
			refused.Code, refused.Body.String())
	}
}

// よる認証器のリセットは拒否され、対象ユーザーの認証器は変更されない。
//
//spec:covers EX-AUTHENTICATION-022-02: 他テナントの管理者、または `admin` ロールを持たない操作者に
func TestAdminAuthenticatorResetRefusalLeavesAuthenticatorsUnchanged(t *testing.T) {
	const seededSecret = "MFRGGZDFMZTWQ2LKNNWG23TPOB2XI4TJ"

	// authenticators は対象ユーザーに残っている TOTP と復旧コードの本数を返す。
	authenticators := func(t *testing.T, fixture *authRefusalFixture) (int, int) {
		t.Helper()
		factors, err := fixture.factors.ListBySub(context.Background(), authRefusalAlice)
		if err != nil {
			t.Fatal(err)
		}
		return len(factors), len(fixture.recoveryCodeHashes(t))
	}

	for _, actor := range []struct {
		name     string
		userID   string
		tenantID string
	}{
		{"他テナントの管理者", authRefusalForeignAdmin, authRefusalOtherTenant},
		{"admin ロールを持たない操作者", authRefusalBob, tenancydomain.DefaultTenantID},
	} {
		t.Run(actor.name, func(t *testing.T) {
			fixture := newAuthRefusalServer(t)
			fixture.seedTotpFactor(t, authRefusalAlice, seededSecret)
			fixture.seedRecoveryCodes(t, authRefusalAlice)
			session := fixture.seedSession(t, "sess-actor", actor.tenantID, actor.userID)

			refused := fixture.send(t, authRefusalRequest{
				method: http.MethodPost, csrf: authRefusalCSRF, sessionID: session,
				path: "/api/admin/v1/users/" + authRefusalAlice + "/authenticator-reset",
				body: map[string]any{"targets": []string{"totp", "recovery_code"}},
			})
			if refused.Code == http.StatusOK {
				t.Fatalf("リセットが受理された: body=%s", refused.Body.String())
			}
			if factors, codes := authenticators(t, fixture); factors != 1 || codes != 1 {
				t.Fatalf("拒否されたのに認証器が変わった: totp=%d recovery=%d", factors, codes)
			}
			for _, event := range *fixture.events {
				if event.EventType() == "AuthenticatorResetCompleted" {
					t.Fatalf("拒否後に AuthenticatorResetCompleted が発行された: %#v", event)
				}
			}
		})
	}

	// 対照: 同一テナントの admin では同じ要求が通り、TOTP と復旧コードが消える。
	fixture := newAuthRefusalServer(t)
	fixture.seedTotpFactor(t, authRefusalAlice, seededSecret)
	fixture.seedRecoveryCodes(t, authRefusalAlice)
	homeAdmin := fixture.seedSession(t, "sess-home-admin", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodPost, csrf: authRefusalCSRF, sessionID: homeAdmin,
		path: "/api/admin/v1/users/" + authRefusalAlice + "/authenticator-reset",
		body: map[string]any{"targets": []string{"totp", "recovery_code"}},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: admin のリセットが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if factors, codes := authenticators(t, fixture); factors != 0 || codes != 0 {
		t.Fatalf("前提が壊れている: リセット後に totp=%d recovery=%d", factors, codes)
	}
}

// seedConnection は既定テナントに外部 IdP 接続を 1 本置く。
// この Rule の拒否はどれも 1 テナント内で完結するので、テナントは固定する。
func (f *authRefusalFixture) seedConnection(t *testing.T, id, displayName string) {
	t.Helper()
	now := time.Now().UTC()
	if err := f.federation.Connections.Save(context.Background(), &federationdomain.IdentityProviderConnection{
		ID: id, TenantID: tenancydomain.DefaultTenantID, DisplayName: displayName,
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionDisabled,
		Issuer: "https://upstream.example", ClientID: "upstream-client",
		AuthorizationEndpoint: "https://upstream.example/auth",
		TokenEndpoint:         "https://upstream.example/token", JWKSURI: "https://upstream.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

// connectionNames は保存されている接続の表示名を id 付きで返す。
func (f *authRefusalFixture) connectionNames(t *testing.T, tenantID string) map[string]string {
	t.Helper()
	connections, err := f.federation.Connections.List(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, connection := range connections {
		names[connection.ID] = connection.DisplayName
	}
	return names
}

// 接続の管理へ到達できない。拒否は `insufficient_scope` で、必要な資格として対話
// セッションを提示する。接続は作成も更新もされない。
//
//spec:covers EX-AUTHENTICATION-025-02: API アクセストークンは、どのスコープを持っていても外部 IdP
func TestApiTokenCannotManageIdentityProviderConnections(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedConnection(t, authRefusalProviderID, "Workforce")
	before := fixture.connectionNames(t, tenancydomain.DefaultTenantID)

	// 管理 API に届きうる粒度スコープを広く持たせる。それでも到達できないことが
	// 「対話セッションに限る」の中身である。
	token := fixture.issueApiToken(t, tenancydomain.DefaultTenantID, authRefusalHomeAdmin,
		apitokendomain.ScopeSettingsWrite, apitokendomain.ScopeUsersWrite,
		apitokendomain.ScopeTenantsWrite, apitokendomain.ScopeSamlWrite)
	body := map[string]any{
		"display_name": "Taken over", "protocol": "oidc", "issuer": "https://attacker.example",
		"client_id": "attacker", "authorization_endpoint": "https://attacker.example/auth",
		"token_endpoint": "https://attacker.example/token", "jwks_uri": "https://attacker.example/jwks",
		"claim_mapping":  map[string]string{"subject": "sub", "username": "email"},
		"linking_policy": "none",
	}
	for _, operation := range []struct{ name, method, path string }{
		{"一覧", http.MethodGet, "/api/admin/v1/identity-providers"},
		{"作成", http.MethodPost, "/api/admin/v1/identity-providers"},
		{"更新", http.MethodPut, "/api/admin/v1/identity-providers/" + authRefusalProviderID},
		{"削除", http.MethodDelete, "/api/admin/v1/identity-providers/" + authRefusalProviderID},
	} {
		refused := fixture.send(t, authRefusalRequest{
			method: operation.method, path: operation.path, bearer: token, body: body,
		})
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s: status=%d body=%s、期待は 403", operation.name, refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "insufficient_scope" {
			t.Fatalf("%s: error=%q、期待は insufficient_scope", operation.name, code)
		}
		if challenge := refused.Header().Get("WWW-Authenticate"); !strings.Contains(challenge, spec.InteractiveSessionScope) {
			t.Fatalf("%s: WWW-Authenticate=%q、必要な資格として対話セッションを提示していない", operation.name, challenge)
		}
		if strings.Contains(refused.Body.String(), "upstream.example") {
			t.Fatalf("%s: 拒否した応答が接続の設定を含む: %s", operation.name, refused.Body.String())
		}
	}
	if after := fixture.connectionNames(t, tenancydomain.DefaultTenantID); len(after) != len(before) ||
		after[authRefusalProviderID] != before[authRefusalProviderID] {
		t.Fatalf("拒否されたのに接続が変わった: before=%v after=%v", before, after)
	}

	// 対照: 同じ変更をブラウザーの管理セッションから行うと通る。
	// 拒否したのが API アクセストークンという資格であって、要求の中身ではないと示す。
	adminSession := fixture.seedSession(t, "sess-admin", tenancydomain.DefaultTenantID, authRefusalHomeAdmin)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodPut, path: "/api/admin/v1/identity-providers/" + authRefusalProviderID,
		sessionID: adminSession, csrf: authRefusalCSRF, body: body,
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 管理セッションの更新が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if name := fixture.connectionNames(t, tenancydomain.DefaultTenantID)[authRefusalProviderID]; name != "Taken over" {
		t.Fatalf("前提が壊れている: 管理セッションの更新後の表示名=%q", name)
	}
}

// よる外部アイデンティティのリンクと解除は拒否され、どちらも反映されない。
//
//spec:covers EX-AUTHENTICATION-003-02: ステップアップ認証が古い、または行われていないセッションに
func TestExternalIdentityLinkAndUnlinkWithoutStepUpChangeNothing(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedConnection(t, authRefusalProviderID, "Workforce")
	fixture.seedFederatedLink(t, tenancydomain.DefaultTenantID, authRefusalProviderID, authRefusalAlice, "external-alice")
	stale := fixture.seedSession(t, "sess-stale-step-up", tenancydomain.DefaultTenantID, authRefusalAlice,
		func(session *sessiondomain.LoginSession) {
			session.AuthTime = time.Now().Add(-time.Hour).UTC().Unix()
		})

	for _, operation := range []struct{ name, method string }{
		{"リンク", http.MethodPost},
		{"解除", http.MethodDelete},
	} {
		refused := fixture.send(t, authRefusalRequest{
			method: operation.method, path: "/api/account/v1/linked-identities/" + authRefusalProviderID,
			sessionID: stale, csrf: authRefusalCSRF, body: map[string]any{},
		})
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s: status=%d body=%s、期待は 403", operation.name, refused.Code, refused.Body.String())
		}
		if code := problemCode(t, refused); code != "step_up_required" {
			t.Fatalf("%s: error=%q、期待は step_up_required", operation.name, code)
		}
		if location := refused.Header().Get("Location"); location != "" {
			t.Fatalf("%s: 拒否したのに上流への遷移が返った: %q", operation.name, location)
		}
	}
	if providers := fixture.linkedProviders(t, authRefusalAlice); len(providers) != 1 {
		t.Fatalf("ステップアップ無しの要求でリンクが変わった: %v", providers)
	}

	// 対照: 直近でステップアップを済ませたセッションでは解除が通り、リンクが消える。
	fresh := fixture.seedSession(t, "sess-fresh-step-up", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodDelete, path: "/api/account/v1/linked-identities/" + authRefusalProviderID,
		sessionID: fresh, csrf: authRefusalCSRF,
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: ステップアップ済みの解除が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if providers := fixture.linkedProviders(t, authRefusalAlice); len(providers) != 0 {
		t.Fatalf("前提が壊れている: 解除後にリンクが残っている: %v", providers)
	}
}

// 残らなくなる解除は締め出しを防ぐために拒否され、そのリンクは残る。
//
//spec:covers EX-AUTHENTICATION-003-03: パスワード資格情報も他の外部アイデンティティのリンクも
func TestSoleFederatedIdentityUnlinkKeepsTheLink(t *testing.T) {
	fixture := newAuthRefusalServer(t)
	fixture.seedConnection(t, authRefusalProviderID, "Workforce")
	fixture.seedFederatedLink(t, tenancydomain.DefaultTenantID, authRefusalProviderID, authRefusalSoleFederated, "external-sole")
	session := fixture.seedSession(t, "sess-sole", tenancydomain.DefaultTenantID, authRefusalSoleFederated, withFreshStepUp)

	refused := fixture.send(t, authRefusalRequest{
		method: http.MethodDelete, path: "/api/account/v1/linked-identities/" + authRefusalProviderID,
		sessionID: session, csrf: authRefusalCSRF,
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s、期待は 403", refused.Code, refused.Body.String())
	}
	if providers := fixture.linkedProviders(t, authRefusalSoleFederated); len(providers) != 1 {
		t.Fatalf("締め出しになる解除が反映された: %v", providers)
	}

	// 対照: 2 本目のリンクを足すと、同じ要求が通って 1 本目が消える。
	// 拒否したのが「最後の 1 本」であって、ステップアップでも所有者判定でもないと示す。
	fixture.seedConnection(t, "backup", "Backup")
	fixture.seedFederatedLink(t, tenancydomain.DefaultTenantID, "backup", authRefusalSoleFederated, "external-sole-backup")
	accepted := fixture.send(t, authRefusalRequest{
		method: http.MethodDelete, path: "/api/account/v1/linked-identities/" + authRefusalProviderID,
		sessionID: session, csrf: authRefusalCSRF,
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 2 本目がある解除が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if providers := fixture.linkedProviders(t, authRefusalSoleFederated); len(providers) != 1 {
		t.Fatalf("前提が壊れている: 解除後のリンク=%v", providers)
	}
}
