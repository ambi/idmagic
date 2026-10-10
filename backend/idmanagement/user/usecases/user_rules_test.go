package usecases_test

// User の作成、更新、無効化、必須操作、削除の予約、復元、完全削除が約束する細部を、
// 保存層とイベントの両方から読んで固定する。戻り値は use case が組み立てるので、
// 状態は保存層から読み直す。

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	trusteddevicedomain "github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

var userRulesNow = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

type recordedNotification struct {
	userID  string
	trigger userports.ProvisioningTrigger
}

type recordingProvisioningNotifier struct{ calls []recordedNotification }

func (n *recordingProvisioningNotifier) NotifyUserMutation(_ context.Context, _, userID string, trigger userports.ProvisioningTrigger, _ time.Time) error {
	n.calls = append(n.calls, recordedNotification{userID: userID, trigger: trigger})
	return nil
}

type userRulesFixture struct {
	deps     userusecases.AdminUserDeps
	users    *usermemory.UserRepository
	sessions *sessionmemory.SessionStore
	devices  *trusteddevicememory.TrustedDeviceRepository
	quota    *tenancymemory.QuotaRepository
	notifier *recordingProvisioningNotifier
	events   *[]spec.DomainEvent
	agents   *agentmemory.AgentRepository
}

func newUserRulesFixture(t *testing.T) *userRulesFixture {
	t.Helper()
	f := &userRulesFixture{
		users: usermemory.NewUserRepository(), sessions: sessionmemory.NewSessionStore(),
		devices: trusteddevicememory.NewTrustedDeviceRepository(), quota: tenancymemory.NewQuotaRepository(),
		notifier: &recordingProvisioningNotifier{}, events: &[]spec.DomainEvent{},
		agents: agentmemory.NewAgentRepository(),
	}
	f.deps = userusecases.AdminUserDeps{
		UserRepo: f.users, SessionStore: f.sessions, TrustedDeviceRepo: f.devices, QuotaRepo: f.quota,
		AgentRepo:      f.agents,
		PasswordHasher: testing_passwords.NewHasher(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		ProvisioningNotifier: f.notifier,
		Emit:                 func(event spec.DomainEvent) error { *f.events = append(*f.events, event); return nil },
	}
	return f
}

// seed は default テナントの User を置く。`mutate` で状態や項目を変える。
func (f *userRulesFixture) seed(id string, mutate func(*userdomain.User)) *userdomain.User {
	email := id + "@example.test"
	name := id
	user := &userdomain.User{
		ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id, PasswordHash: "hash",
		Name: &name, Email: &email, EmailVerified: true, Roles: []string{},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: userRulesNow.Add(-time.Hour), UpdatedAt: userRulesNow.Add(-time.Hour),
	}
	if mutate != nil {
		mutate(user)
	}
	f.users.Seed(user)
	return user
}

func (f *userRulesFixture) stored(t *testing.T, id string) *userdomain.User {
	t.Helper()
	user, err := f.users.FindBySubIncludingDeleted(testing_tenant.Default(context.Background()), id)
	if err != nil || user == nil {
		t.Fatalf("FindBySubIncludingDeleted(%s)=(%v,%v)", id, user, err)
	}
	return user
}

func (f *userRulesFixture) eventsOf(eventType string) []spec.DomainEvent {
	var out []spec.DomainEvent
	for _, event := range *f.events {
		if event.EventType() == eventType {
			out = append(out, event)
		}
	}
	return out
}

func pendingSince(at time.Time) func(*userdomain.User) {
	return func(user *userdomain.User) {
		user.Lifecycle.Status = idmdomain.UserStatusPendingDeletion
		user.Lifecycle.StatusChangedAt = &at
	}
}

//spec:covers EX-IDMANAGEMENT-042-01, EX-IDMANAGEMENT-042-02: 作成がユーザー名の前後の空白を除いて表記のまま保存し、大文字と小文字だけが異なるユーザー名を username_conflict で拒否して User を増やさないこと。
func TestCreateUserTrimsTheUsernameAndComparesItCaseInsensitively(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	ctx := testing_tenant.Default(context.Background())
	carol, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: " Carol ", Password: "initial-password-9182", Now: userRulesNow,
	})
	if err != nil || carol.PreferredUsername != "Carol" {
		t.Fatalf("user=%+v err=%v, want username Carol", carol, err)
	}
	for _, username := range []string{"alice", "Alice", " ALICE "} {
		if _, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
			ActorUserID: "admin", PreferredUsername: username, Password: "initial-password-9182", Now: userRulesNow,
		}); !errors.Is(err, userusecases.ErrUsernameConflict) {
			t.Fatalf("%q: err=%v, want ErrUsernameConflict", username, err)
		}
	}
	all, err := f.users.FindAll(ctx, tenancydomain.DefaultTenantID)
	if err != nil || len(all) != 2 {
		t.Fatalf("users=%d err=%v, want alice and Carol only", len(all), err)
	}
}

//spec:covers REQ-IDMANAGEMENT-042: テナントのパスワードポリシーに違反する作成を拒否し、User を作らないこと。
func TestCreateUserAppliesTheTenantPasswordPolicy(t *testing.T) {
	f := newUserRulesFixture(t)
	twenty := 20
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, PasswordPolicyOverride: &tenancydomain.PasswordPolicyOverride{MinLength: &twenty},
	}, "https://idp.example", "")
	_, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "twelve-chars", Now: userRulesNow,
	})
	if _, ok := errors.AsType[*authusecases.PasswordPolicyError](err); !ok {
		t.Fatalf("err=%v, want PasswordPolicyError", err)
	}
	if user, _ := f.users.FindByUsername(ctx, tenancydomain.DefaultTenantID, "carol"); user != nil {
		t.Fatalf("拒否した作成が User を作った: %+v", user)
	}
}

