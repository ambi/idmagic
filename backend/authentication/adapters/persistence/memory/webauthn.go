package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

// =====================================================================
// WebAuthnCredentialRepository (Authentication) — wi-26 / ADR-087
// =====================================================================

type WebAuthnCredentialRepository struct {
	mu          sync.RWMutex
	credentials map[string]*domain.WebAuthnCredential // key: credential_id
}

func NewWebAuthnCredentialRepository() *WebAuthnCredentialRepository {
	return &WebAuthnCredentialRepository{credentials: map[string]*domain.WebAuthnCredential{}}
}

func (r *WebAuthnCredentialRepository) ListBySub(
	_ context.Context,
	sub string,
) ([]*domain.WebAuthnCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.WebAuthnCredential{}
	for _, c := range r.credentials {
		if c.UserID == sub {
			out = append(out, cloneWebAuthnCredential(c))
		}
	}
	return out, nil
}

func (r *WebAuthnCredentialRepository) FindByCredentialID(
	_ context.Context,
	credentialID string,
) (*domain.WebAuthnCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneWebAuthnCredential(r.credentials[credentialID]), nil
}

func (r *WebAuthnCredentialRepository) Save(
	_ context.Context,
	credential *domain.WebAuthnCredential,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.credentials[credential.CredentialID] = cloneWebAuthnCredential(credential)
	return nil
}

func (r *WebAuthnCredentialRepository) UpdateSignCount(
	_ context.Context,
	credentialID string,
	signCount uint32,
	lastUsedAt time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.credentials[credentialID]; ok {
		c.SignCount = signCount
		used := lastUsedAt
		c.LastUsedAt = &used
	}
	return nil
}

func (r *WebAuthnCredentialRepository) Delete(
	_ context.Context,
	sub string,
	credentialID string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.credentials[credentialID]; ok && c.UserID == sub {
		delete(r.credentials, credentialID)
	}
	return nil
}

func (r *WebAuthnCredentialRepository) DeleteAllForSub(_ context.Context, sub string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, c := range r.credentials {
		if c.UserID == sub {
			delete(r.credentials, id)
		}
	}
	return nil
}

func cloneWebAuthnCredential(c *domain.WebAuthnCredential) *domain.WebAuthnCredential {
	if c == nil {
		return nil
	}
	out := *c
	if c.Transports != nil {
		out.Transports = append([]string(nil), c.Transports...)
	}
	if c.AAGUID != nil {
		aaguid := *c.AAGUID
		out.AAGUID = &aaguid
	}
	if c.Label != nil {
		label := *c.Label
		out.Label = &label
	}
	if c.LastUsedAt != nil {
		lastUsedAt := *c.LastUsedAt
		out.LastUsedAt = &lastUsedAt
	}
	return &out
}
