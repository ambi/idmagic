// Package signinpolicy はサインイン条件の定義、検証、評価と保存を組み立てる。
package signinpolicy

import (
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/signinpolicy/internal/db_memory"
	"github.com/ambi/idmagic/backend/signinpolicy/internal/db_postgres"
	"github.com/ambi/idmagic/backend/signinpolicy/ports"
)

type Module struct {
	ApplicationRepo ports.SignInPolicyRepository
	DefaultRepo     ports.DefaultSignInPolicyRepository
}

func NewMemoryModule() Module {
	return Module{
		ApplicationRepo: db_memory.NewSignInPolicyRepository(),
		DefaultRepo:     db_memory.NewDefaultSignInPolicyRepository(),
	}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		ApplicationRepo: &db_postgres.SignInPolicyRepository{Pool: pool},
		DefaultRepo:     &db_postgres.DefaultSignInPolicyRepository{Pool: pool},
	}
}
