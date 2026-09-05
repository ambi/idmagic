package handlers_http_test

// アカウント API とメールアドレス確認が宣言する拒否について、応答と
// 「拒否が変えなかった状態」の両方を確かめる。組み立てとヘルパーは
// refusal_effects_test.go が持つ。
//
// ここで守られているのはスコープ、トークンのテナント束縛、二重送信 CSRF である。
// いずれも use case を直接呼ぶテストでは 1 つも通らない。

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// emailChangeTokenPattern はメール本文の確認リンクからトークンを取り出す。
var emailChangeTokenPattern = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

// EX-IDMANAGEMENT-002-02: `account:read` だけの API アクセストークンによる変更操作は
// 拒否され、プロフィールもメールアドレスの変更申請も残らない。
//
// 変更申請は 204 で終わるので、応答からは拒否と成功を見分けられない。申請が
// 生む副作用 (確認メールの送信) を数えて、拒否が何も起こしていないことを読む。
func TestAccountReadScopeChangesNoProfileAndRequestsNoEmailChange(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	readOnly := fixture.issueApiToken(
		t, idmRefusalAlice, apitokendomain.ScopeAccountRead,
	)
	before := *fixture.user(t, idmRefusalAlice)

	renamed := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: readOnly,
		body: map[string]any{"name": "renamed-by-account-read"},
	})
	if renamed.Code != http.StatusForbidden {
		t.Fatalf("account:read の変更が status=%d body=%s, want 403", renamed.Code, renamed.Body.String())
	}
	if code := idmProblemCode(t, renamed); code != "insufficient_scope" {
		t.Fatalf("error code=%q, want insufficient_scope", code)
	}
	// 拒否が何も変えていないこと。表示名も更新時刻も動いていない。
	after := fixture.user(t, idmRefusalAlice)
	if after.Name == nil || *after.Name != *before.Name || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("拒否されたのにプロフィールが変わった: name=%v updated_at=%s", after.Name, after.UpdatedAt)
	}

	changed := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/change_request", bearer: readOnly,
		body: map[string]any{"new_email": "moved@example.test"},
	})
	if changed.Code != http.StatusForbidden {
		t.Fatalf("account:read の変更申請が status=%d body=%s, want 403", changed.Code, changed.Body.String())
	}
	// 拒否が申請を残していないこと。確認メールが 1 通も送られていない。
	if sent := len(fixture.emails.Sent); sent != 0 {
		t.Fatalf("拒否されたのに確認メールが %d 通送られた", sent)
	}

	// 対照: account:write を持つトークンなら同じ 2 つが通る。
	writable := fixture.issueApiToken(
		t, idmRefusalAlice, apitokendomain.ScopeAccountWrite,
	)
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: writable,
		body: map[string]any{"name": "renamed-by-account-write"},
	}); accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: account:write の変更が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/change_request", bearer: writable,
		body: map[string]any{"new_email": "moved@example.test"},
	}); accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: account:write の変更申請が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if sent := len(fixture.emails.Sent); sent != 1 {
		t.Fatalf("前提が壊れている: account:write でも確認メールが %d 通", sent)
	}
}

// EX-IDMANAGEMENT-002-03 (テナントの側): トークンのテナントが操作対象と一致しない
// 要求は拒否され、対象ユーザーの属性は変更されない。
//
// テナント束縛は 6 重になっている。最も外側は署名鍵そのものがテナントごとに
// 分かれていることで、これは「1 本外すと落ちる」形の防護ではない。
// 内側の 5 本 (`iss` の照合、`aud` の照合、テナントで絞った `jti` 参照、
// `ResolveAuthentication` のテナント判定、`loadSelf` のテナント判定) をすべて外しても、
// 越境したトークンは acme の鍵では検証できないため拒否は残る。
// このテストが固定するのは境界が保たれていることであり、特定の 1 行の有無ではない。
// 取り外せる判定に対する変更耐性は、同じ具体例の `user_id` の側が持つ。
func TestAccountTokenFromAnotherTenantChangesNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	// default のテナントで alice に固定したトークンを、acme のレルムへ持ち込む。
	foreignToken := fixture.issueApiToken(
		t, idmRefusalAlice,
		apitokendomain.ScopeAccountRead, apitokendomain.ScopeAccountWrite,
	)
	before := *fixture.user(t, idmRefusalAlice)

	for _, refusal := range []struct {
		name, method, path string
		body               any
	}{
		{"参照", http.MethodGet, "/api/account/v1/profile", nil},
		{
			"変更", http.MethodPatch, "/api/account/v1/profile",
			map[string]any{"name": "renamed-across-tenants"},
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path, tenantID: idmRefusalOtherTenant,
				bearer: foreignToken, body: refusal.body,
			})
			if refused.Code < http.StatusBadRequest {
				t.Fatalf("越境したトークンが status=%d body=%s で受理された", refused.Code, refused.Body.String())
			}
			// 拒否が対象の属性を返していないこと。
			if refusal.method == http.MethodGet && strings.Contains(refused.Body.String(), *before.Email) {
				t.Fatalf("越境した参照が対象の属性を返した: %s", refused.Body.String())
			}
		})
	}

	// 拒否が何も変えていないこと。default 側の alice は表示名も更新時刻もそのまま。
	after := fixture.user(t, idmRefusalAlice)
	if after.Name == nil || *after.Name != *before.Name || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("越境した変更が通った: name=%v updated_at=%s", after.Name, after.UpdatedAt)
	}

	// 対照: 発行元のレルムなら同じトークンで同じ変更が通る。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: foreignToken,
		body: map[string]any{"name": "renamed-in-home-realm"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 発行元レルムの変更が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if name := fixture.user(t, idmRefusalAlice).Name; name == nil || *name != "renamed-in-home-realm" {
		t.Fatalf("前提が壊れている: 発行元レルムでも表示名が変わらない: %v", name)
	}
}

