package handlers_http_test

// 予約ロール `system_admin` の保存規則を、production と同じ HTTP の境界で確かめる
// (REQ-IDMANAGEMENT-032)。
//
// 判定はユースケース境界に置くが、観測はここで行う。JSON API、CSV、Agent API の
// どれか 1 本だけへ検査を差し込んだ実装は、ユースケースの単体テストでは通ってしまう。
// 経路を選ぶのは組み立てとルーティングなので、そこを通さないと「配線が 1 本抜けている」
// を検出できない。
//
// 拒否のテストは応答と「拒否が変えなかった状態」の両方を読む。422 を書いてから保存する
// 実装は、ステータスだけを読むテストを素通りする。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// reservedRole は制御面 User のために予約されたロール名。
const reservedRole = "system_admin"

// groupsIn は指定したテナントに残っているグループ名を返す。既定テナントに固定された
// groupNames とは別に要る。予約ロールの拒否は制御面の外側で観測するものだからである。
func (f *idmRefusalFixture) groupsIn(t *testing.T, tenantID string) []string {
	t.Helper()
	groups, err := f.groups.ListAll(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	return names
}

// agentIn は指定したテナントの Agent を読み直す。
func (f *idmRefusalFixture) agentIn(t *testing.T, tenantID, id string) *agentdomain.Agent {
	t.Helper()
	agent, err := f.agents.FindByID(context.Background(), tenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	return agent
}

// decodeJSONBody は応答本文を target へ読み出す。
func decodeJSONBody(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("応答本文を読めない: %v body=%s", err, recorder.Body.String())
	}
}

// emitted は発行済みイベントに指定した型のものが含まれるかを返す。
func emitted[E spec.DomainEvent](events []spec.DomainEvent) bool {
	for _, event := range events {
		if _, ok := event.(E); ok {
			return true
		}
	}
	return false
}

// 制御面 Group へ付与した `system_admin` が、所属する制御面 User の有効ロールに現れる。
//
// 主要ユースケース control-plane-group-grants-system-admin の E2E 証拠。観測点を
// `GET /api/admin/v1/users/{sub}/groups` の `effective_roles` に置くのは、Group が
// システム運用者になるのではなく制御面 User の有効ロールへロールを配る束だという
// 設計を、そのまま読める場所がここだけだからである。
//
//spec:covers EX-IDMANAGEMENT-032-01: 制御面テナントの Group への `system_admin` の付与が受理され、所属させた制御面 User の実効ロールに現れ、`group_roles` 側に載って `direct_roles` は空のままであること。
func TestControlPlaneGroupGrantsSystemAdminToItsMembers(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-cp-admin", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/groups", sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "system-operators", "roles": []string{reservedRole}},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("制御面 Group の作成が status=%d body=%s, want 201", created.Code, created.Body.String())
	}
	group := fixture.groupByName(t, tenancydomain.DefaultTenantID, "system-operators")
	if !slices.Contains(group.Roles, reservedRole) {
		t.Fatalf("作成された Group の roles=%v が %q を含まない", group.Roles, reservedRole)
	}

	joined := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, sessionID: admin, csrf: idmRefusalCSRF,
		path: "/api/admin/v1/groups/" + group.ID + "/members/" + idmRefusalBob,
		body: map[string]any{},
	})
	if joined.Code >= http.StatusBadRequest {
		t.Fatalf("所属の追加が status=%d body=%s", joined.Code, joined.Body.String())
	}

	listed := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users/" + idmRefusalBob + "/groups", sessionID: admin,
	})
	if listed.Code != http.StatusOK {
		t.Fatalf("所属グループの取得が status=%d body=%s", listed.Code, listed.Body.String())
	}
	var view struct {
		DirectRoles    []string `json:"direct_roles"`
		GroupRoles     []string `json:"group_roles"`
		EffectiveRoles []string `json:"effective_roles"`
	}
	decodeJSONBody(t, listed, &view)
	if !slices.Contains(view.EffectiveRoles, reservedRole) {
		t.Fatalf("実効ロール %v が %q を含まない", view.EffectiveRoles, reservedRole)
	}
	if !slices.Contains(view.GroupRoles, reservedRole) {
		t.Fatalf("group_roles %v が %q を含まない", view.GroupRoles, reservedRole)
	}
	if len(view.DirectRoles) != 0 {
		t.Fatalf("direct_roles=%v, want empty", view.DirectRoles)
	}
}

