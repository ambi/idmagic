package relay

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config は idmagic-relay の起動構成。すべて環境変数から組み立てる。
type Config struct {
	Sink          string // RELAY_SINK: kafka | pubsub | log
	DatabaseURL   string
	Brokers       []string
	ClientID      string
	PubSubProject string
	PollInterval  time.Duration
	BatchSize     int
}

func loadConfig() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	sink := envDefault("RELAY_SINK", "kafka")
	cfg := Config{
		Sink:          sink,
		DatabaseURL:   databaseURL,
		Brokers:       splitNonEmpty(os.Getenv("KAFKA_BROKERS")),
		ClientID:      envDefault("RELAY_CLIENT_ID", "idmagic-relay"),
		PubSubProject: os.Getenv("PUBSUB_PROJECT"),
		PollInterval:  time.Duration(envInt("POLL_INTERVAL_MS", 200)) * time.Millisecond,
		BatchSize:     envInt("BATCH_SIZE", 100),
	}
	switch sink {
	case "kafka":
		if len(cfg.Brokers) == 0 {
			return Config{}, errors.New("KAFKA_BROKERS is required when RELAY_SINK=kafka")
		}
	case "pubsub":
		if cfg.PubSubProject == "" {
			return Config{}, errors.New("PUBSUB_PROJECT is required when RELAY_SINK=pubsub")
		}
	case "log":
		// broker/クラウドを用意しないローカル/オンプレ最小構成。追加の必須項目なし。
	default:
		return Config{}, fmt.Errorf("RELAY_SINK must be kafka, pubsub or log, got %q", sink)
	}
	return cfg, nil
}
