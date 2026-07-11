// Package eventlog is the PostgreSQL adapter for event_logs / event_deliveries
// (ADR-094, wi-184 T002/T003). It satisfies backend/shared/eventlog.Recorder.
package eventlog

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/eventlog/sqlcgen"
	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
)

// Repository writes/reads event_logs / event_deliveries. Pool is the
// standalone connection (a pool) used when no transaction is active. When
// the given ctx carries a pgx.Tx (backend/shared/adapters/persistence/postgres.WithTx,
// set by Runner.Run), every method uses that transaction instead — this is
// how business-mutation repositories and the event log recorder share one
// PostgreSQL transaction (ADR-094 EventLogAtomicWithBusinessState, wi-184
// T003) without Repository owning commit/rollback itself.
type Repository struct {
	Pool sqlcgen.DBTX
}

var _ sharedeventlog.Recorder = (*Repository)(nil)

func New(pool sqlcgen.DBTX) *Repository {
	return &Repository{Pool: pool}
}

func (r *Repository) db(ctx context.Context) sqlcgen.DBTX {
	if tx, ok := sharedpg.TxFromContext(ctx); ok {
		return tx
	}
	return r.Pool
}

func (r *Repository) Append(ctx context.Context, rec sharedeventlog.Record) error {
	payload := rec.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.db(ctx)).InsertEventLog(ctx, sqlcgen.InsertEventLogParams{
		EventID:        rec.EventID,
		TenantID:       rec.TenantID,
		Type:           rec.Type,
		Classification: string(rec.Classification),
		Actor:          toText(rec.Actor),
		Subject:        toText(rec.Subject),
		CorrelationID:  rec.CorrelationID,
		OccurredAt:     rec.OccurredAt,
		Payload:        payloadJSON,
	})
}

func (r *Repository) AppendDelivery(ctx context.Context, eventID string) error {
	return sqlcgen.New(r.db(ctx)).InsertEventDelivery(ctx, eventID)
}

// FindByID returns nil, nil when eventID does not exist. Used by contract
// tests and by the transaction-bound emit wiring (wi-184 T003) to confirm a
// write landed.
func (r *Repository) FindByID(ctx context.Context, eventID string) (*sharedeventlog.Record, error) {
	row, err := sqlcgen.New(r.db(ctx)).GetEventLogByID(ctx, eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return nil, err
	}
	return &sharedeventlog.Record{
		EventID:        row.EventID,
		TenantID:       row.TenantID,
		Type:           row.Type,
		Classification: sharedeventlog.Classification(row.Classification),
		Actor:          row.Actor.String,
		Subject:        row.Subject.String,
		CorrelationID:  row.CorrelationID,
		OccurredAt:     row.OccurredAt,
		Payload:        payload,
	}, nil
}

// FindDeliveryByID returns nil, nil when eventID has no event_deliveries row.
func (r *Repository) FindDeliveryByID(ctx context.Context, eventID string) (*sharedeventlog.Delivery, error) {
	row, err := sqlcgen.New(r.db(ctx)).GetEventDeliveryByID(ctx, eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := &sharedeventlog.Delivery{
		EventID:   row.EventID,
		Status:    sharedeventlog.DeliveryStatus(row.Status),
		Attempts:  int(row.Attempts),
		LastError: row.LastError.String,
		UpdatedAt: row.UpdatedAt,
	}
	if row.DeliveredAt.Valid {
		t := row.DeliveredAt.Time
		d.DeliveredAt = &t
	}
	return d, nil
}

func toText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
