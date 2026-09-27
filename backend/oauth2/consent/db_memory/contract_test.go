package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/consent/domain"
	"github.com/ambi/idmagic/backend/oauth2/consent/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewConsentRepository(), TenantID: "tenant-contract", Consent: &domain.Consent{UserID: "user-contract", ClientID: "client-contract", Scopes: []string{"openid"}, State: domain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour)}}
	})
}
