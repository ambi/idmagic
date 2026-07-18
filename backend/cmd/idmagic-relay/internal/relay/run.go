package relay

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// Run は outbox → 選択した transport (kafka | pubsub | log) のリレーを起動する。
// SIGINT/SIGTERM で graceful shutdown。
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

	pub, err := newPublisher(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create publisher: %w", err)
	}
	relay := eventsink.NewRelay(pool, pub)
	defer relay.Close()
	relay.PollInterval = cfg.PollInterval
	relay.BatchSize = cfg.BatchSize
	logging.Info(ctx, "idmagic relay started", "sink", cfg.Sink)
	if err := relay.Run(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("relay: %w", err)
	}
	return nil
}

// newPublisher は RELAY_SINK に応じた Publisher を構築する。pubsub は build タグ
// `pubsub` 付きビルドでのみ利用でき、非タグビルドでは明示エラーになる ([[ADR-120]])。
func newPublisher(ctx context.Context, cfg Config) (eventsink.Publisher, error) {
	switch cfg.Sink {
	case "kafka":
		pub, err := eventsink.NewKafkaPublisher(cfg.Brokers, cfg.ClientID)
		if err != nil {
			return nil, err
		}
		return pub, nil
	case "pubsub":
		return eventsink.NewPubSubPublisher(ctx, cfg.PubSubProject)
	case "log":
		return eventsink.NewLogPublisher(), nil
	default:
		return nil, fmt.Errorf("unknown sink %q", cfg.Sink)
	}
}
