package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpostgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_jose"
	"github.com/ambi/idmagic/backend/tenancy"
)

// KeyStore (OAuth2: 署名鍵)。tenant scope は ctx (tenancy.TenantID) から解決する。
// 秘密鍵マテリアルを app DB に置く dev/test 用の provider。本番は VaultTransit を使う。
type KeyStore struct {
	Pool sharedpostgres.DB
}

func NewKeyStore(_ context.Context, pool sharedpostgres.DB) (*KeyStore, error) {
	return &KeyStore{Pool: pool}, nil
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

func (s *KeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)

	row, err := New(s.Pool).GetActiveKey(ctx, GetActiveKeyParams{
		TenantID: tenantID,
		KeyUsage: string(usage),
		ScopeID:  scopeID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return s.rotateForTenant(ctx, tenantID, time.Now().UTC(), 7*24*time.Hour, nil)
	}
	if err != nil {
		return nil, err
	}

	var publicJWK, privateJWK map[string]any
	if err := json.Unmarshal(row.PublicJwk, &publicJWK); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.PrivateJwk, &privateJWK); err != nil {
		return nil, err
	}
	pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
	if err != nil {
		return nil, err
	}

	return &signingdomain.SigningKey{
		Kid:            row.Kid,
		TenantID:       row.TenantID,
		Alg:            signingdomain.SignatureAlgorithm(row.Alg),
		Provider:       signingdomain.KeyProvider(row.Provider),
		Usage:          signingdomain.KeyUsage(row.KeyUsage),
		ScopeID:        row.ScopeID,
		PublicJWK:      publicJWK,
		PublicKey:      pub,
		PrivateKey:     priv,
		CertificateDER: row.CertificateDer,
		Active:         row.Active,
		CreatedAt:      row.CreatedAt,
		RetiredAt:      timePtr(row.RetiredAt),
		ExpiresAt:      timePtr(row.ExpiresAt),
		ArchivedAt:     timePtr(row.ArchivedAt),
	}, nil
}

func (s *KeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	rows, err := New(s.Pool).GetAllKeys(ctx, GetAllKeysParams{
		TenantID: tenantID,
		KeyUsage: string(usage),
		ScopeID:  scopeID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*signingdomain.SigningKey, 0, len(rows))
	for _, row := range rows {
		var publicJWK, privateJWK map[string]any
		if err := json.Unmarshal(row.PublicJwk, &publicJWK); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.PrivateJwk, &privateJWK); err != nil {
			return nil, err
		}
		pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
		if err != nil {
			return nil, err
		}
		out = append(out, &signingdomain.SigningKey{
			Kid:            row.Kid,
			TenantID:       row.TenantID,
			Alg:            signingdomain.SignatureAlgorithm(row.Alg),
			Provider:       signingdomain.KeyProvider(row.Provider),
			Usage:          signingdomain.KeyUsage(row.KeyUsage),
			ScopeID:        row.ScopeID,
			PublicJWK:      publicJWK,
			PublicKey:      pub,
			PrivateKey:     priv,
			CertificateDER: row.CertificateDer,
			Active:         row.Active,
			CreatedAt:      row.CreatedAt,
			RetiredAt:      timePtr(row.RetiredAt),
			ExpiresAt:      timePtr(row.ExpiresAt),
			ArchivedAt:     timePtr(row.ArchivedAt),
		})
	}
	return out, nil
}

func (s *KeyStore) ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)

	expiresAt := now
	rows, err := New(s.Pool).ListPublicKeys(ctx, ListPublicKeysParams{
		TenantID:  tenantID,
		KeyUsage:  string(usage),
		ScopeID:   scopeID,
		ExpiresAt: timeToPg(&expiresAt),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*signingdomain.SigningKey, 0, len(rows))
	for _, row := range rows {
		var publicJWK, privateJWK map[string]any
		if err := json.Unmarshal(row.PublicJwk, &publicJWK); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.PrivateJwk, &privateJWK); err != nil {
			return nil, err
		}
		pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
		if err != nil {
			return nil, err
		}
		out = append(out, &signingdomain.SigningKey{
			Kid:            row.Kid,
			TenantID:       row.TenantID,
			Alg:            signingdomain.SignatureAlgorithm(row.Alg),
			Provider:       signingdomain.KeyProvider(row.Provider),
			Usage:          signingdomain.KeyUsage(row.KeyUsage),
			ScopeID:        row.ScopeID,
			PublicJWK:      publicJWK,
			PublicKey:      pub,
			PrivateKey:     priv,
			CertificateDER: row.CertificateDer,
			Active:         row.Active,
			CreatedAt:      row.CreatedAt,
			RetiredAt:      timePtr(row.RetiredAt),
			ExpiresAt:      timePtr(row.ExpiresAt),
			ArchivedAt:     timePtr(row.ArchivedAt),
		})
	}
	return out, nil
}

