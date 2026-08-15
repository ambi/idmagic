package db_memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// TrustedDeviceRepository は信頼済みデバイスの in-memory 実装 (wi-91)。
type TrustedDeviceRepository struct {
	mu      sync.Mutex
	devices map[string]*domain.TrustedDevice // key: device id
}

func NewTrustedDeviceRepository() *TrustedDeviceRepository {
	return &TrustedDeviceRepository{devices: map[string]*domain.TrustedDevice{}}
}

func (r *TrustedDeviceRepository) FindBySelector(
	_ context.Context,
	tenantID, selector string,
) (*domain.TrustedDevice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, device := range r.devices {
		if device.Selector == selector && device.TenantID == tenantID {
			return cloneTrustedDevice(device), nil
		}
	}
	return nil, nil // An unknown selector is an absent device, not an error.
}

func (r *TrustedDeviceRepository) FindByID(
	_ context.Context,
	tenantID, userID, deviceID string,
) (*domain.TrustedDevice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	device := r.devices[deviceID]
	if device == nil || device.TenantID != tenantID || device.UserID != userID {
		return nil, nil // Another user's device is intentionally treated as absent.
	}
	return cloneTrustedDevice(device), nil
}

func (r *TrustedDeviceRepository) ListActiveByUser(
	_ context.Context,
	tenantID, userID string,
) ([]*domain.TrustedDevice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	out := []*domain.TrustedDevice{}
	for _, device := range r.devices {
		if device.TenantID != tenantID || device.UserID != userID {
			continue
		}
		if device.RevokedAt != nil || !now.Before(device.ExpiresAt) {
			continue
		}
		out = append(out, cloneTrustedDevice(device))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastUsedAt.After(out[j].LastUsedAt) })
	return out, nil
}

func (r *TrustedDeviceRepository) Save(_ context.Context, device *domain.TrustedDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[device.ID] = cloneTrustedDevice(device)
	return nil
}

func (r *TrustedDeviceRepository) RevokeAllForUser(
	_ context.Context,
	tenantID, userID string,
	reason spec.TrustedDeviceRevokeReason,
	now time.Time,
) ([]*domain.TrustedDevice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	revoked := []*domain.TrustedDevice{}
	for _, device := range r.devices {
		if device.TenantID != tenantID || device.UserID != userID || device.RevokedAt != nil {
			continue
		}
		device.Revoke(reason, now)
		revoked = append(revoked, cloneTrustedDevice(device))
	}
	sort.Slice(revoked, func(i, j int) bool { return revoked[i].ID < revoked[j].ID })
	return revoked, nil
}

func (r *TrustedDeviceRepository) DeleteAllForSub(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, device := range r.devices {
		if device.UserID == userID {
			delete(r.devices, id)
		}
	}
	return nil
}

func cloneTrustedDevice(device *domain.TrustedDevice) *domain.TrustedDevice {
	if device == nil {
		return nil
	}
	out := *device
	if device.RevokedAt != nil {
		revoked := *device.RevokedAt
		out.RevokedAt = &revoked
	}
	if device.RevokeReason != nil {
		reason := *device.RevokeReason
		out.RevokeReason = &reason
	}
	return &out
}
