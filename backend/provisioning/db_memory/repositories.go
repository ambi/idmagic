// Package memory implements the Provisioning bounded context's repositories
// in-memory (demo/test use, mirrors backend/idgovernance/db_memory).
package db_memory

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
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

func (r *ProvisioningConnectionRepository) ListAll(_ context.Context, tenantID string) ([]*domain.ProvisioningConnection, error) {
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

// ProvisioningTaskRepository is the in-memory ports.ProvisioningTaskRepository.
type ProvisioningTaskRepository struct {
	mu          sync.RWMutex
	tasks       map[string]*domain.ProvisioningTask // key: tenantKey(tenant_id, id)
	idempotency map[string]string                   // idempotency key -> task id
	// reservations は猶予期間つき削除の予約。key: tenantKey(tenant_id, id)
	reservations map[string]*domain.ScheduledDeprovision
}

var _ ports.ProvisioningTaskRepository = (*ProvisioningTaskRepository)(nil)

func NewProvisioningTaskRepository() *ProvisioningTaskRepository {
	return &ProvisioningTaskRepository{
		tasks: map[string]*domain.ProvisioningTask{}, idempotency: map[string]string{},
		reservations: map[string]*domain.ScheduledDeprovision{},
	}
}

func taskKey(tenantID, id string) string { return sharedmem.TenantKey(tenantID, id) }

func cloneTask(d *domain.ProvisioningTask) *domain.ProvisioningTask {
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

func (r *ProvisioningTaskRepository) Save(_ context.Context, d *domain.ProvisioningTask) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	idKey := d.IdempotencyKey()
	if _, ok := r.idempotency[idKey]; ok {
		return false, nil
	}
	r.tasks[taskKey(d.TenantID, d.ID)] = cloneTask(d)
	r.idempotency[idKey] = d.ID
	return true, nil
}

func (r *ProvisioningTaskRepository) Find(_ context.Context, tenantID, taskID string) (*domain.ProvisioningTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneTask(r.tasks[taskKey(tenantID, taskID)]), nil
}

func (r *ProvisioningTaskRepository) ListByConnection(_ context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, limit int) ([]*domain.ProvisioningTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningTask{}
	for _, d := range r.tasks {
		if d.TenantID != tenantID || d.ConnectionID != connectionID {
			continue
		}
		if status != nil && d.Status != *status {
			continue
		}
		out = append(out, cloneTask(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListPageByConnection implements ports.ProvisioningTaskRepository.ListPageByConnection
// (wi-159): keyset pagination ordered by (CreatedAt, ID) descending
// — matching ListByConnection's pre-existing "most recent first" order.
func (r *ProvisioningTaskRepository) ListPageByConnection(_ context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, sourceType *domain.ProvisioningSourceType, afterCreatedAt time.Time, afterID string, limit int) ([]*domain.ProvisioningTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningTask{}
	for _, d := range r.tasks {
		if d.TenantID != tenantID || d.ConnectionID != connectionID {
			continue
		}
		if status != nil && d.Status != *status {
			continue
		}
		if sourceType != nil && d.SourceType != *sourceType {
			continue
		}
		out = append(out, cloneTask(d))
	}
	key := func(d *domain.ProvisioningTask) (string, string) {
		return d.CreatedAt.UTC().Format(time.RFC3339Nano), d.ID
	}
	afterPrimary := ""
	if !afterCreatedAt.IsZero() {
		afterPrimary = afterCreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return sharedmem.KeysetPage(out, key, true, afterPrimary, afterID, limit), nil
}

func (r *ProvisioningTaskRepository) ListPageBeforeByConnection(_ context.Context, tenantID, connectionID string, status *domain.ProvisioningTaskStatus, sourceType *domain.ProvisioningSourceType, beforeCreatedAt time.Time, beforeID string, limit int) ([]*domain.ProvisioningTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningTask{}
	for _, d := range r.tasks {
		if d.TenantID != tenantID || d.ConnectionID != connectionID {
			continue
		}
		if status != nil && d.Status != *status {
			continue
		}
		if sourceType != nil && d.SourceType != *sourceType {
			continue
		}
		out = append(out, cloneTask(d))
	}
	key := func(d *domain.ProvisioningTask) (string, string) {
		return d.CreatedAt.UTC().Format(time.RFC3339Nano), d.ID
	}
	beforePrimary := ""
	if !beforeCreatedAt.IsZero() {
		beforePrimary = beforeCreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return sharedmem.KeysetPageBefore(out, key, true, beforePrimary, beforeID, limit), nil
}

func (r *ProvisioningTaskRepository) ListUnenqueued(_ context.Context, limit int) ([]*domain.ProvisioningTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ProvisioningTask{}
	for _, d := range r.tasks {
		if d.Status == domain.TaskPending && d.JobID == nil {
			out = append(out, cloneTask(d))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *ProvisioningTaskRepository) AttachJob(_ context.Context, tenantID, taskID, jobID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.tasks[taskKey(tenantID, taskID)]
	if d == nil || d.JobID != nil || d.Status != domain.TaskPending {
		return false, nil
	}
	d.JobID = &jobID
	d.Status = domain.TaskInFlight
	return true, nil
}

func (r *ProvisioningTaskRepository) UpdateStatus(_ context.Context, tenantID, taskID string, status domain.ProvisioningTaskStatus, lastError *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.tasks[taskKey(tenantID, taskID)]
	if d == nil {
		return nil
	}
	d.Status = status
	d.LastError = lastError
	return nil
}

func (r *ProvisioningTaskRepository) RetryDeadLetter(_ context.Context, tenantID, taskID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.tasks[taskKey(tenantID, taskID)]
	if d == nil || d.Status != domain.TaskDeadLetter {
		return false, nil
	}
	d.Status, d.JobID, d.LastError = domain.TaskPending, nil, nil
	return true, nil
}

func cloneScheduledDeprovision(s *domain.ScheduledDeprovision) *domain.ScheduledDeprovision {
	clone := *s
	if s.TaskID != nil {
		v := *s.TaskID
		clone.TaskID = &v
	}
	return &clone
}

func (r *ProvisioningTaskRepository) ScheduleDeprovision(_ context.Context, s *domain.ScheduledDeprovision) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.reservations {
		if existing.Status == domain.ScheduledDeprovisionScheduled && existing.TenantID == s.TenantID &&
			existing.ConnectionID == s.ConnectionID && existing.UserID == s.UserID {
			return false, nil
		}
	}
	r.reservations[taskKey(s.TenantID, s.ID)] = cloneScheduledDeprovision(s)
	return true, nil
}

func (r *ProvisioningTaskRepository) CancelScheduledDeprovisions(_ context.Context, tenantID, connectionID, userID string, beforeVersion int64, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cancelled := 0
	for _, s := range r.reservations {
		if s.Status != domain.ScheduledDeprovisionScheduled || s.TenantID != tenantID ||
			s.ConnectionID != connectionID || s.UserID != userID || s.SourceVersion >= beforeVersion {
			continue
		}
		s.Status, s.UpdatedAt = domain.ScheduledDeprovisionCancelled, now
		cancelled++
	}
	return cancelled, nil
}

func (r *ProvisioningTaskRepository) ListDueDeprovisions(_ context.Context, now time.Time, limit int) ([]*domain.ScheduledDeprovision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.ScheduledDeprovision{}
	for _, s := range r.reservations {
		if s.IsDue(now) {
			out = append(out, cloneScheduledDeprovision(s))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueAt.Before(out[j].DueAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *ProvisioningTaskRepository) MaterializeDeprovision(_ context.Context, s *domain.ScheduledDeprovision, d *domain.ProvisioningTask) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := r.reservations[taskKey(s.TenantID, s.ID)]
	if stored == nil || stored.Status != domain.ScheduledDeprovisionScheduled {
		return false, nil
	}
	idKey := d.IdempotencyKey()
	if _, ok := r.idempotency[idKey]; !ok {
		r.tasks[taskKey(d.TenantID, d.ID)] = cloneTask(d)
		r.idempotency[idKey] = d.ID
	}
	taskID := r.idempotency[idKey]
	stored.Status, stored.TaskID, stored.UpdatedAt = domain.ScheduledDeprovisionMaterialized, &taskID, d.CreatedAt
	return true, nil
}
