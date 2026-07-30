package usecases

import (
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
)

func baseConnection(status federationdomain.ConnectionStatus) federationdomain.IdentityProviderConnection {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	return federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: "tenant-a", DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: status,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token",
		JWKSURI:               "https://accounts.example.com/jwks",
		ClaimMapping:          federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy:         federationdomain.LinkingNone,
		CreatedAt:             now, UpdatedAt: now,
	}
}

// RED (usecase: UpdateIdentityProviderConnection, ADR-149 §編集時の自動デグレード):
// display_name のみの変更では Active を保持する。
func TestResolveUpdatedStatusPreservesActiveOnNonTrustChange(t *testing.T) {
	existing := baseConnection(federationdomain.ConnectionActive)
	incoming := existing
	incoming.DisplayName = "Google Workspace"
	got := ResolveUpdatedStatus(existing, incoming)
	if got != federationdomain.ConnectionActive {
		t.Fatalf("status=%q, want active preserved", got)
	}
}

// RED: issuer (trust source) の変更は Active から Disabled へ落とす。
func TestResolveUpdatedStatusDegradesOnTrustChange(t *testing.T) {
	existing := baseConnection(federationdomain.ConnectionActive)
	incoming := existing
	incoming.Issuer = "https://other.example.com"
	got := ResolveUpdatedStatus(existing, incoming)
	if got != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q, want disabled after trust source change", got)
	}
}

// RED: SAML 署名証明書一覧の変更も trust source 変更として扱う。
func TestResolveUpdatedStatusDegradesOnCertificateChange(t *testing.T) {
	existing := baseConnection(federationdomain.ConnectionActive)
	existing.Protocol = federationdomain.ProtocolSAML
	existing.SAMLSigningCertificates = []string{"cert-a"}
	incoming := existing
	incoming.SAMLSigningCertificates = []string{"cert-b"}
	got := ResolveUpdatedStatus(existing, incoming)
	if got != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q, want disabled after certificate change", got)
	}
}

// RED: 既に Disabled な接続は trust 変更があっても Disabled のままとする (二重降格しない)。
func TestResolveUpdatedStatusKeepsDisabledOnTrustChange(t *testing.T) {
	existing := baseConnection(federationdomain.ConnectionDisabled)
	incoming := existing
	incoming.Issuer = "https://other.example.com"
	got := ResolveUpdatedStatus(existing, incoming)
	if got != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q, want disabled", got)
	}
}
