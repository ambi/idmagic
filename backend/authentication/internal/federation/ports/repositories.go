package ports

import (
	"context"
	"errors"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
)

var (
	ErrLinkConflict    = errors.New("federated identity link conflicts with an existing link")
	ErrAttemptNotFound = errors.New("federated login attempt not found")
	// ErrAttemptConsumed は消費済みの attempt に加えて、期限切れの attempt も表す。
	ErrAttemptConsumed = errors.New("federated login attempt already consumed")
)

type ConnectionRepository interface {
	Save(context.Context, *federationdomain.IdentityProviderConnection) error
	Find(context.Context, string, string) (*federationdomain.IdentityProviderConnection, error)
	List(context.Context, string) ([]*federationdomain.IdentityProviderConnection, error)
	Delete(context.Context, string, string) error
}

type IdentityRepository interface {
	Create(context.Context, *federationdomain.FederatedIdentity) error
	FindBySubject(context.Context, string, string, string) (*federationdomain.FederatedIdentity, error)
	FindByUserProvider(context.Context, string, string, string) (*federationdomain.FederatedIdentity, error)
	ListByUser(context.Context, string, string) ([]*federationdomain.FederatedIdentity, error)
	// ExistsForProvider は、テナントの接続に連携した外部アイデンティティが 1 つでも残っているかを返す。
	ExistsForProvider(ctx context.Context, tenantID, providerID string) (bool, error)
	Delete(context.Context, string, string, string) error
}

type AttemptStore interface {
	Save(context.Context, *federationdomain.FederatedLoginAttempt) error
	Consume(context.Context, string, string, time.Time) (*federationdomain.FederatedLoginAttempt, error)
}

// ReplayStore はテナントの中で同じ応答 ID を一度だけ予約する。
// 既存の予約の期限切れは、呼び出し側が検証に使った時刻 now で判定し、新しい予約は now+ttl まで有効にする。
// now がゼロ値、または ttl が正でなければ、予約せずにエラーを返す。
type ReplayStore interface {
	Reserve(ctx context.Context, tenantID, id string, now time.Time, ttl time.Duration) (bool, error)
}

type SecretResolver interface {
	Resolve(context.Context, string) (string, error)
}

// SecretCipher envelope-encrypts the client secret at rest. The concrete
// implementation is backend/datakeys.FieldCipher; this port exists so db_postgres does
// not depend on DataKeys' internal packages directly (mirrors
// backend/authentication/totp/ports.SecretCipher).
type SecretCipher interface {
	Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (keyVersion int, ciphertext []byte, err error)
	Decrypt(ctx context.Context, tenantID, recordContext, table, recordID, field string, keyVersion int, ciphertext []byte) (plaintext string, err error)
}
