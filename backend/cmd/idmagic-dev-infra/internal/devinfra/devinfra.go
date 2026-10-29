// Package devinfra provides the Docker-free shared infrastructure used by
// `mise run dev`: an embedded PostgreSQL server. It is development-only; production
// continues to use PostgreSQL managed outside the application process. All
// ephemeral state now lives in PostgreSQL as well, so no separate
// cache/KV endpoint is started.
package devinfra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultPostgresPort uint32 = 55432

type Config struct {
	PostgresPort uint32
	SchemaPath   string
	// SkipSchema publishes an empty database so that psqldef can apply the
	// schema itself (`mise run check-schema`).
	SkipSchema bool
	ReadyFile  string
	RuntimeDir string
	DataPath   string
	Logger     io.Writer
}

type Ready struct {
	DatabaseURL string `json:"database_url"`
}

type Runtime struct {
	postgres *embeddedpostgres.EmbeddedPostgres
	pool     *pgxpool.Pool
	ready    string
	runtime  string
}

func Start(ctx context.Context, cfg Config) (*Runtime, Ready, error) {
	if cfg.PostgresPort == 0 {
		cfg.PostgresPort = DefaultPostgresPort
	}
	if cfg.SchemaPath == "" {
		cfg.SchemaPath = filepath.Join("infra", "schema", "postgres.sql")
	}
	if cfg.RuntimeDir == "" {
		var err error
		cfg.RuntimeDir, err = os.MkdirTemp("", "idmagic-dev-postgres-*")
		if err != nil {
			return nil, Ready{}, fmt.Errorf("create runtime directory: %w", err)
		}
	}
	if cfg.Logger == nil {
		cfg.Logger = io.Discard
	}

	rt := &Runtime{ready: cfg.ReadyFile, runtime: cfg.RuntimeDir}
	fail := func(err error) (*Runtime, Ready, error) {
		_ = rt.Close()
		return nil, Ready{}, err
	}

	dataPath := cfg.DataPath
	if dataPath == "" {
		dataPath = filepath.Join(cfg.RuntimeDir, "data")
	}
	postgresConfig := embeddedpostgres.DefaultConfig().
		Port(cfg.PostgresPort).
		Database("idmagic").
		Username("idmagic").
		Password("idmagic").
		Storage(embeddedpostgres.Storage{
			CacheDir: postgresBinaryDir(),
			WorkDir:  filepath.Join(cfg.RuntimeDir, "runtime"),
			DataDir:  dataPath,
		}).
		StartParameters(map[string]string{
			// The development cluster is recreated logically on every start, so
			// durability settings only slow the local feedback loop.
			"fsync":              "off",
			"full_page_writes":   "off",
			"shared_buffers":     "16MB",
			"synchronous_commit": "off",
		}).
		Logger(cfg.Logger).
		StartTimeout(90 * time.Second)
	pg := embeddedpostgres.NewDatabase(postgresConfig)
	if err := pg.Start(); err != nil {
		return fail(fmt.Errorf("start embedded PostgreSQL: %w", err))
	}
	rt.postgres = pg

	ready := Ready{
		DatabaseURL: fmt.Sprintf("postgres://idmagic:idmagic@127.0.0.1:%d/idmagic?sslmode=disable", cfg.PostgresPort),
	}
	pool, err := pgxpool.New(ctx, ready.DatabaseURL)
	if err != nil {
		return fail(fmt.Errorf("connect embedded PostgreSQL: %w", err))
	}
	rt.pool = pool
	if cfg.DataPath != "" {
		if err := resetSchema(ctx, pool); err != nil {
			return fail(err)
		}
	}
	if !cfg.SkipSchema {
		if err := applySchema(ctx, pool, cfg.SchemaPath); err != nil {
			return fail(err)
		}
	}
	if cfg.ReadyFile != "" {
		if err := writeReadyFile(cfg.ReadyFile, ready); err != nil {
			return fail(err)
		}
	}
	return rt, ready, nil
}

func resetSchema(ctx context.Context, pool *pgxpool.Pool) error {
	//sql:raw スキーマを作り直す DDL であり、問い合わせではない。
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		return fmt.Errorf("reset PostgreSQL schema: %w", err)
	}
	return nil
}

func applySchema(ctx context.Context, pool *pgxpool.Pool, path string) error {
	schema, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read PostgreSQL schema: %w", err)
	}
	//sql:raw infra/schema/postgres.sql の DDL を適用する。
	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		return fmt.Errorf("apply PostgreSQL schema: %w", err)
	}
	return nil
}

func writeReadyFile(path string, ready Ready) error {
	payload, err := json.Marshal(ready)
	if err != nil {
		return fmt.Errorf("encode ready file: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o600); err != nil {
		return fmt.Errorf("write ready file: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("publish ready file: %w", err)
	}
	return nil
}

func postgresBinaryDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".embedded-postgres-go", "v2")
}

func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	if r.ready != "" {
		_ = os.Remove(r.ready)
		_ = os.Remove(r.ready + ".tmp")
	}
	if r.pool != nil {
		r.pool.Close()
	}
	var closeErr error
	if r.postgres != nil {
		closeErr = r.postgres.Stop()
	}
	if r.runtime != "" {
		if err := os.RemoveAll(r.runtime); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	return closeErr
}
