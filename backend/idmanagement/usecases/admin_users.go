package usecases

// 管理者向け User ライフサイクル操作 (Create / Update / Disable / Enable)。
// SCL の IdManagement bounded context が所有する admin インターフェース群:
// CreateAdminUser / UpdateAdminUser / DisableAdminUser / EnableAdminUser。

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	ErrUsernameConflict = errors.New("preferred username already exists")
	ErrInvalidRole      = errors.New("role must not be empty")
	// ErrSelfDeleteForbidden は admin / system_admin が自身を削除しようとした場合に
	// 返る (ADR-036 の自爆防止)。
	ErrSelfDeleteForbidden = errors.New("admins cannot delete themselves")
	// ErrSelfDisableForbidden は admin / system_admin が自身を無効化しようとした
	// 場合に返る。delete 側 (ErrSelfDeleteForbidden) と対称な自爆防止で、誤操作で
	// 自身の管理画面アクセスを即時遮断する事故を防ぐ。enable 方向には適用しない。
	ErrSelfDisableForbidden = errors.New("admins cannot disable themselves")
	// ErrInvalidAttribute は attributes が実効スキーマ (組み込み ∪ tenant) に
	// 適合しない場合に返る (ADR-040)。
	ErrInvalidAttribute = errors.New("attribute does not conform to schema")
)

// deletedPasswordHashSentinel は ADR-036 の tombstone 用に PasswordHash へ設定する
// 非ハッシュ形式の値。Argon2id のフォーマットと一致しないため、どんなパスワードでも
// 認証に通らないが、`z.String().Required()` の schema 制約は満たす。
const deletedPasswordHashSentinel = "$deleted$"

type AdminUserDeps struct {
	UserRepo            idmports.UserRepository
	GroupRepo           idmports.GroupRepository
	AttrSchemaRepo      tenantports.TenantUserAttributeSchemaRepository
	ConsentRepo         oauthports.ConsentRepository
	RefreshStore        oauthports.RefreshTokenStore
	DeviceCodeStore     oauthports.DeviceCodeStore
	SessionStore        authnports.SessionStore
	MfaFactorRepo       authnports.MfaFactorRepository
	PasswordHasher      authnports.PasswordHasher
	PasswordHistoryRepo authnports.PasswordHistoryRepository
	Emit                func(spec.DomainEvent) error
	// UserMutationCommitter は User mutation を確定させる境界 port。IdGovernance が
	// 実装し、User 保存と派生する LifecycleWorkflow run 生成を同一トランザクションで
	// 確定する (wi-237, ADR-117)。nil のとき UserRepo.Save に fallback する。
	UserMutationCommitter idmports.UserMutationCommitter
	// SoftDeleteGraceSeconds は soft-delete の猶予期間 (秒)。0 のとき
	// UserSoftDeleteGracePeriodSeconds を既定として使う。テストで短縮するために注入する。
	SoftDeleteGraceSeconds int
}

// graceSeconds は soft-delete の実効猶予期間 (秒) を返す。未指定 (0) なら既定値。
func (d AdminUserDeps) graceSeconds() int {
	if d.SoftDeleteGraceSeconds > 0 {
		return d.SoftDeleteGraceSeconds
	}
	return UserSoftDeleteGracePeriodSeconds
}

type CreateUserInput struct {
	ActorUserID       string
	PreferredUsername string
	Password          string
	Name              *string
	Email             *string
	EmailVerified     bool
	Roles             []string
	Now               time.Time
}

