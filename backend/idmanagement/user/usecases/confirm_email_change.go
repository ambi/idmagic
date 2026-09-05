package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ErrInvalidEmailChangeToken は、提示されたトークンで確定へ進めないことを表す。用途違い、
// ダイジェスト不一致、期限切れ、消費済みのどれであっても外へはこの一つで返る。拒否理由は
// 包まれたエラーの連鎖に残るので、内部の観測は理由を区別できる。
var ErrInvalidEmailChangeToken = errors.New("email change token is invalid or expired")

// ConfirmEmailChangeDeps / Input は新アドレスへ送ったワンタイムトークンを消費し、
// primary email を確定する (self-service, wi-21)。トークンが所有確認の証左なので
// 認証済みセッションは要求しない (reset password と同方針)。
type ConfirmEmailChangeDeps struct {
	UserRepo   userports.UserRepository
	TokenStore userports.EmailChangeTokenStore
	Emit       func(spec.DomainEvent)
}

type ConfirmEmailChangeInput struct {
	Token string
	Now   time.Time
}

// ConfirmEmailChange は確認リンクのトークンで primary email を確定する。
//
// トークンは検証するだけで、この時点では消費しない。起票から確定までの間に別のユーザーが
// 同じアドレスを取っていた場合、トークンは未使用のまま残るので、利用者は別のアドレスを
// 選び直すのではなく、状況が変わればもう一度同じリンクを使える。使用済み化と保存は
// ConsumeAndApply が同じトランザクションで確定する。
func ConfirmEmailChange(ctx context.Context, deps ConfirmEmailChangeDeps, in ConfirmEmailChangeInput) (*userdomain.User, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stored, err := deps.TokenStore.Find(ctx, actiontoken.Fingerprint(in.Token))
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, ErrInvalidEmailChangeToken
	}
	envelope, err := actiontoken.Verify(actiontoken.VerifyInput{
		RawToken:        in.Token,
		ExpectedPurpose: actiontoken.PurposeEmailChange,
		Stored:          *stored,
		Now:             now,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEmailChangeToken, err)
	}
	payload, err := actiontoken.DecodePayload(userports.EmailChangePayloadCodec{}, envelope)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEmailChangeToken, err)
	}
	user, err := deps.UserRepo.FindBySub(ctx, envelope.Subject)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrInvalidEmailChangeToken
	}
	// 起票から確定までの間に別ユーザが同アドレスを確定していないか再チェックする。
	existing, err := deps.UserRepo.FindByEmail(ctx, user.TenantID, payload.NewEmail)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != user.ID {
		return nil, ErrEmailTaken
	}

	updated := *user
	email := payload.NewEmail
	updated.Email = &email
	updated.EmailVerified = true
	updated.UpdatedAt = now
	clearedVerifyEmail := slices.Contains(updated.Lifecycle.RequiredActions, idmdomain.RequiredActionVerifyEmail)
	if clearedVerifyEmail {
		updated.Lifecycle.RequiredActions = removeRequiredAction(
			updated.Lifecycle.RequiredActions, idmdomain.RequiredActionVerifyEmail,
		)
	}
	if err := deps.TokenStore.ConsumeAndApply(ctx, userports.EmailChangeCommit{
		Digest: envelope.Digest, Now: now, User: &updated,
	}); err != nil {
		if errors.Is(err, actiontoken.ErrAlreadyConsumed) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidEmailChangeToken, err)
		}
		return nil, err
	}
	// 外部作用はトランザクションの外で起こす。
	if deps.Emit != nil {
		deps.Emit(&idmdomain.EmailChanged{At: now, TenantID: user.TenantID, UserID: user.ID})
		if clearedVerifyEmail {
			deps.Emit(&idmdomain.UserRequiredActionCleared{
				At: now, TenantID: user.TenantID, ActorUserID: user.ID, TargetUserID: user.ID,
				Action: string(idmdomain.RequiredActionVerifyEmail),
			})
		}
	}
	return &updated, nil
}
