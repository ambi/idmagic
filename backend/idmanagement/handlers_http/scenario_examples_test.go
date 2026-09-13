package handlers_http_test

// IdManagement が宣言する具体例のうち、通常経路を HTTP 境界で観測するもの。
// 拒否の側は refusal_effects_test.go とその兄弟が持つ。組み立てとヘルパーは
// refusal_effects_test.go の `idmRefusalFixture` を共有する。
//
// ここに置く判断基準は「use case を直接呼ぶテストでは通らないか」である。
// スコープの検査、二重送信 CSRF、ロールによる管理 API の境界はいずれも
// ミドルウェアと配線に宿っており、use case からは見えない。

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 具体例の 2 つの `Then` に 2 つの観測を置く。変更の側は応答だけでなく、保存層と
// 送信済みの確認メールを読み直す。**この向きの観測が要る理由は 002-02 の対偶である。**
// 「だけを許可する」の拒否側は 002-02 が持つが、拒否側だけでは何も通さない実装も通ってしまう。
//
//spec:covers EX-IDMANAGEMENT-002-01: account:read が通す 3 つの参照と、account:write が通すプロフィールの変更とメールアドレス変更の申請。
func TestAccountScopesAllowTheReadsAndWritesTheyName(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	readOnly := fixture.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountRead)

	for _, read := range []struct {
		name, path string
	}{
		{"概要", "/api/account/v1/summary"},
		{"プロフィール", "/api/account/v1/profile"},
		{"データエクスポート", "/api/account/v1/data_export"},
	} {
		t.Run(read.name, func(t *testing.T) {
			response := fixture.send(t, idmRefusalRequest{
				method: http.MethodGet, path: read.path, bearer: readOnly,
			})
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s, want 200", response.Code, response.Body.String())
			}
			// 参照が返すのはトークンが固定した利用者のものであること。
			if !strings.Contains(response.Body.String(), idmRefusalAlice) {
				t.Fatalf("%s の応答が %s を含まない: %s", read.name, idmRefusalAlice, response.Body.String())
			}
		})
	}

	writable := fixture.issueApiToken(t, idmRefusalAlice, apitokendomain.ScopeAccountWrite)

	renamed := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/account/v1/profile", bearer: writable,
		body: map[string]any{"name": "renamed-by-account-write"},
	})
	if renamed.Code != http.StatusOK {
		t.Fatalf("account:write の変更が status=%d body=%s", renamed.Code, renamed.Body.String())
	}
	// 応答は use case の戻り値から組み立てられる。保存されたかどうかは読み直して見る。
	if name := fixture.user(t, idmRefusalAlice).Name; name == nil || *name != "renamed-by-account-write" {
		t.Fatalf("表示名が保存されていない: %v", name)
	}

	requested := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/change_request", bearer: writable,
		body: map[string]any{"new_email": "moved@example.test"},
	})
	if requested.Code != http.StatusNoContent {
		t.Fatalf("account:write の変更申請が status=%d body=%s", requested.Code, requested.Body.String())
	}
	// 変更申請は 204 で終わる。申請が成立したことは、確認メールが出たことでしか読めない。
	if sent := len(fixture.emails.Sent); sent != 1 {
		t.Fatalf("確認メールが %d 通", sent)
	}
}

// **固定値ではなく画面が配った値で送る。** テスト側が知っている定数で二重送信を
// 組み立てると、応答が CSRF トークンも Cookie も返していない実装でも同じテストが通る。
// `SameSite` は Cookie 属性なので、応答の `Set-Cookie` から読む。
//
//spec:covers EX-IDMANAGEMENT-003-01: 未認証で開ける確認画面が CSRF トークンと SameSite 付き Cookie を配ること、その組で確認が受理されること。
func TestEmailVerifyContextEstablishesTheCSRFBoundaryItThenAccepts(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	alice := fixture.seedSession(
		t, "sess-alice-verify-context", tenancydomain.DefaultTenantID, idmRefusalAlice, withFreshStepUp,
	)

	// 未認証で確認の文脈を取る。ここに認証を付けないことが具体例の前提である。
	verifyContext := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/account/v1/email/verify_context",
	})
	if verifyContext.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", verifyContext.Code, verifyContext.Body.String())
	}
	var context struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(verifyContext.Body.Bytes(), &context); err != nil {
		t.Fatalf("decode verify_context: %v body=%s", err, verifyContext.Body.String())
	}
	if context.CSRFToken == "" {
		t.Fatalf("応答に CSRF トークンが無い: %s", verifyContext.Body.String())
	}
	cookies := verifyContext.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("応答に Cookie が無い")
	}
	issued := cookies[0]
	if issued.SameSite == http.SameSiteDefaultMode {
		t.Fatalf("Cookie に SameSite 属性が無い: %+v", issued)
	}

	// 変更申請を出し、確認リンクのトークンを取る。ここは 002-01 が固定する経路。
	requested := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/change_request",
		sessionID: alice, csrf: idmRefusalCSRF, body: map[string]any{"new_email": "moved@example.test"},
	})
	if requested.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 変更申請が status=%d body=%s", requested.Code, requested.Body.String())
	}
	match := emailChangeTokenPattern.FindStringSubmatch(fixture.emails.Sent[0].Text)
	if match == nil {
		t.Fatalf("確認メールにトークンが無い: %s", fixture.emails.Sent[0].Text)
	}

	// 画面が配ったトークンと Cookie の組で確認を送る。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/account/v1/email/verify",
		csrf: context.CSRFToken, csrfCookie: issued.Value, body: map[string]any{"token": match[1]},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("確認が status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
	// 受理は応答ではなく、プライマリメールアドレスが動いたことで読む。
	if email := fixture.user(t, idmRefusalAlice).Email; email == nil || *email != "moved@example.test" {
		t.Fatalf("受理されたのにメールアドレスが変わっていない: %v", email)
	}
}

