package domain

import (
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestClientValidateRequiresGrantTypes(t *testing.T) {
	c := OAuth2Client{
		ClientID:                 "demo",
		ClientType:               spec.ClientConfidential,
		RedirectURIs:             []string{"https://app.example.com/cb"},
		GrantTypes:               nil,
		TokenEndpointAuthMethod:  AuthMethodClientSecretBasic,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              FapiNone,
		CreatedAt:                time.Now().UTC(),
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty grant_types")
	}
}
