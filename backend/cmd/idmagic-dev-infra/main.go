package main

import (
	"context"
	"flag"
	"fmt"
	"net"
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
	flag.Var(&postgresPort, "postgres-port", "embedded PostgreSQL port; 0 picks a free port")
	flag.StringVar(&cfg.SchemaPath, "schema", "infra/schema/postgres.sql", "PostgreSQL schema path")
	flag.BoolVar(&cfg.SkipSchema, "empty", false, "publish an empty database without applying the schema")
	flag.StringVar(&cfg.ReadyFile, "ready-file", "", "ready marker JSON path")
	flag.StringVar(&cfg.DataPath, "data-dir", "", "persistent embedded PostgreSQL data directory; schema is reset on each start")
	flag.Parse()
	cfg.PostgresPort = uint32(postgresPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cfg.PostgresPort == 0 {
		port, err := freePort(ctx)
		if err != nil {
			return err
		}
		cfg.PostgresPort = port
	}
	runtime, ready, err := devinfra.Start(ctx, cfg)
	if err != nil {
		return err
	}
	defer runtime.Close()
	fmt.Printf("development infrastructure ready\nPostgreSQL: %s\n", ready.DatabaseURL)
	<-ctx.Done()
	return nil
}

// freePort lets parallel `check-schema` runs coexist with each other and with
// the development cluster on DefaultPostgresPort. The chosen port appears in
// the ready file's database_url.
func freePort(ctx context.Context) (uint32, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("pick a free PostgreSQL port: %w", err)
	}
	defer func() { _ = listener.Close() }()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}
	return uint32(addr.Port), nil //nolint:gosec // G115: a TCP port fits in uint32
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
