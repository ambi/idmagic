package db_postgres

import (
	"context"
	"encoding/json"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keysJOSE "github.com/ambi/idmagic/backend/signingkeys/internal/keys_jose"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5/pgtype"
)

func lockKeySet(ctx context.Context, q *Queries) error {
	return q.LockSigningKeyRotation(ctx, LockSigningKeyRotationParams{
		Column1: pgtype.Text{String: tenantports.TenantID(ctx), Valid: true},
		Column2: pgtype.Text{String: string(signingports.KeyUsage(ctx)), Valid: true},
		Column3: pgtype.Text{String: signingports.KeyScope(ctx), Valid: true},
	})
}

func (s *KeyStore) Prepare(ctx context.Context, now time.Time) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	if _, err := s.GetActiveKey(ctx); err != nil {
		return signingdomain.RotationResult{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := lockKeySet(ctx, q); err != nil {
		return signingdomain.RotationResult{}, err
	}
	keys, err := loadKeySet(ctx, q)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	if prepared := signingdomain.PreparedKey(keys); prepared != nil {
		return signingdomain.RotationResult{Next: prepared, Previous: signingdomain.ActiveKey(keys)}, nil
	}
	priv, public, private, kid, err := keysJOSE.GenerateRSAJWKPair()
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	certificate, err := keysJOSE.NewFederationCertificate(tenantports.TenantID(ctx), kid, priv, now)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	publicJSON, err := json.Marshal(public)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	privateJSON, err := json.Marshal(private)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	key := &signingdomain.SigningKey{
		TenantID: tenantports.TenantID(ctx), Kid: kid, Alg: signingdomain.SigAlgPS256,
		Provider: signingdomain.KeyProviderDatabase, Usage: signingports.KeyUsage(ctx), ScopeID: signingports.KeyScope(ctx),
		PrivateKey: priv, PublicKey: &priv.PublicKey, PublicJWK: public, CertificateDER: certificate, Prepared: true, CreatedAt: now,
	}
	if err := q.InsertSigningKey(ctx, InsertSigningKeyParams{
		Kid: kid, TenantID: key.TenantID, KeyUsage: string(key.Usage), ScopeID: key.ScopeID,
		PublicJwk: publicJSON, PrivateJwk: privateJSON, CertificateDer: certificate, Prepared: true, CreatedAt: now,
	}); err != nil {
		return signingdomain.RotationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return signingdomain.RotationResult{}, err
	}
	return signingdomain.RotationResult{Next: key, Previous: signingdomain.ActiveKey(keys), Changed: true}, nil
}

func (s *KeyStore) Activate(ctx context.Context, kid string, now time.Time, grace time.Duration) (signingdomain.RotationResult, error) {
	if signingports.KeyUsage(ctx) != signingdomain.KeyUsageXMLFederationSigning {
		return signingdomain.RotationResult{}, signingdomain.ErrKeyNotPrepared
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := lockKeySet(ctx, q); err != nil {
		return signingdomain.RotationResult{}, err
	}
	keys, err := loadKeySet(ctx, q)
	if err != nil {
		return signingdomain.RotationResult{}, err
	}
	result, err := signingdomain.ActivatePrepared(keys, kid, now, grace)
	if err != nil || !result.Changed {
		return result, err
	}
	expires := now.Add(grace)
	if err := q.RetireActiveKey(ctx, RetireActiveKeyParams{TenantID: tenantports.TenantID(ctx), KeyUsage: string(signingports.KeyUsage(ctx)), ScopeID: signingports.KeyScope(ctx), RetiredAt: timeToPg(&now), ExpiresAt: timeToPg(&expires)}); err != nil {
		return signingdomain.RotationResult{}, err
	}
	if err := q.ActivatePreparedKey(ctx, ActivatePreparedKeyParams{Kid: kid, TenantID: tenantports.TenantID(ctx), KeyUsage: string(signingports.KeyUsage(ctx)), ScopeID: signingports.KeyScope(ctx), UpdatedAt: now}); err != nil {
		return signingdomain.RotationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return signingdomain.RotationResult{}, err
	}
	return result, nil
}
