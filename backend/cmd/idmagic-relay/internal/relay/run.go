package relay

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// Run は outbox → Kafka リレーを起動する。SIGINT/SIGTERM で graceful shutdown。
func Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbCfg := postgres.DBConfig{
		MaxConns:        10,
		MinConns:        1,
		MaxConnIdleTime: 30 * time.Second,
		MaxConnLifetime: 1 * time.Hour,
		ConnectTimeout:  5 * time.Second,
		QueryTimeout:    5 * time.Second,
	}

	pool, err := postgres.Open(ctx, cfg.DatabaseURL, dbCfg)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer pool.Close()
	relay, err := eventsink.NewKafkaRelay(pool, cfg.Brokers, cfg.ClientID)
	if err != nil {
		return fmt.Errorf("create relay: %w", err)
	}
	defer relay.Close()
	relay.PollInterval = cfg.PollInterval
	relay.BatchSize = cfg.BatchSize
	logging.Info(ctx, "idmagic relay started", "brokers", strings.Join(cfg.Brokers, ","))
	if err := relay.Run(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("relay: %w", err)
	}
	return nil
}
