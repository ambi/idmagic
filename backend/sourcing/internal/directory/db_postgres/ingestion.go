package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// CorrelationRepository は Connector ごとの User の対応を PostgreSQL に保存する。
type CorrelationRepository struct{ Pool sharedpg.DB }

var (
	_ ports.CorrelationRepository      = (*CorrelationRepository)(nil)
	_ directoryports.CorrelationWriter = (*CorrelationRepository)(nil)
)

func (r *CorrelationRepository) SaveUserCorrelation(ctx context.Context, c directoryports.UserCorrelation) error {
	return New(r.Pool).SaveDirectoryUserCorrelation(ctx, SaveDirectoryUserCorrelationParams{
		TenantID: c.TenantID, ConnectorID: c.ConnectorID, ExternalID: c.ExternalID, UserID: c.UserID, LastSeenRunID: c.LastSeenRunID,
	})
}

func (r *CorrelationRepository) Find(ctx context.Context, tenantID, connectorID, externalID string) (*directoryports.UserCorrelation, error) {
	row, err := New(r.Pool).FindDirectoryUserCorrelation(ctx, FindDirectoryUserCorrelationParams{
		TenantID: tenantID, ConnectorID: connectorID, ExternalID: externalID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return new(correlationToPort(row)), nil
}

func (r *CorrelationRepository) MarkSeen(ctx context.Context, tenantID, connectorID, externalID, runID string) error {
	return New(r.Pool).MarkDirectoryUserCorrelationSeen(ctx, MarkDirectoryUserCorrelationSeenParams{
		RunID: runID, TenantID: tenantID, ConnectorID: connectorID, ExternalID: externalID,
	})
}

func (r *CorrelationRepository) ListNotSeenIn(ctx context.Context, tenantID, connectorID, runID string) ([]directoryports.UserCorrelation, error) {
	rows, err := New(r.Pool).ListDirectoryUserCorrelationsNotSeenIn(ctx, ListDirectoryUserCorrelationsNotSeenInParams{
		TenantID: tenantID, ConnectorID: connectorID, RunID: runID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]directoryports.UserCorrelation, len(rows))
	for n, row := range rows {
		result[n] = correlationToPort(row)
	}
	return result, nil
}

func (r *CorrelationRepository) CorrelatedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error) {
	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		// 列は uuid なので、形式の外の ID は対応を持たないものとして問い合わせない。
		if isUUID(id) {
			ids = append(ids, id)
		}
	}
	result := map[string]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := New(r.Pool).ListDirectoryCorrelatedUserIDs(ctx, ListDirectoryCorrelatedUserIDsParams{TenantID: tenantID, UserIds: ids})
	if err != nil {
		return nil, err
	}
	for _, id := range rows {
		result[id] = true
	}
	return result, nil
}

func correlationToPort(row *DirectoryUserCorrelation) directoryports.UserCorrelation {
	return directoryports.UserCorrelation{
		TenantID: row.TenantID, ConnectorID: row.ConnectorID, ExternalID: row.ExternalID,
		UserID: row.UserID, LastSeenRunID: row.LastSeenRunID,
	}
}

// IngestionRunRepository は取り込みの実行とバッチを PostgreSQL に保存する。
type IngestionRunRepository struct{ Pool sharedpg.DB }

var _ ports.IngestionRunRepository = (*IngestionRunRepository)(nil)

func (r *IngestionRunRepository) Start(ctx context.Context, run domain.IngestionRun) error {
	return pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		q := New(tx)
		if err := q.SupersedeRunningDirectoryIngestionRuns(ctx, SupersedeRunningDirectoryIngestionRunsParams{
			TenantID: run.TenantID, ConnectorID: run.ConnectorID,
		}); err != nil {
			return err
		}
		return q.CreateDirectoryIngestionRun(ctx, CreateDirectoryIngestionRunParams{
			ID: run.ID, TenantID: run.TenantID, ConnectorID: run.ConnectorID,
			Mode: string(run.Mode), Status: string(run.Status), StartedAt: run.StartedAt,
		})
	})
}