// 具体例の 2 つの `Then` に 2 つの観測を置く。発行の側はイベントの記録を読む。
// 状態は変えるが記録を残さない実装は、一覧だけを見るテストでは通る。
//
//spec:covers EX-IDMANAGEMENT-014-01: admin ロールを持つ操作者の作成が UserCreated を発行し、その利用者が同じ操作者の一覧に現れること。
func TestAdminWithRoleCreatesAUserThatThenAppearsInTheList(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-create", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"preferred_username": "bob", "password": "scenario-password-1234"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s, want 201", created.Code, created.Body.String())
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if !emittedFor(fixture, "UserCreated", body.ID) {
		t.Fatalf("UserCreated が %s に対して発行されていない: %v", body.ID, emittedTypes(fixture))
	}

	listed := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users", sessionID: admin,
	})
	if listed.Code != http.StatusOK {
		t.Fatalf("一覧が status=%d body=%s", listed.Code, listed.Body.String())
	}
	if !strings.Contains(listed.Body.String(), `"bob"`) {
		t.Fatalf("一覧に bob が現れない: %s", listed.Body.String())
	}
}

// emittedFor は指定した種類のイベントが、対象を名指して発行されたかを返す。
// イベントの型は Context をまたいで多いので、JSON へ落として対象 id の出現で読む。
// 種類だけを数えると、別の利用者に対する同じ種類のイベントと区別できない。
func emittedFor(fixture *idmRefusalFixture, eventType, targetID string) bool {
	for _, event := range *fixture.events {
		if event.EventType() != eventType {
			continue
		}
		encoded, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if strings.Contains(string(encoded), targetID) {
			return true
		}
	}
	return false
}

func emittedTypes(fixture *idmRefusalFixture) []string {
	types := make([]string, 0, len(*fixture.events))
	for _, event := range *fixture.events {
		types = append(types, event.EventType())
	}
	return types
}

// 具体例は 4 段の連なりで書かれている。段ごとに別のテストへ散らすと、
// 「登録した Agent にバインドし、無効化し、再有効化して一覧に戻る」という
// 連なり自体が誰の持ち物でもなくなる。各段は応答ではなく、その後の参照で読む。
//
//spec:covers EX-IDMANAGEMENT-009-01: 区分を指定した Agent の登録、資格情報のバインド、無効化、再有効化と一覧への再出現。
func TestAgentRegistrationBindingDisableAndEnableRoundTrip(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-agent", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "batch-agent-2", "kind": "supervised"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("登録が status=%d body=%s", created.Code, created.Body.String())
	}
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	// 「指定した区分で登録される」。既定値へ丸める実装は、201 だけでは通る。
	if kind := fixture.agent(t, registered.ID).Kind; string(kind) != "supervised" {
		t.Fatalf("kind=%q, want supervised", kind)
	}

	bound := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents/" + registered.ID + "/credentials",
		sessionID: admin, csrf: idmRefusalCSRF, body: map[string]any{"client_id": idmRefusalClient},
	})
	if bound.Code != http.StatusNoContent {
		t.Fatalf("バインドが status=%d body=%s", bound.Code, bound.Body.String())
	}
	// バインドは 204 で終わる。関連付けが残ったことは参照でしか読めない。
	if ids := agentCredentialIDs(t, fixture, admin, registered.ID); !slices.Contains(ids, idmRefusalClient) {
		t.Fatalf("client_ids=%v, want it to contain %s", ids, idmRefusalClient)
	}

	disabled := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents/" + registered.ID + "/disable",
		sessionID: admin, csrf: idmRefusalCSRF,
	})
	if disabled.Code != http.StatusNoContent {
		t.Fatalf("無効化が status=%d body=%s", disabled.Code, disabled.Body.String())
	}
	if status := fixture.agent(t, registered.ID).Status; string(status) != "disabled" {
		t.Fatalf("status=%q, want disabled", status)
	}

	enabled := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents/" + registered.ID + "/enable",
		sessionID: admin, csrf: idmRefusalCSRF,
	})
	if enabled.Code != http.StatusNoContent {
		t.Fatalf("再有効化が status=%d body=%s", enabled.Code, enabled.Body.String())
	}
	listed := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/agents", sessionID: admin,
	})
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), "batch-agent-2") {
		t.Fatalf("一覧が status=%d body=%s", listed.Code, listed.Body.String())
	}
}

