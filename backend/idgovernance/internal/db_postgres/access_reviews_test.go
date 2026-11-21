package db_postgres_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idgovernance/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	fixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

//spec:covers REQ-IDGOVERNANCE-021, REQ-IDGOVERNANCE-022, REQ-IDGOVERNANCE-025: PostgreSQL のロックとロールバックで開始の原子性と同時回答とテナント分離を保つ。
func TestAccessReviewPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	a := fixtures.SeedTenant(t, db)
	b := fixtures.SeedTenant(t, db)
	// 見直しの保存先も本番と同じモジュールの組み立てから取得する。
	module := idgovernance.NewPostgresModule(db, nil, nil)
	testing_contract.RunAccessReviews(t, testing_contract.AccessReviewFixture{Plans: module.AccessReviewPlans, Reviews: module.AccessReviews, TenantA: a.ID, TenantB: b.ID})
}
