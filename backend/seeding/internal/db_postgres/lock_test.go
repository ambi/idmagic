package db_postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/seeding/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

//spec:covers REQ-SEEDING-006: 独立した DB セッションで同じ要求を直列化し、別の鍵は待たず、キャンセル後も再取得できる。
func TestRequestLockSerializesIndependentConnections(t *testing.T) {
	pgtest.Require(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := domain.Request{Environment: domain.EnvironmentTest, Profile: domain.ProfileTest, TenantID: "lock-test"}
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unlock := func() { once.Do(func() { close(release) }) }
	defer unlock()
	first := make(chan error, 1)
	go func() {
		first <- WithRequestLock(ctx, pgtest.Pool, request, func(ctx context.Context) error {
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	waitCtx, cancelWait := context.WithCancel(ctx)
	defer cancelWait()
	second := make(chan error, 1)
	go func() {
		second <- WithRequestLock(waitCtx, pgtest.Pool, request, func(context.Context) error {
			return errors.New("second apply entered before cancellation")
		})
	}()
	for {
		var waiting bool
		//sql:raw DB のロック待ちを観測し、スケジューラーの速度を判定条件にしない。
		if err := pgtest.Pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted)").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err := <-second:
			t.Fatalf("second request did not wait: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
	other := request
	other.TenantID = "other-lock-test"
	if err := WithRequestLock(ctx, pgtest.Pool, other, func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	cancelWait()
	if err := <-second; err == nil {
		t.Fatal("canceled waiter succeeded")
	}
	unlock()
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	want := errors.New("apply failure")
	if err := WithRequestLock(ctx, pgtest.Pool, request, func(context.Context) error { return want }); !errors.Is(err, want) {
		t.Fatalf("apply error = %v", err)
	}
	if err := WithRequestLock(ctx, pgtest.Pool, request, func(context.Context) error { return nil }); err != nil {
		t.Fatalf("lock remained after failure: %v", err)
	}
}
