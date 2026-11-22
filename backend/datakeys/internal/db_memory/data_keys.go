// Package db_memory: Layer 5 - Adapters (memory)
package db_memory

import (
	"context"
	"strconv"
	"sync"
	"time"

	datakeysdomain "github.com/ambi/idmagic/backend/datakeys/internal/domain"
)

// DataKeyRepository is the in-process reference implementation of
// ports.DataKeyRepository. It is also the reference behavior for tests and
// the local demo (docs/design/data/database.md) — the PostgreSQL
// adapter must never diverge from it.
type DataKeyRepository struct {
	mu       sync.Mutex
	byTenant map[string][]*datakeysdomain.TenantDataEncryptionKey // newest version last
	nextID   int
}

func NewDataKeyRepository() *DataKeyRepository {
	return &DataKeyRepository{byTenant: map[string][]*datakeysdomain.TenantDataEncryptionKey{}}
}

func (r *DataKeyRepository) Bootstrap(_ context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nextVersion := 1
	for _, k := range r.byTenant[tenantID] {
		if k.Status != datakeysdomain.DataKeyStatusDestroyed {
			return nil, datakeysdomain.ErrDataKeyAlreadyBootstrapped
		}
		if k.Version >= nextVersion {
			nextVersion = k.Version + 1
		}
	}

	key := r.newKeyLocked(tenantID, nextVersion, wrappedDEK, masterKeyID, now)
	r.byTenant[tenantID] = append(r.byTenant[tenantID], key)
	return cloneKey(key), nil
}

func (r *DataKeyRepository) Rotate(_ context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*datakeysdomain.TenantDataEncryptionKey, *datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions := r.byTenant[tenantID]
	var previous *datakeysdomain.TenantDataEncryptionKey
	for _, k := range versions {
		if k.Status == datakeysdomain.DataKeyStatusActive {
			previous = k
			break
		}
	}
	if previous == nil {
		return nil, nil, datakeysdomain.ErrNoActiveDataKey
	}

	previous.Status = datakeysdomain.DataKeyStatusRetiring
	next := r.newKeyLocked(tenantID, previous.Version+1, wrappedDEK, masterKeyID, now)
	r.byTenant[tenantID] = append(r.byTenant[tenantID], next)
	return cloneKey(next), cloneKey(previous), nil
}

func (r *DataKeyRepository) Disable(_ context.Context, tenantID string, version int, now time.Time) (*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, datakeysdomain.ErrDataKeyNotFound
	}
	if key.Status == datakeysdomain.DataKeyStatusActive {
		return nil, datakeysdomain.ErrDataKeyIsActive
	}
	if key.Status != datakeysdomain.DataKeyStatusRetiring {
		return nil, datakeysdomain.ErrDataKeyNotDisableable
	}
	key.Status = datakeysdomain.DataKeyStatusDisabled
	disabledAt := now
	key.DisabledAt = &disabledAt
	return cloneKey(key), nil
}

func (r *DataKeyRepository) Destroy(_ context.Context, tenantID string, version int, now time.Time) (*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, datakeysdomain.ErrDataKeyNotFound
	}
	if key.Status == datakeysdomain.DataKeyStatusActive {
		return nil, datakeysdomain.ErrDataKeyIsActive
	}
	if key.Status != datakeysdomain.DataKeyStatusRetiring && key.Status != datakeysdomain.DataKeyStatusDisabled {
		return nil, datakeysdomain.ErrDataKeyNotDestroyable
	}
	key.Status = datakeysdomain.DataKeyStatusDestroyed
	key.WrappedDEK = nil
	destroyedAt := now
	key.DestroyedAt = &destroyedAt
	return cloneKey(key), nil
}

func (r *DataKeyRepository) FindActive(_ context.Context, tenantID string) (*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, k := range r.byTenant[tenantID] {
		if k.Status == datakeysdomain.DataKeyStatusActive {
			return cloneKey(k), nil
		}
	}
	return nil, datakeysdomain.ErrNoActiveDataKey
}

func (r *DataKeyRepository) FindByVersion(_ context.Context, tenantID string, version int) (*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.findLocked(tenantID, version)
	if key == nil {
		return nil, datakeysdomain.ErrDataKeyNotFound
	}
	return cloneKey(key), nil
}

func (r *DataKeyRepository) ListAll(_ context.Context, tenantID string) ([]*datakeysdomain.TenantDataEncryptionKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions := r.byTenant[tenantID]
	out := make([]*datakeysdomain.TenantDataEncryptionKey, len(versions))
	for i, k := range versions {
		out[len(versions)-1-i] = cloneKey(k)
	}
	return out, nil
}

func (r *DataKeyRepository) findLocked(tenantID string, version int) *datakeysdomain.TenantDataEncryptionKey {
	for _, k := range r.byTenant[tenantID] {
		if k.Version == version {
			return k
		}
	}
	return nil
}

func (r *DataKeyRepository) newKeyLocked(tenantID string, version int, wrappedDEK []byte, masterKeyID string, now time.Time) *datakeysdomain.TenantDataEncryptionKey {
	r.nextID++
	activatedAt := now
	return &datakeysdomain.TenantDataEncryptionKey{
		ID:          idFor(tenantID, version),
		TenantID:    tenantID,
		Version:     version,
		Status:      datakeysdomain.DataKeyStatusActive,
		WrappedDEK:  append([]byte(nil), wrappedDEK...),
		MasterKeyID: masterKeyID,
		CreatedAt:   now,
		ActivatedAt: &activatedAt,
	}
}

func idFor(tenantID string, version int) string {
	return tenantID + ":" + strconv.Itoa(version)
}

func cloneKey(k *datakeysdomain.TenantDataEncryptionKey) *datakeysdomain.TenantDataEncryptionKey {
	clone := *k
	clone.WrappedDEK = append([]byte(nil), k.WrappedDEK...)
	return &clone
}
