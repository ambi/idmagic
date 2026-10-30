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

// SsfStreamRepository は SsfStream の in-memory 実装。
type SsfStreamRepository struct {
	mu      sync.RWMutex
	streams map[string]*ssdomain.SsfStream // key: sharedmem.TenantKey(tenant_id, id)
}

func NewSsfStreamRepository() *SsfStreamRepository {
	return &SsfStreamRepository{streams: map[string]*ssdomain.SsfStream{}}
}

func cloneStream(s *ssdomain.SsfStream) *ssdomain.SsfStream {
	cloned := *s
	cloned.EventTypes = slices.Clone(s.EventTypes)
	cloned.AuditEventTypes = slices.Clone(s.AuditEventTypes)
	return &cloned
}

func (r *SsfStreamRepository) ListAll(_ context.Context, tenantID string) ([]*ssdomain.SsfStream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ssdomain.SsfStream, 0)
	for _, s := range r.streams {
		if s.TenantID == tenantID {
			out = append(out, cloneStream(s))
		}
	}
	slices.SortFunc(out, func(a, b *ssdomain.SsfStream) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (r *SsfStreamRepository) FindByID(_ context.Context, tenantID, id string) (*ssdomain.SsfStream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.streams[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
	if s == nil {
		return nil, nil
	}
	return cloneStream(s), nil
}

func (r *SsfStreamRepository) Save(_ context.Context, s *ssdomain.SsfStream) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[sharedmem.ScopedKey(s.TenantID, s.ID, kernel.DefaultTenantID)] = cloneStream(s)
	return nil
}

func (r *SsfStreamRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	return nil
}

// RecordDeliveryFailure は連続の失敗を数え、閾値で無効化したときに true を返す。
func (r *SsfStreamRepository) RecordDeliveryFailure(_ context.Context, tenantID, id string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.streams[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
	if s == nil {
		return false, nil
	}
	s.ConsecutiveFailures++
	if s.FirstFailureAt == nil {
		first := now
		s.FirstFailureAt = &first
	}
	if !s.IsEnabled() || s.ConsecutiveFailures < ssdomain.AutoDisableFailures || now.Sub(*s.FirstFailureAt) < ssdomain.AutoDisableElapsed {
		return false, nil
	}
	s.Status = ssdomain.SsfStreamStatusDisabled
	return true, nil
}

// RecordDeliverySuccess は連続の失敗の記録を消す。
func (r *SsfStreamRepository) RecordDeliverySuccess(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s := r.streams[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]; s != nil {
		s.ConsecutiveFailures = 0
		s.FirstFailureAt = nil
	}
	return nil
}
