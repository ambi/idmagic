package usecases

import (
	"context"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// ClientAdministration は clientports.ClientAdministration を、管理者向けの操作で実装する。
type ClientAdministration struct{ Deps AdminOAuth2ClientDeps }

var _ clientports.ClientAdministration = ClientAdministration{}

func (a ClientAdministration) Create(ctx context.Context, in CreateAdminOAuth2ClientInput) (*RegisterClientResult, error) {
	return CreateAdminOAuth2Client(ctx, a.Deps, in)
}

func (a ClientAdministration) Update(ctx context.Context, in UpdateAdminOAuth2ClientInput) (*domain.OAuth2Client, error) {
	return UpdateAdminOAuth2Client(ctx, a.Deps, in)
}

func (a ClientAdministration) RotateSecret(ctx context.Context, in RotateClientSecretInput) (*RotateClientSecretResult, error) {
	return RotateClientSecret(ctx, a.Deps, in)
}

func (a ClientAdministration) IssueSecret(ctx context.Context, in IssueClientSecretInput) (*IssueClientSecretResult, error) {
	return IssueClientSecret(ctx, a.Deps, in)
}

func (a ClientAdministration) RevokeSecret(ctx context.Context, in RevokeClientSecretInput) (*RevokeClientSecretResult, error) {
	return RevokeClientSecret(ctx, a.Deps, in)
}
