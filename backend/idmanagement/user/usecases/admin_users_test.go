package usecases_test

// 主要ユースケース追跡: REQ-PLATFORM-001、REQ-PLATFORM-002、REQ-IDMANAGEMENT-011。

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 010-01 が言うのは再有効化の 2 つの `Then` である。戻り値は use case が組み立てるので、
// 状態は保存層から読み直す。
//
//spec:covers EX-IDMANAGEMENT-010-01: 無効化した User の再有効化で、保存された状態が Active に戻り UserEnabled が発行されること。
func TestCreateUpdateAndDisableUser(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	var events []spec.DomainEvent
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, PasswordHasher: hasher, PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error { events = append(events, event); return nil },
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	email := "bob@example.com"
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "bob", Password: "initial-password-9182",
		Email: &email, Roles: []string{"support", "support"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Roles) != 1 || user.Roles[0] != "support" {
		t.Fatalf("roles=%v", user.Roles)
	}
	if events[0].EventType() != "UserCreated" {
		t.Fatalf("event=%s", events[0].EventType())
	}
	updatedName := "Bob"
	roles := []string{"admin", "support"}
	user, err = userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Name: &updatedName, Roles: &roles, Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Name == nil || *user.Name != "Bob" || len(user.Roles) != 2 {
		t.Fatalf("updated user=%+v", user)
	}
	user, err = userusecases.SetUserDisabled(
		ctx, deps, "admin", user.ID, true, now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatal("status was not set to disabled")
	}
	if got := events[len(events)-1].EventType(); got != "UserDisabled" {
		t.Fatalf("last event=%s", got)
	}
	user, err = userusecases.SetUserDisabled(
		ctx, deps, "admin", user.ID, false, now.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatal("status was not cleared to active")
	}
	if got := events[len(events)-1].EventType(); got != "UserEnabled" {
		t.Fatalf("last event=%s", got)
	}
	stored, err := userRepo.FindBySub(ctx, user.ID)
	if err != nil || stored == nil {
		t.Fatalf("FindBySub=(%v,%v)", stored, err)
	}
	if stored.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatalf("保存された状態 = %s, want active", stored.Lifecycle.Status)
	}
}

// federatedProvisioningDeps は JIT の入口が読む 4 つの検証源 (一意性、属性スキーマ、
// リソース上限、発行先) をすべて配線した Deps を建てる。既定は上限も属性も通る側に置き、
// 具体例ごとに 1 つだけ外す。userLimit が 0 なら上限を設定しない。
func federatedProvisioningDeps(
	ctx context.Context, t *testing.T, userLimit int,
) (userusecases.AdminUserDeps, *usermemory.UserRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	schemaRepo := usermemory.NewTenantUserAttributeSchemaRepository()
	if err := schemaRepo.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{
			{Key: "department", Type: idmdomain.AttributeTypeString},
		},
	}); err != nil {
		t.Fatalf("Save schema: %v", err)
	}
	quotaRepo := tenancymemory.NewQuotaRepository()
	if userLimit > 0 {
		if err := quotaRepo.SetQuota(
			ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: &userLimit},
		); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
	}
	events := &[]spec.DomainEvent{}
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, AttrSchemaRepo: schemaRepo, QuotaRepo: quotaRepo,
		Emit: func(event spec.DomainEvent) error { *events = append(*events, event); return nil },
	}
	return deps, userRepo, events
}

