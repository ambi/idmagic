package db_memory

import (
	"context"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// メモリ実行の破棄操作は、存在と保護の検証後には失敗しない保存先を組み立てる。
// PostgreSQL では同じ境界を実トランザクションで確定する。
func (r *ApplicationRepository) DeleteWithRelated(ctx context.Context, app *domain.Application, participants ports.DeletionParticipants) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(app.TenantID, app.ID, kernel.DefaultTenantID)
	if r.applications[key] == nil {
		return ports.ErrApplicationNotFound
	}
	app = cloneApplication(r.applications[key])
	consents := 0
	if participants.DeleteProtocol != nil {
		var err error
		consents, err = participants.DeleteProtocol(ctx, app)
		if err != nil {
			return err
		}
	}
	if participants.Assignments != nil {
		if err := participants.Assignments.DeleteByApplication(ctx, app.TenantID, app.ID); err != nil {
			return err
		}
	}
	if participants.Policy != nil {
		if err := participants.Policy.Delete(ctx, app.TenantID, app.ID); err != nil {
			return err
		}
	}
	if participants.Icons != nil {
		if err := participants.Icons.DeleteByApplication(ctx, app.TenantID, app.ID); err != nil {
			return err
		}
	}
	if participants.Quota != nil {
		if err := participants.Quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceApplications, 1); err != nil {
			return err
		}
		if app.Protocol != nil && app.Protocol.Type == domain.ApplicationProtocolOIDC {
			if err := participants.Quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceOAuth2Clients, 1); err != nil {
				return err
			}
		}
		if consents > 0 {
			if err := participants.Quota.Decrement(ctx, app.TenantID, tenantdomain.ResourceConsents, consents); err != nil {
				return err
			}
		}
	}
	delete(r.applications, key)
	return nil
}
