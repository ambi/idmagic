package usecases

import (
	"context"
	"errors"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// CreateSeedUser は固定 ID の初期データを、通常作成と同じ不変条件と作用で保存する。
// 資格情報のない合成 User は初期状態が無効のときだけ受け付ける。
func CreateSeedUser(ctx context.Context, deps AdminUserDeps, desired *userdomain.User, password string) error {
	user := *desired
	user.PreferredUsername = strings.TrimSpace(user.PreferredUsername)
	user.Email = normalizeEmail(user.Email)
	if user.ID == "" {
		return errors.New("seed user id is required")
	}
	roles, err := idmusecases.NormalizeRoles(user.Roles)
	if err != nil {
		return err
	}
	if err := idmusecases.ValidateRoleAssignment(user.TenantID, nil, roles); err != nil {
		return err
	}
	user.Roles = roles
	user.PasswordHash = ""
	if password != "" {
		if deps.ValidateInitialPassword == nil || deps.PasswordHasher == nil {
			return errors.New("initial password dependencies are required")
		}
		if err := deps.ValidateInitialPassword(ctx, password); err != nil {
			return err
		}
		user.PasswordHash, err = deps.PasswordHasher.Hash(password)
		if err != nil {
			return err
		}
	} else if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		return errors.New("credentialless seed user must be disabled")
	}
	_, err = createUser(ctx, deps, newUser{User: user, ActorUserID: "seed", PasswordHistoryHash: user.PasswordHash}, desired.CreatedAt)
	return err
}
