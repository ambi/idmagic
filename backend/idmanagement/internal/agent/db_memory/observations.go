package db_memory

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// AgentObservationRepository は外部の観測をメモリに保存する。PostgreSQL の外部キーと同じ結果に
// なるよう、AgentRepository の削除が確認を解除する。ロックは Agent、観測の順に取る。
type AgentObservationRepository struct {
	agents *AgentRepository
	mu     sync.RWMutex
	rows   map[string]*agentdomain.AgentObservation // key: id
}

// NewAgentObservationRepository は agents の削除に連動する保存先を作る。
func NewAgentObservationRepository(agents *AgentRepository) *AgentObservationRepository {
	repo := &AgentObservationRepository{agents: agents, rows: map[string]*agentdomain.AgentObservation{}}
	agents.observations = repo
	return repo
}

func cloneObservation(row *agentdomain.AgentObservation) *agentdomain.AgentObservation {
	cloned := *row
	cloned.Resources = slices.Clone(row.Resources)
	return &cloned
}

func (r *AgentObservationRepository) Import(_ context.Context, tenantID string, batch []*agentdomain.AgentObservation, maxObservations int) (agentports.ObservationImportCounts, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	byKey := map[[2]string]*agentdomain.AgentObservation{}
	count := 0
	for _, row := range r.rows {
		if row.TenantID == tenantID {
			byKey[[2]string{row.Issuer, row.Subject}] = row
			count++
		}
	}
	var counts agentports.ObservationImportCounts
	created := []*agentdomain.AgentObservation{}
	updates := map[string]*agentdomain.AgentObservation{}
	for _, item := range batch {
		key := [2]string{item.Issuer, item.Subject}
		existing, ok := byKey[key]
		if !ok {
			count++
			if count > maxObservations {
				return agentports.ObservationImportCounts{}, agentports.ErrObservationLimitExceeded
			}
			row := cloneObservation(item)
			byKey[key] = row
			created = append(created, row)
			counts.Created++
			continue
		}
		if !item.ObservedAt.After(existing.ObservedAt) {
			counts.Unchanged++
			continue
		}
		next := cloneObservation(existing)
		next.ExternalRegistrationID, next.DisplayName, next.ResponsibleCandidate = item.ExternalRegistrationID, item.DisplayName, item.ResponsibleCandidate
		next.Resources, next.ObservedAt, next.SourceTokenID = slices.Clone(item.Resources), item.ObservedAt, item.SourceTokenID
		byKey[key] = next
		updates[next.ID] = next
		counts.Updated++
	}
	for _, row := range created {
		r.rows[row.ID] = row
	}
	maps.Copy(r.rows, updates)
	return counts, nil
}

func (r *AgentObservationRepository) FindByID(_ context.Context, tenantID, id string) (*agentdomain.AgentObservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.rows[id]
	if !ok || row.TenantID != tenantID {
		return nil, nil
	}
	return cloneObservation(row), nil
}

func (r *AgentObservationRepository) filtered(tenantID string, filter agentports.ObservationFilter) []*agentdomain.AgentObservation {
	out := []*agentdomain.AgentObservation{}
	for _, row := range r.rows {
		if row.TenantID == tenantID && (filter.Decision == "" || row.Decision() == filter.Decision) {
			out = append(out, cloneObservation(row))
		}
	}
	return out
}

func observationKeyset(row *agentdomain.AgentObservation) (string, string) {
	return row.Subject, row.ID
}

func (r *AgentObservationRepository) ListPage(_ context.Context, tenantID string, filter agentports.ObservationFilter, afterSubject, afterID string, limit int) ([]*agentdomain.AgentObservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sharedmem.KeysetPage(r.filtered(tenantID, filter), observationKeyset, false, afterSubject, afterID, limit), nil
}

func (r *AgentObservationRepository) ListPageBefore(_ context.Context, tenantID string, filter agentports.ObservationFilter, beforeSubject, beforeID string, limit int) ([]*agentdomain.AgentObservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sharedmem.KeysetPageBefore(r.filtered(tenantID, filter), observationKeyset, false, beforeSubject, beforeID, limit), nil
}

func (r *AgentObservationRepository) Count(_ context.Context, tenantID string, filter agentports.ObservationFilter) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.filtered(tenantID, filter))), nil
}

func (r *AgentObservationRepository) ListConfirmed(_ context.Context, tenantID string) ([]*agentdomain.AgentObservation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := r.filtered(tenantID, agentports.ObservationFilter{Decision: agentdomain.ObservationConfirmed})
	slices.SortFunc(out, func(a, b *agentdomain.AgentObservation) int {
		if c := strings.Compare(a.Subject, b.Subject); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (r *AgentObservationRepository) Confirm(_ context.Context, tenantID, id, agentID, actorUserID string, at time.Time) (bool, error) {
	r.agents.mu.RLock()
	defer r.agents.mu.RUnlock()
	if _, ok := r.agents.agents[sharedmem.ScopedKey(tenantID, agentID, kernel.DefaultTenantID)]; !ok {
		return false, agentports.ErrObservationAgentMissing
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok || row.TenantID != tenantID || row.Decision() != agentdomain.ObservationPending {
		return false, nil
	}
	row.ConfirmedAgentID, row.DecidedAt, row.DecidedByUserID = &agentID, &at, &actorUserID
	return true, nil
}

func (r *AgentObservationRepository) Exclude(_ context.Context, tenantID, id, actorUserID string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok || row.TenantID != tenantID || row.Decision() != agentdomain.ObservationPending {
		return false, nil
	}
	row.Excluded, row.DecidedAt, row.DecidedByUserID = true, &at, &actorUserID
	return true, nil
}

func (r *AgentObservationRepository) Reopen(_ context.Context, tenantID, id string) (*agentdomain.AgentObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok || row.TenantID != tenantID {
		return nil, nil
	}
	previous := cloneObservation(row)
	row.Excluded, row.ConfirmedAgentID, row.DecidedAt, row.DecidedByUserID = false, nil, nil, nil
	return previous, nil
}

func (r *AgentObservationRepository) PurgeExpired(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var deleted int64
	for id, row := range r.rows {
		if row.ConfirmedAgentID == nil && row.ObservedAt.Before(cutoff) {
			delete(r.rows, id)
			deleted++
		}
	}
	return deleted, nil
}

// releaseAgent は削除した Agent への確認を解除する。呼び出し側が Agent のロックを持つ。
func (r *AgentObservationRepository) releaseAgent(tenantID, agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.TenantID == tenantID && row.ConfirmedAgentID != nil && *row.ConfirmedAgentID == agentID {
			row.ConfirmedAgentID, row.DecidedAt, row.DecidedByUserID = nil, nil, nil
		}
	}
}
