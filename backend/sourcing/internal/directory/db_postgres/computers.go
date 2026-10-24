package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// ComputerRepository は Connector ごとのコンピューターを PostgreSQL に保存する。有効な記録の間の
// SID と主体名の一意性は、部分一意索引が守る。
type ComputerRepository struct{ Pool sharedpg.DB }

var _ ports.ComputerRepository = (*ComputerRepository)(nil)

// sqlstateUniqueViolation は PostgreSQL の unique_violation の SQLSTATE である。
const sqlstateUniqueViolation = "23505"

func (r *ComputerRepository) Find(ctx context.Context, tenantID, connectorID, externalID string) (*domain.Computer, error) {
	return foundComputer(New(r.Pool).FindDirectoryComputer(ctx, FindDirectoryComputerParams{
		TenantID: tenantID, ConnectorID: connectorID, ExternalID: externalID,
	}))
}

func (r *ComputerRepository) FindEnabledHolders(ctx context.Context, tenantID, objectSID, principalKey string) ([]domain.Computer, error) {
	rows, err := New(r.Pool).ListEnabledDirectoryComputerHolders(ctx, ListEnabledDirectoryComputerHoldersParams{
		TenantID: tenantID, ObjectSid: objectSID, PrincipalKey: principalKey,
	})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Computer, len(rows))
	for n, row := range rows {
		result[n] = computerToDomain(row)
	}
	return result, nil
}

func (r *ComputerRepository) Save(ctx context.Context, c domain.Computer) error {
	err := New(r.Pool).SaveDirectoryComputer(ctx, SaveDirectoryComputerParams{
		TenantID: c.TenantID, ConnectorID: c.ConnectorID, ExternalID: c.ExternalID,
		SamAccountName: c.SAMAccountName, ObjectSid: c.ObjectSID, Domain: c.Domain, DnsHostName: c.DNSHostName,
		PrincipalKey: c.PrincipalKey(), Enabled: c.Enabled, LastSeenRunID: c.LastSeenRunID, UpdatedAt: c.UpdatedAt,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == sqlstateUniqueViolation {
		return domain.ErrComputerConflict
	}
	return err
}

func (r *ComputerRepository) MarkSeen(ctx context.Context, tenantID, connectorID, externalID, runID string) error {
	return New(r.Pool).MarkDirectoryComputerSeen(ctx, MarkDirectoryComputerSeenParams{
		RunID: runID, TenantID: tenantID, ConnectorID: connectorID, ExternalID: externalID,
	})
}

func (r *ComputerRepository) DisableNotSeenIn(ctx context.Context, tenantID, connectorID, runID string, at time.Time) (int, error) {
	disabled, err := New(r.Pool).DisableDirectoryComputersNotSeenIn(ctx, DisableDirectoryComputersNotSeenInParams{
		UpdatedAt: at, TenantID: tenantID, ConnectorID: connectorID, RunID: runID,
	})
	return int(disabled), err
}

func (r *ComputerRepository) FindEnabledBySID(ctx context.Context, tenantID, objectSID string) (*domain.Computer, error) {
	return foundComputer(New(r.Pool).FindEnabledDirectoryComputerBySID(ctx, FindEnabledDirectoryComputerBySIDParams{
		TenantID: tenantID, ObjectSid: objectSID,
	}))
}

func (r *ComputerRepository) FindEnabledByPrincipal(ctx context.Context, tenantID, samAccountName, realm string) (*domain.Computer, error) {
	return foundComputer(New(r.Pool).FindEnabledDirectoryComputerByPrincipal(ctx, FindEnabledDirectoryComputerByPrincipalParams{
		TenantID: tenantID, PrincipalKey: domain.PrincipalKey(samAccountName, realm),
	}))
}

func foundComputer(row *DirectoryComputer, err error) (*domain.Computer, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return new(computerToDomain(row)), nil
}

func computerToDomain(row *DirectoryComputer) domain.Computer {
	return domain.Computer{
		TenantID: row.TenantID, ConnectorID: row.ConnectorID,
		ExternalID: row.ExternalID, Enabled: row.Enabled, SAMAccountName: row.SamAccountName,
		ObjectSID: row.ObjectSid, Domain: row.Domain, DNSHostName: row.DnsHostName,
		LastSeenRunID: row.LastSeenRunID, UpdatedAt: row.UpdatedAt.UTC(),
	}
}
