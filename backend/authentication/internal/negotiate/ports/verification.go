package ports

import (
	"context"
	"errors"
	"time"

	negotiatedomain "github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
)

// ErrUnsupportedEncryption は、チケットかセッションの鍵の暗号方式が AES でないことを表す。
// 呼び出し側は一般のチケットの失敗と区別して監査に残す。
var ErrUnsupportedEncryption = errors.New("negotiate: ticket encryption type is not accepted")

type Verified struct {
	Principal negotiatedomain.Principal
	ReplayID  string
	Reply     []byte
}

// Verifier は鍵、ASN.1、channel binding を呼び出し側へ漏らさない。
type Verifier interface {
	Verify(context.Context, negotiatedomain.Config, []byte, string) (Verified, error)
}

// ReplayStore は全レプリカで同じ認証応答を一度だけ予約する。
// 既存の予約の期限切れは now で判定し、新しい予約は now+ttl まで有効にする。
type ReplayStore interface {
	Reserve(ctx context.Context, tenantID, id string, now time.Time, ttl time.Duration) (bool, error)
}