// 制御面テナントの User へは `system_admin` を直接割り当てられる。
//
// 拒否の側だけを実装すると、対象の所属テナントを見ずに一律で拒否する実装が
// 拒否のテストをすべて通してしまう。許可の側をここで固定する。
//
//spec:covers EX-IDMANAGEMENT-032-02: 制御面テナントの User への `system_admin` の直接付与が受理され、保存後の `roles` に現れること。
func TestReservedRoleStaysAssignableToControlPlaneUsers(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-cp-direct", tenancydomain.DefaultTenantID, idmRefusalAdmin)

	updated := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/admin/v1/users/" + idmRefusalAlice,
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"roles": []string{reservedRole}},
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("制御面 User への直接付与が status=%d body=%s, want 200", updated.Code, updated.Body.String())
	}
	if roles := fixture.user(t, idmRefusalAlice).Roles; !slices.Contains(roles, reservedRole) {
		t.Fatalf("保存後の roles=%v が %q を含まない", roles, reservedRole)
	}
}

// 制御面テナント以外の User への直接付与は拒否され、同じ要求の別項目も保存されない。
//
//spec:covers EX-IDMANAGEMENT-032-03: 制御面テナント以外のテナントの User への `system_admin` の付与が `invalid_role` で拒否され、roles も同じ要求に含めた表示名も `updated_at` も変わらないこと。
func TestReservedRoleRefusedOutsideTheControlPlaneChangesNoUser(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(
		t, "sess-foreign-admin", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)
	before := *fixture.user(t, idmRefusalForeignMember)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/users/" + idmRefusalForeignMember,
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"roles": []string{reservedRole}, "name": "renamed-alongside-the-reserved-role"},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_role" {
		t.Fatalf("error code=%q, want invalid_role", code)
	}
	// 拒否が何も変えていないこと。roles だけでなく、同じ要求に相乗りした表示名も動かない。
	after := fixture.user(t, idmRefusalForeignMember)
	if slices.Contains(after.Roles, reservedRole) {
		t.Fatalf("拒否されたのに roles=%v へ %q が入った", after.Roles, reservedRole)
	}
	if after.Name == nil || *after.Name != *before.Name {
		t.Fatalf("拒否されたのに表示名が変わった: %v", after.Name)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("拒否されたのに updated_at が動いた: %s -> %s", before.UpdatedAt, after.UpdatedAt)
	}

	// 対照: 予約ロール以外の名前なら同じ要求が通る。これが無いと「このテナントでは
	// そもそもロールを更新できない構成だった」と区別できない。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/users/" + idmRefusalForeignMember,
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"roles": []string{"catalog:read"}},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 通常のロール更新が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// 制御面テナント以外の Group への付与は拒否され、Group は作成されない。
//
//spec:covers EX-IDMANAGEMENT-032-04: 制御面テナント以外のテナントで `system_admin` を持つ Group の作成が `invalid_role` で拒否され、Group も "GroupCreated" も生じないこと。
func TestReservedRoleRefusedOutsideTheControlPlaneCreatesNoGroup(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(
		t, "sess-foreign-group", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)
	before := fixture.groupsIn(t, idmRefusalOtherTenant)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/groups",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "escalation", "roles": []string{reservedRole}},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_role" {
		t.Fatalf("error code=%q, want invalid_role", code)
	}
	// 拒否が Group を作っていないこと。422 を書いてから保存する実装と区別する。
	if after := fixture.groupsIn(t, idmRefusalOtherTenant); !sameStrings(before, after) {
		t.Fatalf("拒否されたのにグループの集合が変わった: before=%v after=%v", before, after)
	}
	if emitted[*idmdomain.GroupCreated](*fixture.events) {
		t.Fatal("拒否されたのに GroupCreated が発行された")
	}

	// 対照: 予約ロール以外の名前なら同じテナントで作成が通る。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/groups",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "escalation", "roles": []string{"catalog:read"}},
	})
	if accepted.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: 通常ロールの作成が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// Agent への付与は、制御面テナントであっても拒否される。
