package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
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

// countedLaunch は環境変数を map から読み、依存の組み立てが呼ばれた回数を数える起動入力である。
// 組み立ては実行せず errAssembled を返す。KeyStore は組み立ての中でしか作られないので、
// 呼ばれていなければ鍵のローテーションも作成も起こり得ない。
func countedLaunch(env map[string]string) (launch, *int) {
	calls := 0
	return launch{
		loader: bootstrap.NewConfigLoader(func(key string) string { return env[key] }),
		assemble: func(context.Context, bootstrap.SharedConfig) (*bootstrap.Dependencies, error) {
			calls++
			return nil, errAssembled
		},
	}, &calls
}

var errAssembled = errors.New("dependencies were assembled")

//spec:covers EX-SIGNINGKEYS-003-01: grace-days が cadence-days と等しいか大きい signing-key-lifecycle は設定エラーで終了し、依存を組み立てないので鍵をローテートしない。正しい設定では組み立てが 1 回呼ばれることを対照にする。
func TestSigningKeyLifecycleRefusesGraceNotBelowCadenceBeforeTouchingKeys(t *testing.T) {
	for _, flags := range [][]string{
		{"--cadence-days=7", "--grace-days=7"},
		{"--cadence-days=7", "--grace-days=8"},
	} {
		in, assembled := countedLaunch(map[string]string{})
		err := run(t.Context(), append([]string{"signing-key-lifecycle"}, flags...), in)
		if err == nil || errors.Is(err, errAssembled) || !strings.Contains(err.Error(), "grace-days") {
			t.Fatalf("%v: err = %v, want a configuration error naming grace-days", flags, err)
		}
		if *assembled != 0 {
			t.Fatalf("%v: dependencies assembled %d times, want none before the configuration is accepted", flags, *assembled)
		}
	}

	in, assembled := countedLaunch(map[string]string{})
	if err := run(t.Context(), []string{"signing-key-lifecycle", "--cadence-days=90", "--grace-days=7"}, in); !errors.Is(err, errAssembled) {
		t.Fatalf("valid lifecycle config: err = %v, want the run to reach dependency assembly", err)
	}
	if *assembled != 1 {
		t.Fatalf("valid lifecycle config assembled dependencies %d times, want 1", *assembled)
	}
}

//spec:covers EX-SIGNINGKEYS-012-01: PERSISTENCE=postgres で KEY_PROVIDER が無い起動は KEY_PROVIDER を名指す設定エラーで拒否され、依存を組み立てないので平文の鍵素材を持つ KeyStore も署名鍵も作られない。KEY_PROVIDER=local を明示すれば組み立てへ進むことを対照にする。
func TestBatchRefusesImplicitPlaintextKeyCustodyBeforeAssembling(t *testing.T) {
	env := map[string]string{
		"PERSISTENCE":  "postgres",
		"DATABASE_URL": "postgres://idmagic@db.internal:5432/idmagic",
	}
	in, assembled := countedLaunch(env)
	err := run(t.Context(), []string{"signing-key-lifecycle"}, in)
	if err == nil || errors.Is(err, errAssembled) || !strings.Contains(err.Error(), "KEY_PROVIDER") {
		t.Fatalf("err = %v, want a startup configuration error naming KEY_PROVIDER", err)
	}
	if *assembled != 0 {
		t.Fatalf("dependencies assembled %d times, want none without an explicit KEY_PROVIDER", *assembled)
	}

	env["KEY_PROVIDER"] = "local"
	in, assembled = countedLaunch(env)
	if err := run(t.Context(), []string{"signing-key-lifecycle"}, in); !errors.Is(err, errAssembled) {
		t.Fatalf("explicit KEY_PROVIDER=local: err = %v, want the run to reach dependency assembly", err)
	}
	if *assembled != 1 {
		t.Fatalf("explicit KEY_PROVIDER=local assembled dependencies %d times, want 1", *assembled)
	}
}
