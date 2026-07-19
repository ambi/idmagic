// Package memory implements the Provisioning bounded context's repositories
// in-memory (demo/test use, mirrors backend/idgovernance/adapters/persistence/memory).
package memory

import (
	"context"
	"slices"
	"sort"
	"sync"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// ProvisioningConnectionRepository is the in-memory ports.ProvisioningConnectionRepository.
type ProvisioningConnectionRepository struct {
	mu      sync.RWMutex
	conns   map[string]*domain.ProvisioningConnection // key: tenantKey(tenant_id, application_id)
	secrets map[string]string
}

var _ ports.ProvisioningConnectionRepository = (*ProvisioningConnectionRepository)(nil)

func NewProvisioningConnectionRepository() *ProvisioningConnectionRepository {
	return &ProvisioningConnectionRepository{conns: map[string]*domain.ProvisioningConnection{}, secrets: map[string]string{}}
}

func connKey(tenantID, applicationID string) string {
	return sharedmem.TenantKey(tenantID, applicationID)
}

func cloneConnection(c *domain.ProvisioningConnection) *domain.ProvisioningConnection {
	if c == nil {
		return nil
	}
	clone := *c
	clone.AttributeMappings = slices.Clone(c.AttributeMappings)
	if c.Capabilities != nil {
		capabilities := *c.Capabilities
		clone.Capabilities = &capabilities
	}
	if c.GroupPush != nil {
		gp := *c.GroupPush
		gp.ExplicitGroupIDs = slices.Clone(c.GroupPush.ExplicitGroupIDs)
		clone.GroupPush = &gp
	}
	if c.NotificationEmail != nil {
		v := *c.NotificationEmail
		clone.NotificationEmail = &v
	}
	if c.LastFullSyncAt != nil {
		v := *c.LastFullSyncAt
		clone.LastFullSyncAt = &v
	}
	if c.QuarantinedAt != nil {
		v := *c.QuarantinedAt
		clone.QuarantinedAt = &v
	}
	if c.QuarantineReason != nil {
		v := *c.QuarantineReason
		clone.QuarantineReason = &v
	}
	return &clone
}

func (r *ProvisioningConnectionRepository) Register(_ context.Context, conn *domain.ProvisioningConnection, secret string) error {
	if err := conn.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := connKey(conn.TenantID, conn.ApplicationID)
	if _, ok := r.conns[key]; ok {
		return ports.ErrConnectionAlreadyExists
	}
	r.conns[key] = cloneConnection(conn)
	r.secrets[key] = secret
	return nil
}

func (r *ProvisioningConnectionRepository) Update(_ context.Context, conn *domain.ProvisioningConnection, secret *string) error {
	if err := conn.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := connKey(conn.TenantID, conn.ApplicationID)
	if _, ok := r.conns[key]; !ok {
		return nil
	}
	r.conns[key] = cloneConnection(conn)
	if secret != nil {
		r.secrets[key] = *secret
	}
	return nil
}

func (r *ProvisioningConnectionRepository) Find(_ context.Context, tenantID, applicationID string) (*domain.ProvisioningConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneConnection(r.conns[connKey(tenantID, applicationID)]), nil
}

func (r *ProvisioningConnectionRepository) CredentialSecret(_ context.Context, tenantID, applicationID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.secrets[connKey(tenantID, applicationID)], nil
}

func (r *ProvisioningConnectionRepository) Delete(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := connKey(tenantID, applicationID)
	delete(r.conns, key)
	delete(r.secrets, key)
	return nil
}

func (r *ProvisioningConnectionRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.ProvisioningConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningConnection{}
	for _, c := range r.conns {
		if c.TenantID == tenantID {
			out = append(out, cloneConnection(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ApplicationID < out[j].ApplicationID })
	return out, nil
}

// RemoteResourceLinkRepository is the in-memory ports.RemoteResourceLinkRepository.
type RemoteResourceLinkRepository struct {
	mu    sync.RWMutex
	links map[string]*domain.RemoteResourceLink
}

var _ ports.RemoteResourceLinkRepository = (*RemoteResourceLinkRepository)(nil)

func NewRemoteResourceLinkRepository() *RemoteResourceLinkRepository {
	return &RemoteResourceLinkRepository{links: map[string]*domain.RemoteResourceLink{}}
}

func linkKey(connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) string {
	return connectionID + "|" + string(sourceType) + "|" + sourceID
}

func (r *RemoteResourceLinkRepository) Find(_ context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) (*domain.RemoteResourceLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	link := r.links[linkKey(connectionID, sourceType, sourceID)]
	if link == nil {
		return nil, nil
	}
	clone := *link
	return &clone, nil
}

func (r *RemoteResourceLinkRepository) Upsert(_ context.Context, link *domain.RemoteResourceLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := *link
	r.links[linkKey(link.ConnectionID, link.SourceType, link.SourceID)] = &clone
	return nil
}

// ProvisioningDeliveryRepository is the in-memory ports.ProvisioningDeliveryRepository.
type ProvisioningDeliveryRepository struct {
	mu          sync.RWMutex
	deliveries  map[string]*domain.ProvisioningDelivery // key: tenantKey(tenant_id, id)
	idempotency map[string]string                       // idempotency key -> delivery id
}

var _ ports.ProvisioningDeliveryRepository = (*ProvisioningDeliveryRepository)(nil)

func NewProvisioningDeliveryRepository() *ProvisioningDeliveryRepository {
	return &ProvisioningDeliveryRepository{deliveries: map[string]*domain.ProvisioningDelivery{}, idempotency: map[string]string{}}
}

func deliveryKey(tenantID, id string) string { return sharedmem.TenantKey(tenantID, id) }

func cloneDelivery(d *domain.ProvisioningDelivery) *domain.ProvisioningDelivery {
	if d == nil {
		return nil
	}
	clone := *d
	if d.JobID != nil {
		v := *d.JobID
		clone.JobID = &v
	}
	if d.LastError != nil {
		v := *d.LastError
		clone.LastError = &v
	}
	if d.CompletedAt != nil {
		v := *d.CompletedAt
		clone.CompletedAt = &v
	}
	return &clone
}

func (r *ProvisioningDeliveryRepository) Save(_ context.Context, d *domain.ProvisioningDelivery) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	idKey := d.IdempotencyKey()
	if _, ok := r.idempotency[idKey]; ok {
		return false, nil
	}
	r.deliveries[deliveryKey(d.TenantID, d.ID)] = cloneDelivery(d)
	r.idempotency[idKey] = d.ID
	return true, nil
}

func (r *ProvisioningDeliveryRepository) Find(_ context.Context, tenantID, deliveryID string) (*domain.ProvisioningDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneDelivery(r.deliveries[deliveryKey(tenantID, deliveryID)]), nil
}

func (r *ProvisioningDeliveryRepository) ListByConnection(_ context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, limit int) ([]*domain.ProvisioningDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningDelivery{}
	for _, d := range r.deliveries {
		if d.TenantID != tenantID || d.ConnectionID != connectionID {
			continue
		}
		if status != nil && d.Status != *status {
			continue
		}
		out = append(out, cloneDelivery(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *ProvisioningDeliveryRepository) ListUnenqueued(_ context.Context, limit int) ([]*domain.ProvisioningDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningDelivery{}
	for _, d := range r.deliveries {
		if d.Status == domain.DeliveryPending && d.JobID == nil {
			out = append(out, cloneDelivery(d))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *ProvisioningDeliveryRepository) AttachJob(_ context.Context, tenantID, deliveryID, jobID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.deliveries[deliveryKey(tenantID, deliveryID)]
	if d == nil || d.JobID != nil {
		return false, nil
	}
	d.JobID = &jobID
	return true, nil
}

func (r *ProvisioningDeliveryRepository) UpdateStatus(_ context.Context, tenantID, deliveryID string, status domain.ProvisioningDeliveryStatus, lastError *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.deliveries[deliveryKey(tenantID, deliveryID)]
	if d == nil {
		return nil
	}
	d.Status = status
	d.LastError = lastError
	return nil
}

func (r *ProvisioningDeliveryRepository) RetryDeadLetter(_ context.Context, tenantID, deliveryID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.deliveries[deliveryKey(tenantID, deliveryID)]
	if d == nil || d.Status != domain.DeliveryDeadLetter {
		return false, nil
	}
	d.Status, d.JobID, d.LastError = domain.DeliveryPending, nil, nil
	return true, nil
}
