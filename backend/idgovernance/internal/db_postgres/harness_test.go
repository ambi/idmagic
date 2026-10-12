package db_postgres_test

import (
	"os"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// TestMain は埋め込み PostgreSQL を起動して infra/schema/postgres.sql を適用してから、
// このパッケージのテストを実行する。起動できない環境では DB を使うテストを skip する。
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
