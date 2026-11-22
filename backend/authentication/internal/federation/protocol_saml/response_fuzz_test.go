package protocol_saml

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
)

type fuzzReplayStore struct{}

func (fuzzReplayStore) Reserve(context.Context, string, string, time.Time) (bool, error) {
	return true, nil
}

func FuzzUpstreamSAMLResponse(f *testing.F) {
	f.Add([]byte(`<Response ID="_duplicate"><Assertion ID="_duplicate"/></Response>`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY e SYSTEM "file:///etc/passwd">]><Response>&e;</Response>`))
	f.Add([]byte(`<Response><Assertion>`))

	connection := federationdomain.IdentityProviderConnection{
		ID: "provider", TenantID: "tenant", DisplayName: "Provider",
		Protocol: federationdomain.ProtocolSAML, Status: federationdomain.ConnectionActive,
		Issuer: "https://idmagic.example", SAMLEntityID: "https://idp.example",
		SAMLSSOURL: "https://idp.example/sso", SAMLSigningCertificates: []string{"invalid"},
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "NameID", Username: "username"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	attempt := federationdomain.FederatedLoginAttempt{
		State: "state", TenantID: "tenant", ProviderID: "provider",
		Protocol: federationdomain.ProtocolSAML, RequestID: "_request",
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}
	f.Fuzz(func(t *testing.T, xmlInput []byte) {
		if len(xmlInput) > maxSAMLResponseBytes {
			return
		}
		encoded := base64.StdEncoding.EncodeToString(xmlInput)
		_, _ = ValidateResponse(
			context.Background(), connection, attempt, encoded,
			"https://idmagic.example/callback", time.Now(), fuzzReplayStore{},
		)
	})
}
