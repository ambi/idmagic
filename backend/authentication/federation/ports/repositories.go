package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
)

var (
	ErrLinkConflict    = errors.New("federated identity link conflicts with an existing link")
	ErrAttemptNotFound = errors.New("federated login attempt not found")
	ErrAttemptConsumed = errors.New("federated login attempt already consumed")
)

type ConnectionRepository interface {
	Save(context.Context, *domain.IdentityProviderConnection) error
	Find(context.Context, string, string) (*domain.IdentityProviderConnection, error)
	List(context.Context, string) ([]*domain.IdentityProviderConnection, error)
	Delete(context.Context, string, string) error
}

type IdentityRepository interface {
	Create(context.Context, *domain.FederatedIdentity) error
	FindBySubject(context.Context, string, string, string) (*domain.FederatedIdentity, error)
	FindByUserProvider(context.Context, string, string, string) (*domain.FederatedIdentity, error)
	ListByUser(context.Context, string, string) ([]*domain.FederatedIdentity, error)
	Delete(context.Context, string, string, string) error
}

type AttemptStore interface {
	Save(context.Context, *domain.FederatedLoginAttempt) error
	Consume(context.Context, string, string, time.Time) (*domain.FederatedLoginAttempt, error)
}

type ReplayStore interface {
	Reserve(context.Context, string, string, time.Time) (bool, error)
}

type SecretResolver interface {
	Resolve(context.Context, string) (string, error)
}
