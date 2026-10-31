package server_http

import (
	"context"

	authentication "github.com/ambi/idmagic/backend/authentication"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
)

// idJagIssuerTrust は Authentication の接続と連携を OAuth2 の契約へ写す。OAuth2 と
// Authentication が互いの型を知らずに済むよう、組み立て地点に置く。
type idJagIssuerTrust struct {
	authentication authentication.Module
	fetch          func(ctx context.Context, jwksURI string) ([]map[string]any, error)
}

func (t idJagIssuerTrust) ResolveIssuer(ctx context.Context, tenantID, issuer string) (*oauthports.IDJAGTrustedIssuer, error) {
	found, err := t.authentication.FindIDJAGIssuer(ctx, tenantID, issuer)
	if err != nil || found == nil {
		return nil, err
	}
	jwks, err := t.fetch(ctx, found.JWKSURI)
	if err != nil {
		return nil, err
	}
	return &oauthports.IDJAGTrustedIssuer{ProviderID: found.ProviderID, JWKS: jwks}, nil
}

func (t idJagIssuerTrust) ProviderExists(ctx context.Context, tenantID, providerID string) (bool, error) {
	return t.authentication.FederationProviderExists(ctx, tenantID, providerID)
}

func (t idJagIssuerTrust) ResolveSubject(ctx context.Context, tenantID, providerID, externalSubject string) (string, error) {
	return t.authentication.FederatedLocalUserID(ctx, tenantID, providerID, externalSubject)
}
