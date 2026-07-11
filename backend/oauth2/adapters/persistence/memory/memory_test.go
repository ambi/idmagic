package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestAuthorizationCodeRedeemIsAtomic(t *testing.T) {
	store := NewAuthorizationCodeStore()
	code := &domain.AuthorizationCodeRecord{Code: "code", State: spec.AuthCodeRecordIssued}
	if err := store.Save(context.Background(), code); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			rec, err := store.Redeem(context.Background(), "code", time.Now())
			if err != nil {
				t.Errorf("redeem: %v", err)
			}
			if rec != nil {
				successes.Add(1)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful redeems=%d, want 1", successes.Load())
	}
}

func TestDeviceCodeExchangeIsAtomic(t *testing.T) {
	store := NewDeviceCodeStore()
	rec := &domain.DeviceAuthorization{
		DeviceCodeHash: "hash", UserCode: "CODE", State: spec.DeviceFlowApproved,
	}
	if err := store.Save(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			exchanged, err := store.Exchange(context.Background(), "hash")
			if err != nil {
				t.Errorf("exchange: %v", err)
			}
			if exchanged != nil {
				successes.Add(1)
			}
		})
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful exchanges=%d, want 1", successes.Load())
	}
}

func TestReplayStoreAcceptsJTIOnce(t *testing.T) {
	store := NewDpopReplayStore()
	now := time.Now()
	first, err := store.RecordIfNew(context.Background(), "jti", 60, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RecordIfNew(context.Background(), "jti", 60, now)
	if err != nil {
		t.Fatal(err)
	}
	if !first || second {
		t.Fatalf("first=%v second=%v", first, second)
	}

	// now.IsZero() のケースをテスト
	third, err := store.RecordIfNew(context.Background(), "jti-zero-time", 60, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if !third {
		t.Error("expected true when recording with zero time")
	}

	// 期限切れクリーンアップのケースをテスト
	_, _ = store.RecordIfNew(context.Background(), "jti-expired", 1, now)
	// 2秒進めた時間で別のキーを登録
	future := now.Add(2 * time.Second)
	_, err = store.RecordIfNew(context.Background(), "jti-new", 60, future)
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientAssertionReplayStore(t *testing.T) {
	store := NewClientAssertionReplayStore()
	now := time.Now()
	first, err := store.RecordIfNew(context.Background(), "jti-client", 60, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RecordIfNew(context.Background(), "jti-client", 60, now)
	if err != nil {
		t.Fatal(err)
	}
	if !first || second {
		t.Fatalf("first=%v second=%v", first, second)
	}
}