//
// 制御面テナントの Agent を先に置くのは、テナントの規則からは導けない唯一の行だからである。
// User と Group を許可するテナントで Agent だけが拒否されることを、規則表の代わりに固定する。
//
//spec:covers EX-IDMANAGEMENT-032-05: 制御面テナントの Agent への `system_admin` の付与が `invalid_role` で拒否され、`roles` も `updated_at` も変わらず "AgentUpdated" も発行されないこと。制御面以外のテナントの Agent も同じ扱いになること。
func TestReservedRoleRefusedForAgentsInEveryTenant(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	controlPlane := fixture.seedSession(
		t, "sess-agent-cp", tenancydomain.DefaultTenantID, idmRefusalAdmin,
	)
	foreign := fixture.seedSession(
		t, "sess-agent-foreign", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)

	// 制御面以外のテナントには Agent が居ないので、通常ロールで 1 体登録してから更新を試す。
	registered := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/agents",
		sessionID: foreign, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "foreign-agent", "kind": "autonomous", "roles": []string{"catalog:read"}},
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: 通常ロールの Agent 登録が status=%d body=%s", registered.Code, registered.Body.String())
	}
	var foreignAgent struct {
		ID string `json:"id"`
	}
	decodeJSONBody(t, registered, &foreignAgent)

	for _, target := range []struct {
		name, tenantID, session, agentID string
	}{
		{"制御面テナント", tenancydomain.DefaultTenantID, controlPlane, idmRefusalAgent},
		{"制御面以外のテナント", idmRefusalOtherTenant, foreign, foreignAgent.ID},
	} {
		t.Run(target.name, func(t *testing.T) {
			before := *fixture.agentIn(t, target.tenantID, target.agentID)
			*fixture.events = nil

			refused := fixture.send(t, idmRefusalRequest{
				method: http.MethodPatch, tenantID: target.tenantID,
				path:      "/api/admin/v1/agents/" + target.agentID,
				sessionID: target.session, csrf: idmRefusalCSRF,
				body: map[string]any{"roles": []string{reservedRole}},
			})
			if refused.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "invalid_role" {
				t.Fatalf("error code=%q, want invalid_role", code)
			}
			// 拒否が Agent を変えていないこと。
			after := fixture.agentIn(t, target.tenantID, target.agentID)
			if slices.Contains(after.Roles, reservedRole) {
				t.Fatalf("拒否されたのに roles=%v へ %q が入った", after.Roles, reservedRole)
			}
			if !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Fatalf("拒否されたのに updated_at が動いた: %s -> %s", before.UpdatedAt, after.UpdatedAt)
			}
			if emitted[*idmdomain.AgentUpdated](*fixture.events) {
				t.Fatal("拒否されたのに AgentUpdated が発行された")
			}
		})
	}

	// 登録の側も同じ検査を通ること。更新だけを直した実装と区別する。
	refusedRegistration := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents",
		sessionID: controlPlane, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "privileged-agent", "kind": "autonomous", "roles": []string{reservedRole}},
	})
	if refusedRegistration.Code != http.StatusUnprocessableEntity {
		t.Fatalf("登録が status=%d body=%s, want 422", refusedRegistration.Code, refusedRegistration.Body.String())
	}
	agents, err := fixture.agents.ListAll(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range agents {
		if agent.Name == "privileged-agent" {
			t.Fatal("拒否されたのに Agent が登録された")
		}
	}
}

