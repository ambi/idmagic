package devinfra

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs"
	jobspostgres "github.com/ambi/idmagic/backend/jobs/db_postgres"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

func TestWriteReadyFilePublishesDatabaseURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready.json")
	want := Ready{DatabaseURL: "postgres://example"}
	if err := writeReadyFile(path, want); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Ready
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ready=%+v, want %+v", got, want)
	}
}

func TestApplySchemaRejectsMissingFile(t *testing.T) {
	err := applySchema(t.Context(), nil, filepath.Join(t.TempDir(), "missing.sql"))
	if err == nil {
		t.Fatal("missing schema should fail")
	}
}

func TestRepairIncompletePostgresExtractionForcesReextract(t *testing.T) {
	binariesPath := t.TempDir()
	pgCtlPath := filepath.Join(binariesPath, "bin", "pg_ctl")
	if err := os.MkdirAll(filepath.Dir(pgCtlPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pgCtlPath, []byte("partial extraction"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := repairIncompletePostgresExtraction(binariesPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pgCtlPath); !os.IsNotExist(err) {
		t.Fatalf("pg_ctl should be removed to force re-extraction, stat error=%v", err)
	}
}

// API と `worker` が同じキューを共有することは、両者が同じ投入口と同じ取得口を同じ
// データベースへ向けることで成り立つ。`dev.sh` が両者へ同じ接続先を渡すことは
// TestDevScriptStartsEveryProcessAgainstTheSharedQueue が固定する。
//
//spec:covers EX-JOBS-001-01: 組込み PostgreSQL が起動してスキーマが適用され、API の投入口で積んだ Job を `worker` の Runner が同じキューから取得して succeeded にする。
func TestEmbeddedInfrastructureSharesJobQueueWithRunner(t *testing.T) {
	postgresPort := freePort(t)
	infra, ready, err := Start(t.Context(), Config{
		PostgresPort: uint32(postgresPort),
		SchemaPath:   schemaPath(t),
		RuntimeDir:   t.TempDir(),
		Logger:       io.Discard,
	})
	if err != nil {
		t.Skipf("embedded PostgreSQL unavailable: %v", err)
	}
	t.Cleanup(func() { _ = infra.Close() })

	pool, err := sharedpg.Open(t.Context(), ready.DatabaseURL, sharedpg.DBConfig{
		MaxConns: 5, MinConns: 1, ConnectTimeout: 5 * time.Second,
		MaxConnIdleTime: time.Minute, MaxConnLifetime: time.Minute, QueryTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tenantID := "00000000-0000-4000-8000-000000000000"
	if _, err := pool.Exec(t.Context(), `INSERT INTO tenants
		(id, realm, display_name, status) VALUES ($1, 'default', 'Default', 'active')`, tenantID); err != nil {
		t.Fatal(err)
	}

	// API と同じ投入の入口を通す。レーンは種別の登録から決まり、呼び出し元は指定しない。
	repo := &jobspostgres.JobRepository{Pool: pool}
	job, err := jobsusecases.Enqueue(t.Context(), jobsusecases.EnqueueDeps{Repo: repo}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindNoopEcho, Params: json.RawMessage(`{"smoke":true}`),
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	registry := jobsusecases.NewHandlerRegistry()
	registry.Register(jobsdomain.KindNoopEcho, jobs.NoopEchoHandler)
	ctx, cancel := context.WithCancel(context.Background())
	runner := jobsusecases.NewRunner(jobsusecases.RunnerConfig{
		WorkerID: "dev-smoke", Lane: jobsdomain.LaneDefault, PollInterval: 10 * time.Millisecond, Concurrency: 1,
		LeaseDuration: time.Second, BackoffBase: time.Millisecond, BackoffCap: time.Second,
	}, jobsusecases.RunnerDeps{Repo: repo, Handlers: registry, Now: func() time.Time { return time.Now().UTC() }})
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := repo.Get(t.Context(), job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == jobsdomain.StatusSucceeded {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach succeeded", job.ID)
}

func TestPersistentClusterResetsSchemaOnStart(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "postgres-data")
	schema := schemaPath(t)

	first, firstReady, err := Start(t.Context(), Config{
		PostgresPort: uint32(freePort(t)),
		SchemaPath:   schema,
		RuntimeDir:   t.TempDir(),
		DataPath:     dataPath,
		Logger:       io.Discard,
	})
	if err != nil {
		t.Skipf("embedded PostgreSQL unavailable: %v", err)
	}
	pool, err := sharedpg.Open(t.Context(), firstReady.DatabaseURL, sharedpg.DBConfig{
		MaxConns: 1, ConnectTimeout: 5 * time.Second, QueryTimeout: 5 * time.Second,
	})
	if err != nil {
		_ = first.Close()
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO tenants
		(id, realm, display_name, status) VALUES ('00000000-0000-4000-8000-000000000000', 'default', 'Default', 'active')`); err != nil {
		pool.Close()
		_ = first.Close()
		t.Fatal(err)
	}
	pool.Close()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, secondReady, err := Start(t.Context(), Config{
		PostgresPort: uint32(freePort(t)),
		SchemaPath:   schema,
		RuntimeDir:   t.TempDir(),
		DataPath:     dataPath,
		Logger:       io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })
	pool, err = sharedpg.Open(t.Context(), secondReady.DatabaseURL, sharedpg.DBConfig{
		MaxConns: 1, ConnectTimeout: 5 * time.Second, QueryTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM tenants").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("tenant count after schema reset = %d, want 0", count)
	}
}

// `dev.sh` は準備完了ファイルが現れるまで API と UI を起動しない。失敗した起動が
// このファイルを残せば、壊れたデータベースへ向けて残りのプロセスが起動する。
//
//spec:covers EX-JOBS-001-02: ポートの確保またはスキーマの適用に失敗した起動は、エラーを返して準備完了ファイルを公開しない。
func TestStartFailureLeavesNoReadyFile(t *testing.T) {
	invalidSchema := filepath.Join(t.TempDir(), "invalid.sql")
	if err := os.WriteFile(invalidSchema, []byte("CREATE TABLE broken ("), 0o600); err != nil {
		t.Fatal(err)
	}
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	for name, cfg := range map[string]Config{
		"port already in use":   {PostgresPort: uint32(occupied.Addr().(*net.TCPAddr).Port), SchemaPath: schemaPath(t)},
		"schema fails to apply": {PostgresPort: uint32(freePort(t)), SchemaPath: invalidSchema},
	} {
		t.Run(name, func(t *testing.T) {
			cfg.ReadyFile = filepath.Join(t.TempDir(), "ready.json")
			cfg.RuntimeDir = t.TempDir()
			cfg.Logger = io.Discard
			infra, _, err := Start(t.Context(), cfg)
			if err == nil {
				_ = infra.Close()
				t.Fatal("Start succeeded, want a failure")
			}
			if _, statErr := os.Stat(cfg.ReadyFile); !os.IsNotExist(statErr) {
				t.Fatalf("ready file exists after a failed start (stat error=%v)", statErr)
			}
		})
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// schemaPath はリポジトリルートのスキーマを指す。所在を誤ると Start はスキーマを
// 読めずに失敗し、組込み PostgreSQL が使えない環境と区別できないまま skip に紛れる。
func schemaPath(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "infra", "schema", "postgres.sql")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("schema not found: %v", err)
	}
	return path
}
