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
	"github.com/jackc/pgx/v5/pgtype"
)

// AuthorizationCodeStore は単回使用の認可コードを PostgreSQL に保持する (ADR-139)。
// full record を payload JSONB に持ち、state / redeemed_at / issued_family_id を昇格列にする。
// Redeem は state='issued' の単一列 CAS (UPDATE ... RETURNING) で atomic に redeemed へ倒し、
// read では昇格列を payload に overlay して単文 CAS による payload の陳腐化を隠す。
type AuthorizationCodeStore struct{ Pool sharedpg.DB }

func codeFromPayload(payload []byte) (*domain.AuthorizationCodeRecord, error) {
	var rec domain.AuthorizationCodeRecord
	if err := json.Unmarshal(payload, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func overlayCode(rec *domain.AuthorizationCodeRecord, state string, redeemedAt pgtype.Timestamptz, family pgtype.Text) {
	rec.State = spec.AuthorizationCodeRecordState(state)
	rec.RedeemedAt = timestamptzPtr(redeemedAt)
	rec.IssuedFamilyID = textPtr(family)
}

func (s *AuthorizationCodeStore) Save(ctx context.Context, code *domain.AuthorizationCodeRecord) error {
	code.TenantID = tenancy.TenantID(ctx)
	payload, err := json.Marshal(code)
	if err != nil {
		return err
	}
	return New(s.Pool).SaveAuthorizationCode(ctx, SaveAuthorizationCodeParams{
		Code:           code.Code,
		TenantID:       code.TenantID,
		State:          string(code.State),
		RedeemedAt:     timestamptzOrNil(code.RedeemedAt),
		IssuedFamilyID: textOrNil(code.IssuedFamilyID),
		ExpiresAt:      code.ExpiresAt,
		Payload:        payload,
	})
}

func (s *AuthorizationCodeStore) Find(ctx context.Context, code string) (*domain.AuthorizationCodeRecord, error) {
	row, err := New(s.Pool).FindAuthorizationCode(ctx, FindAuthorizationCodeParams{
		Code:     code,
		TenantID: tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := codeFromPayload(row.Payload)
	if err != nil {
		return nil, err
	}
	overlayCode(rec, row.State, row.RedeemedAt, row.IssuedFamilyID)
	return rec, nil
}

func (s *AuthorizationCodeStore) Redeem(ctx context.Context, code string, now time.Time) (*domain.AuthorizationCodeRecord, error) {
	row, err := New(s.Pool).RedeemAuthorizationCode(ctx, RedeemAuthorizationCodeParams{
		RedeemedAt: pgtype.Timestamptz{Time: now.UTC(), Valid: true},
		Code:       code,
		TenantID:   tenancy.TenantID(ctx),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := codeFromPayload(row.Payload)
	if err != nil {
		return nil, err
	}
	overlayCode(rec, row.State, row.RedeemedAt, row.IssuedFamilyID)
	return rec, nil
}

func (s *AuthorizationCodeStore) LinkFamily(ctx context.Context, code, familyID string) error {
	affected, err := New(s.Pool).LinkAuthorizationCodeFamily(ctx, LinkAuthorizationCodeFamilyParams{
		IssuedFamilyID: pgtype.Text{String: familyID, Valid: true},
		Code:           code,
		TenantID:       tenancy.TenantID(ctx),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("code not found")
	}
	return nil
}

func (s *AuthorizationCodeStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredAuthorizationCodesBatch(ctx, DeleteExpiredAuthorizationCodesBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
