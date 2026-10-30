package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2pg "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ConsentRepository は Consent を PostgreSQL に永続化する。クエリは sqlc 生成
// (wi-173); Pool は oauth2pg.DBTX を構造的に満たす。
type ConsentRepository struct{ Pool sharedpg.DB }

// consentRow は、問い合わせごとに生成される行の型に共通する列である。
type consentRow struct {
	user                  string
	clientID, agentID     pgtype.UUID
	scopes                []byte
	grantedAt, expiresAt  time.Time
	revokedAt, lastUsedAt pgtype.Timestamptz
}

func uuidText(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return value.String()
}

func uuidParam(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(value)
	return id, err
}

func consentFromRow(row consentRow) (*domain.Consent, error) {
	c := &domain.Consent{
		UserID:    row.user,
		ClientID:  uuidText(row.clientID),
		AgentID:   uuidText(row.agentID),
		GrantedAt: row.grantedAt,
		ExpiresAt: row.expiresAt,
	}
	if row.revokedAt.Valid {
		revokedAt := row.revokedAt.Time
		c.RevokedAt = &revokedAt
	}
	if row.lastUsedAt.Valid {
		lastUsedAt := row.lastUsedAt.Time
		c.LastUsedAt = &lastUsedAt
	}
	if err := json.Unmarshal(row.scopes, &c.Scopes); err != nil {
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

func consentsFromRows[R any](rows []*R, toRow func(*R) consentRow) ([]*domain.Consent, error) {
	out := make([]*domain.Consent, 0, len(rows))
	for _, row := range rows {
		c, err := consentFromRow(toRow(row))
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *ConsentRepository) Find(ctx context.Context, tenantID, sub, clientID string) (*domain.Consent, error) {
	client, err := uuidParam(clientID)
	if err != nil {
		return nil, err
	}
	row, err := oauth2pg.New(r.Pool).GetConsent(ctx, oauth2pg.GetConsentParams{UserID: sub, ClientID: client})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return consentFromRow(consentRow{
		user: row.UserID, clientID: row.ClientID, scopes: row.Scopes,
		grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt,
	})
}

func (r *ConsentRepository) FindAll(ctx context.Context, tenantID string) ([]*domain.Consent, error) {
	rows, err := oauth2pg.New(r.Pool).ListConsentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return consentsFromRows(rows, func(row *oauth2pg.ListConsentsByTenantRow) consentRow {
		return consentRow{
			user: row.UserID, clientID: row.ClientID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt,
		}
	})
}

// ListPage implements ports.ConsentRepository.ListPage
// (wi-159): keyset pagination ordered by (user_id, client_id)
// ascending, strictly after the given keyset ("", "" for the first page).
func (r *ConsentRepository) ListPage(ctx context.Context, tenantID, afterUserID, afterClientID string, limit int) ([]*domain.Consent, error) {
	q := oauth2pg.New(r.Pool)
	if afterUserID == "" && afterClientID == "" {
		rows, err := q.ListConsentsByTenantPage(ctx, oauth2pg.ListConsentsByTenantPageParams{
			TenantID:  tenantID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
		if err != nil {
			return nil, err
		}
		return consentsFromRows(rows, func(row *oauth2pg.ListConsentsByTenantPageRow) consentRow {
			return consentRow{
				user: row.UserID, clientID: row.ClientID, scopes: row.Scopes,
				grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt,
			}
		})
	}
	rows, err := q.ListConsentsByTenantPageAfter(ctx, oauth2pg.ListConsentsByTenantPageAfterParams{
		TenantID:      tenantID,
		AfterUserID:   afterUserID,
		AfterClientID: afterClientID,
		PageLimit:     int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	return consentsFromRows(rows, func(row *oauth2pg.ListConsentsByTenantPageAfterRow) consentRow {
		return consentRow{
			user: row.UserID, clientID: row.ClientID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt,
		}
	})
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
	return consentsFromRows(rows, func(row *oauth2pg.ListConsentsByTenantPageBeforeRow) consentRow {
		return consentRow{
			user: row.UserID, clientID: row.ClientID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt,
		}
	})
}

func (r *ConsentRepository) Save(ctx context.Context, tenantID string, c *domain.Consent) error {
	scopes, err := json.Marshal(c.Scopes)
	if err != nil {
		return err
	}
	client, err := uuidParam(c.ClientID)
	if err != nil {
		return err
	}
	var revokedAt pgtype.Timestamptz
	if c.RevokedAt != nil {
		revokedAt = pgtype.Timestamptz{Time: *c.RevokedAt, Valid: true}
	}
	return oauth2pg.New(r.Pool).UpsertConsent(ctx, oauth2pg.UpsertConsentParams{
		UserID:    c.UserID,
		ClientID:  client,
		Scopes:    scopes,
		GrantedAt: c.GrantedAt,
		ExpiresAt: c.ExpiresAt,
		RevokedAt: revokedAt,
	})
}

func (r *ConsentRepository) Revoke(ctx context.Context, tenantID, sub, clientID string) error {
	client, err := uuidParam(clientID)
	if err != nil {
		return err
	}
	return oauth2pg.New(r.Pool).RevokeConsent(ctx, oauth2pg.RevokeConsentParams{UserID: sub, ClientID: client})
}

func (r *ConsentRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return oauth2pg.New(r.Pool).DeleteConsentsForSub(ctx, sub)
}

// FindAgentDelegation は UUID でない agentID を、付与がないものとして扱う。パスから直接
// 届く値で、存在しない Agent と区別する理由がないからである。
func (r *ConsentRepository) FindAgentDelegation(ctx context.Context, tenantID, sub, agentID string) (*domain.Consent, error) {
	agent, err := uuidParam(agentID)
	if err != nil {
		return nil, nil //nolint:nilerr // 上のコメントのとおり
	}
	row, err := oauth2pg.New(r.Pool).GetAgentDelegation(ctx, oauth2pg.GetAgentDelegationParams{UserID: sub, AgentID: agent})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return consentFromRow(consentRow{
		user: row.UserID, agentID: row.AgentID, scopes: row.Scopes,
		grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt, lastUsedAt: row.LastUsedAt,
	})
}

func (r *ConsentRepository) ListAgentDelegations(ctx context.Context, tenantID, sub string) ([]*domain.Consent, error) {
	rows, err := oauth2pg.New(r.Pool).ListAgentDelegationsForUser(ctx, sub)
	if err != nil {
		return nil, err
	}
	return consentsFromRows(rows, func(row *oauth2pg.ListAgentDelegationsForUserRow) consentRow {
		return consentRow{
			user: row.UserID, agentID: row.AgentID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt, lastUsedAt: row.LastUsedAt,
		}
	})
}

func (r *ConsentRepository) ListAgentDelegationPage(ctx context.Context, tenantID, afterUserID, afterAgentID string, limit int) ([]*domain.Consent, error) {
	q := oauth2pg.New(r.Pool)
	if afterUserID == "" && afterAgentID == "" {
		rows, err := q.ListAgentDelegationsByTenantPage(ctx, oauth2pg.ListAgentDelegationsByTenantPageParams{
			TenantID:  tenantID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
		if err != nil {
			return nil, err
		}
		return consentsFromRows(rows, func(row *oauth2pg.ListAgentDelegationsByTenantPageRow) consentRow {
			return consentRow{
				user: row.UserID, agentID: row.AgentID, scopes: row.Scopes,
				grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt, lastUsedAt: row.LastUsedAt,
			}
		})
	}
	rows, err := q.ListAgentDelegationsByTenantPageAfter(ctx, oauth2pg.ListAgentDelegationsByTenantPageAfterParams{
		TenantID: tenantID, AfterUserID: afterUserID, AfterAgentID: afterAgentID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	return consentsFromRows(rows, func(row *oauth2pg.ListAgentDelegationsByTenantPageAfterRow) consentRow {
		return consentRow{
			user: row.UserID, agentID: row.AgentID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt, lastUsedAt: row.LastUsedAt,
		}
	})
}

func (r *ConsentRepository) ListAgentDelegationPageBefore(ctx context.Context, tenantID, beforeUserID, beforeAgentID string, limit int) ([]*domain.Consent, error) {
	rows, err := oauth2pg.New(r.Pool).ListAgentDelegationsByTenantPageBefore(ctx, oauth2pg.ListAgentDelegationsByTenantPageBeforeParams{
		TenantID: tenantID, BeforeUserID: beforeUserID, BeforeAgentID: beforeAgentID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	return consentsFromRows(rows, func(row *oauth2pg.ListAgentDelegationsByTenantPageBeforeRow) consentRow {
		return consentRow{
			user: row.UserID, agentID: row.AgentID, scopes: row.Scopes,
			grantedAt: row.GrantedAt, expiresAt: row.ExpiresAt, revokedAt: row.RevokedAt, lastUsedAt: row.LastUsedAt,
		}
	})
}

func (r *ConsentRepository) SaveAgentDelegation(ctx context.Context, tenantID string, c *domain.Consent) error {
	scopes, err := json.Marshal(c.Scopes)
	if err != nil {
		return err
	}
	agent, err := uuidParam(c.AgentID)
	if err != nil {
		return err
	}
	return oauth2pg.New(r.Pool).UpsertAgentDelegation(ctx, oauth2pg.UpsertAgentDelegationParams{
		UserID: c.UserID, AgentID: agent, Scopes: scopes, GrantedAt: c.GrantedAt, ExpiresAt: c.ExpiresAt,
	})
}

func (r *ConsentRepository) RevokeAgentDelegation(ctx context.Context, tenantID, sub, agentID string) error {
	agent, err := uuidParam(agentID)
	if err != nil {
		return err
	}
	return oauth2pg.New(r.Pool).RevokeAgentDelegation(ctx, oauth2pg.RevokeAgentDelegationParams{UserID: sub, AgentID: agent})
}

func (r *ConsentRepository) TouchAgentDelegation(ctx context.Context, tenantID, sub, agentID string, at time.Time) error {
	agent, err := uuidParam(agentID)
	if err != nil {
		return err
	}
	return oauth2pg.New(r.Pool).TouchAgentDelegation(ctx, oauth2pg.TouchAgentDelegationParams{
		UserID: sub, AgentID: agent, LastUsedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}
