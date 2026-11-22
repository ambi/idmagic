package ports

import (
	"context"
	"errors"
	"time"

	directorydomain "github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
)

var ErrDelegationUnavailable = errors.New("directory delegation is unavailable")

// PasswordChallengeStore は期限、配信と結果の確定を一つの保存先で検査する。
type PasswordChallengeStore interface {
	Enqueue(context.Context, directorydomain.PasswordChallenge, time.Time) error
	Claim(ctx context.Context, tenantID, connectorID string, revision int64, now time.Time) (*directorydomain.PasswordChallenge, error)
	Complete(ctx context.Context, tenantID, connectorID, id string, revision int64, result directorydomain.PasswordResult, now time.Time) error
	Find(ctx context.Context, tenantID, id string) (*directorydomain.PasswordChallenge, error)
	Delete(ctx context.Context, tenantID, id string) error
	Sweep(ctx context.Context, now time.Time) error
}
