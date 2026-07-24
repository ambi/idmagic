package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
)

// WebAuthnSessionStore は WebAuthn ceremony の challenge を PostgreSQL に短命保持する
// (ADR-087 / ADR-139)。Take は GetDel 相当の一度きり消費 (DELETE ... WHERE expires_at > now
// RETURNING) で replay を防ぐ。tenant は ctx から解決する。SessionData は JSONB で持つ。
type WebAuthnSessionStore struct{ Pool sharedpg.DB }

func (s *WebAuthnSessionStore) Save(ctx context.Context, key string, data gowebauthn.SessionData, expiresAt time.Time) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return New(s.Pool).SaveWebauthnSession(ctx, SaveWebauthnSessionParams{
		TenantID:   tenancy.TenantID(ctx),
		SessionKey: key,
		Data:       payload,
		ExpiresAt:  expiresAt,
	})
}

func (s *WebAuthnSessionStore) Take(ctx context.Context, key string) (*gowebauthn.SessionData, error) {
	payload, err := New(s.Pool).TakeWebauthnSession(ctx, TakeWebauthnSessionParams{
		TenantID:   tenancy.TenantID(ctx),
		SessionKey: key,
		Now:        time.Now().UTC(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data gowebauthn.SessionData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *WebAuthnSessionStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredWebauthnSessionsBatch(ctx, DeleteExpiredWebauthnSessionsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