func CreateUser(ctx context.Context, deps AdminUserDeps, in CreateUserInput) (*idmdomain.User, error) {
	username := strings.TrimSpace(in.PreferredUsername)
	if username == "" {
		return nil, errors.New("preferred username is required")
	}
	tenantID := tenancy.TenantID(ctx)
	existing, err := deps.UserRepo.FindByUsername(ctx, tenantID, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameConflict
	}
	result := authusecases.ValidatePassword(in.Password)
	if !result.OK {
		return nil, &authusecases.PasswordPolicyError{Violations: result.Violations}
	}
	roles, err := normalizeRoles(in.Roles)
	if err != nil {
		return nil, err
	}
	passwordHash, err := deps.PasswordHasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	now := normalizedNow(in.Now)
	user := &idmdomain.User{
		ID: id, TenantID: tenantID, PreferredUsername: username, PasswordHash: passwordHash,
		Name: in.Name, Email: in.Email, EmailVerified: in.EmailVerified, Roles: roles,
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	if err := captureUserMutation(ctx, deps, nil, user, nil, now); err != nil {
		return nil, err
	}
	if deps.GroupRepo != nil {
		if err := SyncDynamicGroupsForUser(ctx, DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, user, now); err != nil {
			return nil, err
		}
	}
	if err := deps.PasswordHistoryRepo.Add(ctx, user.ID, passwordHash, now); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.UserCreated{At: now, TenantID: user.TenantID, ActorUserID: in.ActorUserID, TargetUserID: user.ID}); err != nil {
		return nil, err
	}
	return user, nil
}

// captureUserMutation persists the mutated user and any governance side effects
// (lifecycle workflow runs) atomically. When a UserMutationCommitter is wired,
// IdGovernance owns the transactional capture; the nil fallback keeps
// lightweight unit-test wiring usable (wi-237, ADR-117).
func captureUserMutation(ctx context.Context, deps AdminUserDeps, before, after *idmdomain.User, changed []string, now time.Time) error {
	if deps.UserMutationCommitter == nil {
		return deps.UserRepo.Save(ctx, after)
	}
	return deps.UserMutationCommitter.CommitUserMutation(ctx, idmports.UserMutation{Before: before, After: after, Changed: changed, Now: now})
}

type UpdateUserInput struct {
	ActorUserID       string
	Sub               string
	PreferredUsername *string
	Name              *string
	GivenName         *string
	FamilyName        *string
	Email             *string
	EmailVerified     *bool
	Roles             *[]string
	// Attributes は指定時に attributes 全体を置換する (実効スキーマで検証)。
	Attributes *map[string]idmdomain.AttributeValue
	Now        time.Time
}

