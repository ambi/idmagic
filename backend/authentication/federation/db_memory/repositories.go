package db_memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
)

type Repositories struct {
	Connections *ConnectionRepository
	Identities  *IdentityRepository
	Attempts    *AttemptStore
	Replay      *ReplayStore
}

func NewRepositories() Repositories {
	return Repositories{
		Connections: &ConnectionRepository{items: map[string]*domain.IdentityProviderConnection{}},
		Identities:  &IdentityRepository{bySubject: map[string]*domain.FederatedIdentity{}, byUser: map[string]string{}},
		Attempts:    &AttemptStore{items: map[string]*domain.FederatedLoginAttempt{}},
		Replay:      &ReplayStore{items: map[string]time.Time{}},
	}
}

type ConnectionRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.IdentityProviderConnection
}

func (r *ConnectionRepository) Save(_ context.Context, connection *domain.IdentityProviderConnection) error {
	if err := connection.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[connectionKey(connection.TenantID, connection.ID)] = cloneConnection(connection)
	return nil
}

func (r *ConnectionRepository) Find(_ context.Context, tenantID, id string) (*domain.IdentityProviderConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneConnection(r.items[connectionKey(tenantID, id)]), nil
}

func (r *ConnectionRepository) List(_ context.Context, tenantID string) ([]*domain.IdentityProviderConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.IdentityProviderConnection, 0)
	prefix := tenantID + "\x00"
	for key, connection := range r.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			out = append(out, cloneConnection(connection))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	return out, nil
}

func (r *ConnectionRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, connectionKey(tenantID, id))
	return nil
}

type IdentityRepository struct {
	mu        sync.RWMutex
	bySubject map[string]*domain.FederatedIdentity
	byUser    map[string]string
}

func (r *IdentityRepository) Create(_ context.Context, identity *domain.FederatedIdentity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	subjectKey := linkSubjectKey(identity.TenantID, identity.ProviderID, identity.ExternalSubject)
	userKey := linkUserKey(identity.TenantID, identity.ProviderID, identity.LocalUserID)
	if _, exists := r.bySubject[subjectKey]; exists {
		return federationports.ErrLinkConflict
	}
	if _, exists := r.byUser[userKey]; exists {
		return federationports.ErrLinkConflict
	}
	cloned := *identity
	r.bySubject[subjectKey], r.byUser[userKey] = &cloned, subjectKey
	return nil
}

func (r *IdentityRepository) FindBySubject(_ context.Context, tenantID, providerID, subject string) (*domain.FederatedIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneIdentity(r.bySubject[linkSubjectKey(tenantID, providerID, subject)]), nil
}

func (r *IdentityRepository) FindByUserProvider(_ context.Context, tenantID, providerID, userID string) (*domain.FederatedIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneIdentity(r.bySubject[r.byUser[linkUserKey(tenantID, providerID, userID)]]), nil
}

func (r *IdentityRepository) ListByUser(_ context.Context, tenantID, userID string) ([]*domain.FederatedIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.FederatedIdentity, 0)
	for _, identity := range r.bySubject {
		if identity.TenantID == tenantID && identity.LocalUserID == userID {
			out = append(out, cloneIdentity(identity))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProviderID < out[j].ProviderID })
	return out, nil
}

func (r *IdentityRepository) Delete(_ context.Context, tenantID, providerID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	userKey := linkUserKey(tenantID, providerID, userID)
	if subjectKey, ok := r.byUser[userKey]; ok {
		delete(r.bySubject, subjectKey)
		delete(r.byUser, userKey)
	}
	return nil
}

type AttemptStore struct {
	mu    sync.Mutex
	items map[string]*domain.FederatedLoginAttempt
}

func (s *AttemptStore) Save(_ context.Context, attempt *domain.FederatedLoginAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := *attempt
	s.items[attemptKey(attempt.TenantID, attempt.State)] = &cloned
	return nil
}

func (s *AttemptStore) Consume(_ context.Context, tenantID, state string, now time.Time) (*domain.FederatedLoginAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt := s.items[attemptKey(tenantID, state)]
	if attempt == nil {
		return nil, federationports.ErrAttemptNotFound
	}
	if attempt.ConsumedAt != nil {
		return nil, federationports.ErrAttemptConsumed
	}
	if err := attempt.Consume(now); err != nil {
		return nil, err
	}
	cloned := *attempt
	return &cloned, nil
}

type ReplayStore struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func (s *ReplayStore) Reserve(_ context.Context, tenantID, id string, expiresAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, now := tenantID+"\x00"+id, time.Now().UTC()
	if expiry, exists := s.items[key]; exists && expiry.After(now) {
		return false, nil
	}
	s.items[key] = expiresAt
	return true, nil
}

func cloneConnection(connection *domain.IdentityProviderConnection) *domain.IdentityProviderConnection {
	if connection == nil {
		return nil
	}
	cloned := *connection
	cloned.SAMLSigningCertificates = append([]string(nil), connection.SAMLSigningCertificates...)
	cloned.AllowedEmailDomains = append([]string(nil), connection.AllowedEmailDomains...)
	return &cloned
}

func cloneIdentity(identity *domain.FederatedIdentity) *domain.FederatedIdentity {
	if identity == nil {
		return nil
	}
	cloned := *identity
	return &cloned
}

func connectionKey(tenantID, id string) string { return tenantID + "\x00" + id }
func linkSubjectKey(tenantID, providerID, subject string) string {
	return tenantID + "\x00" + providerID + "\x00" + subject
}

func linkUserKey(tenantID, providerID, userID string) string {
	return tenantID + "\x00" + providerID + "\x00" + userID
}
func attemptKey(tenantID, state string) string { return tenantID + "\x00" + state }
