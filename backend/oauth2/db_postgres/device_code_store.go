package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
)

// DeviceCodeStore は RFC 8628 device authorization を PostgreSQL に保持する (ADR-139)。
// full record を payload JSONB に持ち、device_code_hash を PK、(tenant_id,user_code) を
// UNIQUE 鍵、state を Exchange の CAS 述語、user_id を DeleteAllForSub の索引に昇格する。
// Save / Update は同一 upsert。read では state を payload に overlay する。
type DeviceCodeStore struct{ Pool sharedpg.DB }

func deviceFromPayload(payload []byte, state string) (*domain.DeviceAuthorization, error) {
	var rec domain.DeviceAuthorization
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, err
	}
	rec.State = spec.DeviceCodeFlowState(state)
	return &rec, nil
}

func (s *DeviceCodeStore) upsert(ctx context.Context, rec *domain.DeviceAuthorization) error {
	rec.TenantID = tenancy.TenantID(ctx)
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return New(s.Pool).SaveDeviceCode(ctx, SaveDeviceCodeParams{
		DeviceCodeHash: rec.DeviceCodeHash,
		TenantID:       rec.TenantID,
		UserCode:       rec.UserCode,
		UserID:         uuidOrNil(strDeref(rec.UserID)),
		State:          string(rec.State),
		ExpiresAt:      rec.ExpiresAt,
		Payload:        payload,
	})
}

func (s *DeviceCodeStore) Save(ctx context.Context, rec *domain.DeviceAuthorization) error {
	return s.upsert(ctx, rec)
}

func (s *DeviceCodeStore) Update(ctx context.Context, rec *domain.DeviceAuthorization) error {
	return s.upsert(ctx, rec)
}

func (s *DeviceCodeStore) FindByDeviceCodeHash(ctx context.Context, hash string) (*domain.DeviceAuthorization, error) {
	row, err := New(s.Pool).FindDeviceCodeByHash(ctx, FindDeviceCodeByHashParams{
		DeviceCodeHash: hash,
		TenantID:       tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return deviceFromPayload(row.Payload, row.State)
}

func (s *DeviceCodeStore) FindByUserCode(ctx context.Context, userCode string) (*domain.DeviceAuthorization, error) {
	row, err := New(s.Pool).FindDeviceCodeByUserCode(ctx, FindDeviceCodeByUserCodeParams{
		UserCode: userCode,
		TenantID: tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return deviceFromPayload(row.Payload, row.State)
}

func (s *DeviceCodeStore) Exchange(ctx context.Context, deviceCodeHash string) (*domain.DeviceAuthorization, error) {
	row, err := New(s.Pool).ExchangeDeviceCode(ctx, ExchangeDeviceCodeParams{
		DeviceCodeHash: deviceCodeHash,
		TenantID:       tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return deviceFromPayload(row.Payload, row.State)
}

func (s *DeviceCodeStore) DeleteAllForSub(ctx context.Context, sub string) error {
	return New(s.Pool).DeleteDeviceCodesForUser(ctx, DeleteDeviceCodesForUserParams{
		TenantID: tenancy.TenantID(ctx),
		UserID:   uuidOrNil(sub),
	})
}

func (s *DeviceCodeStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredDeviceCodesBatch(ctx, DeleteExpiredDeviceCodesBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
