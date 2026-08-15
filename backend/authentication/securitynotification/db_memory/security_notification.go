// Package db_memory はセキュリティ通知の受信設定と既知の端末の in-memory 実装 (wi-90)。
package db_memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
)

// PreferenceRepository は受信設定の in-memory 実装。
type PreferenceRepository struct {
	mu          sync.Mutex
	preferences map[string]domain.Preferences // key: user id
}

func NewPreferenceRepository() *PreferenceRepository {
	return &PreferenceRepository{preferences: map[string]domain.Preferences{}}
}

func (r *PreferenceRepository) Find(_ context.Context, userID string) (*domain.Preferences, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.preferences[userID]
	if !ok {
		return nil, nil // 行が無いことは「すべて有効」であり、エラーではない。
	}
	stored.Disabled = slices.Clone(stored.Disabled)
	return &stored, nil
}

func (r *PreferenceRepository) Save(_ context.Context, preferences domain.Preferences) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	preferences.Disabled = slices.Clone(preferences.Disabled)
	r.preferences[preferences.UserID] = preferences
	return nil
}

// KnownDeviceRepository は既知の端末の in-memory 実装。
type KnownDeviceRepository struct {
	mu      sync.Mutex
	devices map[string]ports.KnownDevice // key: user id + "\x00" + device hash
}

func NewKnownDeviceRepository() *KnownDeviceRepository {
	return &KnownDeviceRepository{devices: map[string]ports.KnownDevice{}}
}

func knownDeviceKey(userID, deviceHash string) string { return userID + "\x00" + deviceHash }

func (r *KnownDeviceRepository) Observe(_ context.Context, device ports.KnownDevice) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := knownDeviceKey(device.UserID, device.DeviceHash)
	_, existed := r.devices[key]
	device.SeenAt = device.SeenAt.UTC()
	r.devices[key] = device
	return !existed, nil
}

func (r *KnownDeviceRepository) DeleteIdleBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deleted int64
	for key, device := range r.devices {
		if device.SeenAt.Before(cutoff.UTC()) {
			delete(r.devices, key)
			deleted++
		}
	}
	return deleted, nil
}
