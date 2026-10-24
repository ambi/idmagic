package sourcing

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	directorywrites "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	directorymemory "github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	directorypostgres "github.com/ambi/idmagic/backend/sourcing/internal/directory/db_postgres"
	directoryports "github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/source_idmanagement"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/jackc/pgx/v5"
)

func NewMemoryModule() Module {
	correlations := directorymemory.NewCorrelationRepository()
	return Module{
		ScimRepo: db_memory.NewScimRepository(), connectors: directorymemory.NewConnectorRepository(),
		correlations: correlations, memoryCorrelations: correlations, runs: directorymemory.NewIngestionRunRepository(),
		computers: directorymemory.NewComputerRepository(),
	}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		ScimRepo: &db_postgres.ScimRepository{Pool: pool}, begin: pool.Begin,
		connectors:   &directorypostgres.ConnectorRepository{Pool: pool},
		correlations: &directorypostgres.CorrelationRepository{Pool: pool},
		runs:         &directorypostgres.IngestionRunRepository{Pool: pool},
		computers:    &directorypostgres.ComputerRepository{Pool: pool},
	}
}

// UserOwnershipGuard は、SCIM か Directory Connector の対応を持つ User を取り込み元の管理下とする。
func (m Module) UserOwnershipGuard() userports.UserSourceOwnershipGuard {
	return sourceOwnership{scim: source_idmanagement.UserOwnershipGuard{Repository: m.ScimRepo}, directory: m.correlations}
}

type sourceOwnership struct {
	scim      userports.UserSourceOwnershipGuard
	directory directoryports.CorrelationRepository
}

func (o sourceOwnership) SourceManagedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error) {
	managed, err := o.scim.SourceManagedUserIDs(ctx, tenantID, userIDs)
	if err != nil || o.directory == nil {
		return managed, err
	}
	correlated, err := o.directory.CorrelatedUserIDs(ctx, tenantID, userIDs)
	if err != nil {
		return nil, err
	}
	for id := range correlated {
		managed[id] = true
	}
	return managed, nil
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
	return m.inTransaction(ctx, func(tx pgx.Tx) error {
		if tx == nil {
			return create(nil, m.ScimRepo)
		}
		return create(tx, &db_postgres.ScimRepository{Pool: postgres.BindTransaction(tx)})
	})
}

// CreateDirectoryUserInTransaction は、所有者の作成と Directory Connector の対応を同じ Tx で確定する。
// メモリの対応には I/O と保存の失敗がなく、tx は nil になる。
func (m Module) CreateDirectoryUserInTransaction(ctx context.Context, create func(pgx.Tx, directorywrites.CorrelationWriter) error) error {
	return m.inTransaction(ctx, func(tx pgx.Tx) error {
		if tx == nil {
			return create(nil, m.memoryCorrelations)
		}
		return create(tx, &directorypostgres.CorrelationRepository{Pool: postgres.BindTransaction(tx)})
	})
}

func (m Module) inTransaction(ctx context.Context, body func(pgx.Tx) error) error {
	if m.begin == nil {
		return body(nil)
	}
	tx, err := m.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := body(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
