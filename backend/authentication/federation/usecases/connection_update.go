package usecases

import (
	"slices"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
)

// ResolveUpdatedStatus decides whether an update should preserve the existing
// connection's status or degrade it to Disabled. Only a change to the trust
// source (issuer, protocol, or the OIDC/SAML endpoints and certificates that
// establish it) forces a degrade; non-trust fields (display name, claim
// mapping, linking policy, JIT provisioning, allowed email domains) never
// change the status. A connection that is already Disabled stays Disabled.
func ResolveUpdatedStatus(existing, incoming federationdomain.IdentityProviderConnection) federationdomain.ConnectionStatus {
	if existing.Status != federationdomain.ConnectionActive {
		return existing.Status
	}
	if trustSourceChanged(existing, incoming) {
		return federationdomain.ConnectionDisabled
	}
	return existing.Status
}

func trustSourceChanged(existing, incoming federationdomain.IdentityProviderConnection) bool {
	return existing.Protocol != incoming.Protocol ||
		existing.Issuer != incoming.Issuer ||
		existing.ClientID != incoming.ClientID ||
		existing.AuthorizationEndpoint != incoming.AuthorizationEndpoint ||
		existing.TokenEndpoint != incoming.TokenEndpoint ||
		existing.JWKSURI != incoming.JWKSURI ||
		existing.SAMLSSOURL != incoming.SAMLSSOURL ||
		existing.SAMLEntityID != incoming.SAMLEntityID ||
		!slices.Equal(existing.SAMLSigningCertificates, incoming.SAMLSigningCertificates)
}
