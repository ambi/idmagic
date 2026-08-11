package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// REQ-IDMANAGEMENT-024: 管理者はグループの連絡先メールとカスタム属性を、テナント定義の
// スキーマに従って設定できる。

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
}

func TestCreateGroupRejectsMalformedEmail(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales", Email: new("not-an-email"), Now: now,
	}); !errors.Is(err, groupusecases.ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestCreateGroupRejectsUndefinedAttributeKey(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	_, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "sales",
		Attributes: map[string]userdomain.AttributeValue{
			"unknown": {Type: idmdomain.AttributeTypeString, String: new("x")},
		},
		Now: now,
	})
	if !errors.Is(err, groupusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute, got %v", err)
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
