package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// WebAuthnCredentialRepository (Authentication) — wi-26 / ADR-087
type WebAuthnCredentialRepository struct{ Pool DB }

const webAuthnCredentialSelect = `SELECT credential_id,user_id,public_key,sign_count,transports,aaguid,label,backup_eligible,backup_state,created_at,last_used_at FROM webauthn_credentials`

func (r *WebAuthnCredentialRepository) ListBySub(
	ctx context.Context,
	sub string,
) ([]*spec.WebAuthnCredential, error) {
	rows, err := r.Pool.Query(ctx, webAuthnCredentialSelect+" WHERE user_id=$1 ORDER BY created_at", sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.WebAuthnCredential{}
	for rows.Next() {
		credential, err := scanWebAuthnCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, credential)
	}
	return out, rows.Err()
}

func (r *WebAuthnCredentialRepository) FindByCredentialID(
	ctx context.Context,
	credentialID string,
) (*spec.WebAuthnCredential, error) {
	return scanWebAuthnCredential(r.Pool.QueryRow(ctx, webAuthnCredentialSelect+" WHERE credential_id=$1", credentialID))
}

func (r *WebAuthnCredentialRepository) Save(ctx context.Context, c *spec.WebAuthnCredential) error {
	transports := c.Transports
	if transports == nil {
		transports = []string{}
	}
	_, err := r.Pool.Exec(ctx, `
INSERT INTO webauthn_credentials (credential_id,user_id,public_key,sign_count,transports,aaguid,label,backup_eligible,backup_state,created_at,last_used_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (credential_id) DO UPDATE SET sign_count=EXCLUDED.sign_count,label=EXCLUDED.label,last_used_at=EXCLUDED.last_used_at,updated_at=now()`,
		c.CredentialID, c.UserID, c.PublicKey, int64(c.SignCount), transports, c.AAGUID, c.Label,
		c.BackupEligible, c.BackupState, c.CreatedAt, c.LastUsedAt)
	return err
}

func (r *WebAuthnCredentialRepository) UpdateSignCount(
	ctx context.Context,
	credentialID string,
	signCount uint32,
	lastUsedAt time.Time,
) error {
	_, err := r.Pool.Exec(ctx,
		"UPDATE webauthn_credentials SET sign_count=$2,last_used_at=$3,updated_at=now() WHERE credential_id=$1",
		credentialID, int64(signCount), lastUsedAt)
	return err
}

func (r *WebAuthnCredentialRepository) Delete(ctx context.Context, sub, credentialID string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM webauthn_credentials WHERE user_id=$1 AND credential_id=$2", sub, credentialID)
	return err
}

func (r *WebAuthnCredentialRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM webauthn_credentials WHERE user_id=$1", sub)
	return err
}

func scanWebAuthnCredential(row RowScanner) (*spec.WebAuthnCredential, error) {
	var c spec.WebAuthnCredential
	var signCount int64
	err := row.Scan(&c.CredentialID, &c.UserID, &c.PublicKey, &signCount, &c.Transports,
		&c.AAGUID, &c.Label, &c.BackupEligible, &c.BackupState, &c.CreatedAt, &c.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// sign_count は uint32 として書き込むため通常は範囲内だが、DB 破損に備えて範囲を丸める。
	if signCount < 0 || signCount > math.MaxUint32 {
		signCount = 0
	}
	c.SignCount = uint32(signCount)
	if c.Transports == nil {
		c.Transports = []string{}
	}
	return &c, c.Validate()
}
