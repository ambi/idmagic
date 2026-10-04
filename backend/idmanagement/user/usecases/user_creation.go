package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// newUser は管理者による作成と JIT が createUser へ渡す、経路ごとに決まった部分である。
type newUser struct {
	// User は ID、時刻、状態を除いた作成する User。ユーザー名とメールアドレスは正規化済みとする。
	User        userdomain.User
	ActorUserID string
	// PasswordHistoryHash が空なら、パスワードの履歴を作らない。
	PasswordHistoryHash string
}

// createUser は、User を作るすべての経路が共有する検証と作用をまとめる (REQ-IDMANAGEMENT-089)。
// CSV の適用は一行ずつ不可分に確定する別のコミッターを通るので、ここを通らず、
// 同じ比較キーと syncDynamicGroups を使う。
func createUser(ctx context.Context, deps AdminUserDeps, in newUser, now time.Time) (*userdomain.User, error) {
	user := in.User
	if err := ensureUserIdentityAvailable(ctx, deps.UserRepo, user.TenantID, "", user.PreferredUsername, user.Email); err != nil {
		return nil, err
	}
	if len(user.Attributes) > 0 {
		defs, err := effectiveUserAttributeDefs(ctx, deps.AttrSchemaRepo, user.TenantID)
		if err != nil {
			return nil, err
		}
		if err := userdomain.ValidateAttributes(user.Attributes, defs); err != nil {
			return nil, errors.Join(ErrInvalidAttribute, err)
		}
	}
	if err := idmusecases.CheckQuotaAndAudit(ctx, deps.QuotaRepo, deps.Emit, user.TenantID, tenancydomain.ResourceUsers, now); err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	user.ID = id
	user.Lifecycle = userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}
	user.CreatedAt, user.UpdatedAt = now, now
	if err := user.Validate(); err != nil {
		return nil, err
	}
	if err := captureUserMutation(ctx, deps, nil, &user, nil, now); err != nil {
		return nil, err
	}
	if err := syncDynamicGroups(ctx, deps, &user, now); err != nil {
		return nil, err
	}
	if in.PasswordHistoryHash != "" {
		if err := deps.PasswordHistoryRepo.Add(ctx, user.ID, in.PasswordHistoryHash, now); err != nil {
			return nil, err
		}
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.UserCreated{At: now, TenantID: user.TenantID, ActorUserID: in.ActorUserID, TargetUserID: user.ID}); err != nil {
		return nil, err
	}
	notifyProvisioning(ctx, deps, user.TenantID, user.ID, userports.ProvisioningUserCreated, now)
	return &user, nil
}

// ensureUserIdentityAvailable は、ユーザー名とメールアドレスが同じテナントのほかの User と
// 比較キーで一致しないことを確かめる。selfID の User 自身との一致は衝突にしない。
// email が nil なら、メールアドレスを調べない。
func ensureUserIdentityAvailable(ctx context.Context, repo userports.UserRepository, tenantID, selfID, username string, email *string) error {
	if username != "" {
		existing, err := repo.FindByUsername(ctx, tenantID, username)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != selfID {
			return ErrUsernameConflict
		}
	}
	if email != nil {
		existing, err := repo.FindByEmail(ctx, tenantID, *email)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != selfID {
			return ErrEmailTaken
		}
	}
	return nil
}

// normalizeEmail は前後の空白を除き、空になるメールアドレスを未設定にする。
func normalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*email)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// syncDynamicGroups は User を有効な動的グループの規則で評価し直す。
// GroupRepo が未配線なら何もしない。
func syncDynamicGroups(ctx context.Context, deps AdminUserDeps, user *userdomain.User, now time.Time) error {
	if deps.GroupRepo == nil {
		return nil
	}
	return groupusecases.SyncDynamicGroupsForUser(ctx, groupusecases.DynamicGroupDeps{GroupRepo: deps.GroupRepo, UserRepo: deps.UserRepo, SchemaRepo: deps.AttrSchemaRepo}, user, now)
}
