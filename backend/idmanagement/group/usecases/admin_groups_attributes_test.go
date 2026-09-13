package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

//spec:covers REQ-IDMANAGEMENT-024: 管理者はグループの連絡先メールとカスタム属性を、テナント定義の
// スキーマに従って設定できる。

// storedGroup は保存層からグループを読み直す。応答は use case の戻り値から組み立てられる
// ので、戻り値だけを見るテストは「組み立てるが保存しない」実装をそのまま通す。
func storedGroup(
	ctx context.Context, t *testing.T, deps groupusecases.AdminGroupDeps, id string,
) *groupdomain.Group {
	t.Helper()
	group, _, err := groupusecases.GetGroup(ctx, deps, id)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	return group
}

// groupNames は保存層に在るグループ名の集合。拒否が何も作っていないことを、
// 「特定の 1 件が無い」ではなく「増えていない」として読むために要る。
func groupNames(
	ctx context.Context, t *testing.T, deps groupusecases.AdminGroupDeps,
) []string {
	t.Helper()
	views, err := groupusecases.ListGroups(ctx, deps, "", "", 100)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	names := make([]string, 0, len(views))
	for _, view := range views {
		names = append(names, view.Group.Name)
	}
	return names
}

func TestCreateGroupWithEmailAndValidAttributes(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := tenancyusecases.UpdateGroupAttributeSchema(ctx, deps.GroupAttrSchemaRepo, tenancydomain.DefaultTenantID,
		tenancyGroupAttributeDef(t), now); err != nil {
		t.Fatalf("seed schema: %v", err)
	}

	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales", Email: new("sales@example.test"),
		Attributes: map[string]userdomain.AttributeValue{
			"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-100")},
		},
		Now: now,
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if group.Email == nil || *group.Email != "sales@example.test" {
		t.Fatalf("email = %v", group.Email)
	}
	if group.Attributes["cost_center"].String == nil || *group.Attributes["cost_center"].String != "CC-100" {
		t.Fatalf("attributes = %#v", group.Attributes)
	}
	if len(eventTypes(*events)) == 0 || eventTypes(*events)[len(*events)-1] != "GroupCreated" {
		t.Fatalf("events = %v", eventTypes(*events))
	}
	// 「指定どおりに保存され」は保存層から読み直して確かめる。
	stored := storedGroup(ctx, t, deps, group.ID)
	if stored.Email == nil || *stored.Email != "sales@example.test" {
		t.Fatalf("保存された email = %v", stored.Email)
	}
	if got := stored.Attributes["cost_center"]; got.String == nil || *got.String != "CC-100" {
		t.Fatalf("保存された attributes = %#v", stored.Attributes)
	}
}

// 拒否の型だけでなく効果の不在を見る。形式検査を通してから保存する実装は、
// エラーの型を返しつつグループを残せてしまう。
//
//spec:covers EX-IDMANAGEMENT-024-02: メールアドレスの形式を満たさない作成が InvalidEmailError で拒否され、グループを 1 件も残さないこと。
func TestCreateGroupRejectsMalformedEmail(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales", Email: new("not-an-email"), Now: now,
	}); !errors.Is(err, groupusecases.ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
	if names := groupNames(ctx, t, deps); len(names) != 0 {
		t.Fatalf("拒否されたのにグループが残った: %v", names)
	}
	if types := eventTypes(*events); len(types) != 0 {
		t.Fatalf("拒否されたのにイベントが出た: %v", types)
	}
}

