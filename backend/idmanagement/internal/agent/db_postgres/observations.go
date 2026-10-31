package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const pgForeignKeyViolation = "23503"

func isUUID(value string) bool { return uuid.Validate(value) == nil }

// AgentObservationRepository は外部の観測を PostgreSQL に永続化する。confirmed_agent_id は
// agents への ON DELETE SET NULL の外部キーなので、Agent の削除と同じ確定で確認が解除される。
type AgentObservationRepository struct{ Pool sharedpg.DB }

func observationFromRow(row *AgentObservation) (*agentdomain.AgentObservation, error) {
	observation := &agentdomain.AgentObservation{
		ID: row.ID, TenantID: row.TenantID, Issuer: row.Issuer, Subject: row.Subject,
		ObservedAt: row.ObservedAt, SourceTokenID: row.SourceTokenID, Excluded: row.Excluded, CreatedAt: row.CreatedAt,
	}
	for _, field := range []struct {
		from pgtype.Text
		to   **string
	}{
		{row.ExternalRegistrationID, &observation.ExternalRegistrationID},
		{row.DisplayName, &observation.DisplayName},
		{row.ResponsibleCandidate, &observation.ResponsibleCandidate},
		{row.DecidedByUserID, &observation.DecidedByUserID},
	} {
		if field.from.Valid {
			value := field.from.String
			*field.to = &value
		}
	}
	if row.ConfirmedAgentID.Valid {
		agentID := uuid.UUID(row.ConfirmedAgentID.Bytes).String()
		observation.ConfirmedAgentID = &agentID
	}
	if row.DecidedAt.Valid {
		decidedAt := row.DecidedAt.Time
		observation.DecidedAt = &decidedAt
	}
	if err := json.Unmarshal(row.Resources, &observation.Resources); err != nil {
		return nil, err
	}
	return observation, nil
}

func observationsFromRows(rows []*AgentObservation) ([]*agentdomain.AgentObservation, error) {
	out := make([]*agentdomain.AgentObservation, 0, len(rows))
	for _, row := range rows {
		observation, err := observationFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, observation)
	}
	return out, nil
}

// Import は同じテナントの取り込みを助言ロックで直列にし、件数の上限をバッチの途中でも確かめる。
func (r *AgentObservationRepository) Import(ctx context.Context, tenantID string, batch []*agentdomain.AgentObservation, maxObservations int) (agentports.ObservationImportCounts, error) {
	var counts agentports.ObservationImportCounts
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return counts, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.LockAgentObservationsTenant(ctx, tenantID); err != nil {
		return counts, err
	}
	total, err := q.CountAgentObservations(ctx, CountAgentObservationsParams{TenantID: tenantID})
	if err != nil {
		return counts, err
	}
	for _, item := range batch {
		resources, err := json.Marshal(item.Resources)
		if err != nil {
			return counts, err
		}
		inserted, err := q.UpsertAgentObservation(ctx, UpsertAgentObservationParams{
			ID: item.ID, TenantID: tenantID, Issuer: item.Issuer, Subject: item.Subject,
			ExternalRegistrationID: textOrNil(item.ExternalRegistrationID), DisplayName: textOrNil(item.DisplayName),
			Resources: resources, ResponsibleCandidate: textOrNil(item.ResponsibleCandidate),
			ObservedAt: item.ObservedAt, SourceTokenID: item.SourceTokenID, CreatedAt: item.CreatedAt,
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			counts.Unchanged++
		case err != nil:
			return agentports.ObservationImportCounts{}, err
		case inserted:
			counts.Created++
			total++
			if total > int64(maxObservations) {
				return agentports.ObservationImportCounts{}, agentports.ErrObservationLimitExceeded
			}
		default:
			counts.Updated++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return agentports.ObservationImportCounts{}, err
	}
	return counts, nil
}

func (r *AgentObservationRepository) FindByID(ctx context.Context, tenantID, id string) (*agentdomain.AgentObservation, error) {
	if !isUUID(id) {
		return nil, nil
	}
	row, err := New(r.Pool).FindAgentObservationByID(ctx, FindAgentObservationByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return observationFromRow(row)
}

func (r *AgentObservationRepository) ListPage(ctx context.Context, tenantID string, filter agentports.ObservationFilter, afterSubject, afterID string, limit int) ([]*agentdomain.AgentObservation, error) {
	rows, err := New(r.Pool).ListAgentObservationsPage(ctx, ListAgentObservationsPageParams{
		TenantID: tenantID, Decision: string(filter.Decision), AfterSubject: afterSubject, AfterID: afterID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	return observationsFromRows(rows)
}

func (r *AgentObservationRepository) ListPageBefore(ctx context.Context, tenantID string, filter agentports.ObservationFilter, beforeSubject, beforeID string, limit int) ([]*agentdomain.AgentObservation, error) {
	rows, err := New(r.Pool).ListAgentObservationsPageBefore(ctx, ListAgentObservationsPageBeforeParams{
		TenantID: tenantID, Decision: string(filter.Decision), BeforeSubject: beforeSubject, BeforeID: beforeID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	return observationsFromRows(rows)
}

func (r *AgentObservationRepository) Count(ctx context.Context, tenantID string, filter agentports.ObservationFilter) (int64, error) {
	return New(r.Pool).CountAgentObservations(ctx, CountAgentObservationsParams{TenantID: tenantID, Decision: string(filter.Decision)})
}

func (r *AgentObservationRepository) ListConfirmed(ctx context.Context, tenantID string) ([]*agentdomain.AgentObservation, error) {
	rows, err := New(r.Pool).ListConfirmedAgentObservations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return observationsFromRows(rows)
}

func (r *AgentObservationRepository) Confirm(ctx context.Context, tenantID, id, agentID, actorUserID string, at time.Time) (bool, error) {
	agentUUID, err := uuid.Parse(agentID)
	if err != nil {
		return false, agentports.ErrObservationAgentMissing
	}
	rows, err := New(r.Pool).ConfirmAgentObservation(ctx, ConfirmAgentObservationParams{
		AgentID: pgtype.UUID{Bytes: agentUUID, Valid: true}, DecidedAt: pgtype.Timestamptz{Time: at, Valid: true},
		DecidedByUserID: pgtype.Text{String: actorUserID, Valid: true}, TenantID: tenantID, ID: id,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return false, agentports.ErrObservationAgentMissing
	}
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *AgentObservationRepository) Exclude(ctx context.Context, tenantID, id, actorUserID string, at time.Time) (bool, error) {
	rows, err := New(r.Pool).ExcludeAgentObservation(ctx, ExcludeAgentObservationParams{
		DecidedAt:       pgtype.Timestamptz{Time: at, Valid: true},
		DecidedByUserID: pgtype.Text{String: actorUserID, Valid: true}, TenantID: tenantID, ID: id,
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *AgentObservationRepository) Reopen(ctx context.Context, tenantID, id string) (*agentdomain.AgentObservation, error) {
	if !isUUID(id) {
		return nil, nil
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	row, err := q.FindAgentObservationByIDForUpdate(ctx, FindAgentObservationByIDForUpdateParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	previous, err := observationFromRow(row)
	if err != nil {
		return nil, err
	}
	if err := q.ReopenAgentObservation(ctx, ReopenAgentObservationParams{TenantID: tenantID, ID: id}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return previous, nil
}

func (r *AgentObservationRepository) PurgeExpired(ctx context.Context, cutoff time.Time) (int64, error) {
	return New(r.Pool).PurgeExpiredAgentObservations(ctx, cutoff)
}
