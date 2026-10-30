package db_memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// SecurityEventDeliveryRepository は SecurityEventDelivery (outbound SET outbox) の
// in-memory 実装。
type SecurityEventDeliveryRepository struct {
	mu         sync.RWMutex
	deliveries map[string]*ssdomain.SecurityEventDelivery // key: sharedmem.TenantKey(tenant_id, id)
}

func NewSecurityEventDeliveryRepository() *SecurityEventDeliveryRepository {
	return &SecurityEventDeliveryRepository{deliveries: map[string]*ssdomain.SecurityEventDelivery{}}
}

func cloneDelivery(d *ssdomain.SecurityEventDelivery) *ssdomain.SecurityEventDelivery {
	cloned := *d
	return &cloned
}

func (r *SecurityEventDeliveryRepository) ListByStream(_ context.Context, tenantID, streamID string) ([]*ssdomain.SecurityEventDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ssdomain.SecurityEventDelivery, 0)
	for _, d := range r.deliveries {
		if d.TenantID == tenantID && d.StreamID == streamID {
			out = append(out, cloneDelivery(d))
		}
	}
	slices.SortFunc(out, func(a, b *ssdomain.SecurityEventDelivery) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

// RedeliverDeadLettered は stream の dead_letter の配送を pending へ戻し、created_at 昇順で返す。
func (r *SecurityEventDeliveryRepository) RedeliverDeadLettered(_ context.Context, tenantID, streamID string) ([]*ssdomain.SecurityEventDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*ssdomain.SecurityEventDelivery, 0)
	for _, d := range r.deliveries {
		if d.TenantID != tenantID || d.StreamID != streamID || d.Status != ssdomain.SecurityEventDeliveryStatusDeadLetter {
			continue
		}
		d.Status, d.AttemptCount, d.NextAttemptAt, d.LastError = ssdomain.SecurityEventDeliveryStatusPending, 0, nil, nil
		out = append(out, cloneDelivery(d))
	}
	slices.SortFunc(out, func(a, b *ssdomain.SecurityEventDelivery) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (r *SecurityEventDeliveryRepository) Save(_ context.Context, d *ssdomain.SecurityEventDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliveries[sharedmem.ScopedKey(d.TenantID, d.ID, kernel.DefaultTenantID)] = cloneDelivery(d)
	return nil
}

// ClaimDue は配送の時刻が来た配送を、ストリームごとに perStream 件まで取得し、時刻を leaseUntil へ進める。
// ストリームの状態を知らないので、無効化の判定は呼び出し側が行う。
func (r *SecurityEventDeliveryRepository) ClaimDue(_ context.Context, now, leaseUntil time.Time, perStream, limit int) ([]*ssdomain.SecurityEventDelivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	due := make([]*ssdomain.SecurityEventDelivery, 0)
	for _, d := range r.deliveries {
		if d.IsSettled() || (d.NextAttemptAt != nil && d.NextAttemptAt.After(now)) {
			continue
		}
		due = append(due, d)
	}
	slices.SortFunc(due, func(a, b *ssdomain.SecurityEventDelivery) int { return a.CreatedAt.Compare(b.CreatedAt) })
	counts := map[string]int{}
	out := make([]*ssdomain.SecurityEventDelivery, 0)
	for _, d := range due {
		key := d.TenantID + "/" + d.StreamID
		if counts[key] >= perStream || len(out) >= limit {
			continue
		}
		counts[key]++
		lease := leaseUntil
		d.NextAttemptAt = &lease
		out = append(out, cloneDelivery(d))
	}
	return out, nil
}

// PurgeSettled は before より前に作られた終端の配送を消す。
func (r *SecurityEventDeliveryRepository) PurgeSettled(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deleted int64
	for key, d := range r.deliveries {
		if d.IsSettled() && d.CreatedAt.Before(before) {
			delete(r.deliveries, key)
			deleted++
		}
	}
	return deleted, nil
}
