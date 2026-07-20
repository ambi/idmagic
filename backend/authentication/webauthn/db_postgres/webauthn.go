package db_postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// WebAuthnCredentialRepository (Authentication) — wi-26 / ADR-087
type WebAuthnCredentialRepository struct{ Pool sharedpg.DB }

func (r *WebAuthnCredentialRepository) queries() *Queries { return New(r.Pool) }

func (r *WebAuthnCredentialRepository) ListBySub(
	ctx context.Context,
	sub string,
) ([]*domain.WebAuthnCredential, error) {
	rows, err := r.queries().ListWebAuthnCredentialsBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.WebAuthnCredential, 0, len(rows))
	for _, row := range rows {
		c, err := webAuthnCredentialFromRow(
			row.CredentialID, row.UserID, row.PublicKey, row.SignCount, row.Transports,
			row.Aaguid, row.Label, row.BackupEligible, row.BackupState, row.CreatedAt, row.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *WebAuthnCredentialRepository) FindByCredentialID(
	ctx context.Context,
	credentialID string,
) (*domain.WebAuthnCredential, error) {
	row, err := r.queries().GetWebAuthnCredentialByID(ctx, credentialID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return webAuthnCredentialFromRow(
		row.CredentialID, row.UserID, row.PublicKey, row.SignCount, row.Transports,
		row.Aaguid, row.Label, row.BackupEligible, row.BackupState, row.CreatedAt, row.LastUsedAt,
	)
}

func (r *WebAuthnCredentialRepository) Save(ctx context.Context, c *domain.WebAuthnCredential) error {
	transports := c.Transports
	if transports == nil {
		transports = []string{}
	}
	return r.queries().UpsertWebAuthnCredential(ctx, UpsertWebAuthnCredentialParams{
		CredentialID:   c.CredentialID,
		UserID:         c.UserID,
		PublicKey:      c.PublicKey,
		SignCount:      int64(c.SignCount),
		Transports:     transports,
		Aaguid:         textOrNil(c.AAGUID),
		Label:          textOrNil(c.Label),
		BackupEligible: c.BackupEligible,
		BackupState:    c.BackupState,
		CreatedAt:      c.CreatedAt,
		LastUsedAt:     timestamptzOrNil(c.LastUsedAt),
	})
}

func (r *WebAuthnCredentialRepository) UpdateSignCount(
	ctx context.Context,
	credentialID string,
	signCount uint32,
	lastUsedAt time.Time,
) error {
	return r.queries().UpdateWebAuthnCredentialSignCount(ctx, UpdateWebAuthnCredentialSignCountParams{
		CredentialID: credentialID,
		SignCount:    int64(signCount),
		LastUsedAt:   timestamptzOrNil(&lastUsedAt),
	})
}

func (r *WebAuthnCredentialRepository) Delete(ctx context.Context, sub, credentialID string) error {
	return r.queries().DeleteWebAuthnCredential(ctx, DeleteWebAuthnCredentialParams{
		UserID: sub, CredentialID: credentialID,
	})
}

func (r *WebAuthnCredentialRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeleteWebAuthnCredentialsForSub(ctx, sub)
}

func webAuthnCredentialFromRow(
	credentialID, userID, publicKey string,
	signCount int64,
	transports []string,
	aaguid, label pgtype.Text,
	backupEligible, backupState bool,
	createdAt time.Time,
	lastUsedAt pgtype.Timestamptz,
) (*domain.WebAuthnCredential, error) {
	// sign_count は uint32 として書き込むため通常は範囲内だが、DB 破損に備えて範囲を丸める。
	if signCount < 0 || signCount > math.MaxUint32 {
		signCount = 0
	}
	if transports == nil {
		transports = []string{}
	}
	c := &domain.WebAuthnCredential{
		CredentialID:   credentialID,
		UserID:         userID,
		PublicKey:      publicKey,
		SignCount:      uint32(signCount),
		Transports:     transports,
		AAGUID:         textPtr(aaguid),
		Label:          textPtr(label),
		BackupEligible: backupEligible,
		BackupState:    backupState,
		CreatedAt:      createdAt,
		LastUsedAt:     timestamptzPtr(lastUsedAt),
	}
	return c, c.Validate()
}
