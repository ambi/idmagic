// Package db_memory: Layer 5 - Adapters (memory)
package db_memory

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
)

// DataKeyRepository is the in-process reference implementation of
// ports.DataKeyRepository. It is also the reference behavior for tests and
// the local demo (ARCHITECTURE.md Persistence policy) — the PostgreSQL
// adapter must never diverge from it.
type DataKeyRepository struct {
	mu       sync.Mutex
	byTenant map[string][]*domain.TenantDataEncryptionKey // newest version last
	nextID   int
}

func NewDataKeyRepository() *DataKeyRepository {
	return &DataKeyRepository{byTenant: map[string][]*domain.TenantDataEncryptionKey{}}
}

func (r *DataKeyRepository) Bootstrap(_ context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nextVersion := 1
	for _, k := range r.byTenant[tenantID] {
		if k.Status != domain.DataKeyStatusDestroyed {
			return nil, domain.ErrDataKeyAlreadyBootstrapped
		}
		if k.Version >= nextVersion {
			nextVersion = k.Version + 1
		}
	}

	key := r.newKeyLocked(tenantID, nextVersion, wrappedDEK, masterKeyID, now)
	r.byTenant[tenantID] = append(r.byTenant[tenantID], key)
	return cloneKey(key), nil
}

func (r *DataKeyRepository) Rotate(_ context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*domain.TenantDataEncryptionKey, *domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions := r.byTenant[tenantID]
	var previous *domain.TenantDataEncryptionKey
	for _, k := range versions {
		if k.Status == domain.DataKeyStatusActive {
			previous = k
			break
		}
	}
	if previous == nil {
		return nil, nil, domain.ErrNoActiveDataKey
	}

	previous.Status = domain.DataKeyStatusRetiring
	next := r.newKeyLocked(tenantID, previous.Version+1, wrappedDEK, masterKeyID, now)
	r.byTenant[tenantID] = append(r.byTenant[tenantID], next)
	return cloneKey(next), cloneKey(previous), nil
}

func (r *DataKeyRepository) Disable(_ context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, domain.ErrDataKeyNotFound
	}
	if key.Status == domain.DataKeyStatusActive {
		return nil, domain.ErrDataKeyIsActive
	}
	if key.Status != domain.DataKeyStatusRetiring {
		return nil, domain.ErrDataKeyNotDisableable
	}
	key.Status = domain.DataKeyStatusDisabled
	disabledAt := now
	key.DisabledAt = &disabledAt
	return cloneKey(key), nil
}

func (r *DataKeyRepository) Destroy(_ context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, domain.ErrDataKeyNotFound
	}
	if key.Status == domain.DataKeyStatusActive {
		return nil, domain.ErrDataKeyIsActive
	}
	if key.Status != domain.DataKeyStatusRetiring && key.Status != domain.DataKeyStatusDisabled {
		return nil, domain.ErrDataKeyNotDestroyable
	}
	key.Status = domain.DataKeyStatusDestroyed
	key.WrappedDEK = nil
	destroyedAt := now
	key.DestroyedAt = &destroyedAt
	return cloneKey(key), nil
}

func (r *DataKeyRepository) FindActive(_ context.Context, tenantID string) (*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, k := range r.byTenant[tenantID] {
		if k.Status == domain.DataKeyStatusActive {
			return cloneKey(k), nil
		}
	}
	return nil, domain.ErrNoActiveDataKey
}

func (r *DataKeyRepository) FindByVersion(_ context.Context, tenantID string, version int) (*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, domain.ErrDataKeyNotFound
	}
	return cloneKey(key), nil
}

func (r *DataKeyRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions := r.byTenant[tenantID]
	out := make([]*domain.TenantDataEncryptionKey, len(versions))
	for i, k := range versions {
		out[len(versions)-1-i] = cloneKey(k)
	}
	return out, nil
}

func (r *DataKeyRepository) findLocked(tenantID string, version int) *domain.TenantDataEncryptionKey {
	for _, k := range r.byTenant[tenantID] {
		if k.Version == version {
			return k
		}
	}
	return nil
}

func (r *DataKeyRepository) newKeyLocked(tenantID string, version int, wrappedDEK []byte, masterKeyID string, now time.Time) *domain.TenantDataEncryptionKey {
	r.nextID++
	activatedAt := now
	return &domain.TenantDataEncryptionKey{
		ID:          idFor(tenantID, version),
		TenantID:    tenantID,
		Version:     version,
		Status:      domain.DataKeyStatusActive,
		WrappedDEK:  append([]byte(nil), wrappedDEK...),
		MasterKeyID: masterKeyID,
		CreatedAt:   now,
		ActivatedAt: &activatedAt,
	}
}

func idFor(tenantID string, version int) string {
	return tenantID + ":" + strconv.Itoa(version)
}

func cloneKey(k *domain.TenantDataEncryptionKey) *domain.TenantDataEncryptionKey {
	clone := *k
	clone.WrappedDEK = append([]byte(nil), k.WrappedDEK...)
	return &clone
}
