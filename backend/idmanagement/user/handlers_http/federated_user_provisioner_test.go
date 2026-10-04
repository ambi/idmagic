package handlers_http_test

import (
	"context"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/user/handlers_http"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-IDMANAGEMENT-089: 管理 API と同じ依存から組み立てた JIT の入口が、規則に一致する属性の User を動的グループに所属させること。
func TestFederatedUserProvisionerEvaluatesDynamicGroups(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "dyn-eng", TenantID: tenancydomain.DefaultTenantID, Name: "dyn-eng",
		MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{
		GroupID: "dyn-eng", TenantID: tenancydomain.DefaultTenantID, Expression: `user.department == "Engineering"`,
		Enabled: true, Version: 1, ReferencedAttributes: []string{"department"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	if err := schemas.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID:   tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate}},
	}); err != nil {
		t.Fatal(err)
	}

	provision := userhttp.FederatedUserProvisioner(userhttp.Deps{Authenticator: &support.Authenticator{}, UserRepo: users, GroupRepo: groups, AttrSchemaRepo: schemas})
	engineering := "Engineering"
	user, err := provision(ctx, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "eve", Now: now,
		Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &engineering}},
	})
	if err != nil {
		t.Fatal(err)
	}
	members, err := groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, "dyn-eng")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].UserID != user.ID {
		t.Fatalf("members=%+v, want the JIT user %s", members, user.ID)
	}
}
