package db_postgres

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/jackc/pgx/v5"
)

func (r *ApplicationRepository) DeleteWithRelated(ctx context.Context, app *domain.Application, participants ports.DeletionParticipants) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(tx)
	if _, err := queries.LockApplicationForDeletion(ctx, LockApplicationForDeletionParams{TenantID: app.TenantID, ID: app.ID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrApplicationNotFound
		}
		return err
	}
	// 削除対象の関係はロックした保存状態から読み直し、呼び出し側の古い射影を使わない。
	current := &ApplicationRepository{Pool: sharedpg.BindTransaction(tx), ProtocolLinkers: r.ProtocolLinkers}
	app, err = current.FindByID(ctx, app.TenantID, app.ID)
	if err != nil {
		return err
	}
	if app == nil {
		return ports.ErrApplicationNotFound
	}
	consents := 0
	if app.Protocol != nil && app.Protocol.Type == domain.ApplicationProtocolOIDC {
		if r.ProtocolLinkers.DeleteOIDC == nil {
			return errors.New("application deletion is not wired to the OIDC owner")
		}
		consents, err = r.ProtocolLinkers.DeleteOIDC(ctx, tx, app.TenantID, app.Protocol.ClientID)
		if err != nil {
			return err
		}
	}
	if err := queries.DeleteApplication(ctx, DeleteApplicationParams{TenantID: app.TenantID, ID: app.ID}); err != nil {
		return err
	}
	if participants.Quota != nil {
		if r.ProtocolLinkers.QuotaInTx == nil {
			return errors.New("application deletion is not wired to transactional quotas")
		}
		quota := r.ProtocolLinkers.QuotaInTx(tx)
		if err := quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceApplications, 1); err != nil {
			return err
		}
		if app.Protocol != nil && app.Protocol.Type == domain.ApplicationProtocolOIDC {
			if err := quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceOAuth2Clients, 1); err != nil {
				return err
			}
		}
		if consents > 0 {
			if err := quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceConsents, consents); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