// agentCredentialIDs は管理 API 経由で Agent に残っている関連付けを読む。
func agentCredentialIDs(
	t *testing.T, fixture *idmRefusalFixture, sessionID, agentID string,
) []string {
	t.Helper()
	response := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/agents/" + agentID, sessionID: sessionID,
	})
	if response.Code != http.StatusOK {
		t.Fatalf("Agent の参照が status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		ClientIDs []string `json:"client_ids"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	return body.ClientIDs
}

// 具体例は「テナント acme の Agent にテナント default の client_id を指定する」と書く。
// 拒否だけでなく、越境が関連付けを残さないことまで読む。バインドは 204 で終わるので、
// 「拒否を書いてから関連付ける」実装は応答では見分けられない。
//
//spec:covers REQ-IDMANAGEMENT-009: 別テナントの client_id を指定したバインドが拒否され、Agent に関連付けが残らないこと。拒否の形 (EX-IDMANAGEMENT-009-04 が言う InvalidRequestError) は wi-573 が持つ。
func TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	foreignAdmin := fixture.seedSession(
		t, "sess-admin-acme-bind", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)
	// acme のテナントに Agent を 1 つ建てる。越境の対象になる側である。
	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/agents",
		sessionID: foreignAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "acme-agent", "kind": "autonomous"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: acme の Agent 登録が status=%d body=%s", created.Code, created.Body.String())
	}
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register: %v", err)
	}

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/agents/" + registered.ID + "/credentials",
		sessionID: foreignAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"client_id": idmRefusalClient},
	})
	if refused.Code == http.StatusNoContent {
		t.Fatalf("越境した client_id のバインドが受理された: %s", refused.Body.String())
	}
	// 拒否が関連付けを残していないこと。
	ids := agentCredentialIDsIn(t, fixture, idmRefusalOtherTenant, foreignAdmin, registered.ID)
	if len(ids) != 0 {
		t.Fatalf("拒否されたのに client_ids=%v", ids)
	}

	// 対照: 同じテナントの client_id なら同じ要求が通る。これが無いと
	// 「そもそもバインドが動かない構成だった」と区別できない。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/agents/" + registered.ID + "/credentials",
		sessionID: foreignAdmin, csrf: idmRefusalCSRF,
		body: map[string]any{"client_id": idmRefusalForeignClient},
	})
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: 同テナントのバインドが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

