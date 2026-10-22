package usecases_test

// unique と定義した属性の値を、管理者の作成と更新、本人の更新のどれもがほかの User と重ねられない
// ことを、組み立て地点と同じく一意性を守るメモリの保存先で観測する。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

func anchorValue(value string) map[string]userdomain.AttributeValue {
	return map[string]userdomain.AttributeValue{
		"source_anchor": {Type: idmdomain.AttributeTypeString, String: &value},
	}
}

//spec:covers REQ-IDMANAGEMENT-089: 管理者の更新と本人の更新のどちらも、同じテナントの削除されていないほかの User と unique の属性の値が同じ User を保存しないこと。
//spec:covers REQ-IDMANAGEMENT-090: ほかの User が持つ値へ unique の属性を変える管理者の更新を拒否し、User を変えないこと。
//spec:covers REQ-IDMANAGEMENT-051: ほかの User が持つ値へ unique の属性を変える本人の更新を拒否し、どの項目も変えないこと。
func TestUniqueAttributeValuesStayUniqueOnEveryWrite(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Now().UTC()
	users, schemas := usermemory.NewUserStore()
	if err := schemas.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{{
			Key: "source_anchor", Type: idmdomain.AttributeTypeString, Unique: true, EditableByUser: true,
			Visibility: idmdomain.AttrVisibilitySelfReadable,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	admin := userusecases.AdminUserDeps{
		ValidateInitialPassword: authentication.ValidateInitialPassword,
		UserCredentials:         authentication.Module{}.UserCredentials(),
		UserRepo:                users,
		AttrSchemaRepo:          schemas,
		PasswordHasher:          testing_passwords.NewHasher(),
		PasswordHistoryRepo:     authentication.NewMemoryModule(nil).PasswordHistoryRepo,
		Emit:                    func(spec.DomainEvent) error { return nil },
	}
	// 管理者による作成は属性を受け取らないので、作ってから管理者の更新で値を付ける。
	create := func(username, anchor string) (*userdomain.User, error) {
		created, err := userusecases.CreateUser(ctx, admin, userusecases.CreateUserInput{
			ActorUserID: "admin", PreferredUsername: username, Password: "initial-password-9182", Now: now,
		})
		if err != nil {
			return nil, err
		}
		return userusecases.UpdateUser(ctx, admin, userusecases.UpdateUserInput{
			ActorUserID: "admin", Sub: created.ID, Attributes: new(anchorValue(anchor)), Now: now,
		})
	}
	wantConflict := func(err error, when string) {
		t.Helper()
		var conflict *userports.UniqueAttributeConflictError
		if !errors.As(err, &conflict) || conflict.Key != "source_anchor" {
			t.Fatalf("%s: err = %v, want a source_anchor conflict", when, err)
		}
	}
	anchorOf := func(id string) string {
		t.Helper()
		u, err := users.FindBySub(ctx, id)
		if err != nil || u == nil {
			t.Fatalf("FindBySub(%s) = (%+v, %v)", id, u, err)
		}
		return *u.Attributes["source_anchor"].String
	}

	alice, err := create("alice", "anchor-alice")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := create("bob", "anchor-bob")
	if err != nil {
		t.Fatal(err)
	}

	_, err = userusecases.UpdateUser(ctx, admin, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: bob.ID, Attributes: new(anchorValue("anchor-alice")), Now: now,
	})
	wantConflict(err, "admin update")
	if got := anchorOf(bob.ID); got != "anchor-bob" {
		t.Fatalf("refused admin update changed the value to %q", got)
	}

	displayName := "Bob the impostor"
	_, _, err = userusecases.UpdateUserProfile(ctx, userusecases.AccountProfileDeps{
		UserRepo: users, AttrSchemaRepo: schemas, Emit: func(spec.DomainEvent) error { return nil },
	}, userusecases.UpdateUserProfileInput{Sub: bob.ID, Name: &displayName, Attributes: new(anchorValue("anchor-alice")), Now: now})
	wantConflict(err, "self-service update")
	if got, err := users.FindBySub(ctx, bob.ID); err != nil || got == nil || got.Name != nil || *got.Attributes["source_anchor"].String != "anchor-bob" {
		t.Fatalf("refused self-service update changed the user: (%+v, %v)", got, err)
	}
	if got := anchorOf(alice.ID); got != "anchor-alice" {
		t.Fatalf("the holder lost its value: %q", got)
	}
}