func (s *KeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
	row, err := New(s.Pool).FindKeyByKID(ctx, FindKeyByKIDParams{
		Kid:      kid,
		TenantID: tenantID,
		KeyUsage: string(usage),
		ScopeID:  scopeID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var publicJWK, privateJWK map[string]any
	if err := json.Unmarshal(row.PublicJwk, &publicJWK); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.PrivateJwk, &privateJWK); err != nil {
		return nil, err
	}
	pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
	if err != nil {
		return nil, err
	}

	return &signingdomain.SigningKey{
		Kid:            row.Kid,
		TenantID:       row.TenantID,
		Alg:            signingdomain.SignatureAlgorithm(row.Alg),
		Provider:       signingdomain.KeyProvider(row.Provider),
		Usage:          signingdomain.KeyUsage(row.KeyUsage),
		ScopeID:        row.ScopeID,
		PublicJWK:      publicJWK,
		PublicKey:      pub,
		PrivateKey:     priv,
		CertificateDER: row.CertificateDer,
		Active:         row.Active,
		CreatedAt:      row.CreatedAt,
		RetiredAt:      timePtr(row.RetiredAt),
		ExpiresAt:      timePtr(row.ExpiresAt),
		ArchivedAt:     timePtr(row.ArchivedAt),
	}, nil
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

func (s *KeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	key, err := s.FindByKID(ctx, kid)
	if err != nil || key == nil {
		return nil, err
	}
	if key.Active {
		return nil, signingdomain.ErrActiveSigningKeyCannotBeDisabled
	}
	if err := New(s.Pool).DisableKey(ctx, DisableKeyParams{
		Kid:      kid,
		TenantID: tenancy.TenantID(ctx),
	}); err != nil {
		return nil, err
	}
	key.Active = false
	return key, nil
}

func (s *KeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)

	expiresAt := before.UTC()
	rows, err := New(s.Pool).ArchiveExpiredKeys(ctx, ArchiveExpiredKeysParams{
		ArchivedAt: timeToPg(&expiresAt),
		TenantID:   tenantID,
		KeyUsage:   string(usage),
		ScopeID:    scopeID,
	})
	if err != nil {
		return nil, err
	}

	archived := make([]*signingdomain.SigningKey, 0, len(rows))
	for _, row := range rows {
		var publicJWK, privateJWK map[string]any
		if err := json.Unmarshal(row.PublicJwk, &publicJWK); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.PrivateJwk, &privateJWK); err != nil {
			return nil, err
		}
		pub, priv, err := signingcrypto.ImportRSAJWK(publicJWK, privateJWK)
		if err != nil {
			return nil, err
		}
		archived = append(archived, &signingdomain.SigningKey{
			Kid:            row.Kid,
			TenantID:       row.TenantID,
			Alg:            signingdomain.SignatureAlgorithm(row.Alg),
			Provider:       signingdomain.KeyProvider(row.Provider),
			Usage:          signingdomain.KeyUsage(row.KeyUsage),
			ScopeID:        row.ScopeID,
			PublicJWK:      publicJWK,
			PublicKey:      pub,
			PrivateKey:     priv,
			CertificateDER: row.CertificateDer,
			Active:         row.Active,
			CreatedAt:      row.CreatedAt,
			RetiredAt:      timePtr(row.RetiredAt),
			ExpiresAt:      timePtr(row.ExpiresAt),
			ArchivedAt:     timePtr(row.ArchivedAt),
		})
	}
	return archived, nil
}

func (s *KeyStore) Provider() signingdomain.KeyProvider { return signingdomain.KeyProviderDatabase }

func (s *KeyStore) Healthy(ctx context.Context) bool { return s.Pool.Ping(ctx) == nil }

func (s *KeyStore) rotateForTenant(ctx context.Context, tenantID string, now time.Time, grace time.Duration, dueBefore *time.Time) (*signingdomain.SigningKey, error) {
	usage := signingports.KeyUsage(ctx)
	scopeID := signingports.KeyScope(ctx)
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
	var certificateDER []byte
	if usage == signingdomain.KeyUsageXMLFederationSigning {
		certificateDER, err = signingcrypto.NewFederationCertificate(tenantID, kid, priv, now)
		if err != nil {
			return nil, err
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.LockSigningKeyRotation(ctx, LockSigningKeyRotationParams{
		Column1: pgtype.Text{String: tenantID, Valid: true},
		Column2: pgtype.Text{String: string(usage), Valid: true},
		Column3: pgtype.Text{String: scopeID, Valid: true},
	}); err != nil {
		return nil, err
	}
	if dueBefore != nil {
		createdAt, err := q.GetActiveKeyCreatedAtForUpdate(ctx, GetActiveKeyCreatedAtForUpdateParams{
			TenantID: tenantID,
			KeyUsage: string(usage),
			ScopeID:  scopeID,
		})
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

	retireExpires := expires
	if err := q.RetireActiveKey(ctx, RetireActiveKeyParams{
		RetiredAt: timeToPg(&now),
		ExpiresAt: timeToPg(&retireExpires),
		TenantID:  tenantID,
		KeyUsage:  string(usage),
		ScopeID:   scopeID,
	}); err != nil {
		return nil, err
	}

	if err := q.InsertSigningKey(ctx, InsertSigningKeyParams{
		Kid:            kid,
		TenantID:       tenantID,
		KeyUsage:       string(usage),
		ScopeID:        scopeID,
		PublicJwk:      publicJSON,
		PrivateJwk:     privateJSON,
		CertificateDer: certificateDER,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &signingdomain.SigningKey{
		TenantID: tenantID, Kid: kid, Alg: signingdomain.SigAlgPS256,
		Provider: signingdomain.KeyProviderDatabase, Usage: usage,
		ScopeID:    scopeID,
		PrivateKey: priv, PublicKey: &priv.PublicKey,
		PublicJWK: publicJWK, CertificateDER: certificateDER, Active: true, CreatedAt: now,
	}, nil
}
