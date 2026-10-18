package sourcing

import (
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/source_idmanagement"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
)

func NewMemoryModule() Module {
	return Module{ScimRepo: db_memory.NewScimRepository()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{ScimRepo: &db_postgres.ScimRepository{Pool: pool}}
}

func (m Module) UserOwnershipGuard() userports.UserSourceOwnershipGuard {
	return source_idmanagement.UserOwnershipGuard{Repository: m.ScimRepo}
}

// SCIMUserLookup は任意の由来表示を組み立てる。未配線なら照会も未配線のままにする。
func (m Module) SCIMUserLookup() userports.SCIMUserLookup {
	if m.ScimRepo == nil {
		return nil
	}
	return source_idmanagement.UserLookup{Repository: m.ScimRepo}
}

func (m Module) GroupOwnershipGuard() groupports.GroupSourceOwnershipGuard {
	return source_idmanagement.GroupOwnershipGuard{Repository: m.ScimRepo}
}

// SCIMGroupLookup は任意の Group の由来表示を組み立てる。
func (m Module) SCIMGroupLookup() groupports.SCIMGroupLookup {
	if m.ScimRepo == nil {
		return nil
	}
	return source_idmanagement.GroupLookup{Repository: m.ScimRepo}
}

// Usecases は HTTP と同じライフサイクル操作を使う SCIM の呼び出し口を組み立てる。
func (m Module) Usecases(users userports.UserRepository, groups groupports.GroupRepository, emit func(spec.DomainEvent)) *usecases.Usecases {
	return usecases.NewUsecases(m.ScimRepo, users, groups, m.UserLifecycle, emit)
}
