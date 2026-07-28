// Package db_postgres: Layer 5 - Adapters (PostgreSQL)
package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpostgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// DataKeyRepository persists TenantDataEncryptionKey per ports.DataKeyRepository.
// It must never diverge in behavior from db_memory.DataKeyRepository, which is
// the reference implementation used by tests and the local demo.
type DataKeyRepository struct {
	Pool sharedpostgres.DB
}

func NewDataKeyRepository(_ context.Context, pool sharedpostgres.DB) (*DataKeyRepository, error) {
	return &DataKeyRepository{Pool: pool}, nil
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func timeToPg(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func (r *DataKeyRepository) Bootstrap(ctx context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)

	if err := q.LockDataKeyRotation(ctx, pgtype.Text{String: tenantID, Valid: true}); err != nil {
		return nil, err
	}
	count, err := q.CountNonDestroyedDataKeys(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, domain.ErrDataKeyAlreadyBootstrapped
	}
	maxVersion, err := q.MaxDataKeyVersion(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	version := maxVersion + 1

	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	if err := q.InsertDataKey(ctx, InsertDataKeyParams{
		ID:          id,
		TenantID:    tenantID,
		Version:     version,
		WrappedDek:  wrappedDEK,
		MasterKeyID: masterKeyID,
		CreatedAt:   now,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return newKeyRow(id, tenantID, version, domain.DataKeyStatusActive, wrappedDEK, masterKeyID, now, &now, nil, nil), nil
}

func (r *DataKeyRepository) Rotate(ctx context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*domain.TenantDataEncryptionKey, *domain.TenantDataEncryptionKey, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)

	if err := q.LockDataKeyRotation(ctx, pgtype.Text{String: tenantID, Valid: true}); err != nil {
		return nil, nil, err
	}
	activeRow, err := q.GetActiveDataKey(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrNoActiveDataKey
	}
	if err != nil {
		return nil, nil, err
	}

	if err := q.RetireActiveDataKey(ctx, RetireActiveDataKeyParams{TenantID: tenantID, UpdatedAt: now}); err != nil {
		return nil, nil, err
	}

	nextVersion := activeRow.Version + 1
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, nil, err
	}
	if err := q.InsertDataKey(ctx, InsertDataKeyParams{
		ID:          id,
		TenantID:    tenantID,
		Version:     nextVersion,
		WrappedDek:  wrappedDEK,
		MasterKeyID: masterKeyID,
		CreatedAt:   now,
	}); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	next := newKeyRow(id, tenantID, nextVersion, domain.DataKeyStatusActive, wrappedDEK, masterKeyID, now, &now, nil, nil)
	previous := activeDataKeyRowToDomain(activeRow)
	previous.Status = domain.DataKeyStatusRetiring
	return next, previous, nil
}

func (r *DataKeyRepository) Disable(ctx context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	key, err := r.FindByVersion(ctx, tenantID, version)
	if err != nil {
		return nil, err
	}
	if key.Status == domain.DataKeyStatusActive {
		return nil, domain.ErrDataKeyIsActive
	}
	if key.Status != domain.DataKeyStatusRetiring {
		return nil, domain.ErrDataKeyNotDisableable
	}
	if err := New(r.Pool).DisableDataKey(ctx, DisableDataKeyParams{
		TenantID:   tenantID,
		Version:    int32(version), //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
		DisabledAt: timeToPg(&now),
	}); err != nil {
		return nil, err
	}
	key.Status = domain.DataKeyStatusDisabled
	key.DisabledAt = &now
	return key, nil
}

func (r *DataKeyRepository) Destroy(ctx context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	key, err := r.FindByVersion(ctx, tenantID, version)
	if err != nil {
		return nil, err
	}
	if key.Status == domain.DataKeyStatusActive {
		return nil, domain.ErrDataKeyIsActive
	}
	if key.Status != domain.DataKeyStatusRetiring && key.Status != domain.DataKeyStatusDisabled {
		return nil, domain.ErrDataKeyNotDestroyable
	}
	if err := New(r.Pool).DestroyDataKey(ctx, DestroyDataKeyParams{
		TenantID:    tenantID,
		Version:     int32(version), //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
		DestroyedAt: timeToPg(&now),
	}); err != nil {
		return nil, err
	}
	key.Status = domain.DataKeyStatusDestroyed
	key.WrappedDEK = nil
	key.DestroyedAt = &now
	return key, nil
}

func (r *DataKeyRepository) FindActive(ctx context.Context, tenantID string) (*domain.TenantDataEncryptionKey, error) {
	row, err := New(r.Pool).GetActiveDataKey(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNoActiveDataKey
	}
	if err != nil {
		return nil, err
	}
	return activeDataKeyRowToDomain(row), nil
}

func (r *DataKeyRepository) FindByVersion(ctx context.Context, tenantID string, version int) (*domain.TenantDataEncryptionKey, error) {
	row, err := New(r.Pool).GetDataKeyByVersion(ctx, GetDataKeyByVersionParams{TenantID: tenantID, Version: int32(version)}) //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrDataKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return byVersionRowToDomain(row), nil
}

func (r *DataKeyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.TenantDataEncryptionKey, error) {
	rows, err := New(r.Pool).ListDataKeysByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.TenantDataEncryptionKey, len(rows))
	for i, row := range rows {
		out[i] = &domain.TenantDataEncryptionKey{
			ID:          row.ID,
			TenantID:    row.TenantID,
			Version:     int(row.Version),
			Status:      domain.DataKeyStatus(row.Status),
			WrappedDEK:  row.WrappedDek,
			MasterKeyID: row.MasterKeyID,
			CreatedAt:   row.CreatedAt,
			ActivatedAt: timePtr(row.ActivatedAt),
			DisabledAt:  timePtr(row.DisabledAt),
			DestroyedAt: timePtr(row.DestroyedAt),
		}
	}
	return out, nil
}

func newKeyRow(id, tenantID string, version int32, status domain.DataKeyStatus, wrappedDEK []byte, masterKeyID string, createdAt time.Time, activatedAt, disabledAt, destroyedAt *time.Time) *domain.TenantDataEncryptionKey {
	return &domain.TenantDataEncryptionKey{
		ID:          id,
		TenantID:    tenantID,
		Version:     int(version),
		Status:      status,
		WrappedDEK:  wrappedDEK,
		MasterKeyID: masterKeyID,
		CreatedAt:   createdAt,
		ActivatedAt: activatedAt,
		DisabledAt:  disabledAt,
		DestroyedAt: destroyedAt,
	}
}

func activeDataKeyRowToDomain(row *GetActiveDataKeyRow) *domain.TenantDataEncryptionKey {
	return &domain.TenantDataEncryptionKey{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Version:     int(row.Version),
		Status:      domain.DataKeyStatus(row.Status),
		WrappedDEK:  row.WrappedDek,
		MasterKeyID: row.MasterKeyID,
		CreatedAt:   row.CreatedAt,
		ActivatedAt: timePtr(row.ActivatedAt),
		DisabledAt:  timePtr(row.DisabledAt),
		DestroyedAt: timePtr(row.DestroyedAt),
	}
}

func byVersionRowToDomain(row *GetDataKeyByVersionRow) *domain.TenantDataEncryptionKey {
	return &domain.TenantDataEncryptionKey{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Version:     int(row.Version),
		Status:      domain.DataKeyStatus(row.Status),
		WrappedDEK:  row.WrappedDek,
		MasterKeyID: row.MasterKeyID,
		CreatedAt:   row.CreatedAt,
		ActivatedAt: timePtr(row.ActivatedAt),
		DisabledAt:  timePtr(row.DisabledAt),
		DestroyedAt: timePtr(row.DestroyedAt),
	}
}
