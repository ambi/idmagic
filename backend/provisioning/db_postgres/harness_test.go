package db_postgres

import (
	"os"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// TestMain starts embedded-postgres and applies infra/schema/postgres.sql
// before running this package's tests (pgtest.Main, wi-172). Environments that
// cannot start embedded-postgres skip DB-backed tests to stay green.
func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}
