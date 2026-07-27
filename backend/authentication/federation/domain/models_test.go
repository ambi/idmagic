package domain

import (
	"testing"
	"time"
)

func TestIdentityProviderConnectionValidateAndLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	connection := IdentityProviderConnection{
		ID: "google", TenantID: "tenant-a", DisplayName: "Google",
		Protocol: ProtocolOIDC, Status: ConnectionDraft,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		SecretReference:       "env:GOOGLE_CLIENT_SECRET",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token",
		JWKSURI:               "https://accounts.example.com/jwks",
		ClaimMapping:          ClaimMapping{Subject: "sub", Username: "email", Email: "email", EmailVerified: "email_verified"},
		LinkingPolicy:         LinkingVerifiedEmail, JITProvisioning: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := connection.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if err := connection.Activate(now.Add(time.Minute)); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if connection.Status != ConnectionActive {
		t.Fatalf("status=%q, want active", connection.Status)
	}
	connection.Disable(now.Add(2 * time.Minute))
	if connection.Status != ConnectionDisabled {
		t.Fatalf("status=%q, want disabled", connection.Status)
	}
}

func TestIdentityProviderConnectionRejectsUnsafeOrIncompleteTrust(t *testing.T) {
	tests := []IdentityProviderConnection{
		{
			ID: "oidc", TenantID: "tenant-a", DisplayName: "OIDC",
			Protocol: ProtocolOIDC, Status: ConnectionDraft,
			Issuer: "http://idp.example.com", ClientID: "client",
			AuthorizationEndpoint: "https://idp.example.com/auth",
			TokenEndpoint:         "https://idp.example.com/token", JWKSURI: "https://idp.example.com/jwks",
			ClaimMapping: ClaimMapping{Subject: "sub", Username: "sub"},
		},
		{
			ID: "saml", TenantID: "tenant-a", DisplayName: "SAML",
			Protocol: ProtocolSAML, Status: ConnectionDraft,
			Issuer:       "https://idp.example.com",
			SAMLEntityID: "https://idp.example.com/metadata",
			SAMLSSOURL:   "https://idp.example.com/sso",
			ClaimMapping: ClaimMapping{Subject: "NameID", Username: "mail"},
		},
	}
	for i := range tests {
		if err := tests[i].Validate(); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestFederatedIdentityValidate(t *testing.T) {
	link := FederatedIdentity{
		TenantID: "tenant-a", ProviderID: "google", ExternalSubject: "external-1",
		LocalUserID: "user-1", LinkedAt: time.Now().UTC(),
	}
	if err := link.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	link.ExternalSubject = ""
	if err := link.Validate(); err == nil {
		t.Fatal("empty external subject must be rejected")
	}
}

func TestFederatedLoginAttemptConsumeOnce(t *testing.T) {
	now := time.Now().UTC()
	attempt := FederatedLoginAttempt{
		State: "state", TenantID: "tenant-a", ProviderID: "google", Protocol: ProtocolOIDC,
		Nonce: "nonce", PKCEVerifier: "verifier", CreatedAt: now, ExpiresAt: now.Add(10 * time.Minute),
	}
	if err := attempt.Consume(now.Add(time.Minute)); err != nil {
		t.Fatalf("first Consume: %v", err)
	}
	if err := attempt.Consume(now.Add(2 * time.Minute)); err == nil {
		t.Fatal("second consume must fail")
	}
}
