// Package txrunner declares the command transaction boundary used by
// wi-184 T003's transaction-bound mutations. It has no adapter import so
// every context's HTTP handler layer can depend on it directly.
package txrunner

import "context"

// Runner executes fn once within a single command's transaction scope. The
// concrete implementation decides what "transaction" means: a real
// PostgreSQL transaction for the postgres_valkey runtime (see
// backend/shared/adapters/persistence/postgres.Runner), or a direct
// passthrough for the memory runtime used in tests/local demo (see
// backend/shared/adapters/persistence/memory/txrunner), which has no
// durable transaction to offer.
//
// fn must not perform external I/O (Kafka/SMTP/HTTP/CSV-bulk) — ADR-094
// keeps that outside the transaction so the command commits quickly and
// external-dependency failures cannot hold a PostgreSQL connection open.
type Runner interface {
	Run(ctx context.Context, fn func(ctx context.Context) error) error
}
