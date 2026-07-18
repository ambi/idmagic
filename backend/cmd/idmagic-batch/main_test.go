package main

import (
	"testing"
	"time"
)

func TestParseSigningKeyLifecycleConfig(t *testing.T) {
	cfg, err := parseSigningKeyLifecycleConfig([]string{"--cadence-days=30", "--grace-days=3"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.cadence != 30*24*time.Hour || cfg.grace != 3*24*time.Hour {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestParseSigningKeyLifecycleConfigRejectsGraceAtCadence(t *testing.T) {
	if _, err := parseSigningKeyLifecycleConfig([]string{"--cadence-days=7", "--grace-days=7"}); err == nil {
		t.Fatal("expected invalid lifecycle config")
	}
}
