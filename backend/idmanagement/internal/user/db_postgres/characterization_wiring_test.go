package db_postgres

import (
	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

// 特性化テストの操作列と期待値を固定したまま、現在の必須依存を組み立てる。
func NewUserImportRowCommitter(pool sharedpg.DB, quotasInTx func(pgx.Tx) tenantports.QuotaRepository) UserImportRowCommitter {
	return NewUserImportCommitter(pool, quotasInTx, audit.RecordWriterInTx, authentication.PasswordHistoryWriterInTx)
}
