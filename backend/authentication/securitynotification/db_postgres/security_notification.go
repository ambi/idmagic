// Package db_postgres はセキュリティ通知の受信設定と既知の端末の PostgreSQL 実装 (wi-90)。
package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PreferenceRepository は受信設定の PostgreSQL 実装。
type PreferenceRepository struct{ Pool sharedpg.DB }

func (r *PreferenceRepository) queries() *Queries { return New(r.Pool) }

func (r *PreferenceRepository) Find(ctx context.Context, userID string) (*domain.Preferences, error) {
	row, err := r.queries().FindNotificationPreference(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // 行が無いことは「すべて有効」であり、エラーではない。
	}
	if err != nil {
		return nil, err
	}
	disabled := make([]domain.Category, 0, len(row.DisabledCategories))
	for _, value := range row.DisabledCategories {
		disabled = append(disabled, domain.Category(value))
	}
	return &domain.Preferences{UserID: row.UserID, Disabled: disabled, UpdatedAt: row.UpdatedAt}, nil
}

func (r *PreferenceRepository) Save(ctx context.Context, preferences domain.Preferences) error {
	disabled := make([]string, 0, len(preferences.Disabled))
	for _, category := range preferences.Disabled {
		disabled = append(disabled, string(category))
	}
	return r.queries().UpsertNotificationPreference(ctx, UpsertNotificationPreferenceParams{
		UserID: preferences.UserID, DisabledCategories: disabled, UpdatedAt: preferences.UpdatedAt,
	})
}

// KnownDeviceRepository は既知の端末の PostgreSQL 実装。
type KnownDeviceRepository struct{ Pool sharedpg.DB }

func (r *KnownDeviceRepository) queries() *Queries { return New(r.Pool) }

// Observe は挿入を先に試し、挿入できた場合だけ「新しい端末」を返す。既知の行では
// last_seen_at を進めるだけである。同じ端末から同時に 2 つのサインインが来ても、
// 挿入に成功するのはちょうど 1 つなので通知は 1 通に収まる。
func (r *KnownDeviceRepository) Observe(ctx context.Context, device ports.KnownDevice) (bool, error) {
	seenAt := device.SeenAt.UTC()
	inserted, err := r.queries().InsertKnownSignInDevice(ctx, InsertKnownSignInDeviceParams{
		UserID: device.UserID, DeviceHash: device.DeviceHash,
		Label: textOrNil(device.Label), FirstSeenAt: seenAt,
	})
	if err != nil {
		return false, err
	}
	if inserted > 0 {
		return true, nil
	}
	if err := r.queries().TouchKnownSignInDevice(ctx, TouchKnownSignInDeviceParams{
		UserID: device.UserID, DeviceHash: device.DeviceHash,
		LastSeenAt: seenAt, Label: textOrNil(device.Label),
	}); err != nil {
		return false, err
	}
	return false, nil
}

func (r *KnownDeviceRepository) DeleteIdleBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return r.queries().DeleteIdleKnownSignInDevices(ctx, cutoff.UTC())
}

func textOrNil(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
