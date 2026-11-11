package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/ports"
)

// EntraFederationObservationRepository は観測記録の PostgreSQL の保存先である。
type EntraFederationObservationRepository struct{ Pool sharedpg.DB }

var _ ports.EntraFederationObservationRepository = (*EntraFederationObservationRepository)(nil)

func observationRecordFromRow(row *WsfedEntraFederationObservation) (*domain.EntraFederationObservationRecord, error) {
	record := &domain.EntraFederationObservationRecord{
		TenantID: row.TenantID, Domain: row.Domain, ExpectedFingerprint: row.ExpectedFingerprint,
		SigningCertificateThumbprint:     row.SigningCertificateThumbprint,
		NextSigningCertificateThumbprint: row.NextSigningCertificateThumbprint,
		Matched:                          row.Matched, CheckedAt: row.CheckedAt,
	}
	if err := json.Unmarshal(row.Observation, &record.Observation); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.Differences, &record.Differences); err != nil {
		return nil, err
	}
	record.Observation.RetrievedAt = row.RetrievedAt
	return record, nil
}

func (r *EntraFederationObservationRepository) Find(ctx context.Context, tenantID, domainName string) (*domain.EntraFederationObservationRecord, error) {
	row, err := New(r.Pool).GetEntraFederationObservation(ctx, GetEntraFederationObservationParams{TenantID: tenantID, Domain: domainName})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return observationRecordFromRow(row)
}

func (r *EntraFederationObservationRepository) ListAll(ctx context.Context, tenantID string) ([]*domain.EntraFederationObservationRecord, error) {
	rows, err := New(r.Pool).ListEntraFederationObservations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.EntraFederationObservationRecord, 0, len(rows))
	for _, row := range rows {
		record, err := observationRecordFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (r *EntraFederationObservationRepository) Save(ctx context.Context, record domain.EntraFederationObservationRecord) error {
	observation, err := json.Marshal(record.Observation)
	if err != nil {
		return err
	}
	differences := record.Differences
	if differences == nil {
		differences = []domain.EntraSettingDifference{}
	}
	differencesJSON, err := json.Marshal(differences)
	if err != nil {
		return err
	}
	saved, err := New(r.Pool).SaveEntraFederationObservation(ctx, SaveEntraFederationObservationParams{
		TenantID: record.TenantID, Domain: record.Domain, Observation: observation,
		ExpectedFingerprint:              record.ExpectedFingerprint,
		SigningCertificateThumbprint:     record.SigningCertificateThumbprint,
		NextSigningCertificateThumbprint: record.NextSigningCertificateThumbprint,
		Matched:                          record.Matched, Differences: differencesJSON,
		RetrievedAt: record.Observation.RetrievedAt, CheckedAt: record.CheckedAt,
	})
	if err != nil {
		return err
	}
	if saved == 0 {
		return ports.ErrEntraObservationOutdated
	}
	return nil
}

func (r *EntraFederationObservationRepository) Delete(ctx context.Context, tenantID, domainName string) error {
	return New(r.Pool).DeleteEntraFederationObservation(ctx, DeleteEntraFederationObservationParams{TenantID: tenantID, Domain: domainName})
}
