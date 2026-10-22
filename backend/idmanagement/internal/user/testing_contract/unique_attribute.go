package testing_contract

import (
	"context"
	"errors"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// withString は、属性 key に文字列 value を持つ User を返す。
func withString(u *userdomain.User, key, value string) *userdomain.User {
	if u.Attributes == nil {
		u.Attributes = map[string]userdomain.AttributeValue{}
	}
	v := value
	u.Attributes[key] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: &v}
	return u
}

func uniqueSchema(tenantID string, f Fixture, defs ...userdomain.UserAttributeDef) *userdomain.TenantUserAttributeSchema {
	return &userdomain.TenantUserAttributeSchema{TenantID: tenantID, Attributes: defs, CreatedAt: f.Now, UpdatedAt: f.Now}
}

func stringDef(key string, unique bool) userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{
		Key: key, Type: idmdomain.AttributeTypeString, Unique: unique, Visibility: idmdomain.AttrVisibilityAdminReadable,
	}
}

func wantConflict(t *testing.T, err error, key, when string) {
	t.Helper()
	var conflict *userports.UniqueAttributeConflictError
	if !errors.As(err, &conflict) || conflict.Key != key {
		t.Fatalf("%s: err = %v, want UniqueAttributeConflictError for %q", when, err, key)
	}
}

// runUniqueAttributeValues は、unique の属性の値を保存のたびに一意に保ち、属性を unique に変える
// スキーマの保存が既存の値を写すことを確かめる。
func runUniqueAttributeValues(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	mustSave := func(u *userdomain.User, when string) {
		t.Helper()
		if err := f.Users.Save(ctx, u); err != nil {
			t.Fatalf("%s: Save: %v", when, err)
		}
	}
	mustSaveSchema := func(s *userdomain.TenantUserAttributeSchema, when string) {
		t.Helper()
		if err := f.Schemas.Save(ctx, s); err != nil {
			t.Fatalf("%s: Save schema: %v", when, err)
		}
	}

	mustSaveSchema(uniqueSchema(f.TenantA, f, stringDef("source_anchor", true), stringDef("employee_no", false)), "unique source_anchor")
	first := withString(user(t, f.TenantA, "first", f.Now), "source_anchor", "anchor-1")
	mustSave(first, "first holder")

	// 作成も更新も、ほかの User の値を奪えない。拒否した保存は User を変えない。
	second := withString(user(t, f.TenantA, "second", f.Now), "source_anchor", "anchor-1")
	wantConflict(t, f.Users.Save(ctx, second), "source_anchor", "create with a taken value")
	if got, err := f.Users.FindBySub(ctx, second.ID); err != nil || got != nil {
		t.Fatalf("refused create stored the user: (%+v, %v)", got, err)
	}
	mustSave(withString(second, "source_anchor", "anchor-2"), "second with its own value")
	_, err := f.Users.Update(ctx, second.ID, func(u *userdomain.User) (bool, error) {
		withString(u, "source_anchor", "anchor-1")
		return true, nil
	})
	wantConflict(t, err, "source_anchor", "update to a taken value")
	if got, err := f.Users.FindBySub(ctx, second.ID); err != nil || got == nil || *got.Attributes["source_anchor"].String != "anchor-2" {
		t.Fatalf("refused update changed the user: (%+v, %v)", got, err)
	}
	// 保存し直しは自分の値と衝突しない。
	mustSave(first, "first saved again")

	// 一意性はテナントの中に限り、unique でない属性は重なってよい。
	mustSaveSchema(uniqueSchema(f.TenantB, f, stringDef("source_anchor", true)), "tenant B schema")
	mustSave(withString(user(t, f.TenantB, "other-tenant", f.Now), "source_anchor", "anchor-1"), "same value in another tenant")
	mustSave(withString(withString(user(t, f.TenantA, "employee-a", f.Now), "source_anchor", "anchor-a"), "employee_no", "E-1"), "employee a")
	duplicate := withString(withString(user(t, f.TenantA, "employee-b", f.Now), "source_anchor", "anchor-b"), "employee_no", "E-1")
	mustSave(duplicate, "a shared non-unique value")

	// 削除した User は値を譲る。
	first.Lifecycle.Status = idmdomain.UserStatusDeleted
	mustSave(first, "delete the first holder")
	mustSave(withString(user(t, f.TenantA, "successor", f.Now), "source_anchor", "anchor-1"), "successor takes the freed value")

	// 重なる値を持つ属性は unique にできず、スキーマは変わらない。
	err = f.Schemas.Save(ctx, uniqueSchema(f.TenantA, f, stringDef("source_anchor", true), stringDef("employee_no", true)))
	var notUnique *userports.AttributeValuesNotUniqueError
	if !errors.As(err, &notUnique) || notUnique.Key != "employee_no" {
		t.Fatalf("unique over duplicates: err = %v, want AttributeValuesNotUniqueError for employee_no", err)
	}
	if got, err := f.Schemas.FindByTenant(ctx, f.TenantA); err != nil || got == nil || got.Attributes[1].Unique {
		t.Fatalf("refused schema was stored: (%+v, %v)", got, err)
	}

	// 重なりを解けば unique にでき、それまでの値も以後の保存の重複を止める。
	mustSave(withString(duplicate, "employee_no", "E-2"), "resolve the duplicate")
	mustSaveSchema(uniqueSchema(f.TenantA, f, stringDef("source_anchor", true), stringDef("employee_no", true)), "unique employee_no")
	wantConflict(t, f.Users.Save(ctx, withString(withString(user(t, f.TenantA, "employee-c", f.Now), "source_anchor", "anchor-c"), "employee_no", "E-1")),
		"employee_no", "a value written before the attribute became unique")

	// unique でなくした属性は、重なる値を受け付ける。
	mustSaveSchema(uniqueSchema(f.TenantA, f, stringDef("source_anchor", false), stringDef("employee_no", true)), "source_anchor no longer unique")
	mustSave(withString(user(t, f.TenantA, "again", f.Now), "source_anchor", "anchor-2"), "a value shared after unique was removed")
}
