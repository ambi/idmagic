package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/jackc/pgx/v5"
)

// SsfStreamRepository は SsfStream (ADR-057) を PostgreSQL に永続化する。
type SsfStreamRepository struct{ Pool sharedpg.DB }

func streamFromRow(row *SsfStream) (*ssdomain.SsfStream, error) {
	s := &ssdomain.SsfStream{
		ID: row.ID, TenantID: row.TenantID,
		Direction: ssdomain.SsfStreamDirection(row.Direction),
		Status:    ssdomain.SsfStreamStatus(row.Status),
		CreatedAt: row.CreatedAt, UpdatedAt: &row.UpdatedAt,
	}
	if err := json.Unmarshal(row.EventTypes, &s.EventTypes); err != nil {
		return nil, err
	}
	return s, s.Validate()
}

func (r *SsfStreamRepository) ListByTenant(ctx context.Context, tenantID string) ([]*ssdomain.SsfStream, error) {
	rows, err := New(r.Pool).ListSsfStreamsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*ssdomain.SsfStream, 0, len(rows))
	for _, row := range rows {
		s, err := streamFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *SsfStreamRepository) FindByID(ctx context.Context, tenantID, id string) (*ssdomain.SsfStream, error) {
	row, err := New(r.Pool).FindSsfStreamByID(ctx, FindSsfStreamByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return streamFromRow(row)
}

func (r *SsfStreamRepository) Save(ctx context.Context, s *ssdomain.SsfStream) error {
	eventTypes := s.EventTypes
	if eventTypes == nil {
		eventTypes = []ssdomain.CaepEventType{}
	}
	eventTypesJSON, err := json.Marshal(eventTypes)
	if err != nil {
		return err
	}
	updatedAt := s.CreatedAt
	if s.UpdatedAt != nil {
		updatedAt = *s.UpdatedAt
	}
	return New(r.Pool).SaveSsfStream(ctx, SaveSsfStreamParams{
		ID: s.ID, TenantID: s.TenantID, Direction: string(s.Direction),
		EventTypes: eventTypesJSON, Status: string(s.Status),
		CreatedAt: s.CreatedAt, UpdatedAt: updatedAt,
	})
}

func (r *SsfStreamRepository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteSsfStream(ctx, DeleteSsfStreamParams{TenantID: tenantID, ID: id})
}
