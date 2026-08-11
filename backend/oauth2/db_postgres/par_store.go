package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
)

// PARStore は Pushed Authorization Request を PostgreSQL に短命保持する。
// full record を payload JSONB に持ち、used 列を Consume の CAS 述語に昇格して read で
// payload に overlay する。tenant は ctx から解決し fail-closed 述語に含める。Find は期限
// フィルタを付けない (memory adapter とのパリティ: 期限判定は呼び出し側の domain)。
type PARStore struct{ Pool sharedpg.DB }

func parFromPayload(payload []byte) (*domain.PARRecord, error) {
	var rec domain.PARRecord
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *PARStore) Save(ctx context.Context, rec *domain.PARRecord) error {
	rec.TenantID = tenancy.TenantID(ctx)
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return New(s.Pool).SavePARRequest(ctx, SavePARRequestParams{
		RequestUri: rec.RequestURI,
		TenantID:   rec.TenantID,
		Used:       rec.Used,
		ExpiresAt:  rec.ExpiresAt,
		Payload:    payload,
	})
}

func (s *PARStore) Find(ctx context.Context, requestURI string) (*domain.PARRecord, error) {
	row, err := New(s.Pool).FindPARRequest(ctx, FindPARRequestParams{
		RequestUri: requestURI,
		TenantID:   tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := parFromPayload(row.Payload)
	if err != nil {
		return nil, err
	}
	rec.Used = row.Used
	return rec, nil
}

func (s *PARStore) Consume(ctx context.Context, requestURI string) (*domain.PARRecord, error) {
	payload, err := New(s.Pool).ConsumePARRequest(ctx, ConsumePARRequestParams{
		RequestUri: requestURI,
		TenantID:   tenancy.TenantID(ctx),
		Now:        time.Now().UTC(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := parFromPayload(payload)
	if err != nil {
		return nil, err
	}
	rec.Used = true
	return rec, nil
}

func (s *PARStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredPARRequestsBatch(ctx, DeleteExpiredPARRequestsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
