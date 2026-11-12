package db_postgres

import (
	"fmt"
	"testing"
	"time"
	"uuid"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-026: PostgreSQL の行の排他で、単回配信、暗号文の除去、期限と結果の単回確定を共有する。
func TestPasswordChallengePersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	ids := []string{uuid.NewV4().String(), uuid.NewV4().String()}
	for i, id := range ids {
		c, err := domain.NewConnector(tenant.ID, id, "AD", fmt.Sprintf("%064x", i+1), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := (&ConnectorRepository{Pool: db}).Create(t.Context(), c); err != nil {
			t.Fatal(err)
		}
	}
	testing_contract.RunPasswordChallenges(t, testing_contract.PasswordFixture{Store: &PasswordChallengeStore{Pool: db}, TenantID: tenant.ID, OtherTenantID: other.ID, ConnectorID: ids[0], OtherConnectorID: ids[1]})
}
