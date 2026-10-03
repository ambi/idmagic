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
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
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
	user, err := f.users.FindBySubIncludingDeleted(context.Background(), id)
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

//spec:covers EX-IDMANAGEMENT-042-01, EX-IDMANAGEMENT-042-02: 作成がユーザー名の前後の空白を除き、大文字と小文字だけが異なるユーザー名を別の User として受け付け、完全に同じユーザー名を username_conflict で拒否すること。
func TestCreateUserTrimsTheUsernameAndComparesItCaseSensitively(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	ctx := context.Background()
	carol, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: " carol ", Password: "initial-password-9182", Now: userRulesNow,
	})
	if err != nil || carol.PreferredUsername != "carol" {
		t.Fatalf("user=%+v err=%v, want username carol", carol, err)
	}
	upper, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "Alice", Password: "initial-password-9182", Now: userRulesNow,
	})
	if err != nil || upper.ID == "alice" {
		t.Fatalf("Alice: user=%+v err=%v, want a second user", upper, err)
	}
	if _, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "alice", Password: "initial-password-9182", Now: userRulesNow,
	}); !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("同じユーザー名: err=%v, want ErrUsernameConflict", err)
	}
}

//spec:covers REQ-IDMANAGEMENT-042: テナントのパスワードポリシーに違反する作成を拒否し、User を作らないこと。
func TestCreateUserAppliesTheTenantPasswordPolicy(t *testing.T) {
	f := newUserRulesFixture(t)
	twenty := 20
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{
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

//spec:covers REQ-IDMANAGEMENT-042: 管理者の作成がほかの User と同じメールアドレスを拒否しないこと。
func TestCreateUserAcceptsAnEmailAnotherUserAlreadyHas(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	email := "alice@example.test"
	if _, err := userusecases.CreateUser(context.Background(), f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "initial-password-9182", Email: &email, Now: userRulesNow,
	}); err != nil {
		t.Fatalf("err=%v, want accepted", err)
	}
}

//spec:covers EX-IDMANAGEMENT-043-01, REQ-IDMANAGEMENT-043: JIT が大文字と小文字を区別せずにメールアドレスの衝突を拒否し、空白だけのメールアドレスを未設定として扱い、identity-broker を操作者とすること。
func TestProvisionFederatedUserNormalizesTheEmailAndRejectsACaseInsensitiveConflict(t *testing.T) {
	f := newUserRulesFixture(t)
	f.seed("alice", nil)
	ctx := context.Background()
	conflict := "ALICE@example.test"
	if _, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "alice-federated", Email: &conflict, Now: userRulesNow,
	}); !errors.Is(err, userusecases.ErrEmailConflict) {
		t.Fatalf("衝突: err=%v, want ErrEmailConflict", err)
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

//spec:covers EX-IDMANAGEMENT-043-03: JIT が作成の時点で動的グループの規則を評価せず、規則に一致する User も所属させないこと。
func TestProvisionFederatedUserDoesNotEvaluateDynamicGroups(t *testing.T) {
	f := newUserRulesFixture(t)
	ctx := context.Background()
	groups, schemas := dynamicDepartmentGroup(t)
	f.deps.GroupRepo, f.deps.AttrSchemaRepo = groups, schemas
	engineering := "Engineering"
	user, err := userusecases.ProvisionFederatedUser(ctx, f.deps, userusecases.ProvisionFederatedUserInput{
		PreferredUsername: "eve", Now: userRulesNow,
		Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &engineering}},
	})
	if err != nil {
		t.Fatal(err)
	}
	members, err := groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, "dyn-eng")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 0 {
		t.Fatalf("JIT の User %s が動的グループに所属した: %+v", user.ID, members)
	}

	// 対照: 管理者の作成は同じ属性の User を所属させる。
	admin, err := userusecases.CreateUser(ctx, f.deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "frank", Password: "initial-password-9182", Now: userRulesNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	attrs := map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &engineering}}
	if _, err := userusecases.UpdateUser(ctx, f.deps, userusecases.UpdateUserInput{ActorUserID: "admin", Sub: admin.ID, Attributes: &attrs, Now: userRulesNow}); err != nil {
		t.Fatal(err)
	}
	members, _ = groups.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, "dyn-eng")
	if len(members) != 1 || members[0].UserID != admin.ID {
		t.Fatalf("members=%+v, want the admin-created user only", members)
	}
}

