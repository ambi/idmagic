package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

type ConnectionRepository struct{ Pool sharedpg.DB }

func (r *ConnectionRepository) Save(ctx context.Context, c *domain.IdentityProviderConnection) error {
	if err := c.Validate(); err != nil {
		return err
	}
	certs, err := json.Marshal(c.SAMLSigningCertificates)
	if err != nil {
		return err
	}
	mapping, err := json.Marshal(c.ClaimMapping)
	if err != nil {
		return err
	}
	domains, err := json.Marshal(c.AllowedEmailDomains)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
INSERT INTO identity_provider_connections (
  tenant_id, provider_id, display_name, protocol, status, issuer, client_id, secret_reference,
  authorization_endpoint, token_endpoint, jwks_uri, saml_sso_url, saml_entity_id,
  saml_signing_certificates, claim_mapping, linking_policy, jit_provisioning,
  allowed_email_domains, metadata_refreshed_at, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),
  NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),$14,$15,$16,$17,$18,$19,$20,$21)
ON CONFLICT (tenant_id, provider_id) DO UPDATE SET
  display_name=EXCLUDED.display_name, protocol=EXCLUDED.protocol, status=EXCLUDED.status,
  issuer=EXCLUDED.issuer, client_id=EXCLUDED.client_id,
  secret_reference=COALESCE(EXCLUDED.secret_reference, identity_provider_connections.secret_reference),
  authorization_endpoint=EXCLUDED.authorization_endpoint, token_endpoint=EXCLUDED.token_endpoint,
  jwks_uri=EXCLUDED.jwks_uri, saml_sso_url=EXCLUDED.saml_sso_url,
  saml_entity_id=EXCLUDED.saml_entity_id,
  saml_signing_certificates=EXCLUDED.saml_signing_certificates,
  claim_mapping=EXCLUDED.claim_mapping, linking_policy=EXCLUDED.linking_policy,
  jit_provisioning=EXCLUDED.jit_provisioning, allowed_email_domains=EXCLUDED.allowed_email_domains,
  metadata_refreshed_at=EXCLUDED.metadata_refreshed_at, updated_at=EXCLUDED.updated_at`,
		c.TenantID, c.ID, c.DisplayName, c.Protocol, c.Status, c.Issuer, c.ClientID,
		c.SecretReference, c.AuthorizationEndpoint, c.TokenEndpoint, c.JWKSURI, c.SAMLSSOURL,
		c.SAMLEntityID, certs, mapping, c.LinkingPolicy, c.JITProvisioning, domains,
		c.MetadataRefreshedAt, c.CreatedAt, c.UpdatedAt)
	return err
}

const connectionColumns = `tenant_id, provider_id, display_name, protocol, status, issuer,
COALESCE(client_id,''), COALESCE(secret_reference,''), COALESCE(authorization_endpoint,''),
COALESCE(token_endpoint,''), COALESCE(jwks_uri,''), COALESCE(saml_sso_url,''),
COALESCE(saml_entity_id,''), saml_signing_certificates, claim_mapping, linking_policy,
jit_provisioning, allowed_email_domains, metadata_refreshed_at, created_at, updated_at`

func scanConnection(row sharedpg.RowScanner) (*domain.IdentityProviderConnection, error) {
	c := &domain.IdentityProviderConnection{}
	var certs, mapping, domains []byte
	err := row.Scan(
		&c.TenantID, &c.ID, &c.DisplayName, &c.Protocol, &c.Status, &c.Issuer,
		&c.ClientID, &c.SecretReference, &c.AuthorizationEndpoint, &c.TokenEndpoint, &c.JWKSURI,
		&c.SAMLSSOURL, &c.SAMLEntityID, &certs, &mapping, &c.LinkingPolicy,
		&c.JITProvisioning, &domains, &c.MetadataRefreshedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(certs, &c.SAMLSigningCertificates); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(mapping, &c.ClaimMapping); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(domains, &c.AllowedEmailDomains); err != nil {
		return nil, err
	}
	return c, c.Validate()
}

func (r *ConnectionRepository) Find(ctx context.Context, tenantID, id string) (*domain.IdentityProviderConnection, error) {
	c, err := scanConnection(r.Pool.QueryRow(ctx, `SELECT `+connectionColumns+`
FROM identity_provider_connections WHERE tenant_id=$1 AND provider_id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *ConnectionRepository) List(ctx context.Context, tenantID string) ([]*domain.IdentityProviderConnection, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+connectionColumns+`
FROM identity_provider_connections WHERE tenant_id=$1 ORDER BY display_name, provider_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.IdentityProviderConnection, 0)
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ConnectionRepository) Delete(ctx context.Context, tenantID, id string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM identity_provider_connections WHERE tenant_id=$1 AND provider_id=$2`, tenantID, id)
	return err
}

type IdentityRepository struct{ Pool sharedpg.DB }

