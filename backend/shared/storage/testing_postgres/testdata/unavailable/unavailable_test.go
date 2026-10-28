package unavailable_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) {
	// 実行環境の DB 可否によらず、Main がランタイムディレクトリを作れない状態にする。
	if err := os.Setenv("TMPDIR", filepath.Join(os.DevNull, "pgtest-unavailable")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(pgtest.Main(m))
}

func TestRequire(t *testing.T) {
	pgtest.Require(t)
}

func TestWithoutDatabase(t *testing.T) {
	t.Log("DB を使わないテストも実行する")
}
