package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/testing_contract"
)

func TestDenylistPersistenceContract(t *testing.T) {
	testing_contract.RunDenylist(t, func(*testing.T) testing_contract.DenylistFixture {
		return testing_contract.DenylistFixture{Denylist: NewAccessTokenDenylist(), Context: context.Background(), Now: time.Now().UTC()}
	})
}
