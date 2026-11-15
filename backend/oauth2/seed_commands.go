package oauth2

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	clientusecases "github.com/ambi/idmagic/backend/oauth2/internal/client/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// CreateSeedClient は固定 ID と資格情報を保ち、登録と共通の検証と使用量管理へ通す。
func (m Module) CreateSeedClient(ctx context.Context, c *domain.OAuth2Client, quota tenantports.QuotaRepository, emit func(spec.DomainEvent)) error {
	return clientusecases.CreateSeedClient(ctx, clientusecases.RegisterClientDeps{ClientRepo: m.ClientRepo, QuotaRepo: quota, Emit: emit}, c)
}

func (m Module) CreateSeedAuthorizationDetailType(ctx context.Context, desired *domain.AuthorizationDetailType) error {
	if err := desired.Validate(); err != nil {
		return err
	}
	existing, err := m.AuthzDetailTypeRepo.FindByType(ctx, desired.TenantID, desired.Type)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("authorization detail type already exists")
	}
	return m.AuthzDetailTypeRepo.Save(ctx, desired)
}
