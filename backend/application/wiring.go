package application

import (
	"github.com/ambi/idmagic/backend/application/internal/db_memory"
	"github.com/ambi/idmagic/backend/application/internal/db_postgres"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// NewMemoryModule は Application が所有する保存先をメモリで作る。外向きのプロビジョニングの通知は、
// 割り当ての保存先を読む Provisioning から組み立て地点が設定する。
func NewMemoryModule() Module {
	return Module{
		Repo:                    db_memory.NewApplicationRepository(),
		IconStore:               db_memory.NewApplicationIconStore(),
		AssignmentRepo:          db_memory.NewApplicationAssignmentRepository(),
		OrderingRepo:            db_memory.NewApplicationOrderingRepository(),
		CategoryRepo:            db_memory.NewApplicationCategoryRepository(),
		SignInPolicyRepo:        db_memory.NewSignInPolicyRepository(),
		DefaultSignInPolicyRepo: db_memory.NewDefaultSignInPolicyRepository(),
	}
}

// NewPostgresModule は Application が所有する保存先を PostgreSQL で作る。外向きのプロビジョニングの
// 通知は、割り当ての保存先を読む Provisioning から組み立て地点が設定する。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		Repo:                    &db_postgres.ApplicationRepository{Pool: pool},
		IconStore:               &db_postgres.ApplicationIconStore{Pool: pool},
		AssignmentRepo:          &db_postgres.ApplicationAssignmentRepository{Pool: pool},
		OrderingRepo:            &db_postgres.ApplicationOrderingRepository{Pool: pool},
		CategoryRepo:            &db_postgres.ApplicationCategoryRepository{Pool: pool},
		SignInPolicyRepo:        &db_postgres.SignInPolicyRepository{Pool: pool},
		DefaultSignInPolicyRepo: &db_postgres.DefaultSignInPolicyRepository{Pool: pool},
	}
}