// 応答の組み立てだけでは通らないよう、保存層から読み直した User の `password_hash` と
// 状態を見て、発行されたイベントの種類と対象 User まで突き合わせる。任意の名前・
// メールアドレス・属性がそのまま保存されることも、同じ読み直しで固定する。
//
//spec:covers EX-IDMANAGEMENT-001-01: 上流の検証を終えた JIT が password_hash の空な Active User を作り、UserCreated を発行すること。
func TestProvisionFederatedUserCreatesCredentiallessActiveUser(t *testing.T) {
	ctx := context.Background()
	deps, userRepo, events := federatedProvisioningDeps(ctx, t, 0)
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	email := "federated@example.com"
	name := "Federated Example"
	user, err := userusecases.ProvisionFederatedUser(
		ctx,
		deps,
		userusecases.ProvisionFederatedUserInput{
			PreferredUsername: "federated", Name: &name, Email: &email, EmailVerified: true,
			Attributes: map[string]userdomain.AttributeValue{
				"department": {Type: idmdomain.AttributeTypeString, String: new("engineering")},
			},
			Now: now,
		},
	)
	if err != nil {
		t.Fatalf("ProvisionFederatedUser: %v", err)
	}
	found, err := userRepo.FindByUsername(ctx, user.TenantID, user.PreferredUsername)
	if err != nil || found == nil || found.ID != user.ID {
		t.Fatalf("persisted user=(%+v,%v)", found, err)
	}
	if found.PasswordHash != "" {
		t.Fatalf("persisted password_hash=%q, want credentialless", found.PasswordHash)
	}
	if !found.IsActive() || !found.EmailVerified {
		t.Fatalf("persisted user=%+v", found)
	}
	if found.Name == nil || *found.Name != name || found.Email == nil || *found.Email != email {
		t.Fatalf("persisted name/email=%+v", found)
	}
	if got := found.Attributes["department"]; got.String == nil || *got.String != "engineering" {
		t.Fatalf("persisted attributes=%+v", found.Attributes)
	}
	created := lastEventOfType(*events, "UserCreated")
	if created == nil {
		t.Fatalf("UserCreated was not emitted, events=%v", eventTypes(*events))
	}
	if got := created.(*idmdomain.UserCreated).TargetUserID; got != user.ID {
		t.Fatalf("UserCreated target=%s, want %s", got, user.ID)
	}
}

// **拒否の型だけでなく効果の不在を見る。** 応答を組み立てる前に保存してしまう実装を
// 落とすため、4 つの経路それぞれで保存層の対象が増えていないことと、`UserCreated` が
// 1 件も出ていないことを確かめる。
//
//spec:covers EX-IDMANAGEMENT-001-02: 一意性・リソース上限・属性スキーマのいずれかに反する JIT が、User を作らずエラーを返すこと。
func TestProvisionFederatedUserRejectsWithoutCreatingTheUser(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	takenEmail := "taken@example.com"

	// arrange は「先に居る User」を宣言で書く。クロージャーで書くと具体例ごとの
	// 差が手続きに埋もれ、どの検証源を外しているかが読み取れなくなる。
	type seed struct {
		username string
		email    *string
	}
	tests := []struct {
		name string
		// userLimit が 0 なら上限を設定しない。
		userLimit int
		seeds     []seed
		input     userusecases.ProvisionFederatedUserInput
		wantErr   error
	}{
		{
			name:    "ユーザー名が衝突する",
			seeds:   []seed{{username: "collide"}},
			input:   userusecases.ProvisionFederatedUserInput{PreferredUsername: "collide", Now: now},
			wantErr: userusecases.ErrUsernameConflict,
		},
		{
			name:  "メールアドレスが衝突する",
			seeds: []seed{{username: "first", email: &takenEmail}},
			input: userusecases.ProvisionFederatedUserInput{
				PreferredUsername: "second", Email: &takenEmail, Now: now,
			},
			wantErr: userusecases.ErrEmailConflict,
		},
		{
			name:      "リソース上限を超える",
			userLimit: 1,
			seeds:     []seed{{username: "within-limit"}},
			input:     userusecases.ProvisionFederatedUserInput{PreferredUsername: "over-limit", Now: now},
			wantErr:   nil, // *tenancydomain.QuotaExceededError は下で個別に確かめる
		},
		{
			name: "属性スキーマに違反する",
			input: userusecases.ProvisionFederatedUserInput{
				PreferredUsername: "bad-attributes", Now: now,
				Attributes: map[string]userdomain.AttributeValue{
					"undeclared": {Type: idmdomain.AttributeTypeString, String: new("x")},
				},
			},
			wantErr: userusecases.ErrInvalidAttribute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deps, userRepo, events := federatedProvisioningDeps(ctx, t, tc.userLimit)
			for _, arranged := range tc.seeds {
				provisionFederated(ctx, t, deps, arranged.username, arranged.email, now)
			}
			before := len(*events)
			// ユーザー名の衝突では、同じ名前の User が arrange 済みで存在する。
			// 「存在しないこと」ではなく「増えていないこと」が具体例の言う効果の不在である。
			existing, err := userRepo.FindByUsername(
				ctx, tenancydomain.DefaultTenantID, tc.input.PreferredUsername,
			)
			if err != nil {
				t.Fatalf("FindByUsername before: %v", err)
			}

			user, err := userusecases.ProvisionFederatedUser(ctx, deps, tc.input)
			if err == nil {
				t.Fatalf("expected refusal, got user=%+v", user)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				var qErr *tenancydomain.QuotaExceededError
				if !errors.As(err, &qErr) || qErr.Resource != tenancydomain.ResourceUsers {
					t.Fatalf("err=%v, want *QuotaExceededError for users", err)
				}
			}
			found, err := userRepo.FindByUsername(
				ctx, tenancydomain.DefaultTenantID, tc.input.PreferredUsername,
			)
			if err != nil {
				t.Fatalf("FindByUsername after: %v", err)
			}
			switch {
			case existing == nil && found != nil:
				t.Fatalf("refused provisioning persisted %+v", found)
			case existing != nil && (found == nil || found.ID != existing.ID):
				t.Fatalf("refused provisioning replaced %+v with %+v", existing, found)
			}
			for _, event := range (*events)[before:] {
				if event.EventType() == "UserCreated" {
					t.Fatalf("refused provisioning emitted UserCreated")
				}
			}
		})
	}
}