// EX-IDMANAGEMENT-002-03 (`user_id` の側): トークンの `user_id` が操作対象と
// 一致しない要求は拒否され、どちらの利用者の属性も変更されない。
//
// アカウント API の操作対象はトークン自身の主体なので、この不一致は保存された
// トークンの `user_id` と JWT の `sub` が食い違ったときにだけ起こる。判定は
// フェイルクローズで二重にある。どちらが選ばれても操作が通ってはならないのは、
// 「トークンが名乗る主体」と「サーバが記録した主体」のどちらを信じるかを
// 決めきれていない状態だからである。
func TestAccountTokenWithMismatchedUserIDChangesNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	token := fixture.issueApiToken(
		t, idmRefusalAlice,
		apitokendomain.ScopeAccountRead, apitokendomain.ScopeAccountWrite,
	)
	// 保存されたトークンの主体だけを bob へ書き換える。JWT の `sub` は alice のまま。
	stored, err := fixture.apiTokenDB.List(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("前提が壊れている: 保存されたトークンが %d 件", len(stored))
	}
	stored[0].UserID = idmRefusalBob
	if err := fixture.apiTokenDB.Save(context.Background(), stored[0]); err != nil {
		t.Fatal(err)
	}

	beforeAlice := *fixture.user(t, idmRefusalAlice)
	beforeBob := *fixture.user(t, idmRefusalBob)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: token,
		body: map[string]any{"name": "renamed-by-mismatched-token"},
	})
	if refused.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_token" {
		t.Fatalf("error code=%q, want invalid_token", code)
	}
	// 拒否がどちらの利用者も変えていないこと。どちらか一方でも動けば、
	// サーバは食い違いをどちらかへ丸めて操作を通したことになる。
	for _, expected := range []struct {
		sub    string
		before userdomain.User
	}{{idmRefusalAlice, beforeAlice}, {idmRefusalBob, beforeBob}} {
		after := fixture.user(t, expected.sub)
		if after.Name == nil || *after.Name != *expected.before.Name ||
			!after.UpdatedAt.Equal(expected.before.UpdatedAt) {
			t.Fatalf("%s が変わった: name=%v updated_at=%s", expected.sub, after.Name, after.UpdatedAt)
		}
	}

	// 対照: 食い違いを戻すと同じトークンで同じ変更が通る。
	stored[0].UserID = idmRefusalAlice
	if err := fixture.apiTokenDB.Save(context.Background(), stored[0]); err != nil {
		t.Fatal(err)
	}
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: token,
		body: map[string]any{"name": "renamed-by-consistent-token"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 一致したトークンの変更が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if name := fixture.user(t, idmRefusalAlice).Name; name == nil || *name != "renamed-by-consistent-token" {
		t.Fatalf("前提が壊れている: 一致したトークンでも表示名が変わらない: %v", name)
	}
}

// EX-IDMANAGEMENT-003-02: CSRF トークンと Cookie が一致しないメールアドレスの確認は
// `InvalidRequestError` で拒否され、メールアドレスは確認済みにならない。
//
// 確認は一度きりのトークンを消費する。したがって「拒否が変えなかったもの」には、
// 対象のメールアドレスに加えて、そのトークンが今も使えることが含まれる。
// 拒否のついでにトークンを消費する実装は、状態だけを見ると成功と区別できない。
func TestEmailConfirmationWithMismatchedCSRFVerifiesNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	alice := fixture.seedSession(t, "sess-alice-email", tenancydomain.DefaultTenantID, idmRefusalAlice, withFreshStepUp)

	// 未認証でも開かれる確認画面が CSRF 境界を張る。ここは正常系の前提。
	verifyContext := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/account/v1/email/verify_context",
	})
	if verifyContext.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 確認の文脈が status=%d body=%s", verifyContext.Code, verifyContext.Body.String())
	}

	requested := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/change_request",
		sessionID: alice, csrf: idmRefusalCSRF, body: map[string]any{"new_email": "moved@example.test"},
	})
	if requested.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 変更申請が status=%d body=%s", requested.Code, requested.Body.String())
	}
	if len(fixture.emails.Sent) != 1 {
		t.Fatalf("前提が壊れている: 確認メールが %d 通", len(fixture.emails.Sent))
	}
	match := emailChangeTokenPattern.FindStringSubmatch(fixture.emails.Sent[0].Text)
	if match == nil {
		t.Fatalf("確認メールにトークンが無い: %s", fixture.emails.Sent[0].Text)
	}
	token := match[1]

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/verify",
		csrf: idmRefusalCSRF, csrfCookie: idmRefusalCSRF + "-other",
		body: map[string]any{"token": token},
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "csrf_failed" {
		t.Fatalf("error code=%q, want csrf_failed", code)
	}
	// 拒否が何も変えていないこと。プライマリメールアドレスは元のままである。
	if email := fixture.user(t, idmRefusalAlice).Email; email == nil || *email != idmRefusalAlice+"@example.test" {
		t.Fatalf("CSRF を拒否したのにメールアドレスが変わった: %v", email)
	}

	// 対照: 同じトークンを一致した CSRF で送ると確認は通る。拒否がトークンを
	// 消費していれば、ここで通らない。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/verify",
		csrf: idmRefusalCSRF, body: map[string]any{"token": token},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 一致した CSRF の確認が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if email := fixture.user(t, idmRefusalAlice).Email; email == nil || *email != "moved@example.test" {
		t.Fatalf("前提が壊れている: 一致した CSRF でもメールアドレスが変わらない: %v", email)
	}
}
