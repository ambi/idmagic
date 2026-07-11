package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	ErrCurrentPasswordMismatch = errors.New("current password does not match")
	ErrPasswordReused          = errors.New("new password matches a recent password")
	ErrUserNotFound            = errors.New("user not found")
)

type ChangePasswordInput struct {
	Sub             string
	CurrentPassword string
	NewPassword     string
	Now             time.Time
}

type ChangePasswordDeps struct {
	UserRepo            idmports.UserRepository
	PasswordHasher      authnports.PasswordHasher
	PasswordHistoryRepo authnports.PasswordHistoryRepository
	Emit                func(spec.DomainEvent) error
	HistoryDepth        int                    // Deprecated: use Policy 指定。後方互換のためのフォールバック。
	Policy              PasswordPolicySnapshot // テナント解決済みのしきい値。ゼロ値は global default。
}

func ChangePassword(ctx context.Context, deps ChangePasswordDeps, in ChangePasswordInput) (*idmdomain.User, error) {
	user, err := deps.UserRepo.FindBySub(ctx, in.Sub)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	ok, err := deps.PasswordHasher.Verify(in.CurrentPassword, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCurrentPasswordMismatch
	}

	snap := resolveSnapshot(deps.Policy, deps.HistoryDepth)
	result := ValidatePasswordWith(in.NewPassword, snap)
	if !result.OK {
		return nil, &PasswordPolicyError{Violations: result.Violations}
	}

	depth := snap.HistoryDepth
	recent, err := deps.PasswordHistoryRepo.Recent(ctx, user.ID, depth)
	if err != nil {
		return nil, err
	}
	for _, entry := range recent {
		matched, err := deps.PasswordHasher.Verify(in.NewPassword, entry.Encoded)
		if err != nil {
			return nil, err
		}
		if matched {
			return nil, ErrPasswordReused
		}
	}
	matched, err := deps.PasswordHasher.Verify(in.NewPassword, user.PasswordHash)
	if err != nil {
		return nil, err
	}
	if matched {
		return nil, ErrPasswordReused
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	encoded, err := deps.PasswordHasher.Hash(in.NewPassword)
	if err != nil {
		return nil, err
	}

	updated := *user
	updated.PasswordHash = encoded
	updated.UpdatedAt = now
	updated.Lifecycle.PasswordChangedAt = &now
	// 本人がパスワードを変更したので update_password の強制アクションは自動解除する。
	clearedUpdatePassword := slices.Contains(updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword)
	if clearedUpdatePassword {
		updated.Lifecycle.RequiredActions = removeRequiredAction(
			updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword,
		)
	}
	if err := deps.UserRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := deps.PasswordHistoryRepo.Add(ctx, user.ID, encoded, now); err != nil {
		return nil, err
	}
	if deps.Emit != nil {
		if err := deps.Emit(&spec.PasswordChanged{At: now, TenantID: user.TenantID, UserID: user.ID}); err != nil {
			return nil, err
		}
		if clearedUpdatePassword {
			// 自動解除なので ActorUserID は本人 (system 操作ではなく能動的解除)。
			if err := deps.Emit(&spec.UserRequiredActionCleared{
				At: now, TenantID: user.TenantID, ActorUserID: user.ID, TargetUserID: user.ID,
				Action: string(idmdomain.RequiredActionUpdatePassword),
			}); err != nil {
				return nil, err
			}
		}
	}
	return &updated, nil
}