func UpdateUser(ctx context.Context, deps AdminUserDeps, in UpdateUserInput) (*idmdomain.User, error) {
	user, err := deps.UserRepo.FindBySub(ctx, in.Sub)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrUserNotFound
	}
	updated := *user
	changed := []string{}
	if in.PreferredUsername != nil {
		username := strings.TrimSpace(*in.PreferredUsername)
		if username == "" {
			return nil, errors.New("preferred username must not be empty")
		}
		if username != user.PreferredUsername {
			existing, err := deps.UserRepo.FindByUsername(ctx, user.TenantID, username)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != user.ID {
				return nil, ErrUsernameConflict
			}
			updated.PreferredUsername = username
			changed = append(changed, "preferred_username")
		}
	}
	if in.Name != nil && !equalOptionalString(user.Name, in.Name) {
		updated.Name = in.Name
		changed = append(changed, "name")
	}
	if in.GivenName != nil && !equalOptionalString(user.GivenName, in.GivenName) {
		updated.GivenName = in.GivenName
		changed = append(changed, "given_name")
	}
	if in.FamilyName != nil && !equalOptionalString(user.FamilyName, in.FamilyName) {
		updated.FamilyName = in.FamilyName
		changed = append(changed, "family_name")
	}
	if in.Attributes != nil {
		defs, err := effectiveUserAttributeDefs(ctx, deps.AttrSchemaRepo, user.TenantID)
		if err != nil {
			return nil, err
		}
		if err := idmdomain.ValidateAttributes(*in.Attributes, defs); err != nil {
			return nil, errors.Join(ErrInvalidAttribute, err)
		}
		updated.Attributes = *in.Attributes
		changed = append(changed, changedAttributeFields(user.Attributes, updated.Attributes)...)
	}
	if in.Email != nil && !equalOptionalString(user.Email, in.Email) {
		updated.Email = in.Email
		changed = append(changed, "email")
	}
	if in.EmailVerified != nil && *in.EmailVerified != user.EmailVerified {
		updated.EmailVerified = *in.EmailVerified
		changed = append(changed, "email_verified")
	}
	if in.Roles != nil {
		roles, err := normalizeRoles(*in.Roles)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(roles, user.Roles) {
			updated.Roles = roles
			changed = append(changed, "roles")
		}
	}
	if len(changed) == 0 {
		return &updated, nil
	}
	now := normalizedNow(in.Now)
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := captureUserMutation(ctx, deps, user, &updated, changed, now); err != nil {
		return nil, err
	}
	if deps.GroupRepo != nil {
		if err := SyncDynamicGroupsForUser(ctx, DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, &updated, now); err != nil {
			return nil, err
		}
	}
	if err := adminEmit(deps.Emit, &idmdomain.UserUpdated{
		At: now, TenantID: user.TenantID, ActorUserID: in.ActorUserID, TargetUserID: user.ID, ChangedFields: changed,
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func SetUserDisabled(
	ctx context.Context,
	deps AdminUserDeps,
	actorUserID, targetUserID string,
	disabled bool,
	now time.Time,
) (*idmdomain.User, error) {
	user, err := deps.UserRepo.FindBySub(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrUserNotFound
	}
	if disabled && actorUserID == user.ID && hasPrivilegedRole(user.Roles) {
		return nil, ErrSelfDisableForbidden
	}
	updated := *user
	now = normalizedNow(now)
	if disabled {
		if updated.Lifecycle.Status == idmdomain.UserStatusDisabled {
			return &updated, nil
		}
		updated.Lifecycle.Status = idmdomain.UserStatusDisabled
		updated.Lifecycle.StatusChangedAt = &now
	} else {
		if updated.Lifecycle.Status == idmdomain.UserStatusActive {
			return &updated, nil
		}
		updated.Lifecycle.Status = idmdomain.UserStatusActive
		updated.Lifecycle.StatusChangedAt = &now
	}
	updated.UpdatedAt = now
	if err := captureUserMutation(ctx, deps, user, &updated, []string{"status"}, now); err != nil {
		return nil, err
	}
	if deps.GroupRepo != nil {
		if err := SyncDynamicGroupsForUser(ctx, DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, &updated, now); err != nil {
			return nil, err
		}
	}
	var emitErr error
	if disabled {
		emitErr = adminEmit(deps.Emit, &idmdomain.UserDisabled{At: now, TenantID: updated.TenantID, ActorUserID: actorUserID, TargetUserID: targetUserID})
	} else {
		emitErr = adminEmit(deps.Emit, &idmdomain.UserEnabled{At: now, TenantID: updated.TenantID, ActorUserID: actorUserID, TargetUserID: targetUserID})
	}
	if emitErr != nil {
		return nil, emitErr
	}
	return &updated, nil
}

// ErrInvalidRequiredAction は RequiredAction enum に無い値が指定された場合に返る。
var ErrInvalidRequiredAction = errors.New("required action is not in enum")

// SetUserRequiredAction は対象ユーザに次回ログイン時の強制アクションを付与する
// (admin 専用 / wi-19)。既に付与済みの場合は冪等に no-op で返す。
func SetUserRequiredAction(
	ctx context.Context,
	deps AdminUserDeps,
	actorUserID, targetUserID string,
	action idmdomain.RequiredAction,
	now time.Time,
) (*idmdomain.User, error) {
	if !action.Valid() {
		return nil, ErrInvalidRequiredAction
	}
	user, err := loadTenantUser(ctx, deps, targetUserID)
	if err != nil {
		return nil, err
	}
	if slices.Contains(user.Lifecycle.RequiredActions, action) {
		return user, nil
	}
	updated := *user
	updated.Lifecycle.RequiredActions = append(slices.Clone(user.Lifecycle.RequiredActions), action)
	now = normalizedNow(now)
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.UserRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.UserRequiredActionSet{
		At: now, TenantID: updated.TenantID, ActorUserID: actorUserID, TargetUserID: targetUserID, Action: string(action),
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ClearUserRequiredAction は強制アクションを解除する (admin 専用 / wi-19)。
// 未付与の場合は冪等に no-op で返す。本人のパスワード変更に伴う自動解除は
// clearRequiredAction (change_password.go) を使う。
func ClearUserRequiredAction(
	ctx context.Context,
	deps AdminUserDeps,
	actorUserID, targetUserID string,
	action idmdomain.RequiredAction,
	now time.Time,
) (*idmdomain.User, error) {
	if !action.Valid() {
		return nil, ErrInvalidRequiredAction
	}
	user, err := loadTenantUser(ctx, deps, targetUserID)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(user.Lifecycle.RequiredActions, action) {
		return user, nil
	}
	updated := *user
	updated.Lifecycle.RequiredActions = removeRequiredAction(user.Lifecycle.RequiredActions, action)
	now = normalizedNow(now)
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.UserRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.UserRequiredActionCleared{
		At: now, TenantID: updated.TenantID, ActorUserID: actorUserID, TargetUserID: targetUserID, Action: string(action),
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// loadTenantUser は現在のテナント内の user を取得する。存在しない / 別テナントなら
// ErrUserNotFound。admin user 操作の共通プレリュード。
func loadTenantUser(ctx context.Context, deps AdminUserDeps, sub string) (*idmdomain.User, error) {
	user, err := deps.UserRepo.FindBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// removeRequiredAction は action を除いた新しいスライスを返す (元を破壊しない)。
func removeRequiredAction(actions []idmdomain.RequiredAction, action idmdomain.RequiredAction) []idmdomain.RequiredAction {
	out := make([]idmdomain.RequiredAction, 0, len(actions))
	for _, a := range actions {
		if a != action {
			out = append(out, a)
		}
	}
	return out
}

func normalizeRoles(roles []string) ([]string, error) {
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			return nil, ErrInvalidRole
		}
		if !slices.Contains(out, role) {
			out = append(out, role)
		}
	}
	slices.Sort(out)
	return out, nil
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func equalOptionalString(left, right *string) bool {
	return left == nil && right == nil ||
		left != nil && right != nil && *left == *right
}

func changedAttributeFields(before, after map[string]idmdomain.AttributeValue) []string {
	keys := make(map[string]struct{}, len(before)+len(after))
	for key := range before {
		keys[key] = struct{}{}
	}
	for key := range after {
		keys[key] = struct{}{}
	}
	changed := make([]string, 0, len(keys))
	for key := range keys {
		if !reflect.DeepEqual(before[key], after[key]) {
			changed = append(changed, key)
		}
	}
	slices.Sort(changed)
	return changed
}

func adminEmit(sink func(spec.DomainEvent) error, event spec.DomainEvent) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}

// DeleteUserInput は ADR-036 の DeleteUser use case 入力。
type DeleteUserInput struct {
	ActorUserID string
	Sub         string
	Reason      string
	Now         time.Time
}

// DeleteUser は ADR-036 の anonymize cascade を実行する。
//   - 対象 user の PII フィールドを tombstone 値で置換する (`deleted_at` 設定)。
//   - 関連 aggregate (Consent / RefreshToken / Session / PasswordHistory /
//     MfaFactor / DeviceAuthorization) を物理削除する。
//   - `user.deleted` を 1 度だけ emit する (冪等)。
//
// 既に削除済の user に対しては no-op で nil を返す (audit event も emit しない)。
// actor.Sub == target.Sub かつ target が admin / system_admin role を持つ場合は
// ErrSelfDeleteForbidden を返し、cascade は実施しない。
func DeleteUser(ctx context.Context, deps AdminUserDeps, in DeleteUserInput) error {
	user, err := deps.UserRepo.FindBySubIncludingDeleted(ctx, in.Sub)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	if user.TenantID != tenancy.TenantID(ctx) {
		return ErrUserNotFound
	}
	if user.IsDeleted() {
		return nil
	}
	if in.ActorUserID == user.ID && hasPrivilegedRole(user.Roles) {
		return ErrSelfDeleteForbidden
	}
	now := normalizedNow(in.Now)
	tombstone := anonymizeUser(user, now)
	if err := tombstone.Validate(); err != nil {
		return err
	}
	if err := deps.UserRepo.Save(ctx, tombstone); err != nil {
		return err
	}
	if err := cascadeDeleteForSub(ctx, deps, user.ID); err != nil {
		return err
	}
	return adminEmit(deps.Emit, &idmdomain.UserDeleted{
		At: now, TenantID: user.TenantID, ActorUserID: in.ActorUserID, TargetUserID: user.ID, Reason: in.Reason,
	})
}

func hasPrivilegedRole(roles []string) bool {
	return slices.Contains(roles, "admin") || slices.Contains(roles, "system_admin")
}

// UserSoftDeleteGracePeriodSeconds は SoftDelete (PendingDeletion) から自動 Purge
// までの既定猶予期間 (秒)。SCL objectives.UserSoftDeleteGracePeriod (30 日) と一致する。
const UserSoftDeleteGracePeriodSeconds = 30 * 24 * 60 * 60

// 自動 purge の audit 用 actor / reason。lazy-on-access で猶予期間経過後の
// PendingDeletion user を Purge するときに UserDeleted へ記録する。
const (
	autoPurgeActor  = "system"
	autoPurgeReason = "auto_purge"
)

var (
	// ErrUserNotPendingDeletion は Restore 対象が PendingDeletion でない場合に返る。
	ErrUserNotPendingDeletion = errors.New("user is not pending deletion")
	// ErrRestoreGracePeriodExpired は猶予期間を過ぎた user の Restore で返る。
	ErrRestoreGracePeriodExpired = errors.New("soft-delete grace period has expired")
)

// SoftDeleteUserInput は soft-delete (削除予約) の入力。
type SoftDeleteUserInput struct {
	ActorUserID string
	Sub         string
	Reason      string
	Now         time.Time
}

// SoftDeleteUser は user を PendingDeletion に遷移させ UserSoftDeleted を emit する。
// PII / Consent / RefreshToken / Session は温存し (cascade しない)、猶予期間内は
// RestoreUser で Active に復元できる。既に PendingDeletion なら冪等に no-op で返す。
// actor.Sub == target.Sub かつ admin / system_admin role の場合は ErrSelfDeleteForbidden。
func SoftDeleteUser(ctx context.Context, deps AdminUserDeps, in SoftDeleteUserInput) error {
	user, err := loadTenantUser(ctx, deps, in.Sub)
	if err != nil {
		return err
	}
	if in.ActorUserID == user.ID && hasPrivilegedRole(user.Roles) {
		return ErrSelfDeleteForbidden
	}
	if user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusPendingDeletion {
		return nil
	}
	now := normalizedNow(in.Now)
	updated := *user
	updated.Lifecycle.Status = idmdomain.UserStatusPendingDeletion
	updated.Lifecycle.StatusChangedAt = &now
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return err
	}
	if err := captureUserMutation(ctx, deps, user, &updated, []string{"status"}, now); err != nil {
		return err
	}
	if deps.GroupRepo != nil {
		if err := SyncDynamicGroupsForUser(ctx, DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, &updated, now); err != nil {
			return err
		}
	}
	return adminEmit(deps.Emit, &idmdomain.UserSoftDeleted{
		At: now, TenantID: updated.TenantID, ActorUserID: in.ActorUserID, TargetUserID: updated.ID, Reason: in.Reason,
	})
}

// RestoreUser は PendingDeletion の user を Active に戻し UserRestored を emit する。
// PII / credential は温存されたままなのでログインは通常どおり再開する。PendingDeletion
// でない場合は ErrUserNotPendingDeletion、猶予期間を過ぎている場合は
// ErrRestoreGracePeriodExpired を返す。自分自身 (admin/system_admin) は reject する。
func RestoreUser(
	ctx context.Context, deps AdminUserDeps, actorUserID, targetUserID string, now time.Time,
) (*idmdomain.User, error) {
	user, err := loadTenantUser(ctx, deps, targetUserID)
	if err != nil {
		return nil, err
	}
	if actorUserID == user.ID && hasPrivilegedRole(user.Roles) {
		return nil, ErrSelfDeleteForbidden
	}
	if user.Lifecycle.EffectiveStatus() != idmdomain.UserStatusPendingDeletion {
		return nil, ErrUserNotPendingDeletion
	}
	now = normalizedNow(now)
	if softDeleteExpired(user, now, deps.graceSeconds()) {
		return nil, ErrRestoreGracePeriodExpired
	}
	updated := *user
	updated.Lifecycle.Status = idmdomain.UserStatusActive
	updated.Lifecycle.StatusChangedAt = &now
	updated.UpdatedAt = now
	if err := captureUserMutation(ctx, deps, user, &updated, []string{"status"}, now); err != nil {
		return nil, err
	}
	if deps.GroupRepo != nil {
		if err := SyncDynamicGroupsForUser(ctx, DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, &updated, now); err != nil {
			return nil, err
		}
	}
	if err := adminEmit(deps.Emit, &idmdomain.UserRestored{
		At: now, TenantID: updated.TenantID, ActorUserID: actorUserID, TargetUserID: updated.ID,
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PurgeExpiredSoftDeleted は猶予期間を過ぎた PendingDeletion user を lazy-on-access で
// Purge する。admin のユーザー一覧取得時に呼ばれ、対象を DeleteUser (anonymize cascade)
// にかけて UserDeleted (reason=auto_purge) を emit する。専用スケジューラは別 WI。
func PurgeExpiredSoftDeleted(ctx context.Context, deps AdminUserDeps, now time.Time) error {
	now = normalizedNow(now)
	users, err := deps.UserRepo.FindAll(ctx, tenancy.TenantID(ctx))
	if err != nil {
		return err
	}
	grace := deps.graceSeconds()
	for _, user := range users {
		if user.Lifecycle.EffectiveStatus() != idmdomain.UserStatusPendingDeletion {
			continue
		}
		if !softDeleteExpired(user, now, grace) {
			continue
		}
		if err := DeleteUser(ctx, deps, DeleteUserInput{
			ActorUserID: autoPurgeActor, Sub: user.ID, Reason: autoPurgeReason, Now: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// softDeleteExpired は PendingDeletion の user が猶予期間を過ぎたかを返す。
// status_changed_at が無い場合は期限切れ扱いにしない (安全側)。
func softDeleteExpired(user *idmdomain.User, now time.Time, graceSeconds int) bool {
	changed := user.Lifecycle.StatusChangedAt
	if changed == nil {
		return false
	}
	return now.After(changed.Add(time.Duration(graceSeconds) * time.Second))
}

// effectiveUserAttributeDefs は組み込み属性 + tenant 固有 schema を結合した実効定義を返す。
// AttrSchemaRepo 未配線 (nil) の場合は組み込み属性のみで検証する。
func effectiveUserAttributeDefs(
	ctx context.Context, repo tenantports.TenantUserAttributeSchemaRepository, tenantID string,
) ([]idmdomain.UserAttributeDef, error) {
	defs := idmdomain.BuiltinUserAttributeDefs()
	if repo == nil {
		return defs, nil
	}
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		defs = append(defs, schema.Attributes...)
	}
	return defs, nil
}

func anonymizeUser(user *idmdomain.User, now time.Time) *idmdomain.User {
	tombstone := *user
	tombstone.PreferredUsername = "deleted:" + user.ID
	tombstone.PasswordHash = deletedPasswordHashSentinel
	tombstone.Name = nil
	tombstone.GivenName = nil
	tombstone.FamilyName = nil
	tombstone.Email = nil
	tombstone.EmailVerified = false
	tombstone.MfaEnrolled = false
	tombstone.Roles = []string{}
	tombstone.Attributes = nil
	tombstone.UpdatedAt = now
	tombstone.Lifecycle = idmdomain.UserLifecycle{Status: idmdomain.UserStatusDeleted, StatusChangedAt: &now}
	return &tombstone
}

func cascadeDeleteForSub(ctx context.Context, deps AdminUserDeps, sub string) error {
	if deps.ConsentRepo != nil {
		if err := deps.ConsentRepo.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	if deps.RefreshStore != nil {
		if err := deps.RefreshStore.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	if deps.SessionStore != nil {
		if err := deps.SessionStore.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	if deps.PasswordHistoryRepo != nil {
		if err := deps.PasswordHistoryRepo.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	if deps.MfaFactorRepo != nil {
		if err := deps.MfaFactorRepo.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	if deps.DeviceCodeStore != nil {
		if err := deps.DeviceCodeStore.DeleteAllForSub(ctx, sub); err != nil {
			return err
		}
	}
	return nil
}
