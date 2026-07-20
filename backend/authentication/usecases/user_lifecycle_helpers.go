package usecases

import (
	"context"
	"errors"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ErrUserNotFound は自己サービス経路で対象 user が存在しない、または tenant が
// 不一致な場合。password/totp/webauthn/mfa/session/recovery 各 feature から共有で
// 参照されるため context ルートに置く (ADR-130 決定 6 と同方針)。
var ErrUserNotFound = errors.New("user not found")

// LoadSelfUser は self 経路で対象 user を取得する。tenant 不一致は ErrUserNotFound に潰す。
// webauthn/mfa/recovery の各 feature から使われる横断ヘルパー。
func LoadSelfUser(ctx context.Context, repo userports.UserRepository, sub string) (*userdomain.User, error) {
	user, err := repo.FindBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func NormalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
