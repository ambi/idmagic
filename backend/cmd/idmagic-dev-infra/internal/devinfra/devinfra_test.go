package devinfra

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs"
	jobspostgres "github.com/ambi/idmagic/backend/jobs/db_postgres"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

func TestWriteReadyFilePublishesURLs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready.json")
	want := Ready{DatabaseURL: "postgres://example", ValkeyURL: "valkey://example"}
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

func TestEmbeddedInfrastructureSharesJobQueueWithRunner(t *testing.T) {
	postgresPort := freePort(t)
	valkeyPort := freePort(t)
	runtime, ready, err := Start(t.Context(), Config{
		PostgresPort: uint32(postgresPort),
		ValkeyPort:   valkeyPort,
		SchemaPath:   filepath.Join("..", "..", "infra", "schema", "postgres.sql"),
		RuntimeDir:   t.TempDir(),
		Logger:       io.Discard,
	})
	if err != nil {
		t.Skipf("embedded PostgreSQL unavailable: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })

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

	repo := &jobspostgres.JobRepository{Pool: pool}
	job, _, err := repo.Enqueue(t.Context(), jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindNoopEcho,
		Params: json.RawMessage(`{"smoke":true}`), Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	registry := jobsusecases.NewHandlerRegistry()
	registry.Register(jobsdomain.KindNoopEcho, jobs.NoopEchoHandler)
	ctx, cancel := context.WithCancel(context.Background())
	runner := jobsusecases.NewRunner(jobsusecases.RunnerConfig{
		WorkerID: "dev-smoke", PollInterval: 10 * time.Millisecond, Concurrency: 1,
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
	schemaPath := filepath.Join("..", "..", "infra", "schema", "postgres.sql")

	first, firstReady, err := Start(t.Context(), Config{
		PostgresPort: uint32(freePort(t)),
		ValkeyPort:   freePort(t),
		SchemaPath:   schemaPath,
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
		ValkeyPort:   freePort(t),
		SchemaPath:   schemaPath,
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

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