//spec:covers REQ-IDMANAGEMENT-042, REQ-IDMANAGEMENT-089: 管理者の作成が、ほかの User と大文字と小文字だけが異なるメールアドレスを拒否し、User も使用量も増やさないこと。
func TestCreateUserRejectsAnEmailAnotherUserHas(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	ctx := testing_tenant.Default(context.Background())
	email := " ALICE@example.test "
	if _, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "initial-password-9182", Email: &email, Now: userRulesNow,
	}); !errors.Is(err, userusecases.ErrEmailTaken) {
		t.Fatalf("err=%v, want ErrEmailTaken", err)
	}
	if user, _ := f.users.FindByUsername(ctx, tenancydomain.DefaultTenantID, "carol"); user != nil {
		t.Fatalf("拒否した作成が User を作った: %+v", user)
	}
	if usage, _ := f.quota.GetUsage(ctx, tenancydomain.DefaultTenantID); usage.Users != 0 {
		t.Fatalf("usage=%d, want 0", usage.Users)
	}
}

//spec:covers REQ-IDMANAGEMENT-090: 管理者の更新が、自分のユーザー名の表記だけを変える更新を受け付け、ほかの User と名前またはメールアドレスが同じ値への更新を拒否して User を変えないこと。
func TestUpdateUserRejectsAUsernameOrEmailAnotherUserHas(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	f.seed("bob", nil)
	ctx := testing_tenant.Default(context.Background())
	recased := "Bob"
	updated, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{ActorUserID: "admin", Sub: "bob", PreferredUsername: &recased, Now: userRulesNow})
	if err != nil || updated.PreferredUsername != "Bob" {
		t.Fatalf("user=%+v err=%v, want username Bob", updated, err)
	}
	taken := "ALICE"
	if _, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{ActorUserID: "admin", Sub: "bob", PreferredUsername: &taken, Now: userRulesNow}); !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("username: err=%v, want ErrUsernameConflict", err)
	}
	email := "Alice@Example.test"
	if _, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{ActorUserID: "admin", Sub: "bob", Email: &email, Now: userRulesNow}); !errors.Is(err, userusecases.ErrEmailTaken) {
		t.Fatalf("email: err=%v, want ErrEmailTaken", err)
	}
	if stored := f.stored(t, "bob"); stored.PreferredUsername != "Bob" || *stored.Email != "bob@example.test" {
		t.Fatalf("stored=%+v, want the rejected updates not stored", stored)
	}
}

//spec:covers EX-IDMANAGEMENT-089-01, REQ-IDMANAGEMENT-043: JIT が大文字と小文字を区別せずにメールアドレスの衝突を拒否し、空白だけのメールアドレスを未設定として扱い、identity-broker を操作者とすること。
func TestProvisionFederatedUserNormalizesTheEmailAndRejectsACaseInsensitiveConflict(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	ctx := testing_tenant.Default(context.Background())
	conflict := "ALICE@example.test"
	if _, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "alice-federated", Email: &conflict, Now: userRulesNow,
	}); !errors.Is(err, userusecases.ErrEmailTaken) {
		t.Fatalf("衝突: err=%v, want ErrEmailTaken", err)
	}
	if user, _ := f.users.FindByUsername(ctx, tenancydomain.DefaultTenantID, "alice-federated"); user != nil {
		t.Fatalf("衝突した JIT が User を作った: %+v", user)
	}
	blank := "  "
	user, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "bob-federated", Email: &blank, Now: userRulesNow,
	})
	if err != nil || user.Email != nil || len(user.Roles) != 0 {
		t.Fatalf("user=%+v err=%v, want no email and no roles", user, err)
	}
	created, ok := f.eventsOf("UserCreated")[0].(*idmdomain.UserCreated)
	if !ok || created.ActorUserID != "identity-broker" {
		t.Fatalf("UserCreated=%+v, want actor identity-broker", f.eventsOf("UserCreated"))
	}
}

//spec:covers EX-IDMANAGEMENT-089-02: JIT が作った User を作成の時点で動的グループの規則で評価し、規則に一致する User だけを所属させること。
func TestProvisionFederatedUserEvaluatesDynamicGroups(t *testing.T) {
	f := newUserRulesFixture(t)
	ctx := testing_tenant.Default(context.Background())
	groups, schemas := dynamicDepartmentGroup(t)
	f.deps.GroupRepo, f.deps.AttrSchemaRepo = groups, schemas
	engineering, sales := "Engineering", "Sales"
	eve, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "eve", Now: userRulesNow,
		Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &engineering}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "sam", Now: userRulesNow,
		Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &sales}},
	}); err != nil {
		t.Fatal(err)
	}
	members, err := groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, "dyn-eng")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].UserID != eve.ID {
		t.Fatalf("members=%+v, want only the JIT user %s whose department matches", members, eve.ID)
	}
}

