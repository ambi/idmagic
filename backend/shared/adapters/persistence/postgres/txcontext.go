package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txContextKey struct{}

// WithTx returns a context carrying tx. Repositories built against the same
// pool observe this transaction through TxFromContext instead of falling
// back to their own pool connection. This is how Runner.Run lets
// independent context-owned repositories and the event log recorder share
// one PostgreSQL transaction (ADR-094, wi-184 T003) without any change to
// their port interfaces.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext returns the pgx.Tx stored by WithTx, if any.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}
