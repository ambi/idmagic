package memory

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAuthnRequestReplayStoreRecordsOnceAndExpires(t *testing.T) {
	s := NewAuthnRequestReplayStore()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	if ok, err := s.RecordIfNew(context.Background(), "a", "sp", "request", time.Minute, now); err != nil || !ok {
		t.Fatalf("first=%v, %v", ok, err)
	}
	if ok, err := s.RecordIfNew(context.Background(), "a", "sp", "request", time.Minute, now); err != nil || ok {
		t.Fatalf("duplicate=%v, %v", ok, err)
	}
	if ok, err := s.RecordIfNew(context.Background(), "a", "sp", "request", time.Minute, now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("expired=%v, %v", ok, err)
	}
}

func TestAuthnRequestReplayStoreIsTenantScopedAndAtomic(t *testing.T) {
	s := NewAuthnRequestReplayStore()
	now := time.Now().UTC()
	if ok, _ := s.RecordIfNew(context.Background(), "a", "sp", "request", time.Minute, now); !ok {
		t.Fatal("first reservation failed")
	}
	if ok, _ := s.RecordIfNew(context.Background(), "b", "sp", "request", time.Minute, now); !ok {
		t.Fatal("tenant isolation failed")
	}
	var wg sync.WaitGroup
	twins := 0
	var mu sync.Mutex
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := s.RecordIfNew(context.Background(), "a", "sp", "parallel", time.Minute, now); ok {
				mu.Lock()
				twins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if twins != 1 {
		t.Fatalf("reservations=%d, want 1", twins)
	}
}
