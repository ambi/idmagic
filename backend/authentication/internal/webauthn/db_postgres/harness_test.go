package db_postgres

import (
	"os"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// TestMain は embedded-postgres を起動して infra/schema/postgres.sql を投入する。
func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }
