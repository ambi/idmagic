package db_memory

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

type correlationKey struct{ tenantID, connectorID, externalID string }

// CorrelationRepository は Connector ごとの User の対応をメモリに保存する。
type CorrelationRepository struct {
	mu           sync.Mutex
	correlations map[correlationKey]directoryports.UserCorrelation
}

var (
	_ ports.CorrelationRepository      = (*CorrelationRepository)(nil)
	_ directoryports.CorrelationWriter = (*CorrelationRepository)(nil)
)

func NewCorrelationRepository() *CorrelationRepository {
	return &CorrelationRepository{correlations: map[correlationKey]directoryports.UserCorrelation{}}
}

func (r *CorrelationRepository) SaveUserCorrelation(_ context.Context, c directoryports.UserCorrelation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.correlations[correlationKey{c.TenantID, c.ConnectorID, c.ExternalID}] = c
	return nil
}

func (r *CorrelationRepository) Find(_ context.Context, tenantID, connectorID, externalID string) (*directoryports.UserCorrelation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.correlations[correlationKey{tenantID, connectorID, externalID}]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (r *CorrelationRepository) MarkSeen(_ context.Context, tenantID, connectorID, externalID, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := correlationKey{tenantID, connectorID, externalID}
	if c, ok := r.correlations[k]; ok {
		c.LastSeenRunID = runID
		r.correlations[k] = c
	}
	return nil
}

func (r *CorrelationRepository) ListNotSeenIn(_ context.Context, tenantID, connectorID, runID string) ([]directoryports.UserCorrelation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []directoryports.UserCorrelation
	for k, c := range r.correlations {
		if k.tenantID == tenantID && k.connectorID == connectorID && c.LastSeenRunID != runID {
			result = append(result, c)
		}
	}
	slices.SortFunc(result, func(a, b directoryports.UserCorrelation) int { return cmp.Compare(a.ExternalID, b.ExternalID) })
	return result, nil
}

func (r *CorrelationRepository) CorrelatedUserIDs(_ context.Context, tenantID string, userIDs []string) (map[string]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := map[string]bool{}
	for k, c := range r.correlations {
		if k.tenantID == tenantID && slices.Contains(userIDs, c.UserID) {
			result[c.UserID] = true
		}
	}
	return result, nil
}

type runKey struct{ tenantID, runID string }

// IngestionRunRepository は取り込みの実行とバッチをメモリに保存する。
type IngestionRunRepository struct {
	mu      sync.Mutex
	runs    map[runKey]domain.IngestionRun
	batches map[runKey][]domain.IngestionBatch
}

var _ ports.IngestionRunRepository = (*IngestionRunRepository)(nil)

func NewIngestionRunRepository() *IngestionRunRepository {
	return &IngestionRunRepository{runs: map[runKey]domain.IngestionRun{}, batches: map[runKey][]domain.IngestionBatch{}}
}

func (r *IngestionRunRepository) Start(_ context.Context, run domain.IngestionRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, existing := range r.runs {
		if existing.TenantID == run.TenantID && existing.ConnectorID == run.ConnectorID && existing.Status == domain.RunRunning {
			existing.Status = domain.RunSuperseded
			r.runs[k] = existing
		}
	}
	r.runs[runKey{run.TenantID, run.ID}] = run
	return nil
}

func (r *IngestionRunRepository) Find(_ context.Context, tenantID, connectorID, runID string) (*domain.IngestionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[runKey{tenantID, runID}]
	if !ok || run.ConnectorID != connectorID {
		return nil, nil
	}
	return &run, nil
}

func (r *IngestionRunRepository) List(_ context.Context, tenantID, connectorID string, limit int) ([]domain.IngestionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := []domain.IngestionRun{}
	for _, run := range r.runs {
		if run.TenantID == tenantID && run.ConnectorID == connectorID {
			result = append(result, run)
		}
	}
	slices.SortFunc(result, func(a, b domain.IngestionRun) int {
		return cmp.Or(b.StartedAt.Compare(a.StartedAt), cmp.Compare(b.ID, a.ID))
	})
	return result[:min(limit, len(result))], nil
}

func (r *IngestionRunRepository) FindBatch(_ context.Context, tenantID, runID, batchID string) (*domain.IngestionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, batch := range r.batches[runKey{tenantID, runID}] {
		if batch.BatchID == batchID {
			return new(cloneBatch(batch)), nil
		}
	}
	return nil, nil
}

func (r *IngestionRunRepository) RecordBatch(_ context.Context, batch domain.IngestionBatch) (domain.IngestionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := runKey{batch.TenantID, batch.RunID}
	for _, recorded := range r.batches[k] {
		if recorded.BatchID == batch.BatchID {
			return cloneBatch(recorded), nil
		}
	}
	batch = cloneBatch(batch)
	r.batches[k] = append(r.batches[k], batch)
	if run, ok := r.runs[k]; ok {
		run.Counts = run.Counts.Add(batch.Results)
		r.runs[k] = run
	}
	return cloneBatch(batch), nil
}

func (r *IngestionRunRepository) ListBatches(_ context.Context, tenantID, runID string) ([]domain.IngestionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	batches := r.batches[runKey{tenantID, runID}]
	result := make([]domain.IngestionBatch, len(batches))
	for n, batch := range batches {
		result[n] = cloneBatch(batch)
	}
	return result, nil
}

func (r *IngestionRunRepository) Complete(_ context.Context, tenantID, runID string, completedAt time.Time, disabled int) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := runKey{tenantID, runID}
	run, ok := r.runs[k]
	if !ok || run.Status != domain.RunRunning {
		return false, nil
	}
	run.Status, run.CompletedAt, run.Counts.Disabled = domain.RunCompleted, &completedAt, disabled
	r.runs[k] = run
	return true, nil
}

func cloneBatch(batch domain.IngestionBatch) domain.IngestionBatch {
	batch.Results = slices.Clone(batch.Results)
	return batch
}