func agentCredentialIDsIn(
	t *testing.T, fixture *idmRefusalFixture, tenantID, sessionID, agentID string,
) []string {
	t.Helper()
	response := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, tenantID: tenantID,
		path: "/api/admin/v1/agents/" + agentID, sessionID: sessionID,
	})
	if response.Code != http.StatusOK {
		t.Fatalf("Agent の参照が status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		ClientIDs []string `json:"client_ids"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	return body.ClientIDs
}

// 具体例は 4 つの `Then` で「どのスコープが何を通すか」を言う。拒否の側は
// 025-02 から 025-05 が持つが、**拒否だけでは何も通さない実装も通ってしまう。**
// ここは逆向きに、名指された操作が `insufficient_scope` にならないことを読む。
//
// CSV インポートの適用のように、スコープを通ったあとで別の理由 (プレビューが無い)
// で落ちる経路がある。そこはスコープの検査を抜けたことだけを主張し、経路の正しさは
// 026-05 と 029-08 が持つ。
//
//spec:covers EX-IDMANAGEMENT-025-01: users:read が User の参照と CSV エクスポートの 4 操作を、groups:read が Group の参照と動的規則のプレビューを、groups:write が Group の変更と CSV インポートのプレビューと適用を、agents:write が Agent のキルと削除を通すこと。
func TestManagementScopesAllowTheOperationsTheyName(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	usersRead := fixture.issueApiToken(t, idmRefusalAdmin, apitokendomain.ScopeUsersRead)
	groupsRead := fixture.issueApiToken(t, idmRefusalAdmin, apitokendomain.ScopeGroupsRead)
	groupsWrite := fixture.issueApiToken(t, idmRefusalAdmin, apitokendomain.ScopeGroupsWrite)
	agentsWrite := fixture.issueApiToken(t, idmRefusalAdmin, apitokendomain.ScopeAgentsWrite)

	t.Run("users:read は参照と CSV エクスポートを通す", func(t *testing.T) {
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodGet, path: "/api/admin/v1/users", bearer: usersRead,
		})
		// エクスポートは User を変更しないので users:read で開始できる。
		started := fixture.send(t, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/users/exports", bearer: usersRead,
			body: map[string]any{"columns": []string{"preferred_username", "email"}},
		})
		exportID := startedExportID(t, started)
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID, bearer: usersRead,
		})
		fixture.runExport(t, exportID)
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodGet, path: "/api/admin/v1/users/exports/" + exportID + "/file", bearer: usersRead,
		})
		// 取り消しは終端前にしか意味がないので、別の 1 件を queued のまま取り消す。
		cancelable := startedExportID(t, fixture.send(t, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/users/exports", bearer: usersRead,
			body: map[string]any{"columns": []string{"preferred_username"}},
		}))
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/users/exports/" + cancelable + "/cancel",
			bearer: usersRead,
		})
	})

	t.Run("groups:read は参照と動的規則のプレビューを通す", func(t *testing.T) {
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodGet, path: "/api/admin/v1/groups", bearer: groupsRead,
		})
		// プレビューは Group を変更しないので groups:read で通る。
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost,
			path:   "/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/dynamic-rule/preview",
			bearer: groupsRead,
			body:   map[string]any{"expression": `user.email_verified == true`, "user_ids": []string{idmRefusalAlice}},
		})
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/groups/exports", bearer: groupsRead,
			body: map[string]any{"columns": []string{"id", "name"}},
		})
	})

	t.Run("groups:write は変更と CSV インポートを通す", func(t *testing.T) {
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/groups", bearer: groupsWrite,
			body: map[string]any{"name": "scope-created-group"},
		})
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/groups/imports", bearer: groupsWrite,
			contentType: "text/csv", rawBody: "name,roles\nsales,catalog:read\n",
		})
		// 適用はスコープを抜けたあとにプレビューの解決で落ちる。ここが主張するのは
		// 「スコープでは拒否されない」ことだけである。
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/groups/imports/any-preview-job/apply",
			bearer: groupsWrite,
		})
	})

	t.Run("agents:write はキルと削除の両方を通す", func(t *testing.T) {
		// キルは取り返しが付かず、キル済みの Agent は削除できない。2 体に分ける。
		second := fixture.registerAgent(t, agentsWrite, "scope-agent")
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/agents/" + idmRefusalAgent + "/kill",
			bearer: agentsWrite,
		})
		assertScopeAdmits(t, fixture, idmRefusalRequest{
			method: http.MethodDelete, path: "/api/admin/v1/agents/" + second, bearer: agentsWrite,
		})
	})
}

// assertScopeAdmits は、要求がスコープの検査で拒否されないことだけを主張する。
// **成功そのものは主張しない。** 経路の正しさは各具体例が別に持っており、ここが
// 落とすのは「そのスコープでは 1 つも通さない」実装である。
func assertScopeAdmits(t *testing.T, fixture *idmRefusalFixture, request idmRefusalRequest) {
	t.Helper()
	response := fixture.send(t, request)
	if response.Code == http.StatusForbidden {
		t.Fatalf("%s %s がスコープで拒否された: %s", request.method, request.path, response.Body.String())
	}
	if code := idmProblemCode(t, response); code == "insufficient_scope" {
		t.Fatalf("%s %s が insufficient_scope を返した", request.method, request.path)
	}
}

// registerAgent は API アクセストークンで Agent を 1 体登録し、その id を返す。
func (f *idmRefusalFixture) registerAgent(t *testing.T, bearer, name string) string {
	t.Helper()
	response := f.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents", bearer: bearer,
		body: map[string]any{"name": name, "kind": "autonomous"},
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("Agent の登録が status=%d body=%s", response.Code, response.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("Agent id を読めない: %s", response.Body.String())
	}
	return created.ID
}
