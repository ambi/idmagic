package db_memory

import (
	"context"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// ReceivedSecurityEventRepository は inbound SET 受理記録の in-memory 実装。
// set_jti は stream 内で一意 (key に含めることで replay を検知する)。
type ReceivedSecurityEventRepository struct {
	mu     sync.RWMutex
	events map[string]*ssdomain.ReceivedSecurityEvent // key: sharedmem.TenantKey(tenant_id, stream_id+"|"+set_jti)
}

func NewReceivedSecurityEventRepository() *ReceivedSecurityEventRepository {
	return &ReceivedSecurityEventRepository{events: map[string]*ssdomain.ReceivedSecurityEvent{}}
}

func receivedEventKey(tenantID, streamID, setJTI string) string {
	return sharedmem.TenantKey(tenantID, streamID+"|"+setJTI)
}

func (r *ReceivedSecurityEventRepository) ExistsByJTI(_ context.Context, tenantID, streamID, setJTI string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.events[receivedEventKey(tenantID, streamID, setJTI)]
	return ok, nil
}

func (r *ReceivedSecurityEventRepository) Save(_ context.Context, e *ssdomain.ReceivedSecurityEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *e
	r.events[receivedEventKey(e.TenantID, e.StreamID, e.SetJTI)] = &cloned
	return nil
}
