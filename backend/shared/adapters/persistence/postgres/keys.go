package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// KeyStore (OAuth2: 署名鍵)。tenant scope は ctx (tenancy.TenantID) から解決する。
// 秘密鍵マテリアルを app DB に置く dev/test 用の provider。本番は VaultTransit を使う。
type KeyStore struct{ Pool DB }

func NewKeyStore(ctx context.Context, pool DB) (*KeyStore, error) {
	store := &KeyStore{Pool: pool}
	// default テナントの active 鍵が無ければ 1 本作る (後方互換)。
	var exists bool
	if err := pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM signing_keys WHERE active AND tenant_id=$1)",
		spec.DefaultTenantID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		if _, err := store.rotateForTenant(ctx, spec.DefaultTenantID); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *KeyStore) GetActiveKey(ctx context.Context) (*ports.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	key, err := scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE active=TRUE AND tenant_id=$1 LIMIT 1", tenantID))
	if err != nil {
		return nil, err
	}
	if key == nil {
		// まだ鍵の無いテナントは初回に遅延生成する。
		return s.rotateForTenant(ctx, tenantID)
	}
	return key, nil
}

func (s *KeyStore) GetAllKeys(ctx context.Context) ([]*ports.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	rows, err := s.Pool.Query(ctx,
		keySelect+" WHERE archived_at IS NULL AND tenant_id=$1 ORDER BY created_at DESC", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ports.SigningKey{}
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *KeyStore) FindByKID(ctx context.Context, kid string) (*ports.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	return scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE kid=$1 AND tenant_id=$2", kid, tenantID))
}

func (s *KeyStore) Rotate(ctx context.Context) (*ports.SigningKey, error) {
	return s.rotateForTenant(ctx, tenancy.TenantID(ctx))
}

// Disable は ctx テナントの鍵 1 件を archive し JWKS から即時に外す。
func (s *KeyStore) Disable(ctx context.Context, kid string) (*ports.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	key, err := scanSigningKey(s.Pool.QueryRow(ctx,
		keySelect+" WHERE kid=$1 AND tenant_id=$2", kid, tenantID))
	if err != nil || key == nil {
		return nil, err
	}
	if _, err := s.Pool.Exec(ctx,
		"UPDATE signing_keys SET active=FALSE,archived_at=now(),updated_at=now() WHERE kid=$1 AND tenant_id=$2",
		kid, tenantID); err != nil {
		return nil, err
	}
	key.Active = false
	return key, nil
}

func (s *KeyStore) Provider() spec.KeyProvider { return spec.KeyProviderPostgres }

func (s *KeyStore) Healthy(ctx context.Context) bool { return s.Pool.Ping(ctx) == nil }

func (s *KeyStore) rotateForTenant(ctx context.Context, tenantID string) (*ports.SigningKey, error) {
	priv, publicJWK, privateJWK, kid, err := crypto.GenerateRSAJWKPair()
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
	if _, err := tx.Exec(ctx,
		"UPDATE signing_keys SET active=FALSE,rotated_at=now(),updated_at=now() WHERE active AND tenant_id=$1",
		tenantID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO signing_keys
(kid,tenant_id,alg,provider,key_usage,public_jwk,private_jwk,active)
VALUES ($1,$2,'PS256','Postgres','Signing',$3,$4,TRUE)`,
		kid, tenantID, string(publicJSON), string(privateJSON)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &ports.SigningKey{
		TenantID: tenantID, Kid: kid, Alg: spec.SigAlgPS256,
		Provider: spec.KeyProviderPostgres, Usage: spec.KeyUsageSigning,
		PrivateKey: priv, PublicKey: &priv.PublicKey,
		PublicJWK: publicJWK, Active: true, CreatedAt: time.Now().UTC(),
	}, nil
}

const keySelect = `SELECT kid,tenant_id,alg,provider,key_usage,public_jwk,private_jwk,active,created_at FROM signing_keys`

func scanSigningKey(row RowScanner) (*ports.SigningKey, error) {
	var key ports.SigningKey
	var publicJSON, privateJSON []byte
	err := row.Scan(&key.Kid, &key.TenantID, &key.Alg, &key.Provider, &key.Usage,
		&publicJSON, &privateJSON, &key.Active, &key.CreatedAt)
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
	pub, priv, err := crypto.ImportRSAJWK(publicJWK, privateJWK)
	if err != nil {
		return nil, err
	}
	key.PublicJWK, key.PublicKey, key.PrivateKey = publicJWK, pub, priv
	return &key, nil
}
