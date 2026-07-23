package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ambi/idmagic/backend/saml/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// SamlServiceProviderRepository は SAML 2.0 SP trust を PostgreSQL に永続化する。
// URI 識別子と claim policy は tenant scope の行と JSONB に閉じる。
type SamlServiceProviderRepository struct{ Pool sharedpg.DB }

func samlServiceProviderFromRow(row *SamlServiceProvider) (*domain.SamlServiceProvider, error) {
	var sp domain.SamlServiceProvider
	sp.TenantID, sp.EntityID, sp.DisplayName, sp.SLOURL, sp.Audience = row.TenantID, row.EntityID, row.DisplayName, row.SloUrl, row.Audience
	sp.ApplicationID = sharedpg.UUIDString(row.ApplicationID)
	sp.SignAssertion, sp.SignResponse, sp.WantAuthnRequestsSigned = row.SignAssertion, row.SignResponse, row.WantAuthnRequestsSigned
	sp.AuthnRequestSigningCertificatePEM, sp.CreatedAt, sp.UpdatedAt = row.AuthnRequestSigningCertificatePem, row.CreatedAt, row.UpdatedAt
	if err := json.Unmarshal(row.AcsUrls, &sp.ACSURLs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.ClaimPolicy, &sp.ClaimPolicy); err != nil {
		return nil, err
	}
	return &sp, nil
}

func (r *SamlServiceProviderRepository) FindByEntityID(ctx context.Context, tenantID, entityID string) (*domain.SamlServiceProvider, error) {
	row, err := New(r.Pool).GetSamlServiceProvider(ctx, GetSamlServiceProviderParams{TenantID: tenantID, EntityID: entityID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return samlServiceProviderFromRow(row)
}

func (r *SamlServiceProviderRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.SamlServiceProvider, error) {
	rows, err := New(r.Pool).ListSamlServiceProvidersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.SamlServiceProvider, 0, len(rows))
	for _, row := range rows {
		sp, err := samlServiceProviderFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, nil
}

func (r *SamlServiceProviderRepository) Save(ctx context.Context, sp *domain.SamlServiceProvider) error {
	acsURLs := sp.ACSURLs
	if acsURLs == nil {
		acsURLs = []string{}
	}
	encodedACSURLs, err := json.Marshal(acsURLs)
	if err != nil {
		return err
	}
	encodedClaimPolicy, err := json.Marshal(sp.ClaimPolicy)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertSamlServiceProvider(ctx, UpsertSamlServiceProviderParams{TenantID: sp.TenantID, EntityID: sp.EntityID, DisplayName: sp.DisplayName, AcsUrls: encodedACSURLs, SloUrl: sp.SLOURL, Audience: sp.Audience, ClaimPolicy: encodedClaimPolicy, SignAssertion: sp.SignAssertion, SignResponse: sp.SignResponse, WantAuthnRequestsSigned: sp.WantAuthnRequestsSigned, AuthnRequestSigningCertificatePem: sp.AuthnRequestSigningCertificatePEM, CreatedAt: sp.CreatedAt, UpdatedAt: sp.UpdatedAt})
}

func (r *SamlServiceProviderRepository) Delete(ctx context.Context, tenantID, entityID string) error {
	return New(r.Pool).DeleteSamlServiceProvider(ctx, DeleteSamlServiceProviderParams{TenantID: tenantID, EntityID: entityID})
}
