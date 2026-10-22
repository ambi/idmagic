package db_postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// BindTransaction は既存の保存先を呼び出し側の Tx へ束縛する。
// Begin は pgx の savepoint になり、内側の Commit では外側の Tx を確定しない。
func BindTransaction(tx pgx.Tx) DB { return transactionDB{Tx: tx} }

type transactionDB struct{ pgx.Tx }

func (d transactionDB) Ping(ctx context.Context) error { return d.Conn().Ping(ctx) }
