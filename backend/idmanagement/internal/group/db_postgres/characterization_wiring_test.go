package db_postgres

import (
	"github.com/ambi/idmagic/backend/audit"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

// 特性化テストの操作列と期待値を固定したまま、現在の必須依存を組み立てる。
func NewGroupImportRowCommitter(pool sharedpg.DB, quotasInTx func(pgx.Tx) tenantports.QuotaRepository,
	enqueuersInTx func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer, emit func(spec.DomainEvent),
) GroupImportRowCommitter {
	return NewGroupImportCommitter(pool, quotasInTx, enqueuersInTx, emit, audit.RecordWriterInTx)
}
