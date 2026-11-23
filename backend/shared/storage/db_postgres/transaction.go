package db_postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// BindTransaction は既存の保存先を呼び出し側の Tx へ束縛する。
// Begin は pgx の savepoint になり、内側の Commit では外側の Tx を確定しない。
func BindTransaction(tx pgx.Tx) DB { return transactionDB{Tx: tx} }

// InTransaction は run が返す結果を、確定に成功した場合だけ呼び出し元へ返す。
func InTransaction[T any](ctx context.Context, db DB, run func(pgx.Tx) (T, error)) (result T, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	value, err := run(tx)
	if err != nil {
		return result, err
	}
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	return value, nil
}

type transactionDB struct{ pgx.Tx }

func (d transactionDB) Ping(ctx context.Context) error { return d.Conn().Ping(ctx) }
