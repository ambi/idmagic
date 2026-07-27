package protocol_saml

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
)

type fuzzReplayStore struct{}

func (fuzzReplayStore) Reserve(context.Context, string, string, time.Time) (bool, error) {
	return true, nil
}

func FuzzUpstreamSAMLResponse(f *testing.F) {
	f.Add([]byte(`<Response ID="_duplicate"><Assertion ID="_duplicate"/></Response>`))
	f.Add([]byte(`<!DOCTYPE x [<!ENTITY e SYSTEM "file:///etc/passwd">]><Response>&e;</Response>`))
	f.Add([]byte(`<Response><Assertion>`))

	connection := domain.IdentityProviderConnection{
		ID: "provider", TenantID: "tenant", DisplayName: "Provider",
		Protocol: domain.ProtocolSAML, Status: domain.ConnectionActive,
		Issuer: "https://idmagic.example", SAMLEntityID: "https://idp.example",
		SAMLSSOURL: "https://idp.example/sso", SAMLSigningCertificates: []string{"invalid"},
		ClaimMapping:  domain.ClaimMapping{Subject: "NameID", Username: "username"},
		LinkingPolicy: domain.LinkingNone, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	attempt := domain.FederatedLoginAttempt{
		State: "state", TenantID: "tenant", ProviderID: "provider",
		Protocol: domain.ProtocolSAML, RequestID: "_request",
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
