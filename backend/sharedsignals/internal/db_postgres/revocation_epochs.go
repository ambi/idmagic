package db_postgres

import (
	"context"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AgentRevocationEpochRepository は AgentRevocationEpoch を PostgreSQL に
// 永続化する。Advance は epoch の単調増加を SQL の条件付き ON CONFLICT で fail-closed に
// 強制する (WHERE EXCLUDED.epoch >= 既存 epoch でなければ更新自体が起きず 0 行になる)。
type AgentRevocationEpochRepository struct{ Pool sharedpg.DB }

func revocationEpochFromRow(row *AgentRevocationEpoch) *ssdomain.AgentRevocationEpoch {
	return &ssdomain.AgentRevocationEpoch{
		AgentID: row.AgentID, TenantID: row.TenantID, Epoch: row.Epoch,
		Reason: ssdomain.RevocationReason(row.Reason), AdvancedAt: row.AdvancedAt,
		SourceEventID: textPtrOrNil(row.SourceEventID),
	}
}

func (r *AgentRevocationEpochRepository) FindByAgent(ctx context.Context, tenantID, agentID string) (*ssdomain.AgentRevocationEpoch, error) {
	row, err := New(r.Pool).FindAgentRevocationEpoch(ctx, FindAgentRevocationEpochParams{TenantID: tenantID, AgentID: agentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return revocationEpochFromRow(row), nil
}

func (r *AgentRevocationEpochRepository) Advance(ctx context.Context, epoch ssdomain.AgentRevocationEpoch) error {
	_, err := New(r.Pool).AdvanceAgentRevocationEpoch(ctx, AdvanceAgentRevocationEpochParams{
		AgentID: epoch.AgentID, TenantID: epoch.TenantID, Epoch: epoch.Epoch,
		Reason: string(epoch.Reason), AdvancedAt: epoch.AdvancedAt,
		SourceEventID: textOrNilPtr(epoch.SourceEventID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ssdomain.ErrEpochNotAdvancing
	}
	return err
}

// UserRevocationEpochRepository は UserRevocationEpoch を PostgreSQL に永続化する。
// 単調増加は AgentRevocationEpochRepository と同じ条件付き ON CONFLICT で強制する。
type UserRevocationEpochRepository struct{ Pool sharedpg.DB }

// FindByUser は UUID でない userID を未失効として扱う。トークンの sub から届く値で、
// users の行を指せないものは失効の記録も持たないからである。存在の判定は呼び出し側が User で行う。
func (r *UserRevocationEpochRepository) FindByUser(ctx context.Context, tenantID, userID string) (*ssdomain.UserRevocationEpoch, error) {
	if uuid.Validate(userID) != nil {
		return nil, nil //nolint:nilerr // 上のコメントのとおり
	}
	row, err := New(r.Pool).FindUserRevocationEpoch(ctx, FindUserRevocationEpochParams{TenantID: tenantID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ssdomain.UserRevocationEpoch{
		UserID: row.UserID, TenantID: row.TenantID, Epoch: row.Epoch,
		Reason: ssdomain.RevocationReason(row.Reason), AdvancedAt: row.AdvancedAt,
		SourceEventID: textPtrOrNil(row.SourceEventID),
	}, nil
}

func (r *UserRevocationEpochRepository) Advance(ctx context.Context, epoch ssdomain.UserRevocationEpoch) error {
	_, err := New(r.Pool).AdvanceUserRevocationEpoch(ctx, AdvanceUserRevocationEpochParams{
		UserID: epoch.UserID, TenantID: epoch.TenantID, Epoch: epoch.Epoch,
		Reason: string(epoch.Reason), AdvancedAt: epoch.AdvancedAt,
		SourceEventID: textOrNilPtr(epoch.SourceEventID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ssdomain.ErrEpochNotAdvancing
	}
	return err
}
