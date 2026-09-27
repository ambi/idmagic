package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/testing_contract"
)

func TestRefreshPersistenceContract(t *testing.T) {
	testing_contract.RunRefresh(t, func(*testing.T) testing_contract.RefreshFixture {
		return testing_contract.RefreshFixture{Store: NewRefreshTokenStore(), TenantID: "tenant-contract", ClientID: "client-contract", UserID: "user-contract", Now: time.Now().UTC()}
	})
}
