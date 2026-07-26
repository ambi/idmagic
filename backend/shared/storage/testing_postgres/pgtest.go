// Package pgtest は per-context postgres アダプタのテストが共通で使う
// embedded-postgres ハーネスを提供する (wi-172)。各 context の postgres
// テストパッケージは自身の TestMain から Main を呼び、DB 依存テストは
// Require で利用可否を確認する。embedded-postgres を起動できない環境
// (ネットワーク遮断された CI 等) ではテストをスキップしグリーンを維持する。
package testing_postgres

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool は Main が起動した embedded-postgres への共有接続プール。
// 起動できなかった場合は nil のままとなり、Require がテストをスキップする。
var Pool *pgxpool.Pool

// Main は embedded-postgres を起動し infra/schema/postgres.sql を投入した上で
// m.Run() を実行する。呼び出し側の TestMain から os.Exit(pgtest.Main(m)) の形で使う。
func Main(m *testing.M) int {
	pool, cleanup := start()
	Pool = pool
	defer cleanup()
	return m.Run()
}

func start() (*pgxpool.Pool, func()) {
	noop := func() {}
	runtimePath, err := os.MkdirTemp("", "idmagic-pgtest-*")
	if err != nil {
		warn("cannot create isolated runtime directory", err)
		return nil, noop
	}
	removeRuntimePath := func() {
		if err := os.RemoveAll(runtimePath); err != nil {
			fmt.Fprintf(os.Stderr, "pgtest: cannot remove runtime directory %s: %v\n", runtimePath, err)
		}
	}

	port, err := freePort(context.Background())
	if err != nil {
		warn("cannot allocate port", err)
		removeRuntimePath()
		return nil, noop
	}
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(port).
			// `go test ./...` runs package test binaries concurrently. The library's
			// default runtime/data path is shared across processes, so one package
			// can otherwise remove another package's live PostgreSQL files.
			RuntimePath(runtimePath).
			// The immutable binaries are safe to share and expensive to extract.
			// Keep only mutable runtime/data files in the isolated directory.
			BinariesPath(postgresBinariesPath()).
			Logger(nil).
			StartTimeout(90 * time.Second),
	)
	if err := pg.Start(); err != nil {
		warn("embedded-postgres unavailable", err)
		removeRuntimePath()
		return nil, noop
	}
	stopAndRemove := func() {
		if err := pg.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "pgtest: cannot stop embedded-postgres: %v; preserving runtime directory %s\n", err, runtimePath)
			return
		}
		removeRuntimePath()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		warn("parse config failed", err)
		return nil, stopAndRemove
	}
	// 本番の Open と同じく uuid 列を string で扱えるよう codec を登録する (ADR-084)。
	// sharedpg.RegisterUUIDAsText と同一ロジックをここに持つ (import cycle 回避のため
	// pgtest は sharedpg に依存しない、wi-172)。
	poolConfig.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "uuid",
			OID:   pgtype.UUIDOID,
			Codec: pgtype.TextCodec{},
		})
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		warn("connect failed", err)
		return nil, stopAndRemove
	}
	if err := loadSchema(ctx, pool); err != nil {
		warn("schema load failed", err)
		pool.Close()
		return nil, stopAndRemove
	}
	return pool, func() {
		pool.Close()
		stopAndRemove()
	}
}

func warn(msg string, err error) {
	fmt.Fprintf(os.Stderr, "pgtest: %s: %v; skipping DB tests\n", msg, err)
}

func loadSchema(ctx context.Context, pool *pgxpool.Pool) error {
	sql, err := os.ReadFile(schemaPath())
	if err != nil {
		return err
	}
	// pgx は引数なしの Exec を simple query protocol で送るため、
	// セミコロン区切りの複数ステートメントをまとめて実行できる。
	_, err = pool.Exec(ctx, string(sql))
	return err
}

// schemaPath は呼び出し元パッケージの深さに依存しないよう、本ファイル自身の
// 位置からリポジトリルートの infra/schema/postgres.sql を解決する。
func schemaPath() string {
	_, file, _, _ := runtime.Caller(0)
	// backend/shared/storage/testing_postgres/pgtest.go からリポジトリルートまでは 4 階層。
	// (testing_postgres → storage → shared → backend → <root>)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "infra", "schema", "postgres.sql")
}

func postgresBinariesPath() string {
	cachePath, err := os.UserCacheDir()
	if err != nil {
		cachePath = os.TempDir()
	}
	return filepath.Join(
		cachePath,
		"idmagic",
		"embedded-postgres",
		string(embeddedpostgres.V18),
		runtime.GOOS+"-"+runtime.GOARCH,
	)
}

func freePort(ctx context.Context) (uint32, error) {
	l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", l.Addr())
	}
	// addr.Port は net.Listen("tcp", ...) が割り当てる OS 選択のエフェメラルポートで、
	// 常に [0, 65535] に収まる (net パッケージの不変条件)。
	return uint32(addr.Port), nil //nolint:gosec // G115: bounded by TCP port range, see above
}

// Require は DB を利用できない環境でテストをスキップし、利用できる場合は
// 共有プールを返す (*pgxpool.Pool は呼び出し側の DB interface を構造的に満たす)。
func Require(tb testing.TB) *pgxpool.Pool {
	tb.Helper()
	if Pool == nil {
		tb.Skip("embedded-postgres unavailable; skipping DB-backed test")
	}
	return Pool
}

// Now は TIMESTAMPTZ の分解能 (マイクロ秒) に切り捨てた現在時刻を返す。
// PostgreSQL は保存時にナノ秒を落とすため、time.Now() をそのまま期待値に
// 使うと read-back との比較が Linux (ナノ秒分解能) でのみ失敗する。
// DB に書き込む時刻をテストで組み立てる際は必ずこれを使う。
func Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