// dynamicDepartmentGroup は `department == "Engineering"` の有効な規則を持つ動的グループ
// `dyn-eng` と、`department` を定義した属性スキーマを返す。
func dynamicDepartmentGroup(t *testing.T) (*groupmemory.GroupRepository, *usermemory.TenantUserAttributeSchemaRepository) {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	groups := groupmemory.NewGroupRepository()
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "dyn-eng", TenantID: tenancydomain.DefaultTenantID, Name: "dyn-eng",
		MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: userRulesNow, UpdatedAt: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{
		GroupID: "dyn-eng", TenantID: tenancydomain.DefaultTenantID, Expression: `user.department == "Engineering"`,
		Enabled: true, Version: 1, ReferencedAttributes: []string{"department"}, CreatedAt: userRulesNow, UpdatedAt: userRulesNow,
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
	return groups, schemas
}

//spec:covers EX-IDMANAGEMENT-044-02: 期限切れの削除予約の完全削除が、猶予期間ちょうどの User と時刻を持たない User を残し、過ぎた User だけを system と auto_purge で完全削除すること。
func TestPurgeExpiredSoftDeletedKeepsUsersAtTheExactBoundary(t *testing.T) {
	f := newUserRulesFixture(t)
	grace := time.Duration(userusecases.UserSoftDeleteGracePeriodSeconds) * time.Second
	f.seed("at-boundary", pendingSince(userRulesNow.Add(-grace)))
	f.seed("expired", pendingSince(userRulesNow.Add(-grace-time.Second)))
	f.seed("no-timestamp", func(user *userdomain.User) { user.Lifecycle.Status = idmdomain.UserStatusPendingDeletion })
	if err := userusecases.PurgeExpiredSoftDeleted(testing_tenant.Default(context.Background()), f.deps, userRulesNow); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"at-boundary", "no-timestamp"} {
		if status := f.stored(t, id).Lifecycle.Status; status != idmdomain.UserStatusPendingDeletion {
			t.Fatalf("%s: status=%s, want pending_deletion", id, status)
		}
	}
	if !f.stored(t, "expired").IsDeleted() {
		t.Fatalf("猶予期間を過ぎた User が完全削除されていない")
	}
	deleted := f.eventsOf("UserDeleted")
	if len(deleted) != 1 {
		t.Fatalf("UserDeleted=%d, want 1", len(deleted))
	}
	if event := deleted[0].(*idmdomain.UserDeleted); event.ActorUserID != "system" || event.Reason != "auto_purge" || event.TargetUserID != "expired" {
		t.Fatalf("UserDeleted=%+v, want system / auto_purge for expired", event)
	}
}

