package domain_test

import (
	"testing"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validReceiverConfig() ssdomain.SsfReceiverConfig {
	return ssdomain.SsfReceiverConfig{
		StreamID:          "stream_1",
		TrustedIssuer:     "https://issuer.example",
		JWKSURI:           new("https://issuer.example/.well-known/jwks.json"),
		AcceptedAudiences: []string{"https://idmagic.example/ssf"},
	}
}

// TestSsfReceiverConfigValidateHappyAndFailure — scenario
// `署名不正のSETは反映されず拒否される` の前提となる SsfReceiverConfig の構造的妥当性を検証する
// (spec/contexts/sharedsignals.yaml SsfReceiverConfig constraints)。
func TestSsfReceiverConfigValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.SsfReceiverConfig)
		wantErr bool
	}{
		{"ok", func(*ssdomain.SsfReceiverConfig) {}, false},
		{"issuer not https", func(c *ssdomain.SsfReceiverConfig) { c.TrustedIssuer = "http://issuer.example" }, true},
		{"no jwks_uri and no jwks", func(c *ssdomain.SsfReceiverConfig) { c.JWKSURI = nil; c.JWKS = nil }, true},
		{"jwks inline only is ok", func(c *ssdomain.SsfReceiverConfig) {
			c.JWKSURI = nil
			c.JWKS = map[string]any{"keys": []any{}}
		}, false},
		{"empty accepted_audiences", func(c *ssdomain.SsfReceiverConfig) { c.AcceptedAudiences = nil }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := validReceiverConfig()
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
