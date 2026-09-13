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
	"github.com/jackc/pgx/v5/pgtype"
)

// 期限は Scan まで生きていなければならず、Scan を終えたら解放されなければならない。
// 前半だけを見る検査は cancel を呼ばない実装を通し、後半だけを見る検査は Scan の前に
// cancel する実装を通す。どちらも 1 要求では見えず、前者は接続の枯渇として、後者は
// context canceled として、負荷が上がってから現れる。
//
//spec:covers EX-SYSTEM-012-01: 単一行クエリの期限が Scan の完了まで維持され、完了と同時に解放されること。
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
	if err := pool.rowCtx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("query context error after Scan = %v, want canceled", err)
	}
}

//spec:covers EX-SYSTEM-012-01: 複数行クエリの期限が反復処理の完了まで維持され、結果を閉じると解放されること。
func TestResilientDBQueryDoesNotCancelBeforeRowsClose(t *testing.T) {
	pool := &contextCheckingDB{}
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)

	rows, err := db.Query(context.Background(), "SELECT 42")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if !rows.Next() {
		rows.Close()
		t.Fatalf("Rows.Next() = false, error = %v, want one row", rows.Err())
	}
	var value int
	if err := rows.Scan(&value); err != nil {
		rows.Close()
		t.Fatalf("Rows.Scan() error = %v, want nil", err)
	}
	if value != 42 {
		rows.Close()
		t.Fatalf("value = %d, want 42", value)
	}
	if err := pool.rowsCtx.Err(); err != nil {
		rows.Close()
		t.Fatalf("query context error before Rows.Close() = %v, want nil", err)
	}
	rows.Close()
	if err := pool.rowsCtx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("query context error after Rows.Close() = %v, want canceled", err)
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

// 該当行が無いことは正常なクエリ結果である。2 回続けて no rows を返しても、失敗率 1 件で
// 開くサーキットブレーカーが開かないことで「失敗率を増加させない」を観測する。応答だけを
// 見る検査は、no rows を失敗として数える実装を通す。その実装では、存在しない id の照会が
// 集まるだけで健全な PostgreSQL への経路が遮断される。
//
//spec:covers EX-SYSTEM-012-03: 該当行が無い単一行クエリは no rows を返し、それがサーキットブレーカーの失敗率を増加させないこと。
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

// 結果を読み終えたあとに接続が返っていることを、取得済み接続数の基準からの差で見る。
// 具体例は「結果が context canceled にならず返され、接続が解放される」と 2 つを言っており、
// 戻り値だけを見る検査は、cancel を呼ばずに接続を握り続ける実装を通す。その実装は
// 1 要求ごとに 1 本ずつ接続を漏らし、プールが尽きるまで平常時のテストには現れない。
//
//spec:covers EX-SYSTEM-012-01: 期限内に Scan を終えた単一行クエリが context canceled にならず値を返し、接続が解放されること。
func TestResilientDBQueryRowKeepsTimeoutContextUntilScan(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)
	acquired := pool.Stat().AcquiredConns()

	var slept any
	var value int
	err := db.QueryRow(context.Background(), "SELECT pg_sleep(0.05), 42").Scan(&slept, &value)
	if err != nil {
		t.Fatalf("QueryRow().Scan() error = %v, want nil", err)
	}
	if value != 42 {
		t.Fatalf("value = %d, want 42", value)
	}
	if got := pool.Stat().AcquiredConns(); got != acquired {
		t.Fatalf("acquired connections after Scan = %d, want %d", got, acquired)
	}
}

//spec:covers EX-SYSTEM-012-01: 期限内に反復処理を終えた複数行クエリがすべての行を返し、結果を閉じると接続が解放されること。
func TestResilientDBQueryKeepsTimeoutContextUntilRowsClose(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), time.Second)
	acquired := pool.Stat().AcquiredConns()

	rows, err := db.Query(context.Background(), "SELECT pg_sleep(0.05), value FROM generate_series(1, 2) AS value")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	var got []int
	for rows.Next() {
		var slept any
		var value int
		if err := rows.Scan(&slept, &value); err != nil {
			rows.Close()
			t.Fatalf("Rows.Scan() error = %v", err)
		}
		got = append(got, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatalf("Rows.Err() = %v, want nil", err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		rows.Close()
		t.Fatalf("values = %v, want [1 2]", got)
	}
	rows.Close()
	if got := pool.Stat().AcquiredConns(); got != acquired {
		t.Fatalf("acquired connections after Rows.Close() = %d, want %d", got, acquired)
	}
}

// 期限へ到達した読み取りは deadline exceeded で中断され、結果を閉じたあとは接続と
// タイムアウトの資源が解放される。前者だけを見る検査は、中断したあと接続を握り続ける
// 実装を通す。飽和しているときに起きる中断でこそ接続が返る必要がある。
//
//spec:covers EX-SYSTEM-012-02: 結果の読み取り中に期限へ到達すると deadline exceeded で中断され、結果を閉じると接続が解放されること。
func TestResilientDBQueryRowReportsDeadlineExceeded(t *testing.T) {
	pool := pgtest.Require(t)
	db := NewResilientDB(pool, testCircuitBreaker(), 20*time.Millisecond)
	acquired := pool.Stat().AcquiredConns()

	var slept any
	var value int
	err := db.QueryRow(context.Background(), "SELECT pg_sleep(0.1), 42").Scan(&slept, &value)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("QueryRow().Scan() error = %v, want context deadline exceeded", err)
	}
	if got := pool.Stat().AcquiredConns(); got != acquired {
		t.Fatalf("acquired connections after the interrupted Scan = %d, want %d", got, acquired)
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
	// アダプターが下位へ渡した context を残す。呼び出し側からは見えない期限の解放を、
	// 渡された側から観測するための唯一の足場である。
	rowCtx  context.Context
	rowsCtx context.Context
}

func (db *contextCheckingDB) Query(ctx context.Context, _ string, _ ...any) (pgx.Rows, error) {
	db.rowsCtx = ctx
	return &contextCheckingRows{ctx: ctx}, nil
}

func (db *contextCheckingDB) QueryRow(ctx context.Context, _ string, _ ...any) pgx.Row {
	db.queryRowCalls++
	db.rowCtx = ctx
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
func (*contextCheckingRows) TypeMap() *pgtype.Map   { return nil }
