package usecases

import (
	"context"
	"slices"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
)

// BindApplication は、カタログの作成後に保存側の関連付けを同期する。
// 既存の PostgreSQL の linker と確定順を保ち、未取得なら補償を加えない。
func (a ClientAdministration) BindApplication(ctx context.Context, tenantID, clientID, applicationID string) error {
	client, err := a.Deps.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil || client == nil {
		return err
	}
	client.ApplicationID = applicationID
	return a.Deps.ClientRepo.Save(ctx, client)
}

// UpdateClaims は通常設定の確定後にクレームだけを保存する。イベントはカタログが発行する。
func (a ClientAdministration) UpdateClaims(ctx context.Context, tenantID, clientID string, in clientports.ClientClaimUpdate) error {
	client, err := a.Deps.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if client == nil {
		return clientports.ErrCatalogClientNotFound
	}
	policy := claimdomain.ClaimMappingPolicy{}
	if client.ClaimPolicy != nil {
		policy = *client.ClaimPolicy
	}
	if policy.NameID.Format == "" {
		policy.NameID.Format = "user_id"
	}
	if in.SubSourceAttribute != nil {
		sub := strings.TrimSpace(*in.SubSourceAttribute)
		if sub == "" {
			sub = "user_id"
		}
		policy.NameID.SourceAttribute = sub
	} else if policy.NameID.SourceAttribute == "" {
		policy.NameID.SourceAttribute = "user_id"
	}
	if in.Rules != nil {
		policy.Rules = slices.Clone(*in.Rules)
	}
	client.ClaimPolicy = &policy
	client.UpdatedAt = in.Now
	return a.Deps.ClientRepo.Save(ctx, client)
}

func (a ClientAdministration) DeleteForApplication(ctx context.Context, tenantID, clientID string) error {
	return a.Deps.ClientRepo.Delete(ctx, tenantID, clientID)
}
