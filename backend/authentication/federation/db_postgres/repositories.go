package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

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
	return New(r.Pool).SaveIdentityProviderConnection(ctx, SaveIdentityProviderConnectionParams{
		TenantID:                c.TenantID,
		ProviderID:              c.ID,
		DisplayName:             c.DisplayName,
		Protocol:                string(c.Protocol),
		Status:                  string(c.Status),
		Issuer:                  c.Issuer,
		Column7:                 c.ClientID,
		Column8:                 c.SecretReference,
		Column9:                 c.AuthorizationEndpoint,
		Column10:                c.TokenEndpoint,
		Column11:                c.JWKSURI,
		Column12:                c.SAMLSSOURL,
		Column13:                c.SAMLEntityID,
		SamlSigningCertificates: certs,
		ClaimMapping:            mapping,
		LinkingPolicy:           string(c.LinkingPolicy),
		JitProvisioning:         c.JITProvisioning,
		AllowedEmailDomains:     domains,
		MetadataRefreshedAt:     toTimestamptz(c.MetadataRefreshedAt),
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	})
}

func mapConnection(row *FindIdentityProviderConnectionRow) (*domain.IdentityProviderConnection, error) {
	c := &domain.IdentityProviderConnection{
		TenantID:              row.TenantID,
		ID:                    row.ProviderID,
		DisplayName:           row.DisplayName,
		Protocol:              domain.Protocol(row.Protocol),
		Status:                domain.ConnectionStatus(row.Status),
		Issuer:                row.Issuer,
		ClientID:              row.ClientID,
		SecretReference:       row.SecretReference,
		AuthorizationEndpoint: row.AuthorizationEndpoint,
		TokenEndpoint:         row.TokenEndpoint,
		JWKSURI:               row.JwksUri,
		SAMLSSOURL:            row.SamlSsoUrl,
		SAMLEntityID:          row.SamlEntityID,
		LinkingPolicy:         domain.LinkingPolicy(row.LinkingPolicy),
		JITProvisioning:       row.JitProvisioning,
		MetadataRefreshedAt:   fromTimestamptz(row.MetadataRefreshedAt),
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if err := json.Unmarshal(row.SamlSigningCertificates, &c.SAMLSigningCertificates); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.ClaimMapping, &c.ClaimMapping); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.AllowedEmailDomains, &c.AllowedEmailDomains); err != nil {
		return nil, err
	}
	return c, c.Validate()
}

func mapConnectionListRow(row *ListIdentityProviderConnectionsRow) (*domain.IdentityProviderConnection, error) {
	c := &domain.IdentityProviderConnection{
		TenantID:              row.TenantID,
		ID:                    row.ProviderID,
		DisplayName:           row.DisplayName,
		Protocol:              domain.Protocol(row.Protocol),
		Status:                domain.ConnectionStatus(row.Status),
		Issuer:                row.Issuer,
		ClientID:              row.ClientID,
		SecretReference:       row.SecretReference,
		AuthorizationEndpoint: row.AuthorizationEndpoint,
		TokenEndpoint:         row.TokenEndpoint,
		JWKSURI:               row.JwksUri,
		SAMLSSOURL:            row.SamlSsoUrl,
		SAMLEntityID:          row.SamlEntityID,
		LinkingPolicy:         domain.LinkingPolicy(row.LinkingPolicy),
		JITProvisioning:       row.JitProvisioning,
		MetadataRefreshedAt:   fromTimestamptz(row.MetadataRefreshedAt),
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
	if err := json.Unmarshal(row.SamlSigningCertificates, &c.SAMLSigningCertificates); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.ClaimMapping, &c.ClaimMapping); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.AllowedEmailDomains, &c.AllowedEmailDomains); err != nil {
		return nil, err
	}
	return c, c.Validate()
}