func (r *IngestionRunRepository) Find(ctx context.Context, tenantID, connectorID, runID string) (*domain.IngestionRun, error) {
	if !isUUID(runID) {
		return nil, nil
	}
	row, err := New(r.Pool).FindDirectoryIngestionRun(ctx, FindDirectoryIngestionRunParams{TenantID: tenantID, ConnectorID: connectorID, ID: runID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return new(runToDomain(row)), nil
}

func (r *IngestionRunRepository) List(ctx context.Context, tenantID, connectorID string, limit int) ([]domain.IngestionRun, error) {
	rows, err := New(r.Pool).ListDirectoryIngestionRuns(ctx, ListDirectoryIngestionRunsParams{
		TenantID: tenantID, ConnectorID: connectorID, Limit: int32(limit), //nolint:gosec // 上限は呼び出し側の小さな定数。
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.IngestionRun, len(rows))
	for n, row := range rows {
		result[n] = runToDomain(row)
	}
	return result, nil
}

func (r *IngestionRunRepository) FindBatch(ctx context.Context, tenantID, runID, batchID string) (*domain.IngestionBatch, error) {
	row, err := New(r.Pool).FindDirectoryIngestionBatch(ctx, FindDirectoryIngestionBatchParams{TenantID: tenantID, RunID: runID, BatchID: batchID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	batch, err := batchToDomain(row)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// RecordBatch は、バッチの挿入と実行の件数の加算を同じトランザクションで確定する。
// 同じ ID のバッチが先に挿入されていれば、件数を足さずに先のバッチを返す。
func (r *IngestionRunRepository) RecordBatch(ctx context.Context, batch domain.IngestionBatch) (domain.IngestionBatch, error) {
	results, err := json.Marshal(batch.Results)
	if err != nil {
		return domain.IngestionBatch{}, err
	}
	err = pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		q := New(tx)
		inserted, err := q.InsertDirectoryIngestionBatch(ctx, InsertDirectoryIngestionBatchParams{
			TenantID: batch.TenantID, RunID: batch.RunID, BatchID: batch.BatchID,
			Digest: batch.Digest, Results: results, RecordedAt: batch.RecordedAt,
		})
		if err != nil || inserted == 0 {
			return err
		}
		counts := domain.RunCounts{}.Add(batch.Results)
		return q.AddDirectoryIngestionRunCounts(ctx, AddDirectoryIngestionRunCountsParams{
			Items: int32(counts.Items), Created: int32(counts.Created), Updated: int32(counts.Updated), //nolint:gosec // 1 バッチは 100 件まで。
			Unchanged: int32(counts.Unchanged), Deleted: int32(counts.Deleted), Failed: int32(counts.Failed), //nolint:gosec // 同上。
			TenantID: batch.TenantID, ID: batch.RunID,
		})
	})
	if err != nil {
		return domain.IngestionBatch{}, err
	}
	recorded, err := r.FindBatch(ctx, batch.TenantID, batch.RunID, batch.BatchID)
	if err != nil {
		return domain.IngestionBatch{}, err
	}
	if recorded == nil {
		return domain.IngestionBatch{}, errors.New("directory ingestion: the recorded batch disappeared")
	}
	return *recorded, nil
}

func (r *IngestionRunRepository) ListBatches(ctx context.Context, tenantID, runID string) ([]domain.IngestionBatch, error) {
	rows, err := New(r.Pool).ListDirectoryIngestionBatches(ctx, ListDirectoryIngestionBatchesParams{TenantID: tenantID, RunID: runID})
	if err != nil {
		return nil, err
	}
	result := make([]domain.IngestionBatch, len(rows))
	for n, row := range rows {
		if result[n], err = batchToDomain(row); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *IngestionRunRepository) Complete(ctx context.Context, tenantID, runID string, completedAt time.Time, disabled int) (bool, error) {
	changed, err := New(r.Pool).CompleteDirectoryIngestionRun(ctx, CompleteDirectoryIngestionRunParams{
		CompletedAt: instant(completedAt), Disabled: int32(disabled), TenantID: tenantID, ID: runID, //nolint:gosec // 対応の数は int32 に収まる。
	})
	return changed > 0, err
}

func runToDomain(row *DirectoryIngestionRun) domain.IngestionRun {
	return domain.IngestionRun{
		TenantID: row.TenantID, ID: row.ID, ConnectorID: row.ConnectorID,
		Mode: domain.IngestionMode(row.Mode), Status: domain.IngestionRunStatus(row.Status),
		StartedAt: row.StartedAt.UTC(), CompletedAt: pointer(row.CompletedAt),
		Counts: domain.RunCounts{
			Batches: int(row.Batches), Items: int(row.Items), Created: int(row.Created), Updated: int(row.Updated),
			Unchanged: int(row.Unchanged), Deleted: int(row.Deleted), Failed: int(row.Failed), Disabled: int(row.Disabled),
		},
	}
}

func batchToDomain(row *DirectoryIngestionBatch) (domain.IngestionBatch, error) {
	batch := domain.IngestionBatch{
		TenantID: row.TenantID, RunID: row.RunID, BatchID: row.BatchID, Digest: row.Digest, RecordedAt: row.RecordedAt.UTC(),
	}
	if err := json.Unmarshal(row.Results, &batch.Results); err != nil {
		return domain.IngestionBatch{}, err
	}
	return batch, nil
}
