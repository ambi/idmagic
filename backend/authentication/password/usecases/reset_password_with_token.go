package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ErrInvalidResetToken は、提示されたトークンで作用へ進めないことを表す。用途違い、
// ダイジェスト不一致、期限切れ、消費済みのどれであっても外へはこの一つで返る。拒否理由は
// 包まれたエラーの連鎖に残るので、内部の観測は理由を区別できる。
var ErrInvalidResetToken = errors.New("reset token is invalid or expired")

type ResetPasswordWithTokenDeps struct {
	UserRepo                userports.UserRepository
	TokenStore              passwordports.PasswordResetTokenStore
	PasswordHasher          passwordports.PasswordHasher
	PasswordHistoryRepo     passwordports.PasswordHistoryRepository
	BreachedPasswordChecker passwordports.BreachedPasswordChecker
	Emit                    func(spec.DomainEvent)
	HistoryDepth            int                    // Deprecated: use Policy 指定。後方互換のためのフォールバック。
	Policy                  PasswordPolicySnapshot // テナント解決済みのしきい値。ゼロ値は global default。
}

type ResetPasswordWithTokenInput struct {
	Token       string
	NewPassword string
	Now         time.Time
}

// ResetPasswordWithToken はリセットリンクのトークンでパスワードを設定する。
//
// 順序が重要である。トークンは検証するだけで、この時点では消費しない。パスワード規則、
// 既知漏洩、履歴の再利用でここから先へ進めなかった場合、トークンは未使用のまま残り、
// 利用者は同じリンクをもう一度使える。使用済み化と保存は ConsumeAndApply が同じ
// トランザクションで確定するので、同じトークンで作用が二回成功することはない。
func ResetPasswordWithToken(
	ctx context.Context,
	deps ResetPasswordWithTokenDeps,
	in ResetPasswordWithTokenInput,
) (*userdomain.User, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stored, err := deps.TokenStore.Find(ctx, actiontoken.Fingerprint(in.Token))
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, ErrInvalidResetToken
	}
	envelope, err := actiontoken.Verify(actiontoken.VerifyInput{
		RawToken:        in.Token,
		ExpectedPurpose: actiontoken.PurposePasswordReset,
		Stored:          *stored,
		Now:             now,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResetToken, err)
	}
	user, err := deps.UserRepo.FindBySub(ctx, envelope.Subject)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrInvalidResetToken
	}

	snap := resolveSnapshot(deps.Policy, deps.HistoryDepth)
	result := ValidatePasswordWith(in.NewPassword, snap)
	if !result.OK {
		return nil, &PasswordPolicyError{Violations: result.Violations}
	}
	if deps.BreachedPasswordChecker != nil &&
		deps.BreachedPasswordChecker.IsBreached(ctx, in.NewPassword) {
		return nil, &PasswordPolicyError{Violations: []PasswordPolicyViolation{ViolationBreached}}
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

	encoded, err := deps.PasswordHasher.Hash(in.NewPassword)
	if err != nil {
		return nil, err
	}
	updated := *user
	updated.PasswordHash = encoded
	updated.UpdatedAt = now
	updated.Lifecycle.PasswordChangedAt = &now
	// リセットで新パスワードを設定したので update_password 強制アクションを自動解除する。
	clearedUpdatePassword := slices.Contains(updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword)
	if clearedUpdatePassword {
		updated.Lifecycle.RequiredActions = removeRequiredAction(
			updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword,
		)
	}
	if err := deps.TokenStore.ConsumeAndApply(ctx, passwordports.PasswordResetCommit{
		Digest: envelope.Digest, Now: now, User: &updated, PasswordEncoded: encoded,
	}); err != nil {
		if errors.Is(err, actiontoken.ErrAlreadyConsumed) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidResetToken, err)
		}
		return nil, err
	}
	// 外部作用はトランザクションの外で起こす。
	if deps.Emit != nil {
		deps.Emit(&authdomain.PasswordChanged{At: now, TenantID: user.TenantID, UserID: user.ID})
		if clearedUpdatePassword {
			deps.Emit(&idmdomain.UserRequiredActionCleared{
				At: now, TenantID: user.TenantID, ActorUserID: user.ID, TargetUserID: user.ID,
				Action: string(idmdomain.RequiredActionUpdatePassword),
			})
		}
	}
	return &updated, nil
}