//spec:covers EX-IDMANAGEMENT-045-01, REQ-IDMANAGEMENT-045: 更新が値の変わった属性のキーだけを昇順で changed_fields に載せ、何も変わらない更新は updated_at を進めず UserUpdated を発行しないこと。
func TestUpdateUserRecordsOnlyChangedFields(t *testing.T) {
	f := newUserRulesFixture(t)
	f.deps.AttrSchemaRepo = departmentAndTitleSchema(t)
	sales := "Sales"
	alice := f.seed("alice", func(user *userdomain.User) {
		user.Attributes = map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &sales}}
	})
	engineering, lead := "Engineering", "Lead"
	attrs := map[string]userdomain.AttributeValue{
		"title":      {Type: idmdomain.AttributeTypeString, String: &lead},
		"department": {Type: idmdomain.AttributeTypeString, String: &engineering},
	}
	if _, err := userusecases.UpdateUser(testing_tenant.Default(context.Background()), f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: alice.ID, Name: alice.Name, Attributes: &attrs, Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	updates := f.eventsOf("UserUpdated")
	if len(updates) != 1 || !slices.Equal(updates[0].(*idmdomain.UserUpdated).ChangedFields, []string{"department", "title"}) {
		t.Fatalf("UserUpdated=%+v, want changed_fields [department title]", updates)
	}

	before := f.stored(t, alice.ID).UpdatedAt
	if _, err := userusecases.UpdateUser(testing_tenant.Default(context.Background()), f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: alice.ID, Name: alice.Name, Now: userRulesNow.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if after := f.stored(t, alice.ID).UpdatedAt; !after.Equal(before) {
		t.Fatalf("何も変わらない更新が updated_at を %v から %v へ進めた", before, after)
	}
	if updates := f.eventsOf("UserUpdated"); len(updates) != 1 {
		t.Fatalf("何も変わらない更新が UserUpdated を発行した: %d 件", len(updates))
	}
}

func departmentAndTitleSchema(t *testing.T) *usermemory.TenantUserAttributeSchemaRepository {
	t.Helper()
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	if err := schemas.Save(testing_tenant.Default(context.Background()), &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{
			{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
			{Key: "title", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
		},
	}); err != nil {
		t.Fatal(err)
	}
	return schemas
}

//spec:covers EX-IDMANAGEMENT-045-03, REQ-IDMANAGEMENT-045: email_verified を指定せずに確認済みのメールアドレスを変える更新が email_verified を false にして changed_fields に載せ、同じ要求で指定した email_verified は保存すること。
func TestUpdateUserResetsEmailVerifiedWhenTheAddressChanges(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	bob := f.seed("bob", nil)
	ctx := testing_tenant.Default(context.Background())
	email := "alice@new.example.test"
	if _, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: alice.ID, Email: &email, Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	stored := f.stored(t, alice.ID)
	if *stored.Email != email || stored.EmailVerified {
		t.Fatalf("email=%q verified=%v, want the new address unverified", *stored.Email, stored.EmailVerified)
	}
	updates := f.eventsOf("UserUpdated")
	if len(updates) != 1 || !slices.Equal(updates[0].(*idmdomain.UserUpdated).ChangedFields, []string{"email", "email_verified"}) {
		t.Fatalf("UserUpdated=%+v, want changed_fields [email email_verified]", updates)
	}

	bobEmail, verified := "bob@new.example.test", true
	if _, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: bob.ID, Email: &bobEmail, EmailVerified: &verified, Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	if stored := f.stored(t, bob.ID); *stored.Email != bobEmail || !stored.EmailVerified {
		t.Fatalf("email=%q verified=%v, want the explicitly verified address kept verified", *stored.Email, stored.EmailVerified)
	}
	if updates := f.eventsOf("UserUpdated"); len(updates) != 2 || !slices.Equal(updates[1].(*idmdomain.UserUpdated).ChangedFields, []string{"email"}) {
		t.Fatalf("UserUpdated=%+v, want changed_fields [email] for the explicitly verified update", updates)
	}
}

//spec:covers REQ-IDMANAGEMENT-046: すでに Disabled の User の無効化と Active の User の再有効化が、時刻を進めずイベントを発行しないこと。
func TestSetUserDisabledDoesNothingWhenAlreadyInThatState(t *testing.T) {
	f := newUserRulesFixture(t)
	changedAt := userRulesNow.Add(-time.Hour)
	disabled := f.seed("alice", func(user *userdomain.User) {
		user.Lifecycle.Status = idmdomain.UserStatusDisabled
		user.Lifecycle.StatusChangedAt = &changedAt
	})
	active := f.seed("bob", nil)
	ctx := testing_tenant.Default(context.Background())
	if _, err := userusecases.SetUserDisabled(ctx, f.deps, "admin", disabled.ID, true, userRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.SetUserDisabled(ctx, f.deps, "admin", active.ID, false, userRulesNow); err != nil {
		t.Fatal(err)
	}
	if stored := f.stored(t, disabled.ID); !stored.Lifecycle.StatusChangedAt.Equal(changedAt) || !stored.UpdatedAt.Equal(disabled.UpdatedAt) {
		t.Fatalf("無効化済みの User の時刻が動いた: %+v", stored.Lifecycle)
	}
	if !f.stored(t, active.ID).UpdatedAt.Equal(active.UpdatedAt) {
		t.Fatalf("有効な User の updated_at が動いた")
	}
	if len(*f.events) != 0 {
		t.Fatalf("イベントが発行された: %v", eventTypes(*f.events))
	}
}

//spec:covers REQ-IDMANAGEMENT-046: admin を持つ管理者が自分自身を無効化する操作を self_disable_forbidden で拒否し、自分自身の再有効化は拒否しないこと。
func TestSetUserDisabledRefusesAnAdministratorDisablingThemselves(t *testing.T) {
	f := newUserRulesFixture(t)
	operator := f.seed("operator", func(user *userdomain.User) { user.Roles = []string{"admin"} })
	ctx := testing_tenant.Default(context.Background())
	if _, err := userusecases.SetUserDisabled(ctx, f.deps, operator.ID, operator.ID, true, userRulesNow); !errors.Is(err, userusecases.ErrSelfDisableForbidden) {
		t.Fatalf("err=%v, want ErrSelfDisableForbidden", err)
	}
	if status := f.stored(t, operator.ID).Lifecycle.Status; status != idmdomain.UserStatusActive {
		t.Fatalf("status=%s, want active", status)
	}
	if _, err := userusecases.SetUserDisabled(ctx, f.deps, operator.ID, operator.ID, false, userRulesNow); err != nil {
		t.Fatalf("自分自身の再有効化: err=%v, want accepted", err)
	}
}

// REQ-IDMANAGEMENT-046 の主要な使い方：削除予約中の User は無効化も再有効化もできない。
//
//spec:covers REQ-IDMANAGEMENT-046: 削除予約中の User の無効化と再有効化を user_pending_deletion で拒否し、User を PendingDeletion のまま残しイベントを発行しないこと。
func TestSetUserDisabledRefusesAPendingDeletionUser(t *testing.T) {
	f := newUserRulesFixture(t)
	scheduledAt := userRulesNow.Add(-time.Hour)
	alice := f.seed("alice", func(user *userdomain.User) {
		user.Lifecycle.Status = idmdomain.UserStatusPendingDeletion
		user.Lifecycle.StatusChangedAt = &scheduledAt
	})
	ctx := testing_tenant.Default(context.Background())
	for _, disabled := range []bool{true, false} {
		if _, err := userusecases.SetUserDisabled(ctx, f.deps, "admin", alice.ID, disabled, userRulesNow); !errors.Is(err, userusecases.ErrUserPendingDeletion) {
			t.Fatalf("disabled=%v: err=%v, want ErrUserPendingDeletion", disabled, err)
		}
		stored := f.stored(t, alice.ID)
		if stored.Lifecycle.Status != idmdomain.UserStatusPendingDeletion || !stored.Lifecycle.StatusChangedAt.Equal(scheduledAt) {
			t.Fatalf("disabled=%v: 削除予約が変わった: %+v", disabled, stored.Lifecycle)
		}
	}
	if len(*f.events) != 0 {
		t.Fatalf("イベントが発行された: %v", eventTypes(*f.events))
	}
}

//spec:covers REQ-IDMANAGEMENT-010: 無効化がその User の記憶済みの端末を失効させること。
func TestSetUserDisabledRevokesTrustedDevices(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	ctx := testing_tenant.Default(context.Background())
	if err := f.devices.Save(ctx, &trusteddevicedomain.TrustedDevice{
		ID: "device-1", TenantID: tenancydomain.DefaultTenantID, UserID: alice.ID, Selector: "selector-1",
		VerifierHash: "hash", CreatedAt: userRulesNow.Add(-time.Hour), LastUsedAt: userRulesNow.Add(-time.Hour),
		ExpiresAt: userRulesNow.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.SetUserDisabled(ctx, f.deps, "admin", alice.ID, true, userRulesNow); err != nil {
		t.Fatal(err)
	}
	active, err := f.devices.ListActiveByUser(ctx, tenancydomain.DefaultTenantID, alice.ID, userRulesNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("無効化の後も端末が有効: %+v", active)
	}
}

//spec:covers REQ-IDMANAGEMENT-047: 付与済みの必須操作の付与と未付与の必須操作の解除がイベントを発行せず、定義されていない必須操作を拒否すること。
func TestRequiredActionsAreIdempotentAndClosed(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", func(user *userdomain.User) {
		user.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}
	})
	ctx := testing_tenant.Default(context.Background())
	if _, err := userusecases.SetUserRequiredAction(ctx, f.deps, "admin", alice.ID, idmdomain.RequiredActionUpdatePassword, userRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.ClearUserRequiredAction(ctx, f.deps, "admin", alice.ID, idmdomain.RequiredActionVerifyEmail, userRulesNow); err != nil {
		t.Fatal(err)
	}
	if len(*f.events) != 0 {
		t.Fatalf("イベントが発行された: %v", eventTypes(*f.events))
	}
	if _, err := userusecases.SetUserRequiredAction(ctx, f.deps, "admin", alice.ID, idmdomain.RequiredAction("reboot"), userRulesNow); !errors.Is(err, userusecases.ErrInvalidRequiredAction) {
		t.Fatalf("err=%v, want ErrInvalidRequiredAction", err)
	}
}

//spec:covers REQ-IDMANAGEMENT-011, EX-IDMANAGEMENT-013-03: 削除予約済みの User の削除の予約が成功してイベントを発行せず、管理者自身の予約は削除予約済みでも self_delete_forbidden で拒否すること。
func TestSoftDeleteUserIsIdempotentButChecksSelfDeletionFirst(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", pendingSince(userRulesNow.Add(-time.Hour)))
	operator := f.seed("operator", func(user *userdomain.User) {
		user.Roles = []string{"admin"}
		pendingSince(userRulesNow.Add(-time.Hour))(user)
	})
	ctx := testing_tenant.Default(context.Background())
	if err := userusecases.SoftDeleteUser(ctx, f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
		t.Fatal(err)
	}
	if len(*f.events) != 0 || len(f.notifier.calls) != 0 {
		t.Fatalf("events=%v notifications=%+v, want none", eventTypes(*f.events), f.notifier.calls)
	}
	if err := userusecases.SoftDeleteUser(ctx, f.deps, userusecases.SoftDeleteUserInput{ActorUserID: operator.ID, Sub: operator.ID, Now: userRulesNow}); !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("err=%v, want ErrSelfDeleteForbidden", err)
	}
}

//spec:covers REQ-IDMANAGEMENT-048: 削除の予約が理由を UserSoftDeleted に記録し、下流のプロビジョニングへ User の削除として通知すること。
func TestSoftDeleteUserRecordsTheReasonAndNotifiesProvisioning(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	if err := userusecases.SoftDeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: alice.ID, Reason: "left the company", Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	if event := f.eventsOf("UserSoftDeleted"); len(event) != 1 || event[0].(*idmdomain.UserSoftDeleted).Reason != "left the company" {
		t.Fatalf("UserSoftDeleted=%+v, want reason recorded", event)
	}
	want := []recordedNotification{{userID: alice.ID, trigger: userports.ProvisioningUserDeleted}}
	if !slices.Equal(f.notifier.calls, want) {
		t.Fatalf("notifications=%+v, want %+v", f.notifier.calls, want)
	}
}

//spec:covers EX-IDMANAGEMENT-049-01, EX-IDMANAGEMENT-049-02: 猶予期間の終わりちょうどの復元を受け付け、一秒過ぎた復元を restore_grace_expired で拒否し、時刻を持たない User はいつでも復元できること。
func TestRestoreUserAcceptsTheExactEndOfTheGracePeriod(t *testing.T) {
	f := newUserRulesFixture(t)
	grace := time.Duration(userusecases.UserSoftDeleteGracePeriodSeconds) * time.Second
	f.seed("at-boundary", pendingSince(userRulesNow.Add(-grace)))
	f.seed("expired", pendingSince(userRulesNow.Add(-grace-time.Second)))
	f.seed("no-timestamp", func(user *userdomain.User) { user.Lifecycle.Status = idmdomain.UserStatusPendingDeletion })
	ctx := testing_tenant.Default(context.Background())
	for _, id := range []string{"at-boundary", "no-timestamp"} {
		if _, err := userusecases.RestoreUser(ctx, f.deps, "admin", id, userRulesNow); err != nil {
			t.Fatalf("%s: err=%v, want restored", id, err)
		}
		if status := f.stored(t, id).Lifecycle.Status; status != idmdomain.UserStatusActive {
			t.Fatalf("%s: status=%s, want active", id, status)
		}
	}
	if _, err := userusecases.RestoreUser(ctx, f.deps, "admin", "expired", userRulesNow); !errors.Is(err, userusecases.ErrRestoreGracePeriodExpired) {
		t.Fatalf("err=%v, want ErrRestoreGracePeriodExpired", err)
	}
	if status := f.stored(t, "expired").Lifecycle.Status; status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("拒否した復元が状態を変えた: %s", status)
	}
}

//spec:covers EX-IDMANAGEMENT-049-03, REQ-IDMANAGEMENT-049: 復元が下流のプロビジョニングへ User の再有効化を一度だけ通知すること。
func TestRestoreUserNotifiesProvisioningOfTheReEnabledUser(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", pendingSince(userRulesNow.Add(-time.Hour)))
	if _, err := userusecases.RestoreUser(testing_tenant.Default(context.Background()), f.deps, "admin", alice.ID, userRulesNow); err != nil {
		t.Fatal(err)
	}
	want := []recordedNotification{{userID: alice.ID, trigger: userports.ProvisioningUserEnabled}}
	if !slices.Equal(f.notifier.calls, want) {
		t.Fatalf("notifications=%+v, want %+v", f.notifier.calls, want)
	}
}

//spec:covers EX-IDMANAGEMENT-050-01: 有効な User の完全削除が、ユーザー名を deleted:<sub> にしてロール、属性、必須操作、セッションを消し、どのパスワードでも認証できなくし、使用量を一つ減らすこと。
func TestDeleteUserAnonymizesAnActiveUserAndReleasesItsQuota(t *testing.T) {
	f := newUserRulesFixture(t)
	ctx := testing_tenant.Default(context.Background())
	if err := f.quota.CheckAndIncrement(ctx, tenancydomain.DefaultTenantID, tenancydomain.ResourceUsers, 1); err != nil {
		t.Fatal(err)
	}
	nickname := "ally"
	alice := f.seed("alice", func(user *userdomain.User) {
		user.Roles = []string{"support"}
		user.MfaEnrolled = true
		user.Attributes = map[string]userdomain.AttributeValue{"nickname": {Type: idmdomain.AttributeTypeString, String: &nickname}}
		user.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}
	})
	if err := f.sessions.Save(ctx, &sessiondomain.LoginSession{
		// メモリのセッションストアは実時計で有効期限を判定する。
		ID: "session-1", TenantID: tenancydomain.DefaultTenantID, UserID: alice.ID,
		AuthTime: userRulesNow.Unix(), ExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := userusecases.DeleteUser(ctx, f.deps, userusecases.DeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
		t.Fatal(err)
	}
	stored := f.stored(t, alice.ID)
	if stored.PreferredUsername != "deleted:"+alice.ID || stored.Name != nil || stored.Email != nil ||
		stored.EmailVerified || stored.MfaEnrolled || len(stored.Roles) != 0 || len(stored.Attributes) != 0 ||
		len(stored.Lifecycle.RequiredActions) != 0 || !stored.IsDeleted() {
		t.Fatalf("tombstone=%+v", stored)
	}
	if ok, _ := testing_passwords.NewHasher().Verify("any-password", stored.PasswordHash); ok {
		t.Fatalf("完全削除した User のパスワードが照合に通った")
	}
	if sessions, _ := f.sessions.ListBySub(ctx, alice.ID); len(sessions) != 0 {
		t.Fatalf("sessions=%+v, want none", sessions)
	}
	usage, err := f.quota.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Users != 0 {
		t.Fatalf("users usage=%d, want 0", usage.Users)
	}
}

//spec:covers REQ-IDMANAGEMENT-050: 削除済みの User の完全削除が成功し、UserDeleted を再発行しないこと。
func TestDeleteUserDoesNothingForAnAlreadyDeletedUser(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	ctx := testing_tenant.Default(context.Background())
	for range 2 {
		if err := userusecases.DeleteUser(ctx, f.deps, userusecases.DeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
			t.Fatal(err)
		}
	}
	if deleted := f.eventsOf("UserDeleted"); len(deleted) != 1 {
		t.Fatalf("UserDeleted=%d, want 1", len(deleted))
	}
}

// failingSessionStore は、failFor の sub についてだけ、関連する記録の削除を失敗させる。
type failingSessionStore struct {
	*sessionmemory.SessionStore
	failFor string
}

var errInjected = errors.New("injected failure")

func (s *failingSessionStore) DeleteAllForSub(ctx context.Context, sub string) error {
	if sub == s.failFor {
		return errInjected
	}
	return s.SessionStore.DeleteAllForSub(ctx, sub)
}

// failUserDeletedOnce は、最初の UserDeleted の発行だけを失敗させる。
func (f *userRulesFixture) failUserDeletedOnce() {
	failed := false
	f.deps.Emit = func(event spec.DomainEvent) error {
		if _, ok := event.(*idmdomain.UserDeleted); ok && !failed {
			failed = true
			return errInjected
		}
		*f.events = append(*f.events, event)
		return nil
	}
}

// seedUsersUsage は、テナントの User の使用量を n にする。
func (f *userRulesFixture) seedUsersUsage(t *testing.T, n int) {
	t.Helper()
	if err := f.quota.CheckAndIncrement(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, tenancydomain.ResourceUsers, n); err != nil {
		t.Fatal(err)
	}
}

func (f *userRulesFixture) usersUsage(t *testing.T) int {
	t.Helper()
	usage, err := f.quota.GetUsage(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Users
}

//spec:covers EX-IDMANAGEMENT-050-02, REQ-IDMANAGEMENT-050: 匿名化の後に失敗した完全削除の再実行が、残った記録を消し、使用量を一度だけ減らし、最初の要求の操作者と理由で UserDeleted を一度だけ発行すること。
func TestDeleteUserResumesAPurgeThatFailedAfterAnonymizing(t *testing.T) {
	purge := func(f *userRulesFixture, actor string) error {
		return userusecases.DeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.DeleteUserInput{
			ActorUserID: actor, Sub: "alice", Reason: "offboarding", Now: userRulesNow,
		})
	}
	t.Run("関連する記録の削除で失敗した", func(t *testing.T) {
		f := newUserRulesFixture(t)
		f.seed("alice", nil)
		f.seedUsersUsage(t, 2)
		sessions := &failingSessionStore{SessionStore: f.sessions, failFor: "alice"}
		f.deps.SessionStore = sessions
		if err := f.sessions.Save(testing_tenant.Default(context.Background()), &sessiondomain.LoginSession{
			// メモリのセッションストアは実時計で有効期限を判定する。
			ID: "session-1", TenantID: tenancydomain.DefaultTenantID, UserID: "alice",
			AuthTime: userRulesNow.Unix(), ExpiresAt: time.Now().Add(24 * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		if err := purge(f, "admin"); !errors.Is(err, errInjected) {
			t.Fatalf("first purge err=%v, want injected failure", err)
		}
		remaining, _ := f.sessions.ListBySub(testing_tenant.Default(context.Background()), "alice")
		if !f.stored(t, "alice").IsDeleted() || len(remaining) != 1 || f.usersUsage(t) != 2 || len(f.eventsOf("UserDeleted")) != 0 {
			t.Fatalf("前提：匿名化の後、関連する記録を消す前に止まっていない")
		}

		sessions.failFor = ""
		if err := purge(f, "admin-2"); err != nil {
			t.Fatal(err)
		}
		if remaining, _ := f.sessions.ListBySub(testing_tenant.Default(context.Background()), "alice"); len(remaining) != 0 {
			t.Fatalf("sessions=%+v, want none", remaining)
		}
		if usage := f.usersUsage(t); usage != 1 {
			t.Fatalf("users usage=%d, want 1", usage)
		}
		assertSingleUserDeleted(t, f, "admin", "offboarding")
	})
	t.Run("UserDeleted の発行で失敗した", func(t *testing.T) {
		f := newUserRulesFixture(t)
		f.seed("alice", nil)
		f.seedUsersUsage(t, 2)
		f.failUserDeletedOnce()
		if err := purge(f, "admin"); !errors.Is(err, errInjected) {
			t.Fatalf("first purge err=%v, want injected failure", err)
		}
		if f.usersUsage(t) != 1 {
			t.Fatalf("前提：使用量の減算の後に止まっていない")
		}

		if err := purge(f, "admin-2"); err != nil {
			t.Fatal(err)
		}
		if usage := f.usersUsage(t); usage != 1 {
			t.Fatalf("users usage=%d, want 1 (released only once)", usage)
		}
		assertSingleUserDeleted(t, f, "admin", "offboarding")
		if err := purge(f, "admin-2"); err != nil {
			t.Fatal(err)
		}
		if deleted := f.eventsOf("UserDeleted"); len(deleted) != 1 {
			t.Fatalf("完全削除を終えた後の再実行が UserDeleted を発行した: %d", len(deleted))
		}
	})
}

func assertSingleUserDeleted(t *testing.T, f *userRulesFixture, actor, reason string) {
	t.Helper()
	deleted := f.eventsOf("UserDeleted")
	if len(deleted) != 1 {
		t.Fatalf("UserDeleted=%d, want 1", len(deleted))
	}
	if event := deleted[0].(*idmdomain.UserDeleted); event.ActorUserID != actor || event.Reason != reason || event.TargetUserID != "alice" {
		t.Fatalf("UserDeleted=%+v, want actor %s and reason %s for alice", event, actor, reason)
	}
}

//spec:covers REQ-IDMANAGEMENT-044: 保持期限の削除が、匿名化の後に失敗した管理者の完全削除を、最初の要求の操作者と理由で完了させること。
func TestPurgeExpiredSoftDeletedResumesAPurgeThatFailedAfterAnonymizing(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	f.seedUsersUsage(t, 1)
	f.failUserDeletedOnce()
	if err := userusecases.DeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: "alice", Reason: "offboarding", Now: userRulesNow,
	}); !errors.Is(err, errInjected) {
		t.Fatalf("admin purge err=%v, want injected failure", err)
	}

	if err := userusecases.PurgeExpiredSoftDeleted(testing_tenant.Default(context.Background()), f.deps, userRulesNow); err != nil {
		t.Fatal(err)
	}
	assertSingleUserDeleted(t, f, "admin", "offboarding")
	if usage := f.usersUsage(t); usage != 0 {
		t.Fatalf("users usage=%d, want 0", usage)
	}
}

//spec:covers REQ-IDMANAGEMENT-044: 一人の User の完全削除が失敗しても、保持期限の削除がほかの期限切れの User を完全削除し、失敗を返すこと。
func TestPurgeExpiredSoftDeletedContinuesPastAFailingUser(t *testing.T) {
	f := newUserRulesFixture(t)
	expiredSince := userRulesNow.Add(-time.Duration(userusecases.UserSoftDeleteGracePeriodSeconds)*time.Second - time.Hour)
	f.seed("broken", pendingSince(expiredSince))
	f.seed("healthy", pendingSince(expiredSince))
	f.deps.SessionStore = &failingSessionStore{SessionStore: f.sessions, failFor: "broken"}

	if err := userusecases.PurgeExpiredSoftDeleted(testing_tenant.Default(context.Background()), f.deps, userRulesNow); !errors.Is(err, errInjected) {
		t.Fatalf("err=%v, want the injected failure", err)
	}
	if !f.stored(t, "healthy").IsDeleted() {
		t.Fatalf("失敗した User の後の User が完全削除されていない")
	}
	if deleted := f.eventsOf("UserDeleted"); len(deleted) != 1 || deleted[0].(*idmdomain.UserDeleted).TargetUserID != "healthy" {
		t.Fatalf("UserDeleted=%+v, want only healthy", deleted)
	}
}

// REQ-IDMANAGEMENT-081 の主要な使い方：所有者の User を止める 3 つの操作のどれでも、所有する Agent が止まる。
//
//spec:covers EX-IDMANAGEMENT-081-01: 所有者の無効化、削除の予約、完全削除が、所有する Active の Agent を Disabled にして AgentDisabled を発行し、Killed の Agent と他人の Agent を変えないこと。
func TestStoppingAUserDisablesTheAgentsTheyOwn(t *testing.T) {
	stops := map[string]func(*userRulesFixture, string) error{
		"無効化": func(f *userRulesFixture, id string) error {
			_, err := userusecases.SetUserDisabled(testing_tenant.Default(context.Background()), f.deps, "admin", id, true, userRulesNow)
			return err
		},
		"削除の予約": func(f *userRulesFixture, id string) error {
			return userusecases.SoftDeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
		},
		"完全削除": func(f *userRulesFixture, id string) error {
			return userusecases.DeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.DeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
		},
	}
	for name, stop := range stops {
		t.Run(name, func(t *testing.T) {
			f := newUserRulesFixture(t)
			alice := f.seed("alice", nil)
			f.seed("bob", nil)
			f.seedAgent("deploy-bot", alice.ID, idmdomain.AgentStatusActive)
			f.seedAgent("old-bot", alice.ID, idmdomain.AgentStatusKilled)
			f.seedAgent("bob-bot", "bob", idmdomain.AgentStatusActive)

			if err := stop(f, alice.ID); err != nil {
				t.Fatal(err)
			}
			for id, want := range map[string]idmdomain.AgentStatus{
				"deploy-bot": idmdomain.AgentStatusDisabled,
				"old-bot":    idmdomain.AgentStatusKilled,
				"bob-bot":    idmdomain.AgentStatusActive,
			} {
				if got := f.agentStatus(t, id); got != want {
					t.Fatalf("Agent %s の状態=%s, want %s", id, got, want)
				}
			}
			var disabled []string
			for _, event := range *f.events {
				if e, ok := event.(*idmdomain.AgentDisabled); ok {
					disabled = append(disabled, e.AgentID)
				}
			}
			if len(disabled) != 1 || disabled[0] != "deploy-bot" {
				t.Fatalf("AgentDisabled=%v, want [deploy-bot]", disabled)
			}
		})
	}
}

//spec:covers EX-IDMANAGEMENT-081-04: 削除の予約で止めた Agent が、所有者の復元の後も Disabled のまま残ること。
func TestRestoringTheOwnerLeavesTheirAgentsDisabled(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	f.seedAgent("deploy-bot", alice.ID, idmdomain.AgentStatusActive)

	if err := userusecases.SoftDeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
		t.Fatal(err)
	}
	if got := f.agentStatus(t, "deploy-bot"); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("削除の予約の後の Agent の状態=%s, want disabled", got)
	}
	restored, err := userusecases.RestoreUser(testing_tenant.Default(context.Background()), f.deps, "admin", alice.ID, userRulesNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if restored.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatalf("復元の後の User の状態=%s, want active", restored.Lifecycle.Status)
	}
	if got := f.agentStatus(t, "deploy-bot"); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("復元の後の Agent の状態=%s, want disabled", got)
	}
}

