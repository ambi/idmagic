package db_postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/idmanagement"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-008 REQ-SOURCING-013
func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &ConnectorRepository{Pool: db}, TenantID: tenant.ID, OtherTenantID: other.ID}
	})
}

//spec:covers REQ-SOURCING-015, REQ-SOURCING-017, REQ-SOURCING-018: PostgreSQL の保存先が、対応の範囲、実行の打ち切り、バッチの一度だけの記録と件数、完了の条件を守る。
func TestIngestionPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	users := idmanagement.NewPostgresModule(db).UserRepo
	testing_contract.RunIngestion(t, testing_contract.IngestionFixture{
		Correlations: &CorrelationRepository{Pool: db}, Runs: &IngestionRunRepository{Pool: db},
		TenantID: tenant.ID, OtherTenantID: other.ID,
		NewConnector: newConnector(db),
		NewUser: func(t *testing.T, tenantID string) string {
			t.Helper()
			return pgfixtures.SeedUser(t, users, tenantID).ID
		},
	})
}

//spec:covers REQ-SOURCING-018, REQ-SOURCING-020: PostgreSQL の保存先が、コンピューターの置き換え、有効な記録の間の一意性、主体名の照会、テナントの範囲、現れなかった記録の無効化を守る。
func TestComputerPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	testing_contract.RunComputers(t, testing_contract.ComputerFixture{
		Computers: &ComputerRepository{Pool: db}, TenantID: tenant.ID, OtherTenantID: other.ID,
		NewConnector: newConnector(db),
	})
}

// newConnector は、外部キーを満たす登録待ちの Connector を作る。
func newConnector(db sharedpg.DB) func(t *testing.T, tenantID string) string {
	connectors := &ConnectorRepository{Pool: db}
	return func(t *testing.T, tenantID string) string {
		t.Helper()
		id := uuid.NewString()
		digest := sha256.Sum256([]byte(id))
		connector, err := domain.NewConnector(tenantID, id, "AD", hex.EncodeToString(digest[:]), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := connectors.Create(t.Context(), connector); err != nil {
			t.Fatal(err)
		}
		return connector.ID
	}
}
