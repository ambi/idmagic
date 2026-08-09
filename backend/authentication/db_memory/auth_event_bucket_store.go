package db_memory

// AuthEventBucketStore は AuthEventBucketStore (wi-20 スライス 3) の in-memory 実装。
// (tenant, kind, keyHash, windowStart) を鍵に 5 分窓の件数を積む。攻撃時の爆発を抑える
// のが目的なので個別行は持たず、窓ごとの集約 1 件だけを保持する。テスト / memory 構成用。

import (
	"context"
	"sync"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

const (
	authEventBucketDefaultListLimit = 100
	authEventBucketMaxListLimit     = 1000
	// authEventBucketMaxBuckets は古い窓が無限に溜まらないようにする上限 (aging)。
	authEventBucketMaxBuckets = 10000
)

type authEventBucketKey struct {
	tenantID    string
	kind        authnports.AuthEventBucketKind
	keyHash     string
	windowStart int64
}

type AuthEventBucketStore struct {
	mu      sync.Mutex
	buckets map[authEventBucketKey]*authnports.AuthEventBucket
}

func NewAuthEventBucketStore() *AuthEventBucketStore {
	return &AuthEventBucketStore{buckets: map[authEventBucketKey]*authnports.AuthEventBucket{}}
}

func (s *AuthEventBucketStore) Record(
	_ context.Context,
	kind authnports.AuthEventBucketKind,
	tenantID, keyHash string,
	now time.Time,
) (authnports.AuthEventBucketResult, error) {
	windowStart := now.UTC().Truncate(authnports.AuthEventBucketWindow)
	key := authEventBucketKey{
		tenantID: tenantID, kind: kind, keyHash: keyHash, windowStart: windowStart.Unix(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket, exists := s.buckets[key]
	first := !exists
	if first {
		bucket = &authnports.AuthEventBucket{
			TenantID: tenantID, Kind: kind, KeyHash: keyHash,
			WindowStart: windowStart, FirstSeen: now.UTC(),
		}
		s.buckets[key] = bucket
		s.evictOldestLocked()
	}
	bucket.Count++
	bucket.LastSeen = now.UTC()
	return authnports.AuthEventBucketResult{Bucket: *bucket, FirstInWindow: first}, nil
}

func bucketKeysetKey(bucket authnports.AuthEventBucket) string {
	return string(bucket.Kind) + "|" + bucket.KeyHash
}

func (s *AuthEventBucketStore) List(
	_ context.Context,
	tenantID string,
	afterWindowStart time.Time,
	afterKey string,
	limit int,
) ([]authnports.AuthEventBucket, error) {
	if limit <= 0 {
		limit = authEventBucketDefaultListLimit
	}
	if limit > authEventBucketMaxListLimit {
		limit = authEventBucketMaxListLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]authnports.AuthEventBucket, 0, len(s.buckets))
	for _, bucket := range s.buckets {
		if tenantID != "" && bucket.TenantID != tenantID {
			continue
		}
		out = append(out, *bucket)
	}
	// windowStart 降順 (新しい窓が先)、同窓は "kind|keyHash" (DESC、Postgres 側の
	// tuple 比較と方向を揃える) で安定化 (wi-159, ADR-158: keyset pagination には
	// count のような非一意キーではなく、PRIMARY KEY (tenant_id, kind, key_hash,
	// window_start) 相当の一意キーが要る)。
	key := func(b authnports.AuthEventBucket) (string, string) {
		return b.WindowStart.UTC().Format(time.RFC3339Nano), bucketKeysetKey(b)
	}
	afterPrimary := ""
	if !afterWindowStart.IsZero() {
		afterPrimary = afterWindowStart.UTC().Format(time.RFC3339Nano)
	}
	out = sharedmem.KeysetPage(out, key, true, afterPrimary, afterKey, limit)
	return out, nil
}

// DeleteOlderThan は windowStart が before より前の bucket を物理削除し、削除件数を返す
// (ADR-045 の保持期間 sweep / 既定 90 日)。idempotent。
func (s *AuthEventBucketStore) DeleteOlderThan(_ context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted int64
	for key, bucket := range s.buckets {
		if bucket.WindowStart.Before(before) {
			delete(s.buckets, key)
			deleted++
		}
	}
	return deleted, nil
}

// evictOldestLocked は bucket 数が上限を超えたら最も古い窓から 1 件落とす。呼び出し前に lock 済み。
func (s *AuthEventBucketStore) evictOldestLocked() {
	if len(s.buckets) <= authEventBucketMaxBuckets {
		return
	}
	var oldestKey authEventBucketKey
	var oldest int64
	found := false
	for k := range s.buckets {
		if !found || k.windowStart < oldest {
			oldestKey, oldest, found = k, k.windowStart, true
		}
	}
	if found {
		delete(s.buckets, oldestKey)
	}
}