// dynamicDepartmentGroup は `department == "Engineering"` の有効な規則を持つ動的グループ
// `dyn-eng` と、`department` を定義した属性スキーマを返す。
func dynamicDepartmentGroup(t *testing.T) (*groupmemory.GroupRepository, *usermemory.TenantUserAttributeSchemaRepository) {
	t.Helper()
	ctx := context.Background()
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
	if err := userusecases.PurgeExpiredSoftDeleted(context.Background(), f.deps, userRulesNow); err != nil {
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
	if _, err := userusecases.UpdateUser(context.Background(), f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: alice.ID, Name: alice.Name, Attributes: &attrs, Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	updates := f.eventsOf("UserUpdated")
	if len(updates) != 1 || !slices.Equal(updates[0].(*idmdomain.UserUpdated).ChangedFields, []string{"department", "title"}) {
		t.Fatalf("UserUpdated=%+v, want changed_fields [department title]", updates)
	}

	before := f.stored(t, alice.ID).UpdatedAt
	if _, err := userusecases.UpdateUser(context.Background(), f.deps, userusecases.UpdateUserInput{
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
	if err := schemas.Save(context.Background(), &userdomain.TenantUserAttributeSchema{
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

//spec:covers EX-IDMANAGEMENT-045-03: 確認済みのメールアドレスを別のアドレスへ変える更新が email_verified を true のまま残すこと。
func TestUpdateUserKeepsEmailVerifiedWhenTheAddressChanges(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", nil)
	email := "alice@new.example.test"
	if _, err := userusecases.UpdateUser(context.Background(), f.deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: alice.ID, Email: &email, Now: userRulesNow,
	}); err != nil {
		t.Fatal(err)
	}
	stored := f.stored(t, alice.ID)
	if *stored.Email != email || !stored.EmailVerified {
		t.Fatalf("email=%q verified=%v, want the new address still verified", *stored.Email, stored.EmailVerified)
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	if err := userusecases.SoftDeleteUser(context.Background(), f.deps, userusecases.SoftDeleteUserInput{
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
	ctx := context.Background()
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

//spec:covers EX-IDMANAGEMENT-049-03: 復元が下流のプロビジョニングへ通知しないこと。
func TestRestoreUserDoesNotNotifyProvisioning(t *testing.T) {
	f := newUserRulesFixture(t)
	alice := f.seed("alice", pendingSince(userRulesNow.Add(-time.Hour)))
	if _, err := userusecases.RestoreUser(context.Background(), f.deps, "admin", alice.ID, userRulesNow); err != nil {
		t.Fatal(err)
	}
	if len(f.notifier.calls) != 0 {
		t.Fatalf("notifications=%+v, want none", f.notifier.calls)
	}
}

//spec:covers EX-IDMANAGEMENT-050-01: 有効な User の完全削除が、ユーザー名を deleted:<sub> にしてロール、属性、必須操作、セッションを消し、どのパスワードでも認証できなくし、使用量を一つ減らすこと。
func TestDeleteUserAnonymizesAnActiveUserAndReleasesItsQuota(t *testing.T) {
	f := newUserRulesFixture(t)
	ctx := context.Background()
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
		ID: "session-1", TenantID: tenancydomain.DefaultTenantID, UserID: alice.ID,
		AuthTime: userRulesNow.Unix(), ExpiresAt: userRulesNow.Add(time.Hour),
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
	ctx := context.Background()
	for range 2 {
		if err := userusecases.DeleteUser(ctx, f.deps, userusecases.DeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
			t.Fatal(err)
		}
	}
	if deleted := f.eventsOf("UserDeleted"); len(deleted) != 1 {
		t.Fatalf("UserDeleted=%d, want 1", len(deleted))
	}
}

// REQ-IDMANAGEMENT-081 の主要な使い方：所有者の User を止める 3 つの操作のどれでも、所有する Agent が止まる。
//
//spec:covers EX-IDMANAGEMENT-081-01: 所有者の無効化、削除の予約、完全削除が、所有する Active の Agent を Disabled にして AgentDisabled を発行し、Killed の Agent と他人の Agent を変えないこと。
func TestStoppingAUserDisablesTheAgentsTheyOwn(t *testing.T) {
	stops := map[string]func(*userRulesFixture, string) error{
		"無効化": func(f *userRulesFixture, id string) error {
			_, err := userusecases.SetUserDisabled(context.Background(), f.deps, "admin", id, true, userRulesNow)
			return err
		},
		"削除の予約": func(f *userRulesFixture, id string) error {
			return userusecases.SoftDeleteUser(context.Background(), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
		},
		"完全削除": func(f *userRulesFixture, id string) error {
			return userusecases.DeleteUser(context.Background(), f.deps, userusecases.DeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
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

	if err := userusecases.SoftDeleteUser(context.Background(), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: userRulesNow}); err != nil {
		t.Fatal(err)
	}
	if got := f.agentStatus(t, "deploy-bot"); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("削除の予約の後の Agent の状態=%s, want disabled", got)
	}
	restored, err := userusecases.RestoreUser(context.Background(), f.deps, "admin", alice.ID, userRulesNow.Add(time.Hour))
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
			_, err := userusecases.SetUserDisabled(context.Background(), f.deps, "admin", id, true, userRulesNow)
			return err
		}},
		"削除の予約": {idmdomain.UserStatusPendingDeletion, func(f *userRulesFixture, id string) error {
			return userusecases.SoftDeleteUser(context.Background(), f.deps, userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: id, Now: userRulesNow})
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
			user, _ := f.users.FindBySub(context.Background(), alice.ID)
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
	if _, err := userusecases.SetUserDisabled(context.Background(), f.deps, "admin", alice.ID, true, userRulesNow); !errors.Is(err, errAgentsUnavailable) {
		t.Fatalf("err=%v, want %v", err, errAgentsUnavailable)
	}
}

func (f *userRulesFixture) seedAgent(id, ownerUserID string, status idmdomain.AgentStatus) {
	if err := f.agents.Save(context.Background(), &agentdomain.Agent{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: ownerUserID, Status: status, Roles: []string{},
		CreatedAt: userRulesNow.Add(-time.Hour), UpdatedAt: userRulesNow.Add(-time.Hour),
	}); err != nil {
		panic(err)
	}
}

func (f *userRulesFixture) agentStatus(t *testing.T, id string) idmdomain.AgentStatus {
	t.Helper()
	agent, err := f.agents.FindByID(context.Background(), tenancydomain.DefaultTenantID, id)
	if err != nil || agent == nil {
		t.Fatalf("FindByID(%s)=(%v,%v)", id, agent, err)
	}
	return agent.Status
}