func (r *ConnectionRepository) Find(ctx context.Context, tenantID, id string) (*domain.IdentityProviderConnection, error) {
	row, err := New(r.Pool).FindIdentityProviderConnection(ctx, FindIdentityProviderConnectionParams{
		TenantID:   tenantID,
		ProviderID: id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapConnection(row)
}

func (r *ConnectionRepository) List(ctx context.Context, tenantID string) ([]*domain.IdentityProviderConnection, error) {
	rows, err := New(r.Pool).ListIdentityProviderConnections(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.IdentityProviderConnection, 0, len(rows))
	for _, row := range rows {
		c, err := mapConnectionListRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *ConnectionRepository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteIdentityProviderConnection(ctx, DeleteIdentityProviderConnectionParams{
		TenantID:   tenantID,
		ProviderID: id,
	})
}

type IdentityRepository struct{ Pool sharedpg.DB }

func (r *IdentityRepository) Create(ctx context.Context, identity *domain.FederatedIdentity) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	err := New(r.Pool).CreateFederatedIdentity(ctx, CreateFederatedIdentityParams{
		TenantID:        identity.TenantID,
		ProviderID:      identity.ProviderID,
		ExternalSubject: identity.ExternalSubject,
		LocalUserID:     identity.LocalUserID,
		LinkedAt:        identity.LinkedAt,
		LastLoginAt:     toTimestamptz(identity.LastLoginAt),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return federationports.ErrLinkConflict
	}
	return err
}

func mapIdentity(row *FederatedIdentity) *domain.FederatedIdentity {
	return &domain.FederatedIdentity{
		TenantID:        row.TenantID,
		ProviderID:      row.ProviderID,
		ExternalSubject: row.ExternalSubject,
		LocalUserID:     row.LocalUserID,
		LinkedAt:        row.LinkedAt,
		LastLoginAt:     fromTimestamptz(row.LastLoginAt),
	}
}

func (r *IdentityRepository) FindBySubject(ctx context.Context, tenantID, providerID, subject string) (*domain.FederatedIdentity, error) {
	row, err := New(r.Pool).FindFederatedIdentityBySubject(ctx, FindFederatedIdentityBySubjectParams{
		TenantID:        tenantID,
		ProviderID:      providerID,
		ExternalSubject: subject,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapIdentity(row), nil
}

func (r *IdentityRepository) FindByUserProvider(ctx context.Context, tenantID, providerID, userID string) (*domain.FederatedIdentity, error) {
	row, err := New(r.Pool).FindFederatedIdentityByUserProvider(ctx, FindFederatedIdentityByUserProviderParams{
		TenantID:    tenantID,
		ProviderID:  providerID,
		LocalUserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapIdentity(row), nil
}

func (r *IdentityRepository) ListByUser(ctx context.Context, tenantID, userID string) ([]*domain.FederatedIdentity, error) {
	rows, err := New(r.Pool).ListFederatedIdentitiesByUser(ctx, ListFederatedIdentitiesByUserParams{
		TenantID:    tenantID,
		LocalUserID: userID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.FederatedIdentity, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapIdentity(row))
	}
	return out, nil
}

func (r *IdentityRepository) Delete(ctx context.Context, tenantID, providerID, userID string) error {
	return New(r.Pool).DeleteFederatedIdentity(ctx, DeleteFederatedIdentityParams{
		TenantID:    tenantID,
		ProviderID:  providerID,
		LocalUserID: userID,
	})
}

type AttemptStore struct{ Pool sharedpg.DB }

func (s *AttemptStore) Save(ctx context.Context, a *domain.FederatedLoginAttempt) error {
	return New(s.Pool).SaveFederatedLoginAttempt(ctx, SaveFederatedLoginAttemptParams{
		TenantID:   a.TenantID,
		State:      a.State,
		ProviderID: a.ProviderID,
		Protocol:   string(a.Protocol),
		Column5:    a.Nonce,
		Column6:    a.PKCEVerifier,
		Column7:    a.RequestID,
		Column8:    a.ReturnTo,
		Column9:    a.LinkUserID,
		CreatedAt:  a.CreatedAt,
		ExpiresAt:  a.ExpiresAt,
		ConsumedAt: toTimestamptz(a.ConsumedAt),
	})
}

func (s *AttemptStore) Consume(ctx context.Context, tenantID, state string, now time.Time) (*domain.FederatedLoginAttempt, error) {
	row, err := New(s.Pool).ConsumeFederatedLoginAttempt(ctx, ConsumeFederatedLoginAttemptParams{
		TenantID:   tenantID,
		State:      state,
		ConsumedAt: toTimestamptz(&now),
	})
	if err == nil {
		var linkUserID string
		if row.LinkUserID != nil {
			if str, ok := row.LinkUserID.(string); ok {
				linkUserID = str
			}
		}
		return &domain.FederatedLoginAttempt{
			State:        row.State,
			TenantID:     row.TenantID,
			ProviderID:   row.ProviderID,
			Protocol:     domain.Protocol(row.Protocol),
			Nonce:        row.Nonce,
			PKCEVerifier: row.PkceVerifier,
			RequestID:    row.RequestID,
			ReturnTo:     row.ReturnTo,
			LinkUserID:   linkUserID,
			CreatedAt:    row.CreatedAt,
			ExpiresAt:    row.ExpiresAt,
			ConsumedAt:   fromTimestamptz(row.ConsumedAt),
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	_, err = New(s.Pool).GetFederatedLoginAttemptConsumedAt(ctx, GetFederatedLoginAttemptConsumedAtParams{
		TenantID: tenantID,
		State:    state,
	})
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
	q := New(s.Pool)
	_ = q.DeleteExpiredReplays(ctx, DeleteExpiredReplaysParams{
		TenantID:   tenantID,
		ResponseID: id,
	})

	_, err := q.ReserveReplay(ctx, ReserveReplayParams{
		TenantID:   tenantID,
		ResponseID: id,
		ExpiresAt:  expiresAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
