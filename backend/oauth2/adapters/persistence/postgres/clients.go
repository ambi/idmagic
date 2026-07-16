package postgres

import (
	"context"
	"encoding/json"
	"errors"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// OAuth2ClientRepository は OAuth2Client を PostgreSQL に永続化する。クエリは sqlc 生成
// (wi-173, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type OAuth2ClientRepository struct{ Pool sharedpg.DB }

func clientFromRow(row *sqlcgen.Client) (*domain.OAuth2Client, error) {
	c := &domain.OAuth2Client{
		TenantID:                           row.TenantID,
		ClientID:                           row.ClientID,
		ClientType:                         spec.ClientType(row.ClientType),
		TokenEndpointAuthMethod:            domain.TokenEndpointAuthMethod(row.TokenEndpointAuthMethod),
		Scope:                              row.Scope,
		IDTokenSignedResponseAlg:           signingdomain.SignatureAlgorithm(row.IDTokenSignedResponseAlg),
		RequirePushedAuthorizationRequests: row.RequirePushedAuthorizationRequests,
		DpopBoundAccessTokens:              row.DpopBoundAccessTokens,
		FapiProfile:                        domain.FapiProfile(row.FapiProfile),
		FirstParty:                         row.FirstParty,
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
	}
	if row.ClientSecretHash.Valid {
		c.ClientSecretHash = &row.ClientSecretHash.String
	}
	if row.ClientName.Valid {
		c.ClientName = &row.ClientName.String
	}
	if row.JwksUri.Valid {
		c.JwksURI = &row.JwksUri.String
	}
	if row.TlsClientAuthSubjectDn.Valid {
		c.TlsClientAuthSubjectDN = &row.TlsClientAuthSubjectDn.String
	}
	if err := json.Unmarshal(row.RedirectUris, &c.RedirectURIs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.GrantTypes, &c.GrantTypes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.ResponseTypes, &c.ResponseTypes); err != nil {
		return nil, err
	}
	if len(row.Jwks) > 0 {
		if err := json.Unmarshal(row.Jwks, &c.JWKS); err != nil {
			return nil, err
		}
	}
	return c, c.Validate()
}

func (r *OAuth2ClientRepository) FindByID(ctx context.Context, tenantID, clientID string) (*domain.OAuth2Client, error) {
	row, err := sqlcgen.New(r.Pool).GetClientByID(ctx, sqlcgen.GetClientByIDParams{TenantID: tenantID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return clientFromRow(row)
}

func (r *OAuth2ClientRepository) FindAll(ctx context.Context, tenantID string) ([]*domain.OAuth2Client, error) {
	rows, err := sqlcgen.New(r.Pool).ListClientsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.OAuth2Client, 0, len(rows))
	for _, row := range rows {
		c, err := clientFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func (r *OAuth2ClientRepository) Save(ctx context.Context, c *domain.OAuth2Client) error {
	redirectURIs, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return err
	}
	grantTypes, err := json.Marshal(c.GrantTypes)
	if err != nil {
		return err
	}
	responseTypes, err := json.Marshal(c.ResponseTypes)
	if err != nil {
		return err
	}
	var jwks []byte
	if c.JWKS != nil {
		jwks, err = json.Marshal(c.JWKS)
		if err != nil {
			return err
		}
	}
	return sqlcgen.New(r.Pool).UpsertClient(ctx, sqlcgen.UpsertClientParams{
		TenantID:                           c.TenantID,
		ClientID:                           c.ClientID,
		ClientSecretHash:                   textOrNil(c.ClientSecretHash),
		ClientName:                         textOrNil(c.ClientName),
		ClientType:                         string(c.ClientType),
		RedirectUris:                       redirectURIs,
		GrantTypes:                         grantTypes,
		ResponseTypes:                      responseTypes,
		TokenEndpointAuthMethod:            string(c.TokenEndpointAuthMethod),
		Scope:                              c.Scope,
		JwksUri:                            textOrNil(c.JwksURI),
		Jwks:                               jwks,
		TlsClientAuthSubjectDn:             textOrNil(c.TlsClientAuthSubjectDN),
		IDTokenSignedResponseAlg:           string(c.IDTokenSignedResponseAlg),
		RequirePushedAuthorizationRequests: c.RequirePushedAuthorizationRequests,
		DpopBoundAccessTokens:              c.DpopBoundAccessTokens,
		FapiProfile:                        string(c.FapiProfile),
		FirstParty:                         c.FirstParty,
		CreatedAt:                          c.CreatedAt,
		UpdatedAt:                          c.UpdatedAt,
	})
}

func (r *OAuth2ClientRepository) Delete(ctx context.Context, tenantID, clientID string) error {
	return sqlcgen.New(r.Pool).DeleteClient(ctx, sqlcgen.DeleteClientParams{TenantID: tenantID, ClientID: clientID})
}
