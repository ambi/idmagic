package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/jackc/pgx/v5"

	sharedpostgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_jose"
	"github.com/ambi/idmagic/backend/tenancy"
)

// KeyStore (OAuth2: 署名鍵)。tenant scope は ctx (tenancy.TenantID) から解決する。
// 秘密鍵マテリアルを app DB に置く dev/test 用の provider。本番は VaultTransit を使う。
type KeyStore struct{ Pool sharedpostgres.DB }

func NewKeyStore(ctx context.Context, pool sharedpostgres.DB) (*KeyStore, error) {
	store := &KeyStore{Pool: pool}
	// default テナントの active 鍵が無ければ 1 本作る (後方互換)。
	var exists bool
	if err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM signing_keys WHERE active AND tenant_id=$1)",
		tenancydomain.DefaultTenantID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		if _, err := store.rotateForTenant(ctx, tenancydomain.DefaultTenantID, time.Now().UTC(), 7*24*time.Hour, nil); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *KeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	key, err := scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE active=TRUE AND tenant_id=$1 LIMIT 1", tenantID))
	if err != nil {
		return nil, err
	}
	if key == nil {
		// まだ鍵の無いテナントは初回に遅延生成する。
		return s.rotateForTenant(ctx, tenantID, time.Now().UTC(), 7*24*time.Hour, nil)
	}
	return key, nil
}

func (s *KeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	rows, err := s.Pool.Query(ctx,
		keySelect+" WHERE archived_at IS NULL AND tenant_id=$1 ORDER BY created_at DESC", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*signingdomain.SigningKey{}
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *KeyStore) ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	rows, err := s.Pool.Query(ctx, keySelect+" WHERE archived_at IS NULL AND tenant_id=$1 AND (active=TRUE OR expires_at>$2) ORDER BY created_at DESC", tenantID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*signingdomain.SigningKey{}
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *KeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	return scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE kid=$1 AND tenant_id=$2", kid, tenantID))
}

func (s *KeyStore) Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	return s.rotateForTenant(ctx, tenancy.TenantID(ctx), now, grace, nil)
}

func (s *KeyStore) RotateIfDue(ctx context.Context, now time.Time, cadence, grace time.Duration) (*signingdomain.SigningKey, error) {
	if cadence <= 0 {
		return nil, errors.New("signing key rotation cadence must be positive")
	}
	dueBefore := now.Add(-cadence)
	return s.rotateForTenant(ctx, tenancy.TenantID(ctx), now, grace, &dueBefore)
}

// Disable は ctx テナントの鍵 1 件を archive し JWKS から即時に外す。
func (s *KeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	key, err := scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE kid=$1 AND tenant_id=$2", kid, tenantID))
	if err != nil || key == nil {
		return nil, err
	}
	if key.Active {
		return nil, signingdomain.ErrActiveSigningKeyCannotBeDisabled
	}
	if _, err := s.Pool.Exec(ctx,
		"UPDATE signing_keys SET active=FALSE,archived_at=now(),updated_at=now() WHERE kid=$1 AND tenant_id=$2",
		kid, tenantID); err != nil {
		return nil, err
	}
	key.Active = false
	return key, nil
}

func (s *KeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	rows, err := s.Pool.Query(ctx, `UPDATE signing_keys
SET archived_at=$2,updated_at=$2
WHERE archived_at IS NULL AND tenant_id=$1 AND expires_at IS NOT NULL AND expires_at<=$2
RETURNING kid,tenant_id,alg,provider,key_usage,public_jwk,private_jwk,active,created_at,retired_at,expires_at,archived_at`,
		tenantID, before.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	archived := []*signingdomain.SigningKey{}
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		archived = append(archived, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return archived, nil
}

func (s *KeyStore) Provider() signingdomain.KeyProvider { return signingdomain.KeyProviderDatabase }

func (s *KeyStore) Healthy(ctx context.Context) bool { return s.Pool.Ping(ctx) == nil }

func (s *KeyStore) rotateForTenant(ctx context.Context, tenantID string, now time.Time, grace time.Duration, dueBefore *time.Time) (*signingdomain.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if grace < 0 {
		return nil, errors.New("signing key grace period must not be negative")
	}
	expires := now.Add(grace)
	priv, publicJWK, privateJWK, kid, err := signingcrypto.GenerateRSAJWKPair()
	if err != nil {
		return nil, err
	}
	publicJSON, err := json.Marshal(publicJWK)
	if err != nil {
		return nil, err
	}
	privateJSON, err := json.Marshal(privateJWK)
	if err != nil {
		return nil, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// advisory lock は tenant 単位に取り、テナント間の回転は直列化しない。
	if _, err := tx.Exec(ctx,
		"SELECT pg_advisory_xact_lock(hashtext('idmagic-signing-key:'||$1))", tenantID); err != nil {
		return nil, err
	}
	if dueBefore != nil {
		var createdAt time.Time
		err := tx.QueryRow(ctx,
			"SELECT created_at FROM signing_keys WHERE active AND tenant_id=$1 LIMIT 1 FOR UPDATE",
			tenantID).Scan(&createdAt)
		if err == nil && createdAt.After(*dueBefore) {
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return nil, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx,
		"UPDATE signing_keys SET active=FALSE,retired_at=$2,expires_at=$3,updated_at=$2 WHERE active AND tenant_id=$1",
		tenantID, now, expires); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO signing_keys
(kid,tenant_id,alg,provider,key_usage,public_jwk,private_jwk,active)
VALUES ($1,$2,'PS256','Database','Signing',$3,$4,TRUE)`,
		kid, tenantID, string(publicJSON), string(privateJSON)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &signingdomain.SigningKey{
		TenantID: tenantID, Kid: kid, Alg: signingdomain.SigAlgPS256,
		Provider: signingdomain.KeyProviderDatabase, Usage: signingdomain.KeyUsageSigning,
		PrivateKey: priv, PublicKey: &priv.PublicKey,
		PublicJWK: publicJWK, Active: true, CreatedAt: now,
	}, nil
}

const keySelect = `SELECT kid,tenant_id,alg,provider,key_usage,public_jwk,private_jwk,active,created_at,retired_at,expires_at,archived_at FROM signing_keys`

func scanSigningKey(row sharedpostgres.RowScanner) (*signingdomain.SigningKey, error) {
	var key signingdomain.SigningKey
	var publicJSON, privateJSON []byte
	err := row.Scan(&key.Kid, &key.TenantID, &key.Alg, &key.Provider, &key.Usage,
		&publicJSON, &privateJSON, &key.Active, &key.CreatedAt, &key.RetiredAt, &key.ExpiresAt, &key.ArchivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var publicJWK, privateJWK map[string]any
	if err := json.Unmarshal(publicJSON, &publicJWK); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(privateJSON, &privateJWK); err != nil {
		return nil, err
	}
	pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
	if err != nil {
		return nil, err
	}
	key.PublicJWK, key.PublicKey, key.PrivateKey = publicJWK, pub, priv
	return &key, nil
}
