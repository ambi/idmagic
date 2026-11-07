package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/jackc/pgx/v5"
)

// WebAuthnSessionStore は WebAuthn ceremony の challenge を PostgreSQL に短命保持する。
// Take は GetDel 相当の一度きり消費 (DELETE ... WHERE expires_at > now
// RETURNING) で replay を防ぐ。tenant は ctx から解決する。セッションの JSON は解釈せずに
// JSONB の列へそのまま持つ。
type WebAuthnSessionStore struct{ Pool sharedpg.DB }

func (s *WebAuthnSessionStore) Save(ctx context.Context, key string, session ports.WebAuthnSession, expiresAt time.Time) error {
	return New(s.Pool).SaveWebauthnSession(ctx, SaveWebauthnSessionParams{
		TenantID:   tenantports.TenantID(ctx),
		SessionKey: key,
		Data:       session.Payload,
		ExpiresAt:  expiresAt,
	})
}

func (s *WebAuthnSessionStore) Take(ctx context.Context, key string) (*ports.WebAuthnSession, error) {
	payload, err := New(s.Pool).TakeWebauthnSession(ctx, TakeWebauthnSessionParams{
		TenantID:   tenantports.TenantID(ctx),
		SessionKey: key,
		Now:        time.Now().UTC(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.WebAuthnSession{Payload: payload}, nil
}

func (s *WebAuthnSessionStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredWebauthnSessionsBatch(ctx, DeleteExpiredWebauthnSessionsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
