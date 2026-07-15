package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ambi/idmagic/backend/cmd/idmagic-dev-infra/internal/devinfra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var cfg devinfra.Config
	postgresPort := uint64Flag(devinfra.DefaultPostgresPort)
	flag.Var(&postgresPort, "postgres-port", "embedded PostgreSQL port")
	flag.IntVar(&cfg.ValkeyPort, "valkey-port", devinfra.DefaultValkeyPort, "development Valkey-compatible port")
	flag.StringVar(&cfg.SchemaPath, "schema", "infra/schema/postgres.sql", "PostgreSQL schema path")
	flag.StringVar(&cfg.ReadyFile, "ready-file", "", "ready marker JSON path")
	flag.StringVar(&cfg.DataPath, "data-dir", "", "persistent embedded PostgreSQL data directory; schema is reset on each start")
	flag.Parse()
	cfg.PostgresPort = uint32(postgresPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	runtime, ready, err := devinfra.Start(ctx, cfg)
	if err != nil {
		return err
	}
	defer runtime.Close()
	fmt.Printf("development infrastructure ready\nPostgreSQL: %s\nValkey-compatible: %s\n", ready.DatabaseURL, ready.ValkeyURL)
	<-ctx.Done()
	return nil
}

type uint64Flag uint32

func (f *uint64Flag) String() string { return fmt.Sprint(*f) }
func (f *uint64Flag) Set(value string) error {
	var parsed uint64
	if _, err := fmt.Sscan(value, &parsed); err != nil {
		return err
	}
	if parsed > 65535 {
		return fmt.Errorf("port out of range: %d", parsed)
	}
	*f = uint64Flag(parsed)
	return nil
}
