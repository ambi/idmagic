package db_valkey

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func testClient(t *testing.T) *goredis.Client {
	t.Helper()
	server := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: server.Addr()})
}
