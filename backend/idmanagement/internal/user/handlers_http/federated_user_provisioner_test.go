package handlers_http_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/internal/user/handlers_http"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-IDMANAGEMENT-089: 管理 API と同じ依存から組み立てた JIT の入口が、規則に一致する属性の User を動的グループに所属させること。
func TestFederatedUserProvisionerEvaluatesDynamicGroups(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
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

//spec:covers REQ-IDMANAGEMENT-001, REQ-IDMANAGEMENT-091: JIT の組み立てを通したローカル属性への書き込みが、User とイベントを作らずに拒否される。
func TestFederatedUserProvisionerRejectsLocalAttributes(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	var defs []userdomain.UserAttributeDef
	if err := json.Unmarshal([]byte(`[{"key":"approval","type":"string","visibility":"admin_readable","write_authority":"local"}]`), &defs); err != nil {
		t.Fatal(err)
	}
	if err := schemas.Save(ctx, &userdomain.TenantUserAttributeSchema{TenantID: tenancydomain.DefaultTenantID, Attributes: defs}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	provision := userhttp.FederatedUserProvisioner(userhttp.Deps{
		Authenticator: &support.Authenticator{}, UserRepo: users, GroupRepo: groups, AttrSchemaRepo: schemas,
		Emit: func(event spec.DomainEvent) { events = append(events, event) },
	})
	got, err := provision(ctx, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "eve", Now: time.Now().UTC(),
		Attributes: map[string]userdomain.AttributeValue{"approval": {Type: idmdomain.AttributeTypeString, String: new("yes")}},
	})
	if !errors.Is(err, userusecases.ErrInvalidAttribute) || got != nil {
		t.Fatalf("JIT result=%+v err=%v; want attribute refusal", got, err)
	}
	if stored, err := users.FindByUsername(ctx, tenancydomain.DefaultTenantID, "eve"); err != nil || stored != nil || len(events) != 0 {
		t.Fatalf("refused JIT left user=%+v err=%v events=%d", stored, err, len(events))
	}
}