func provisionFederated(
	ctx context.Context, t *testing.T, deps userusecases.AdminUserDeps,
	username string, email *string, now time.Time,
) {
	t.Helper()
	if _, err := userusecases.ProvisionFederatedUser(
		ctx, deps,
		userusecases.ProvisionFederatedUserInput{PreferredUsername: username, Email: email, Now: now},
	); err != nil {
		t.Fatalf("arrange ProvisionFederatedUser(%s): %v", username, err)
	}
}

func lastEventOfType(events []spec.DomainEvent, eventType string) spec.DomainEvent {
	for _, event := range slices.Backward(events) {
		if event.EventType() == eventType {
			return event
		}
	}
	return nil
}

func eventTypes(events []spec.DomainEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType())
	}
	return types
}

func TestUpdateUserExtraFieldsAndNoop(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, PasswordHasher: testing_passwords.NewHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "charlie", Password: "initial-password-9182", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "taken", Password: "initial-password-9182", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	givenName := "Charlie"
	familyName := "Example"
	email := "charlie@example.com"
	verified := true
	updated, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID,
		GivenName: &givenName, FamilyName: &familyName, Email: &email, EmailVerified: &verified,
		Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.GivenName == nil || *updated.GivenName != givenName || updated.FamilyName == nil || *updated.FamilyName != familyName ||
		updated.Email == nil || *updated.Email != email || !updated.EmailVerified {
		t.Fatalf("updated user=%+v", updated)
	}

	noChange, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if noChange.ID != user.ID {
		t.Fatalf("no-op returned %+v", noChange)
	}

	taken := "taken"
	if _, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, PreferredUsername: &taken,
	}); !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("expected ErrUsernameConflict, got %v", err)
	}
}

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	repo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	repo.Seed(&userdomain.User{
		ID: "existing", PreferredUsername: "bob", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	_, err := userusecases.CreateUser(context.Background(), userusecases.AdminUserDeps{
		UserRepo: repo, PasswordHasher: testing_passwords.NewHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
	}, userusecases.CreateUserInput{
		PreferredUsername: "bob", Password: "initial-password-9182",
	})
	if !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("error=%v, want ErrUsernameConflict", err)
	}
}

// TestCreateUser_rejectsWhenHardQuotaExceeded is a wi-160 T004.1 RED test for
// the SCL scenario "Hard Quota を超過したリソース作成は拒否される"
// (spec/contexts/tenancy.yaml), applied to the users resource.
func TestCreateUser_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := context.Background()
	repo := usermemory.NewUserRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	deps := userusecases.AdminUserDeps{
		UserRepo: repo, PasswordHasher: testing_passwords.NewHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		QuotaRepo:           quotaRepo,
	}
	now := time.Now().UTC()
	if _, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		PreferredUsername: "alice", Password: "initial-password-9182", Now: now,
	}); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		PreferredUsername: "bob", Password: "initial-password-9182", Now: now,
	})
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %v", err)
	}
	if qErr.Resource != tenancydomain.ResourceUsers {
		t.Fatalf("unexpected resource: %s", qErr.Resource)
	}
	if existing, err := repo.FindByUsername(ctx, tenancydomain.DefaultTenantID, "bob"); err != nil || existing != nil {
		t.Fatalf("expected rejected create to not persist bob, found=%v err=%v", existing, err)
	}
}

