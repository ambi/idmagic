package relay

import (
	"testing"
	"time"
)

// loadConfig は必須の環境変数から Config を組み立て、任意項目は既定値を補う。
func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/idmagic")
	t.Setenv("KAFKA_BROKERS", "broker1:9092, broker2:9092 ,")
	t.Setenv("RELAY_CLIENT_ID", "")
	t.Setenv("POLL_INTERVAL_MS", "")
	t.Setenv("BATCH_SIZE", "")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/idmagic" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if len(cfg.Brokers) != 2 || cfg.Brokers[0] != "broker1:9092" || cfg.Brokers[1] != "broker2:9092" {
		t.Errorf("Brokers = %v, want two trimmed brokers", cfg.Brokers)
	}
	if cfg.ClientID != "idmagic-relay" {
		t.Errorf("ClientID = %q, want default idmagic-relay", cfg.ClientID)
	}
	if cfg.PollInterval != 200*time.Millisecond {
		t.Errorf("PollInterval = %v, want 200ms", cfg.PollInterval)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
}

// 明示指定した任意項目は既定値より優先される。
func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db/x")
	t.Setenv("KAFKA_BROKERS", "b:9092")
	t.Setenv("RELAY_CLIENT_ID", "custom-relay")
	t.Setenv("POLL_INTERVAL_MS", "500")
	t.Setenv("BATCH_SIZE", "42")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.ClientID != "custom-relay" {
		t.Errorf("ClientID = %q, want custom-relay", cfg.ClientID)
	}
	if cfg.PollInterval != 500*time.Millisecond {
		t.Errorf("PollInterval = %v, want 500ms", cfg.PollInterval)
	}
	if cfg.BatchSize != 42 {
		t.Errorf("BatchSize = %d, want 42", cfg.BatchSize)
	}
}

// DATABASE_URL / KAFKA_BROKERS が欠けるとエラーになる。
func TestLoadConfigMissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		dbURL   string
		brokers string
	}{
		{"missing database url", "", "b:9092"},
		{"missing brokers", "postgres://db/x", ""},
		{"missing both", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tc.dbURL)
			t.Setenv("KAFKA_BROKERS", tc.brokers)
			if _, err := loadConfig(); err == nil {
				t.Fatal("loadConfig() expected error, got nil")
			}
		})
	}
}
