package db_memory

import (
	"context"
	"slices"
	"sync"
	"time"

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

// ListDue は status が pending/failed かつ next_attempt_at が now 以前 (または未設定) の
// 配送を、created_at 昇順で最大 limit 件返す。
func (r *SecurityEventDeliveryRepository) ListDue(_ context.Context, now time.Time, limit int) ([]*ssdomain.SecurityEventDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ssdomain.SecurityEventDelivery, 0)
	for _, d := range r.deliveries {
		if d.IsTerminal() {
			continue
		}
		if d.NextAttemptAt != nil && d.NextAttemptAt.After(now) {
			continue
		}
		out = append(out, cloneDelivery(d))
	}
	slices.SortFunc(out, func(a, b *ssdomain.SecurityEventDelivery) int { return a.CreatedAt.Compare(b.CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *SecurityEventDeliveryRepository) Save(_ context.Context, d *ssdomain.SecurityEventDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deliveries[sharedmem.TenantKey(d.TenantID, d.ID)] = cloneDelivery(d)
	return nil
}
