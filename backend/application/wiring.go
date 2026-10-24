package application

import (
	"github.com/ambi/idmagic/backend/application/internal/db_memory"
	"github.com/ambi/idmagic/backend/application/internal/db_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ProtocolLinkers は組み立て地点が各所有者から結ぶ、同一 Tx の結び付け操作である。
type ProtocolLinkers = db_postgres.ProtocolLinkers

// NewMemoryModule は Application が所有する保存先をメモリで作る。外向きのプロビジョニングの通知は、
// 割り当ての保存先を読む Provisioning から組み立て地点が設定する。
func NewMemoryModule() Module {
	return Module{
		Repo:           db_memory.NewApplicationRepository(),
		IconStore:      db_memory.NewApplicationIconStore(),
		AssignmentRepo: db_memory.NewApplicationAssignmentRepository(),
		OrderingRepo:   db_memory.NewApplicationOrderingRepository(),
		CategoryRepo:   db_memory.NewApplicationCategoryRepository(),
	}
}

// NewPostgresModule は Application が所有する保存先を PostgreSQL で作る。外向きのプロビジョニングの
// 通知は、割り当ての保存先を読む Provisioning から組み立て地点が設定する。
func NewPostgresModule(pool postgres.DB, linkers ProtocolLinkers) Module {
	return Module{
		Repo:           db_postgres.NewApplicationRepository(pool, linkers),
		IconStore:      &db_postgres.ApplicationIconStore{Pool: pool},
		AssignmentRepo: &db_postgres.ApplicationAssignmentRepository{Pool: pool},
		OrderingRepo:   &db_postgres.ApplicationOrderingRepository{Pool: pool},
		CategoryRepo:   &db_postgres.ApplicationCategoryRepository{Pool: pool},
	}
}
