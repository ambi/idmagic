package db_postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/resilience"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestResilientDBQueryRowDoesNotCancelBeforeScan(t *testing.T) {
	pool := &contextCheckingDB{}
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)

	var value int
	if err := db.QueryRow(context.Background(), "SELECT 42").Scan(&value); err != nil {
		t.Fatalf("QueryRow().Scan() error = %v, want nil", err)
	}
	if value != 42 {
		t.Fatalf("value = %d, want 42", value)
	}
}

func TestResilientDBQueryDoesNotCancelBeforeRowsClose(t *testing.T) {
	pool := &contextCheckingDB{}
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)

	rows, err := db.Query(context.Background(), "SELECT 42")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("Rows.Next() = false, error = %v, want one row", rows.Err())
	}
	var value int
	if err := rows.Scan(&value); err != nil {
		t.Fatalf("Rows.Scan() error = %v, want nil", err)
	}
	if value != 42 {
		t.Fatalf("value = %d, want 42", value)
	}
}

func TestResilientDBQueryRowReportsScanFailureToCircuitBreaker(t *testing.T) {
	scanErr := errors.New("scan failed")
	pool := &contextCheckingDB{rowErr: scanErr}
	breaker := resilience.NewCircuitBreaker(resilience.Settings{
		Name:             "postgres-scan-test",
		FailureThreshold: 1,
		MinRequests:      1,
		Cooldown:         time.Hour,
	})
	db := NewResilientDB(pool, breaker, time.Second)

	if err := db.QueryRow(context.Background(), "SELECT 42").Scan(new(int)); !errors.Is(err, scanErr) {
		t.Fatalf("first Scan() error = %v, want %v", err, scanErr)
	}
	if err := db.QueryRow(context.Background(), "SELECT 42").Scan(new(int)); !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Fatalf("second Scan() error = %v, want circuit open", err)
	}
	if pool.queryRowCalls != 1 {
		t.Fatalf("underlying QueryRow calls = %d, want 1", pool.queryRowCalls)
	}
}

func TestResilientDBQueryRowDoesNotTripCircuitBreakerOnNoRows(t *testing.T) {
	pool := &contextCheckingDB{rowErr: pgx.ErrNoRows}
	breaker := resilience.NewCircuitBreaker(resilience.Settings{
		Name:             "postgres-no-rows-test",
		FailureThreshold: 1,
		MinRequests:      1,
		Cooldown:         time.Hour,
	})
	db := NewResilientDB(pool, breaker, time.Second)

	for attempt := 1; attempt <= 2; attempt++ {
		if err := db.QueryRow(context.Background(), "SELECT 42").Scan(new(int)); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("Scan() attempt %d error = %v, want no rows", attempt, err)
		}
	}
	if pool.queryRowCalls != 2 {
		t.Fatalf("underlying QueryRow calls = %d, want 2", pool.queryRowCalls)
	}
}

// System scenario: PostgreSQLクエリの期限は結果読取完了まで維持される。
func TestResilientDBQueryRowKeepsTimeoutContextUntilScan(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)

	var slept any
	var value int
	err := db.QueryRow(context.Background(), "SELECT pg_sleep(0.05), 42").Scan(&slept, &value)
	if err != nil {
		t.Fatalf("QueryRow().Scan() error = %v, want nil", err)
	}
	if value != 42 {
		t.Fatalf("value = %d, want 42", value)
	}
}

// System scenario: PostgreSQLクエリの期限は結果読取完了まで維持される。
func TestResilientDBQueryKeepsTimeoutContextUntilRowsClose(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)

	rows, err := db.Query(context.Background(), "SELECT pg_sleep(0.05), value FROM generate_series(1, 2) AS value")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	var got []int
	for rows.Next() {
		var slept any
		var value int
		if err := rows.Scan(&slept, &value); err != nil {
			t.Fatalf("Rows.Scan() error = %v", err)
		}
		got = append(got, value)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows.Err() = %v, want nil", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("values = %v, want [1 2]", got)
	}
}

// System scenario extension: 結果読取中にquery timeoutの期限へ到達する。
func TestResilientDBQueryRowReportsDeadlineExceeded(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), 20*time.Millisecond)

	var slept any
	var value int
	err := db.QueryRow(context.Background(), "SELECT pg_sleep(0.1), 42").Scan(&slept, &value)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("QueryRow().Scan() error = %v, want context deadline exceeded", err)
	}
}

func testCircuitBreaker() *resilience.CircuitBreaker {
	return resilience.NewCircuitBreaker(resilience.Settings{
		Name:             "postgres-test",
		FailureThreshold: 1,
		MinRequests:      100,
	})
}

type contextCheckingDB struct {
	rowErr        error
	queryRowCalls int
}

func (db *contextCheckingDB) Query(ctx context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return &contextCheckingRows{ctx: ctx}, nil
}

func (db *contextCheckingDB) QueryRow(ctx context.Context, _ string, _ ...any) pgx.Row {
	db.queryRowCalls++
	return &contextCheckingRow{ctx: ctx, err: db.rowErr}
}

func (*contextCheckingDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (*contextCheckingDB) Begin(context.Context) (pgx.Tx, error) { return nil, nil }
func (*contextCheckingDB) Ping(context.Context) error            { return nil }

type contextCheckingRow struct {
	ctx context.Context
	err error
}

func (r *contextCheckingRow) Scan(dest ...any) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if r.err != nil {
		return r.err
	}
	*dest[0].(*int) = 42
	return nil
}

type contextCheckingRows struct {
	ctx     context.Context
	err     error
	visited bool
}

func (r *contextCheckingRows) Close() {}
func (r *contextCheckingRows) Err() error {
	return r.err
}
func (*contextCheckingRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (*contextCheckingRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *contextCheckingRows) Next() bool {
	if err := r.ctx.Err(); err != nil {
		r.err = err
		return false
	}
	if r.visited {
		return false
	}
	r.visited = true
	return true
}

func (r *contextCheckingRows) Scan(dest ...any) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	*dest[0].(*int) = 42
	return nil
}
func (*contextCheckingRows) Values() ([]any, error) { return []any{42}, nil }
func (*contextCheckingRows) RawValues() [][]byte    { return nil }
func (*contextCheckingRows) Conn() *pgx.Conn        { return nil }
