package domain_test

import (
	"testing"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validTransmitterConfig() ssdomain.SsfTransmitterConfig {
	return ssdomain.SsfTransmitterConfig{
		StreamID:            "stream_1",
		DeliveryEndpoint:    "https://receiver.example/ssf/events",
		Audience:            "https://receiver.example",
		MaxDeliveryAttempts: ssdomain.DefaultMaxDeliveryAttempts,
	}
}

// TestSsfTransmitterConfigValidateHappyAndFailure — scenario
// `receiver障害はローカル失効を遅らせない` の前提となる構造的妥当性を検証する。
func TestSsfTransmitterConfigValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.SsfTransmitterConfig)
		wantErr bool
	}{
		{"ok", func(*ssdomain.SsfTransmitterConfig) {}, false},
		{"endpoint not https", func(c *ssdomain.SsfTransmitterConfig) { c.DeliveryEndpoint = "http://receiver.example" }, true},
		{"zero max attempts", func(c *ssdomain.SsfTransmitterConfig) { c.MaxDeliveryAttempts = 0 }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := validTransmitterConfig()
			c.mutate(&cfg)
			err := cfg.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}