// 具体例は「未定義のキー」と「定義済みのキーと型が一致しない」の 2 つを並べる。
// 片方だけを見ると、キーの照合はするが型を見ない実装が通る。
//
//spec:covers EX-IDMANAGEMENT-024-03: 未定義のキーと、宣言と型が一致しない値のどちらも InvalidGroupAttributeError で拒否され、グループを 1 件も残さないこと。
func TestCreateGroupRejectsUndefinedAttributeKey(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name       string
		withSchema bool
		attributes map[string]userdomain.AttributeValue
	}{
		{
			name: "未定義のキー",
			attributes: map[string]userdomain.AttributeValue{
				"unknown": {Type: idmdomain.AttributeTypeString, String: new("x")},
			},
		},
		{
			name:       "定義済みのキーと型が一致しない",
			withSchema: true,
			attributes: map[string]userdomain.AttributeValue{
				"cost_center": {Type: idmdomain.AttributeTypeNumber, Number: new(100.0)},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, events := newGroupDeps(t) //nolint:contextcheck // newGroupDeps は seed に自前の context を使う
			if tc.withSchema {
				if _, err := tenancyusecases.UpdateGroupAttributeSchema(
					ctx, deps.GroupAttrSchemaRepo, tenancydomain.DefaultTenantID,
					tenancyGroupAttributeDef(t), now,
				); err != nil {
					t.Fatalf("seed schema: %v", err)
				}
			}
			_, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
				ActorUserID: "operator", Name: "sales", Attributes: tc.attributes, Now: now,
			})
			if !errors.Is(err, groupusecases.ErrInvalidAttribute) {
				t.Fatalf("expected ErrInvalidAttribute, got %v", err)
			}
			if names := groupNames(ctx, t, deps); len(names) != 0 {
				t.Fatalf("拒否されたのにグループが残った: %v", names)
			}
			if types := eventTypes(*events); len(types) != 0 {
				t.Fatalf("拒否されたのにイベントが出た: %v", types)
			}
		})
	}
}

func TestCreateGroupRejectsMissingRequiredAttribute(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := tenancyusecases.UpdateGroupAttributeSchema(ctx, deps.GroupAttrSchemaRepo, tenancydomain.DefaultTenantID,
		tenancyGroupAttributeDef(t), now); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	_, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales", Now: now,
	})
	if !errors.Is(err, groupusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute for missing required attribute, got %v", err)
	}
}

// 具体例は作成と更新の 2 段で書かれている。更新の `Then` は「新しい値が反映される」に
// 加えて「`GroupUpdated` の `changed_fields` に email と attributes が含まれる」まで
// 言っており、種類だけを数えるテストでは後半が観測できない。
//
//spec:covers EX-IDMANAGEMENT-024-01: email と attributes を指定した作成と更新が保存され、GroupUpdated の changed_fields に両方が並ぶこと。
func TestUpdateGroupEmailAndAttributes(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := tenancyusecases.UpdateGroupAttributeSchema(ctx, deps.GroupAttrSchemaRepo, tenancydomain.DefaultTenantID,
		tenancyGroupAttributeDef(t), now); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales",
		Attributes: map[string]userdomain.AttributeValue{
			"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-100")},
		},
		Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	*events = (*events)[:0]
	newEmail := "updated@example.test"
	newAttrs := map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-200")},
	}
	updated, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Email: &newEmail, Attributes: &newAttrs, Now: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if updated.Email == nil || *updated.Email != newEmail {
		t.Fatalf("email = %v", updated.Email)
	}
	if updated.Attributes["cost_center"].String == nil || *updated.Attributes["cost_center"].String != "CC-200" {
		t.Fatalf("attributes = %#v", updated.Attributes)
	}
	events2 := eventTypes(*events)
	if len(events2) != 1 || events2[0] != "GroupUpdated" {
		t.Fatalf("events = %v", events2)
	}
	changed := (*events)[0].(*idmdomain.GroupUpdated).ChangedFields
	for _, field := range []string{"email", "attributes"} {
		if !slices.Contains(changed, field) {
			t.Fatalf("changed_fields = %v, want it to contain %q", changed, field)
		}
	}
	// 「更新後のグループに新しい値が反映される」は保存層から読み直す。
	stored := storedGroup(ctx, t, deps, group.ID)
	if stored.Email == nil || *stored.Email != newEmail {
		t.Fatalf("保存された email = %v", stored.Email)
	}
	if got := stored.Attributes["cost_center"]; got.String == nil || *got.String != "CC-200" {
		t.Fatalf("保存された attributes = %#v", stored.Attributes)
	}
}

func TestUpdateGroupRejectsMalformedEmail(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	bad := "not-an-email"
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Email: &bad,
	}); !errors.Is(err, groupusecases.ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func tenancyGroupAttributeDef(t *testing.T) []groupdomain.GroupAttributeDef {
	t.Helper()
	return []groupdomain.GroupAttributeDef{{Key: "cost_center", Type: idmdomain.AttributeTypeString, Required: true}}
}
