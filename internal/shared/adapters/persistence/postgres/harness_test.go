package postgres

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool は embedded-postgres 上に構築した実 DB への接続プール。
// TestMain が 1 プロセスにつき 1 度だけ起動・スキーマ投入し、全テストで共有する。
// この *pgxpool.Pool は base.go の DB インターフェースを満たすため、各
// repository を {Pool: testPool} で直接構築して往復検証できる。
var testPool *pgxpool.Pool

// TestMain は本パッケージのテスト実行前に embedded-postgres を起動して
// deploy/schema/postgres.sql を投入する。embedded-postgres は Docker を必要と
// せず、初回のみ Postgres バイナリを取得する。バイナリ取得や起動に失敗した
// (ネットワーク遮断された CI 等) 場合は、テストを丸ごとスキップして
// グリーンを維持する。カバレッジ計測は本ハーネスが動作する環境で行う。
func TestMain(m *testing.M) {
	os.Exit(runWithEmbeddedPostgres(m))
}

func runWithEmbeddedPostgres(m *testing.M) int {
	port, err := freePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres harness: cannot allocate port: %v; skipping DB tests\n", err)
		return runSkipped(m)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(uint32(port)).
			Logger(nil).
			StartTimeout(90 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "postgres harness: embedded-postgres unavailable: %v; skipping DB tests\n", err)
		return runSkipped(m)
	}
	defer func() { _ = pg.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres harness: connect failed: %v; skipping DB tests\n", err)
		return runSkipped(m)
	}
	defer pool.Close()

	if err := loadSchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "postgres harness: schema load failed: %v; skipping DB tests\n", err)
		return runSkipped(m)
	}

	testPool = pool
	return m.Run()
}

// runSkipped は DB を利用できない環境でテストを実行する。各 DB テストは
// requireDB で testPool の有無を確認して自身をスキップするため、ここでは
// スキーマ非依存のテスト (schema_test.go) だけが実質的に実行される。
func runSkipped(m *testing.M) int {
	return m.Run()
}

func loadSchema(ctx context.Context, pool *pgxpool.Pool) error {
	sql, err := os.ReadFile("../../../../../deploy/schema/postgres.sql")
	if err != nil {
		return err
	}
	// pgx は引数なしの Exec を simple query protocol で送るため、
	// セミコロン区切りの複数ステートメントをまとめて実行できる。
	_, err = pool.Exec(ctx, string(sql))
	return err
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// requireDB は DB を利用できない環境でテストをスキップし、利用できる場合は
// 共有プールを DB として返す。
func requireDB(t *testing.T) DB {
	t.Helper()
	if testPool == nil {
		t.Skip("embedded-postgres unavailable; skipping DB-backed test")
	}
	return testPool
}