func (r *IdentityRepository) Create(ctx context.Context, identity *domain.FederatedIdentity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	_, err := r.Pool.Exec(ctx, `
INSERT INTO federated_identities
  (tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at)
VALUES ($1,$2,$3,$4,$5,$6)`,
		identity.TenantID, identity.ProviderID, identity.ExternalSubject, identity.LocalUserID,
		identity.LinkedAt, identity.LastLoginAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return federationports.ErrLinkConflict
	}
	return err
}

func scanIdentity(row sharedpg.RowScanner) (*domain.FederatedIdentity, error) {
	identity := &domain.FederatedIdentity{}
	err := row.Scan(&identity.TenantID, &identity.ProviderID, &identity.ExternalSubject,
		&identity.LocalUserID, &identity.LinkedAt, &identity.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return identity, err
}

const identityColumns = `tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at`

func (r *IdentityRepository) FindBySubject(ctx context.Context, tenantID, providerID, subject string) (*domain.FederatedIdentity, error) {
	return scanIdentity(r.Pool.QueryRow(ctx, `SELECT `+identityColumns+`
FROM federated_identities WHERE tenant_id=$1 AND provider_id=$2 AND external_subject=$3`,
		tenantID, providerID, subject))
}

func (r *IdentityRepository) FindByUserProvider(ctx context.Context, tenantID, providerID, userID string) (*domain.FederatedIdentity, error) {
	return scanIdentity(r.Pool.QueryRow(ctx, `SELECT `+identityColumns+`
FROM federated_identities WHERE tenant_id=$1 AND provider_id=$2 AND local_user_id=$3`,
		tenantID, providerID, userID))
}

func (r *IdentityRepository) ListByUser(ctx context.Context, tenantID, userID string) ([]*domain.FederatedIdentity, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+identityColumns+`
FROM federated_identities WHERE tenant_id=$1 AND local_user_id=$2 ORDER BY provider_id`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.FederatedIdentity, 0)
	for rows.Next() {
		identity, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, identity)
	}
	return out, rows.Err()
}

func (r *IdentityRepository) Delete(ctx context.Context, tenantID, providerID, userID string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM federated_identities
WHERE tenant_id=$1 AND provider_id=$2 AND local_user_id=$3`, tenantID, providerID, userID)
	return err
}

type AttemptStore struct{ Pool sharedpg.DB }

func (s *AttemptStore) Save(ctx context.Context, a *domain.FederatedLoginAttempt) error {
	_, err := s.Pool.Exec(ctx, `
INSERT INTO federated_login_attempts
  (tenant_id,state,provider_id,protocol,nonce,pkce_verifier,request_id,return_to,link_user_id,
   created_at,expires_at,consumed_at)
VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,'')::uuid,
  $10,$11,$12)`,
		a.TenantID, a.State, a.ProviderID, a.Protocol, a.Nonce, a.PKCEVerifier, a.RequestID,
		a.ReturnTo, a.LinkUserID, a.CreatedAt, a.ExpiresAt, a.ConsumedAt)
	return err
}

func (s *AttemptStore) Consume(ctx context.Context, tenantID, state string, now time.Time) (*domain.FederatedLoginAttempt, error) {
	a := &domain.FederatedLoginAttempt{}
	err := s.Pool.QueryRow(ctx, `
UPDATE federated_login_attempts SET consumed_at=$3
WHERE tenant_id=$1 AND state=$2 AND consumed_at IS NULL AND expires_at>$3
RETURNING state,tenant_id,provider_id,protocol,COALESCE(nonce,''),COALESCE(pkce_verifier,''),
  COALESCE(request_id,''),COALESCE(return_to,''),COALESCE(link_user_id::text,''),
  created_at,expires_at,consumed_at`, tenantID, state, now).Scan(
		&a.State, &a.TenantID, &a.ProviderID, &a.Protocol, &a.Nonce, &a.PKCEVerifier,
		&a.RequestID, &a.ReturnTo, &a.LinkUserID, &a.CreatedAt, &a.ExpiresAt, &a.ConsumedAt)
	if err == nil {
		return a, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var consumed *time.Time
	err = s.Pool.QueryRow(ctx, `SELECT consumed_at FROM federated_login_attempts
WHERE tenant_id=$1 AND state=$2`, tenantID, state).Scan(&consumed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, federationports.ErrAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	return nil, federationports.ErrAttemptConsumed
}

type ReplayStore struct{ Pool sharedpg.DB }

func (s *ReplayStore) Reserve(ctx context.Context, tenantID, id string, expiresAt time.Time) (bool, error) {
	_, _ = s.Pool.Exec(ctx, `DELETE FROM federated_response_replays
WHERE tenant_id=$1 AND response_id=$2 AND expires_at<=now()`, tenantID, id)
	var responseID string
	err := s.Pool.QueryRow(ctx, `INSERT INTO federated_response_replays
  (tenant_id,response_id,expires_at) VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING RETURNING response_id`, tenantID, id, expiresAt).Scan(&responseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