// TestDeleteUser_decrementsQuotaUsage is a wi-160 T004.1 RED test: hard
// deleting a user must free its quota slot so a subsequent create at the same
// limit succeeds.
func TestDeleteUser_decrementsQuotaUsage(t *testing.T) {
	ctx := context.Background()
	repo := usermemory.NewUserRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	deps := userusecases.AdminUserDeps{
		UserRepo: repo, PasswordHasher: testing_passwords.NewHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		QuotaRepo:           quotaRepo,
	}
	now := time.Now().UTC()
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		PreferredUsername: "alice", Password: "initial-password-9182", Now: now,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: user.ID, Now: now,
	}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		PreferredUsername: "bob", Password: "initial-password-9182", Now: now,
	}); err != nil {
		t.Fatalf("expected create to succeed after delete freed quota, got %v", err)
	}
}

func TestDeleteUserAnonymizesAndCascades(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	deviceStore := oauth2memory.NewDeviceCodeStore()
	sessionStore := sessionmemory.NewSessionStore()
	mfaRepo := totpmemory.NewMfaFactorRepository()
	hasher := testing_passwords.NewHasher()
	var events []spec.DomainEvent
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, ConsentRepo: consentRepo, RefreshStore: refreshStore,
		DeviceCodeStore: deviceStore, SessionStore: sessionStore, MfaFactorRepo: mfaRepo,
		PasswordHasher: hasher, PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error { events = append(events, event); return nil },
	}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "alice", Password: "initial-password-9182",
		Roles: []string{"support"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Seed cascade artifacts.
	_ = consentRepo.Save(ctx, tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: user.ID, ClientID: "client-a",
		Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.AddDate(1, 0, 0),
	})
	_ = refreshStore.Save(ctx, &oauthdomain.RefreshTokenRecord{
		ID: "rt-1", TenantID: tenancydomain.DefaultTenantID, Hash: "hash-1",
		FamilyID: "fam-1", ClientID: "client-a", UserID: user.ID,
		Scopes: []string{"openid"}, IssuedAt: now,
		ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	})
	_ = sessionStore.Save(ctx, &sessiondomain.LoginSession{
		ID: "sess-1", TenantID: tenancydomain.DefaultTenantID, UserID: user.ID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:mace:incommon:iap:silver",
		ExpiresAt: now.Add(time.Hour),
	})
	totpSecret := "JBSWY3DPEHPK3PXP"
	_ = mfaRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: &totpSecret, CreatedAt: now,
	})

	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: user.ID, Reason: "leaving company", Now: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if last, ok := events[len(events)-1].(*idmdomain.UserDeleted); !ok || last.TargetUserID != user.ID || last.Reason != "leaving company" {
		t.Fatalf("expected UserDeleted event with target=%s reason set, got %+v", user.ID, events[len(events)-1])
	}
	tombstone, err := userRepo.FindBySubIncludingDeleted(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tombstone == nil || !tombstone.IsDeleted() {
		t.Fatalf("expected tombstone with status=deleted, got %+v", tombstone)
	}
	if tombstone.PreferredUsername != "deleted:"+user.ID {
		t.Fatalf("preferred_username not anonymized: %s", tombstone.PreferredUsername)
	}
	if tombstone.Email != nil || tombstone.Name != nil || len(tombstone.Roles) != 0 || tombstone.MfaEnrolled {
		t.Fatalf("PII not anonymized: %+v", tombstone)
	}
	if seen, _ := userRepo.FindBySub(ctx, user.ID); seen != nil {
		t.Fatalf("FindBySub returned deleted user")
	}
	// Cascade verification.
	if remaining, _ := consentRepo.FindAll(ctx, tenancydomain.DefaultTenantID); len(remaining) != 0 {
		t.Fatalf("consent cascade leaked: %+v", remaining)
	}
	if rec, _ := refreshStore.FindByHash(ctx, "hash-1"); rec != nil {
		t.Fatalf("refresh cascade leaked: %+v", rec)
	}
	if sess, _ := sessionStore.Find(ctx, "sess-1"); sess != nil {
		t.Fatalf("session cascade leaked: %+v", sess)
	}
	if factors, _ := mfaRepo.ListBySub(ctx, user.ID); len(factors) != 0 {
		t.Fatalf("mfa cascade leaked: %+v", factors)
	}
	// Re-delete is no-op (no new UserDeleted event).
	prev := len(events)
	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: user.ID, Now: now.Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("idempotent delete failed: %v", err)
	}
	if len(events) != prev {
		t.Fatalf("idempotent delete emitted extra events")
	}
}

