package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	oauth2pg "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ConsentRepository は Consent を PostgreSQL に永続化する。クエリは sqlc 生成
// (wi-173, ADR-090); Pool は oauth2pg.DBTX を構造的に満たす。
type ConsentRepository struct{ Pool sharedpg.DB }

func consentFromRow(row *oauth2pg.Consent) (*domain.Consent, error) {
	c := &domain.Consent{
		UserID:    row.UserID,
		ClientID:  row.ClientID,
		GrantedAt: row.GrantedAt,
		ExpiresAt: row.ExpiresAt,
	}
	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		c.RevokedAt = &revokedAt
	}
	if err := json.Unmarshal(row.Scopes, &c.Scopes); err != nil {
		return nil, err
	}
	switch {
	case c.RevokedAt != nil:
		c.State = domain.ConsentRevoked
	case !time.Now().Before(c.ExpiresAt):
		c.State = domain.ConsentExpired
	default:
		c.State = domain.ConsentGranted
	}
	return c, nil
}

func (r *ConsentRepository) Find(ctx context.Context, tenantID, sub, clientID string) (*domain.Consent, error) {
	row, err := oauth2pg.New(r.Pool).GetConsent(ctx, oauth2pg.GetConsentParams{UserID: sub, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return consentFromRow(row)
}

func (r *ConsentRepository) FindAll(ctx context.Context, tenantID string) ([]*domain.Consent, error) {
	rows, err := oauth2pg.New(r.Pool).ListConsentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Consent, 0, len(rows))
	for _, row := range rows {
		c, err := consentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// ListPage implements ports.ConsentRepository.ListPage
// (wi-159, ADR-158): keyset pagination ordered by (user_id, client_id)
// ascending, strictly after the given keyset ("", "" for the first page).
func (r *ConsentRepository) ListPage(ctx context.Context, tenantID, afterUserID, afterClientID string, limit int) ([]*domain.Consent, error) {
	q := oauth2pg.New(r.Pool)
	var rows []*oauth2pg.Consent
	var err error
	if afterUserID == "" && afterClientID == "" {
		rows, err = q.ListConsentsByTenantPage(ctx, oauth2pg.ListConsentsByTenantPageParams{
			TenantID:  tenantID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListConsentsByTenantPageAfter(ctx, oauth2pg.ListConsentsByTenantPageAfterParams{
			TenantID:      tenantID,
			AfterUserID:   afterUserID,
			AfterClientID: afterClientID,
			PageLimit:     int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Consent, 0, len(rows))
	for _, row := range rows {
		c, err := consentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *ConsentRepository) ListPageBefore(ctx context.Context, tenantID, beforeUserID, beforeClientID string, limit int) ([]*domain.Consent, error) {
	rows, err := oauth2pg.New(r.Pool).ListConsentsByTenantPageBefore(ctx, oauth2pg.ListConsentsByTenantPageBeforeParams{
		TenantID: tenantID, BeforeUserID: beforeUserID, BeforeClientID: beforeClientID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*domain.Consent, 0, len(rows))
	for _, row := range rows {
		consent, err := consentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, consent)
	}
	return out, nil
}

func (r *ConsentRepository) Save(ctx context.Context, tenantID string, c *domain.Consent) error {
	scopes, err := json.Marshal(c.Scopes)
	if err != nil {
		return err
	}
	var revokedAt pgtype.Timestamptz
	if c.RevokedAt != nil {
		revokedAt = pgtype.Timestamptz{Time: *c.RevokedAt, Valid: true}
	}
	return oauth2pg.New(r.Pool).UpsertConsent(ctx, oauth2pg.UpsertConsentParams{
		UserID:    c.UserID,
		ClientID:  c.ClientID,
		Scopes:    scopes,
		GrantedAt: c.GrantedAt,
		ExpiresAt: c.ExpiresAt,
		RevokedAt: revokedAt,
	})
}

func (r *ConsentRepository) Revoke(ctx context.Context, tenantID, sub, clientID string) error {
	return oauth2pg.New(r.Pool).RevokeConsent(ctx, oauth2pg.RevokeConsentParams{UserID: sub, ClientID: clientID})
}

func (r *ConsentRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return oauth2pg.New(r.Pool).DeleteConsentsForSub(ctx, sub)
}
