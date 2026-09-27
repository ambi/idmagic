package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/client/testing_contract"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewClientRepository(), Client: &domain.OAuth2Client{TenantID: "tenant-contract", ClientID: "client-contract", ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://client.example/cb"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}, TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic, CreatedAt: now, UpdatedAt: now}}
	})
}
