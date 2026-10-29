package testing_postgres

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequireUnavailable(t *testing.T) {
	// 子の go test が読む testdata は Go の結果キャッシュの鍵に入らないため、実行ごとの値で再利用を防ぐ。
	_ = os.Getenv("VERIFICATION_RUN_ID")
	cases := []struct {
		name       string
		allowSkip  string
		wantFailed bool
		wantCached bool
	}{
		{name: "既定は失敗", wantFailed: true},
		{name: "明示の許可だけスキップ", allowSkip: "1"},
		{name: "許可した結果をキャッシュ", allowSkip: "1", wantCached: true},
		{name: "許可解除はキャッシュを使わず失敗", wantFailed: true},
		{name: "0 は許可しない", allowSkip: "0", wantFailed: true},
		{name: "true は許可しない", allowSkip: "true", wantFailed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PGTEST_ALLOW_SKIP", tc.allowSkip)
			// fixture の Main はランタイムディレクトリを作れず、Pool が nil になる。
			// 子は結果キャッシュを有効にして、許可の変更が鍵に入ることも観測する。
			cmd := exec.CommandContext(t.Context(), "go", "test", "-v", "./testdata/unavailable")
			output, err := cmd.CombinedOutput()
			if (err != nil) != tc.wantFailed {
				t.Fatalf("go test error = %v, want failure = %v\n%s", err, tc.wantFailed, output)
			}
			result := "--- SKIP: TestRequire"
			if tc.wantFailed {
				result = "--- FAIL: TestRequire"
			}
			if !strings.Contains(string(output), result) {
				t.Fatalf("DB test result missing %q:\n%s", result, output)
			}
			if !strings.Contains(string(output), "--- PASS: TestWithoutDatabase") {
				t.Fatalf("DB を使わないテストが成功していない:\n%s", output)
			}
			if tc.wantCached && !strings.Contains(string(output), "(cached)") {
				t.Fatalf("許可した結果がキャッシュされていない:\n%s", output)
			}
		})
	}
}

func TestRequireAvailable(t *testing.T) {
	previous := Pool
	t.Cleanup(func() { Pool = previous })
	Pool = &pgxpool.Pool{}
	for _, allow := range []string{"", "1"} {
		t.Setenv("PGTEST_ALLOW_SKIP", allow)
		if got := Require(t); got != Pool {
			t.Fatalf("Require = %p, want shared pool %p", got, Pool)
		}
	}
}