//spec:covers EX-IDMANAGEMENT-081-05: 止まっている所有者への無効化と削除の予約の再実行が、User とそのイベントを変えずに、残っている Active の Agent だけを Disabled にすること。
func TestStoppingAStoppedUserAgainDisablesTheAgentsLeftActive(t *testing.T) {
	stops := map[string]struct {
		status idmdomain.UserStatus
		stop   func(*userRulesFixture, string) error
	}{
		"無効化": {idmdomain.UserStatusDisabled, func(f *userRulesFixture, id string) error {
			_, err := userusecases.SetUserDisabled(testing_tenant.Default(context.Background()), f.deps, "admin", id, true, userRulesNow)
			return err
		}},
		"削除の予約": {idmdomain.UserStatusPendingDeletion, func(f *userRulesFixture, id string) error {
			return userusecases.SoftDeleteUser(testing_tenant.Default(context.Background()), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
		}},
	}
	for name, tc := range stops {
		t.Run(name, func(t *testing.T) {
			f := newUserRulesFixture(t)
			alice := f.seed("alice", func(u *userdomain.User) { u.Lifecycle.Status = tc.status })
			f.seedAgent("deploy-bot", alice.ID, idmdomain.AgentStatusActive)
			f.seedAgent("old-bot", alice.ID, idmdomain.AgentStatusDisabled)

			if err := tc.stop(f, alice.ID); err != nil {
				t.Fatal(err)
			}
			if got := f.agentStatus(t, "deploy-bot"); got != idmdomain.AgentStatusDisabled {
				t.Fatalf("deploy-bot の状態=%s, want disabled", got)
			}
			user, _ := f.users.FindBySub(testing_tenant.Default(context.Background()), alice.ID)
			if user.Lifecycle.Status != tc.status {
				t.Fatalf("User の状態=%s, want %s", user.Lifecycle.Status, tc.status)
			}
			var types []string
			for _, event := range *f.events {
				types = append(types, event.EventType())
			}
			if len(types) != 1 || types[0] != "AgentDisabled" {
				t.Fatalf("events=%v, want [AgentDisabled]", types)
			}
		})
	}
}

// unreadableAgents は Agent の一覧の読み取りだけを失敗させる。
type unreadableAgents struct {
	*agentmemory.AgentRepository
}

var errAgentsUnavailable = errors.New("agents unavailable")

func (unreadableAgents) ListAll(context.Context, string) ([]*agentdomain.Agent, error) {
	return nil, errAgentsUnavailable
}

// 伝播が失敗したことを、止まっている User への再実行でも呼び出し側へ返す。成功として返すと、回収したと誤認する。
func TestStoppingAStoppedUserAgainReportsAPropagationFailure(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", func(u *userdomain.User) { u.Lifecycle.Status = idmdomain.UserStatusDisabled })
	f.deps.AgentRepo = unreadableAgents{f.agents}
	if _, err := userusecases.SetUserDisabled(testing_tenant.Default(context.Background()), f.deps, "admin", alice.ID, true, userRulesNow); !errors.Is(err, errAgentsUnavailable) {
		t.Fatalf("err=%v, want %v", err, errAgentsUnavailable)
	}
}

func (f *userRulesFixture) seedAgent(id, ownerUserID string, status idmdomain.AgentStatus) {
	if err := f.agents.Save(testing_tenant.Default(context.Background()), &agentdomain.Agent{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: ownerUserID, Status: status, Roles: []string{},
		CreatedAt: userRulesNow.Add(-time.Hour), UpdatedAt: userRulesNow.Add(-time.Hour),
	}); err != nil {
		panic(err)
	}
}

func (f *userRulesFixture) agentStatus(t *testing.T, id string) idmdomain.AgentStatus {
	t.Helper()
	agent, err := f.agents.FindByID(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, id)
	if err != nil || agent == nil {
		t.Fatalf("FindByID(%s)=(%v,%v)", id, agent, err)
	}
	return agent.Status
}
