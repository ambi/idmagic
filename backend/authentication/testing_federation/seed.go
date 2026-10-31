// Package testing_federation は、ほかのモジュールのテストが IdP の接続と連携を置くための補助である。
package testing_federation

import (
	"context"
	"testing"
	"time"

	authentication "github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
)

// SeedOIDCConnection は有効な OIDC の接続を置く。
func SeedOIDCConnection(t *testing.T, module authentication.Module, tenantID, id, issuer, jwksURI string) {
	t.Helper()
	now := time.Now().UTC()
	if err := module.FederationConnectionRepo.Save(context.Background(), &domain.IdentityProviderConnection{
		ID: id, TenantID: tenantID, DisplayName: id, Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
		Issuer: issuer, ClientID: "rp", JWKSURI: jwksURI, LinkingPolicy: domain.LinkingNone,
		AuthorizationEndpoint: issuer + "/authorize", TokenEndpoint: issuer + "/token",
		ClaimMapping: domain.ClaimMapping{Subject: "sub", Username: "preferred_username"},
		CreatedAt:    now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed connection: %v", err)
	}
}

// LinkIdentity は外部の主体を利用者へ連携する。
func LinkIdentity(t *testing.T, module authentication.Module, tenantID, providerID, externalSubject, localUserID string) {
	t.Helper()
	if err := module.FederationIdentityRepo.Create(context.Background(), &domain.FederatedIdentity{
		TenantID: tenantID, ProviderID: providerID, ExternalSubject: externalSubject,
		LocalUserID: localUserID, LinkedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("link identity: %v", err)
	}
}
