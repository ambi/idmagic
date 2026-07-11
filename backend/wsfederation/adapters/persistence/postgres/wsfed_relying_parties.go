package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

// WsFedRelyingPartyRepository は WS-Federation RP trust の PostgreSQL 実装。
// クエリは sqlc 生成で管理する (wi-174, ADR-090)。
type WsFedRelyingPartyRepository struct{ Pool sharedpg.DB }

func relyingPartyFromRow(row *sqlcgen.WsfedRelyingParty) (*domain.WsFedRelyingParty, error) {
	rp := &domain.WsFedRelyingParty{
		TenantID: row.TenantID, Wtrealm: row.Wtrealm, DisplayName: row.DisplayName,
		Audience: row.Audience, TokenType: domain.WsFedTokenType(row.TokenType),
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
	row, err := sqlcgen.New(r.Pool).GetWsFedRelyingParty(ctx, sqlcgen.GetWsFedRelyingPartyParams{TenantID: tenantID, Wtrealm: wtrealm})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return relyingPartyFromRow(row)
}

func (r *WsFedRelyingPartyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.WsFedRelyingParty, error) {
	rows, err := sqlcgen.New(r.Pool).ListWsFedRelyingPartiesByTenant(ctx, tenantID)
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
	return sqlcgen.New(r.Pool).UpsertWsFedRelyingParty(ctx, sqlcgen.UpsertWsFedRelyingPartyParams{
		TenantID: rp.TenantID, Wtrealm: rp.Wtrealm, DisplayName: rp.DisplayName, ReplyUrls: replyURLs,
		Audience: rp.Audience, TokenType: string(rp.TokenType), ClaimPolicy: claimPolicy, EntraProfile: entraProfile,
		CreatedAt: rp.CreatedAt, UpdatedAt: rp.UpdatedAt,
	})
}

func (r *WsFedRelyingPartyRepository) Delete(ctx context.Context, tenantID, wtrealm string) error {
	return sqlcgen.New(r.Pool).DeleteWsFedRelyingParty(ctx, sqlcgen.DeleteWsFedRelyingPartyParams{TenantID: tenantID, Wtrealm: wtrealm})
}