func TestDeleteUserRejectsSelfDelete(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	err := userusecases.DeleteUser(ctx, userusecases.AdminUserDeps{UserRepo: userRepo},
		userusecases.DeleteUserInput{ActorUserID: "admin-1", Sub: "admin-1", Now: now})
	if !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("error=%v, want ErrSelfDeleteForbidden", err)
	}
}

func TestSetUserDisabledRejectsSelfDisable(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}

	// admin が自身を無効化しようとすると自爆防止に弾かれる。
	_, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-1", true, now)
	if !errors.Is(err, userusecases.ErrSelfDisableForbidden) {
		t.Fatalf("disable self error=%v, want ErrSelfDisableForbidden", err)
	}

	// enable 方向は自身に対しても許可する (アクセス回復のみで誤操作リスクが低い)。
	if _, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-1", false, now); err != nil {
		t.Fatalf("enable self error=%v, want nil", err)
	}
}

func TestSetUserDisabledAllowsDisablingOtherAdmin(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-2", PreferredUsername: "other-admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}

	user, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-2", true, now)
	if err != nil {
		t.Fatalf("disable other admin error=%v, want nil", err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("status=%v, want disabled", user.Lifecycle.Status)
	}
}

// softDeleteTestDeps は soft-delete 系テスト用に cascade 対象リポジトリを揃えた
// deps と consent リポジトリ (cascade 温存の確認用) を返す。
func softDeleteTestDeps(events *[]spec.DomainEvent) (userusecases.AdminUserDeps, *oauth2memory.ConsentRepository, *usermemory.UserRepository) {
	userRepo := usermemory.NewUserRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, ConsentRepo: consentRepo,
		RefreshStore: oauth2memory.NewRefreshTokenStore(), SessionStore: sessionmemory.NewSessionStore(),
		MfaFactorRepo: totpmemory.NewMfaFactorRepository(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		Emit: func(event spec.DomainEvent) error { *events = append(*events, event); return nil },
	}
	return deps, consentRepo, userRepo
}

func TestSoftDeleteUserSetsPendingDeletionWithoutCascade(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, consentRepo, userRepo := softDeleteTestDeps(&events)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		Roles: []string{"support"}, CreatedAt: now, UpdatedAt: now,
	})
	_ = consentRepo.Save(ctx, tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: "alice-1", ClientID: "client-a",
		Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.AddDate(1, 0, 0),
	})

	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Reason: "maybe leaving", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	last, ok := events[len(events)-1].(*idmdomain.UserSoftDeleted)
	if !ok || last.TargetUserID != "alice-1" || last.Reason != "maybe leaving" {
		t.Fatalf("expected UserSoftDeleted with target/reason, got %+v", events[len(events)-1])
	}
	// status は PendingDeletion で、FindBySub でまだ見える (tombstone と違い可視)。
	user, _ := userRepo.FindBySub(ctx, "alice-1")
	if user == nil || !user.IsSoftDeleted() || user.IsActive() || user.IsDeleted() {
		t.Fatalf("expected visible soft-deleted user, got %+v", user)
	}
	// PII / cascade artifact は温存される。
	if user.Email != nil && *user.Email == "deleted:alice-1" {
		t.Fatal("PII was anonymized on soft-delete")
	}
	if remaining, _ := consentRepo.FindAll(ctx, tenancydomain.DefaultTenantID); len(remaining) != 1 {
		t.Fatalf("consent must be preserved on soft-delete, got %+v", remaining)
	}
	// 冪等: 再 soft-delete は追加イベントを出さない。
	prev := len(events)
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != prev {
		t.Fatal("idempotent soft-delete emitted extra events")
	}
}

