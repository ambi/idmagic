package handlers_http_test

// 自己削除の禁止と動的グループの手動操作の禁止について、応答と
// 「拒否が変えなかった状態」の両方を確かめる。組み立てとヘルパーは
// refusal_effects_test.go が持つ。
//
// どちらも「操作者は管理者として認可されている」ところから先の拒否である。
// ロールとスコープを通ったあとに立つ防護なので、認可の層を落としても現れない。

import (
	"context"
	"net/http"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// EX-IDMANAGEMENT-013-02: 対象が操作者自身であり `admin` または `system_admin` を
// 持つ場合、削除の予約、復元、完全削除のいずれも `self_delete_forbidden` で拒否され、
// 対象ユーザーは `Active` のまま在籍する。
//
// 削除の予約は状態遷移だけで、応答も本文を持たない。予約が通ったかどうかは対象を
// 読み直すほかに知る方法がない。完全削除は取り返しがつかないので、拒否が
// 「応答を書いてから cascade も走る」形になっていないことまで確かめる。
func TestSelfDeleteRefusalKeepsTheAdministratorActive(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-self", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	before := *fixture.user(t, idmRefusalAdmin)

	for _, refusal := range []struct{ name, method, path string }{
		{"削除の予約", http.MethodDelete, "/api/admin/v1/users/" + idmRefusalAdmin},
		{"完全削除", http.MethodDelete, "/api/admin/v1/users/" + idmRefusalAdmin + "?purge=true"},
		{"復元", http.MethodPost, "/api/admin/v1/users/" + idmRefusalAdmin + "/restore"},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path, sessionID: admin, csrf: idmRefusalCSRF,
			})
			if refused.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "self_delete_forbidden" {
				t.Fatalf("error code=%q, want self_delete_forbidden", code)
			}
			// 拒否が何も変えていないこと。状態も、匿名化で消える識別子も、そのままである。
			after := fixture.user(t, idmRefusalAdmin)
			if after.Lifecycle.EffectiveStatus() != idmdomain.UserStatusActive {
				t.Fatalf("拒否されたのに状態が %s になった", after.Lifecycle.EffectiveStatus())
			}
			if after.PreferredUsername != before.PreferredUsername || after.Email == nil ||
				*after.Email != *before.Email {
				t.Fatalf("拒否されたのに匿名化が走った: username=%q email=%v",
					after.PreferredUsername, after.Email)
			}
			// 拒否のあとも管理 API を通れる。締め出されていれば以降の判定が意味を失う。
			if listed := fixture.send(t, idmRefusalRequest{
				method: http.MethodGet, path: "/api/admin/v1/users", sessionID: admin,
			}); listed.Code != http.StatusOK {
				t.Fatalf("拒否のあと管理 API が status=%d になった", listed.Code)
			}
		})
	}

	// 対照: 同じ管理者が他人に対して同じ 3 つを呼ぶと、いずれも通る。
	// これが無いと「そもそも削除できない構成だった」と区別できない。
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodDelete, path: "/api/admin/v1/users/" + idmRefusalBob,
		sessionID: admin, csrf: idmRefusalCSRF,
	}); accepted.Code >= http.StatusBadRequest {
		t.Fatalf("前提が壊れている: 他人の削除予約が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if status := fixture.user(t, idmRefusalBob).Lifecycle.EffectiveStatus(); status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("前提が壊れている: 他人の削除予約でも状態が %s のまま", status)
	}
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/" + idmRefusalBob + "/restore",
		sessionID: admin, csrf: idmRefusalCSRF,
	}); accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 他人の復元が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodDelete, path: "/api/admin/v1/users/" + idmRefusalBob + "?purge=true",
		sessionID: admin, csrf: idmRefusalCSRF,
	}); accepted.Code >= http.StatusBadRequest {
		t.Fatalf("前提が壊れている: 他人の完全削除が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// EX-IDMANAGEMENT-022-01: 未定義の属性または許可外の関数を参照する CEL 式の保存は
// 拒否され、動的グループへの手動メンバー操作も拒否される。規則は保存されず、
// メンバーシップも変わらない。
//
// 規則の保存は成功すると版が上がる。したがって「拒否が変えなかったもの」は、
// 保存されている式と版の両方である。式だけを見ると、版だけ上げて式を捨てる実装を
// 見逃す。手動操作の側は 409 の本文を持つが、拒否したうえで追加も行う実装は
// 応答からは見分けられないので、メンバーの集合を読み直す。
func TestDynamicGroupRefusalsSaveNoRuleAndChangeNoMembership(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-dynamic", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	rulePath := "/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/dynamic-rule"

	saved := fixture.send(t, idmRefusalRequest{
		method: http.MethodPut, path: rulePath, sessionID: admin, csrf: idmRefusalCSRF,
		body: map[string]any{"expression": `user.department == "Engineering"`},
	})
	if saved.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 正しい規則の保存が status=%d body=%s", saved.Code, saved.Body.String())
	}
	before := fixture.dynamicRule(t, idmRefusalDynamicGroup)

	// 動的グループに 1 人置く。除去の拒否は、除去できる相手が居て初めて意味を持つ。
	version := before.Version
	if _, err := fixture.groups.AddMember(context.Background(), &groupdomain.GroupMember{
		GroupID: idmRefusalDynamicGroup, UserID: idmRefusalBob,
		Source: groupdomain.MembershipSourceDynamicRule, RuleVersion: &version, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	beforeMembers := fixture.memberIDs(t, idmRefusalDynamicGroup)

	for _, expression := range []struct{ name, value string }{
		{"未定義の属性", `user.no_such_attribute == "Engineering"`},
		{"許可外の関数", `user.department.upperAscii() == "ENGINEERING"`},
	} {
		t.Run("不正な規則の保存/"+expression.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: http.MethodPut, path: rulePath, sessionID: admin, csrf: idmRefusalCSRF,
				body: map[string]any{"expression": expression.value},
			})
			if refused.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "invalid_dynamic_group_rule" {
				t.Fatalf("error code=%q, want invalid_dynamic_group_rule", code)
			}
			// 拒否が規則を書き換えていないこと。式も版も動いていない。
			after := fixture.dynamicRule(t, idmRefusalDynamicGroup)
			if after.Expression != before.Expression || after.Version != before.Version {
				t.Fatalf("拒否されたのに規則が変わった: expression=%q version=%d",
					after.Expression, after.Version)
			}
		})
	}

	for _, refusal := range []struct{ name, method, path string }{
		{
			"手動での追加", http.MethodPost,
			"/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/members/" + idmRefusalAlice,
		},
		{
			"手動での除去", http.MethodDelete,
			"/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/members/" + idmRefusalBob,
		},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path, sessionID: admin, csrf: idmRefusalCSRF,
			})
			if refused.Code != http.StatusConflict {
				t.Fatalf("status=%d body=%s, want 409", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "dynamic_membership_managed_by_rule" {
				t.Fatalf("error code=%q, want dynamic_membership_managed_by_rule", code)
			}
			// 拒否がメンバーシップを変えていないこと。
			if after := fixture.memberIDs(t, idmRefusalDynamicGroup); !sameStrings(beforeMembers, after) {
				t.Fatalf("拒否されたのにメンバーが変わった: before=%v after=%v", beforeMembers, after)
			}
		})
	}

	// 対照: 手動メンバーシップのグループでは同じ 2 つの操作が通り、集合が変わる。
	manual := "/api/admin/v1/groups/" + idmRefusalManualGroup + "/members/"
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: manual + idmRefusalBob, sessionID: admin, csrf: idmRefusalCSRF,
	}); accepted.Code >= http.StatusBadRequest {
		t.Fatalf("前提が壊れている: 手動グループへの追加が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodDelete, path: manual + idmRefusalAlice, sessionID: admin, csrf: idmRefusalCSRF,
	}); accepted.Code >= http.StatusBadRequest {
		t.Fatalf("前提が壊れている: 手動グループからの除去が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if members := fixture.memberIDs(t, idmRefusalManualGroup); !sameStrings(members, []string{idmRefusalBob}) {
		t.Fatalf("前提が壊れている: 手動グループのメンバーが %v", members)
	}
}

// dynamicRule は保存されている動的グループ規則を読み直す。
func (f *idmRefusalFixture) dynamicRule(t *testing.T, groupID string) *groupdomain.DynamicGroupRule {
	t.Helper()
	rule, err := f.groups.FindDynamicRule(context.Background(), tenancydomain.DefaultTenantID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if rule == nil {
		t.Fatalf("グループ %s に動的規則が無い", groupID)
	}
	return rule
}
