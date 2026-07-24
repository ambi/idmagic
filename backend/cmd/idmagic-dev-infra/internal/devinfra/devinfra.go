// Package devinfra provides the Docker-free shared infrastructure used by
// `just dev`: an embedded PostgreSQL server. It is development-only; production
// continues to use PostgreSQL managed outside the application process. All
// ephemeral state now lives in PostgreSQL as well (ADR-139), so no separate
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

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultPostgresPort uint32 = 55432

type Config struct {
	PostgresPort uint32
	SchemaPath   string
	ReadyFile    string
	RuntimeDir   string
	DataPath     string
	Logger       io.Writer
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

	binariesPath := postgresBinaryDir()
	if err := repairIncompletePostgresExtraction(binariesPath); err != nil {
		return fail(err)
	}
	postgresConfig := embeddedpostgres.DefaultConfig().
		Port(cfg.PostgresPort).
		Database("idmagic").
		Username("idmagic").
		Password("idmagic").
		RuntimePath(filepath.Join(cfg.RuntimeDir, "runtime")).
		BinariesPath(binariesPath).
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
	if cfg.DataPath != "" {
		postgresConfig = postgresConfig.DataPath(cfg.DataPath)
	} else {
		postgresConfig = postgresConfig.DataPath(filepath.Join(cfg.RuntimeDir, "data"))
	}
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
	if err := applySchema(ctx, pool, cfg.SchemaPath); err != nil {
		return fail(err)
	}
	if cfg.ReadyFile != "" {
		if err := writeReadyFile(cfg.ReadyFile, ready); err != nil {
			return fail(err)
		}
	}
	return rt, ready, nil
}

func resetSchema(ctx context.Context, pool *pgxpool.Pool) error {
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

// https://github.com/fergusstrange/embedded-postgres/issues/154
func postgresBinaryDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".embedded-postgres-go", "extracted")
}

// repairIncompletePostgresExtraction makes embedded-postgres retry extraction
// when a previously interrupted extraction left its pg_ctl sentinel behind.
func repairIncompletePostgresExtraction(binariesPath string) error {
	pgCtlPath := filepath.Join(binariesPath, "bin", "pg_ctl")
	if _, err := os.Stat(pgCtlPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect embedded PostgreSQL pg_ctl: %w", err)
	}

	requiredPaths := []string{
		filepath.Join(binariesPath, "bin", "initdb"),
		filepath.Join(binariesPath, "bin", "postgres"),
		filepath.Join(binariesPath, "share", "postgresql", "postgres.bki"),
		filepath.Join(binariesPath, "share", "postgresql", "postgresql.conf.sample"),
	}
	for _, requiredPath := range requiredPaths {
		if _, err := os.Stat(requiredPath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("inspect embedded PostgreSQL binary set: %w", err)
			}
			if err := os.Remove(pgCtlPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("invalidate incomplete embedded PostgreSQL extraction: %w", err)
			}
			return nil
		}
	}
	return nil
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