// 具体例は削除の予約と復元の 2 段で、段ごとに状態とイベントの 2 つを言う。
// 4 つの `Then` に 4 つの観測を置き、状態はいずれも保存層から読み直す。
//
//spec:covers EX-IDMANAGEMENT-011-01: 削除の予約で PendingDeletion と UserSoftDeleted、復元で Active と UserRestored になること。
func TestRestoreUserReturnsToActive(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	email := "alice@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash", Email: &email,
		Roles: []string{"support"}, CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	pending, err := userRepo.FindBySub(ctx, "alice-1")
	if err != nil || pending == nil || !pending.IsSoftDeleted() {
		t.Fatalf("削除の予約後の状態 = (%+v, %v)", pending, err)
	}
	if got := events[len(events)-1].EventType(); got != "UserSoftDeleted" {
		t.Fatalf("last event=%s, want UserSoftDeleted", got)
	}

	restored, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !restored.IsActive() || restored.Email == nil || *restored.Email != email {
		t.Fatalf("expected active restored user with PII intact, got %+v", restored)
	}
	if got := events[len(events)-1].EventType(); got != "UserRestored" {
		t.Fatalf("last event=%s, want UserRestored", got)
	}
	stored, err := userRepo.FindBySub(ctx, "alice-1")
	if err != nil || stored == nil || !stored.IsActive() {
		t.Fatalf("復元後の保存された状態 = (%+v, %v)", stored, err)
	}
}

// 具体例の `Given` は PendingDeletion である。有効な User をそのまま完全削除する経路とは
// 別で、既存のテストが押さえているのは後者と自動 purge だった。
//
//spec:covers EX-IDMANAGEMENT-013-01: PendingDeletion の User を管理者が完全削除すると、状態が Deleted になり UserDeleted が発行されること。
func TestPurgePendingDeletionUserTombstonesAndEmitsUserDeleted(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if pending, _ := userRepo.FindBySub(ctx, "alice-1"); pending == nil || !pending.IsSoftDeleted() {
		t.Fatalf("前提が壊れている: %+v", pending)
	}

	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	tombstone, err := userRepo.FindBySubIncludingDeleted(ctx, "alice-1")
	if err != nil || tombstone == nil || !tombstone.IsDeleted() {
		t.Fatalf("完全削除後の状態 = (%+v, %v)", tombstone, err)
	}
	deleted := lastEventOfType(events, "UserDeleted")
	if deleted == nil {
		t.Fatalf("UserDeleted が発行されていない: %v", eventTypes(events))
	}
	if got := deleted.(*idmdomain.UserDeleted).TargetUserID; got != "alice-1" {
		t.Fatalf("UserDeleted target=%s", got)
	}
}

func TestRestoreUserRejectsNonPendingAndExpired(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	deps.SoftDeleteGraceSeconds = 60
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	// Active user への restore は ErrUserNotPendingDeletion。
	if _, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now); !errors.Is(err, userusecases.ErrUserNotPendingDeletion) {
		t.Fatalf("error=%v, want ErrUserNotPendingDeletion", err)
	}
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	// 猶予期間 (60s) 経過後の restore は ErrRestoreGracePeriodExpired。
	if _, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now.Add(2*time.Minute)); !errors.Is(err, userusecases.ErrRestoreGracePeriodExpired) {
		t.Fatalf("error=%v, want ErrRestoreGracePeriodExpired", err)
	}
}

func TestSoftDeleteAndRestoreRejectSelf(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin-1", Sub: "admin-1", Now: now,
	}); !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("soft-delete self error=%v, want ErrSelfDeleteForbidden", err)
	}
	if _, err := userusecases.RestoreUser(ctx, deps, "admin-1", "admin-1", now); !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("restore self error=%v, want ErrSelfDeleteForbidden", err)
	}
}

func TestPurgeExpiredSoftDeletedAnonymizesAfterGrace(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	deps.SoftDeleteGraceSeconds = 1
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	// 猶予期間内 (grace=1s) の purge は no-op。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	if user, _ := userRepo.FindBySub(ctx, "alice-1"); user == nil || !user.IsSoftDeleted() {
		t.Fatal("user must remain pending within grace")
	}
	// 猶予期間経過後の purge は anonymize cascade を実行し UserDeleted(auto_purge) を emit。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, deps, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	tombstone, _ := userRepo.FindBySubIncludingDeleted(ctx, "alice-1")
	if tombstone == nil || !tombstone.IsDeleted() {
		t.Fatalf("expected tombstone after auto-purge, got %+v", tombstone)
	}
	last, ok := events[len(events)-1].(*idmdomain.UserDeleted)
	if !ok || last.Reason != "auto_purge" {
		t.Fatalf("expected UserDeleted(auto_purge), got %+v", events[len(events)-1])
	}
}
