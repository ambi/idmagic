// Package postgres: 永続化アダプタの PostgreSQL 実装。
// 接続を base.go に置き、リポジトリ実装は
// 境界づけられたコンテキスト単位でファイル分割している (tenants.go / clients.go ...)。
package postgres

import (
	"context"
	"time"

	"github.com/ambi/idmagic/internal/shared/resilience"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig は PostgreSQL 接続プールとレジリエンスの設定を集約する。
type DBConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnIdleTime time.Duration
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

// DB は pgxpool.Pool の主要なクエリメソッドを抽象化するインターフェース。
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
}

// ResilientDB は DB インターフェースを実装し、サーキットブレイカーとタイムアウトを提供する。
type ResilientDB struct {
	pool    *pgxpool.Pool
	cb      *resilience.CircuitBreaker
	timeout time.Duration
}

func NewResilientDB(pool *pgxpool.Pool, cb *resilience.CircuitBreaker, timeout time.Duration) *ResilientDB {
	return &ResilientDB{
		pool:    pool,
		cb:      cb,
		timeout: timeout,
	}
}

func (db *ResilientDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *ResilientDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	var rows pgx.Rows
	err := db.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
		qctx, cancel := db.withTimeout(ctx)
		defer cancel()

		var qerr error
		rows, qerr = db.pool.Query(qctx, sql, args...) //nolint:sqlclosecheck // Rows are closed by repository callers
		return qerr
	})
	return rows, err
}

func (db *ResilientDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	var row pgx.Row
	err := db.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
		// QueryRow 自体は即座にエラーを返さないが、接続確保などを Execute 内で行わせる。
		qctx, cancel := db.withTimeout(ctx)
		defer cancel()

		row = db.pool.QueryRow(qctx, sql, args...)
		return nil
	})
	if err != nil {
		return &resilientRow{err: err}
	}
	return row
}

func (db *ResilientDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	var tag pgconn.CommandTag
	err := db.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
		qctx, cancel := db.withTimeout(ctx)
		defer cancel()

		var qerr error
		tag, qerr = db.pool.Exec(qctx, sql, args...)
		return qerr
	})
	return tag, err
}

func (db *ResilientDB) Begin(ctx context.Context) (pgx.Tx, error) {
	var tx pgx.Tx
	err := db.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
		qctx, cancel := db.withTimeout(ctx)
		defer cancel()

		var qerr error
		tx, qerr = db.pool.Begin(qctx)
		return qerr
	})
	return tx, err
}

func (db *ResilientDB) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if db.timeout > 0 {
		return context.WithTimeout(ctx, db.timeout)
	}
	return ctx, func() {}
}

type resilientRow struct {
	row pgx.Row
	err error
}

func (r *resilientRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return r.row.Scan(dest...)
}

// Open は指定された DSN と設定で接続プールを構築し、リトライ付きで疎通確認を行う。
func Open(ctx context.Context, databaseURL string, cfg DBConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = cfg.MaxConns
	config.MinConns = cfg.MinConns
	config.MaxConnIdleTime = cfg.MaxConnIdleTime
	config.MaxConnLifetime = cfg.MaxConnLifetime

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		RegisterUUIDAsText(conn)
		_, err := conn.Exec(ctx, "SET statement_timeout = '5s'; SET idle_in_transaction_session_timeout = '30s'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	// Exponential Backoff を用いた接続疎通確認（Ping）のリトライ
	err = resilience.RetryWithBackoff(ctx, func() error {
		pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
		defer cancel()
		return pool.Ping(pingCtx)
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// RegisterUUIDAsText は uuid OID に text codec を登録し、UUID 列を Go の string として
// read/write できるようにする (ADR-084)。内部生成 id 列は UUID 型だが Go 側では string で
// 扱うため、接続確立時にこれを登録する。エンコードは text を渡し PostgreSQL が uuid として
// 解釈する。接続プールを構築する全経路 (本番の Open と test harness) で呼ぶ。
func RegisterUUIDAsText(conn *pgx.Conn) {
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  "uuid",
		OID:   pgtype.UUIDOID,
		Codec: pgtype.TextCodec{},
	})
}

// RowScanner は pgx.Row / pgx.Rows の共通スキャンインターフェース。
type RowScanner interface{ Scan(...any) error }
