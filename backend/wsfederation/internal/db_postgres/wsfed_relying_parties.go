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

var _ ports.WsFedRelyingPartyRepository = (*WsFedRelyingPartyRepository)(nil)

// WsFedRelyingPartyRepository は WS-Federation RP trust の PostgreSQL 実装。
// クエリは sqlc 生成で管理する (wi-174)。
type WsFedRelyingPartyRepository struct{ Pool sharedpg.DB }

func relyingPartyFromRow(row *WsfedRelyingParty) (*domain.WsFedRelyingParty, error) {
	rp := &domain.WsFedRelyingParty{
		TenantID: row.TenantID, Wtrealm: row.Wtrealm, DisplayName: row.DisplayName,
		ApplicationID: sharedpg.UUIDString(row.ApplicationID),
		Audience:      row.Audience, TokenType: domain.WsFedTokenType(row.TokenType),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal(row.ReplyUrls, &rp.ReplyURLs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.ClaimPolicy, &rp.ClaimPolicy); err != nil {
		return nil, err
	}
	if len(row.EntraProfile) > 0 && string(row.EntraProfile) != "null" {
		profile := &rp.EntraProfile
		if err := json.Unmarshal(row.EntraProfile, profile); err != nil {
			return nil, err
		}
	}
	return rp, nil
}

func (r *WsFedRelyingPartyRepository) FindByWtrealm(ctx context.Context, tenantID, wtrealm string) (*domain.WsFedRelyingParty, error) {
	row, err := New(r.Pool).GetWsFedRelyingParty(ctx, GetWsFedRelyingPartyParams{TenantID: tenantID, Wtrealm: wtrealm})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return relyingPartyFromRow(row)
}

func (r *WsFedRelyingPartyRepository) ListAll(ctx context.Context, tenantID string) ([]*domain.WsFedRelyingParty, error) {
	rows, err := New(r.Pool).ListWsFedRelyingPartiesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.WsFedRelyingParty, 0, len(rows))
	for _, row := range rows {
		rp, err := relyingPartyFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, nil
}

func (r *WsFedRelyingPartyRepository) Save(ctx context.Context, rp *domain.WsFedRelyingParty) error {
	return writeRelyingParty(ctx, r.Pool, rp)
}

// Update はトランザクションの中で行を FOR UPDATE で固定して読み直し、change の結果を当てる。
// 行がなければ固定するものがないので、change に nil を渡し、作成は Save に任せる。
func (r *WsFedRelyingPartyRepository) Update(ctx context.Context, tenantID, wtrealm string, change ports.RelyingPartyChange) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current *domain.WsFedRelyingParty
	row, err := New(tx).GetWsFedRelyingPartyForUpdate(ctx, GetWsFedRelyingPartyForUpdateParams{TenantID: tenantID, Wtrealm: wtrealm})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return err
	default:
		if current, err = relyingPartyFromRow(row); err != nil {
			return err
		}
	}
	outcome, err := change(current)
	if err != nil {
		return err
	}
	switch {
	case outcome == ports.RelyingPartySaved && current != nil:
		err = writeRelyingParty(ctx, tx, current)
	case outcome == ports.RelyingPartyDeleted:
		err = New(tx).DeleteWsFedRelyingParty(ctx, DeleteWsFedRelyingPartyParams{TenantID: tenantID, Wtrealm: wtrealm})
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func writeRelyingParty(ctx context.Context, db DBTX, rp *domain.WsFedRelyingParty) error {
	replyURLs, err := json.Marshal(rp.ReplyURLs)
	if err != nil {
		return err
	}
	claimPolicy, err := json.Marshal(rp.ClaimPolicy)
	if err != nil {
		return err
	}
	var entraProfile []byte
	if rp.EntraProfile != nil {
		entraProfile, err = json.Marshal(rp.EntraProfile)
		if err != nil {
			return err
		}
	}
	return New(db).UpsertWsFedRelyingParty(ctx, UpsertWsFedRelyingPartyParams{
		TenantID: rp.TenantID, Wtrealm: rp.Wtrealm, DisplayName: rp.DisplayName, ReplyUrls: replyURLs,
		Audience: rp.Audience, TokenType: string(rp.TokenType), ClaimPolicy: claimPolicy, EntraProfile: entraProfile,
		CreatedAt: rp.CreatedAt, UpdatedAt: rp.UpdatedAt,
	})
}

func (r *WsFedRelyingPartyRepository) Delete(ctx context.Context, tenantID, wtrealm string) error {
	return New(r.Pool).DeleteWsFedRelyingParty(ctx, DeleteWsFedRelyingPartyParams{TenantID: tenantID, Wtrealm: wtrealm})
}
