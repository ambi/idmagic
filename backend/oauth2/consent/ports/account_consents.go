package ports

import (
	"context"
	"errors"
	"time"

	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
)

// ErrConsentNotFound は、操作の対象の同意が存在しないことを表す。
var ErrConsentNotFound = errors.New("consent not found")

// AccountConsents は、利用者本人が自分の同意を一覧し撤回する操作である。ほかのモジュールの
// アカウント画面が使い、同意の状態遷移と ConsentRevoked の発行は OAuth2 が行う。
type AccountConsents interface {
	// ListGranted は sub の Granted の同意だけを返す。
	ListGranted(ctx context.Context, sub string) ([]*consentdomain.Consent, error)
	// Revoke は sub 自身を操作者として、sub と clientID の組の同意を撤回する。同意がなければ
	// ErrConsentNotFound を返す。
	Revoke(ctx context.Context, sub, clientID string, now time.Time) error
}
