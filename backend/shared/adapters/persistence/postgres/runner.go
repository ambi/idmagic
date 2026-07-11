package postgres

import "context"

// Runner opens one PostgreSQL transaction per command and runs fn with a
// context carrying it (WithTx/TxFromContext). Repositories and the event
// log recorder built against the same pool automatically participate, so
// fn's business-mutation writes and event_logs append commit or roll back
// together (ADR-094 EventLogAtomicWithBusinessState). fn must not perform
// Kafka/SMTP/HTTP/CSV-bulk I/O (ADR-094 decision 3): keep the transaction
// short and PostgreSQL-only.
type Runner struct {
	Pool DB
}

func (r *Runner) Run(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err = fn(WithTx(ctx, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
