package sourcing

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	directorymemory "github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	directorypostgres "github.com/ambi/idmagic/backend/sourcing/internal/directory/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/source_idmanagement"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/jackc/pgx/v5"
)

func NewMemoryModule() Module {
	return Module{ScimRepo: db_memory.NewScimRepository(), connectors: directorymemory.NewConnectorRepository()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		ScimRepo: &db_postgres.ScimRepository{Pool: pool}, begin: pool.Begin,
		connectors: &directorypostgres.ConnectorRepository{Pool: pool},
	}
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
	return usecases.NewUsecases(m.ScimRepo, users, groups, m.UserLifecycle, emit, m.Writes)
}

// CreateInTransaction は、所有者の作成と SCIM の対応表を同じ Tx で確定する。
// メモリの対応表には I/O と保存の失敗がなく、tx は nil になる。
func (m Module) CreateInTransaction(ctx context.Context, create func(pgx.Tx, ports.ScimRepository) error) error {
	if m.begin == nil {
		return create(nil, m.ScimRepo)
	}
	tx, err := m.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := create(tx, &db_postgres.ScimRepository{Pool: postgres.BindTransaction(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