// groupByName は指定したテナントのグループを名前で読み直す。
func (f *idmRefusalFixture) groupByName(t *testing.T, tenantID, name string) *groupdomain.Group {
	t.Helper()
	groups, err := f.groups.ListAll(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range groups {
		if group.Name == name {
			return group
		}
	}
	t.Fatalf("グループ %q がテナント %q に無い", name, tenantID)
	return nil
}

// 作成の側も同じ検査を通る。更新だけを直した実装はここで落ちる。
//
// 作成は対象が存在しないので「拒否が変えなかった状態」の読み方が更新と違う。
// 対象が保存層に現れないことと、"UserCreated" が発行されないことの両方で読む。
//
//spec:covers EX-IDMANAGEMENT-032-09: 制御面テナント以外のテナントでの `system_admin` を持つ User の作成が `invalid_role` で拒否され、User も "UserCreated" も生じないこと。
func TestReservedRoleRefusedOutsideTheControlPlaneCreatesNoUser(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(
		t, "sess-foreign-create", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/users",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{
			"preferred_username": "mallory", "password": "reserved-role-password-1234",
			"roles": []string{reservedRole},
		},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_role" {
		t.Fatalf("error code=%q, want invalid_role", code)
	}
	// 拒否が User を作っていないこと。
	created, err := fixture.users.FindByUsername(context.Background(), idmRefusalOtherTenant, "mallory")
	if err != nil {
		t.Fatal(err)
	}
	if created != nil {
		t.Fatalf("拒否されたのに User が作られた: %+v", created)
	}
	if emitted[*idmdomain.UserCreated](*fixture.events) {
		t.Fatal("拒否されたのに UserCreated が発行された")
	}

	// 対照: 予約ロール以外の名前なら同じ作成が通る。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/users",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{
			"preferred_username": "mallory", "password": "reserved-role-password-1234",
			"roles": []string{"catalog:read"},
		},
	})
	if accepted.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: 通常ロールの作成が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// Group の更新も同じ検査を通る。作成だけを直した実装はここで落ちる。
//
//spec:covers EX-IDMANAGEMENT-032-10: 制御面テナント以外のテナントの Group の `roles` を `system_admin` へ更新する要求が `invalid_role` で拒否され、`roles` も同じ要求に含めた説明も `updated_at` も変わらないこと。
func TestReservedRoleRefusedOutsideTheControlPlaneChangesNoGroup(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(
		t, "sess-foreign-group-update", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)

	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, tenantID: idmRefusalOtherTenant, path: "/api/admin/v1/groups",
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"name": "engineering", "roles": []string{"catalog:read"}},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: Group の作成が status=%d body=%s", created.Code, created.Body.String())
	}
	before := *fixture.groupByName(t, idmRefusalOtherTenant, "engineering")

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/groups/" + before.ID,
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{
			"roles": []string{reservedRole}, "description": "escalation path",
		},
	})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "invalid_role" {
		t.Fatalf("error code=%q, want invalid_role", code)
	}
	// 拒否が Group を変えていないこと。roles だけでなく相乗りした説明も動かない。
	after := fixture.groupByName(t, idmRefusalOtherTenant, "engineering")
	if !slices.Equal(after.Roles, before.Roles) {
		t.Fatalf("拒否されたのに roles=%v が変わった (before=%v)", after.Roles, before.Roles)
	}
	if after.Description != nil {
		t.Fatalf("拒否されたのに説明 %q が入った", *after.Description)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("拒否されたのに updated_at が動いた: %s -> %s", before.UpdatedAt, after.UpdatedAt)
	}

	// 対照: 予約ロール以外の名前なら同じ更新が通り、説明も入る。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, tenantID: idmRefusalOtherTenant,
		path:      "/api/admin/v1/groups/" + before.ID,
		sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"roles": []string{"catalog:write"}, "description": "escalation path"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 通常ロールの更新が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}
