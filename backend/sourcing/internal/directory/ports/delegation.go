package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
)

var ErrDelegationUnavailable = errors.New("directory delegation is unavailable")

// PasswordChallengeStore は期限、配信と結果の確定を一つの保存先で検査する。
type PasswordChallengeStore interface {
	Enqueue(context.Context, domain.PasswordChallenge, time.Time) error
	Claim(ctx context.Context, tenantID, connectorID string, revision int64, now time.Time) (*domain.PasswordChallenge, error)
	Complete(ctx context.Context, tenantID, connectorID, id string, revision int64, result domain.PasswordResult, now time.Time) error
	Find(ctx context.Context, tenantID, id string) (*domain.PasswordChallenge, error)
	Delete(ctx context.Context, tenantID, id string) error
	Sweep(ctx context.Context, now time.Time) error
}
