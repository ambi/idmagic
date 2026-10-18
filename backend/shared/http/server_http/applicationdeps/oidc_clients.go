package applicationdeps

import (
	"context"
	"errors"
	"slices"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	catalogports "github.com/ambi/idmagic/backend/applicationcatalog/ports"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

type OIDCClients struct {
	repository     clientports.OAuth2ClientRepository
	administration clientports.ClientAdministration
}

var _ catalogports.OIDCClientAdministration = (*OIDCClients)(nil)

func NewOIDCClients(repository clientports.OAuth2ClientRepository, administration clientports.ClientAdministration) *OIDCClients {
	return &OIDCClients{repository: repository, administration: administration}
}

func (a *OIDCClients) Configuration(ctx context.Context, tenantID, clientID string, now time.Time) (*catalogports.OIDCConfiguration, error) {
	if a.repository == nil {
		return nil, nil //nolint:nilnil // 保存先が未配線なら、従来どおり任意の設定表示を省略する。
	}
	client, err := a.repository.FindByID(ctx, tenantID, clientID)
	if err != nil || client == nil {
		return nil, err
	}
	// 資格情報の一覧のエラーは従来どおり設定の表示を失敗させない。
	credentials, _ := a.repository.ListClientSecretCredentials(ctx, clientID)
	return configuration(client, credentials, now), nil
}

func configuration(client *oauthdomain.OAuth2Client, credentials []oauthdomain.ClientSecretCredential, now time.Time) *catalogports.OIDCConfiguration {
	sub := "user_id"
	rules := []claimdomain.ClaimMappingRule{}
	if client.ClaimPolicy != nil {
		if client.ClaimPolicy.NameID.SourceAttribute != "" {
			sub = client.ClaimPolicy.NameID.SourceAttribute
		}
		if client.ClaimPolicy.Rules != nil {
			rules = slices.Clone(client.ClaimPolicy.Rules)
		}
	}
	return &catalogports.OIDCConfiguration{
		ClientID: client.ClientID, ClientType: client.ClientType,
		RedirectURIs: slices.Clone(client.RedirectURIs), GrantTypes: slices.Clone(client.GrantTypes), ResponseTypes: slices.Clone(client.ResponseTypes),
		TokenEndpointAuthMethod: string(client.TokenEndpointAuthMethod), Scope: client.Scope,
		RequirePAR: client.RequirePushedAuthorizationRequests, DpopBoundAccessTokens: client.DpopBoundAccessTokens,
		FapiProfile:           string(client.FapiProfile),
		ClientSecretRotatable: client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretBasic || client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretPost,
		SecretCredentials:     secretMetadata(credentials, now), SubSourceAttribute: sub, Rules: rules,
	}
}

func secretMetadata(credentials []oauthdomain.ClientSecretCredential, now time.Time) []catalogports.ClientSecretMetadata {
	metadata := make([]catalogports.ClientSecretMetadata, 0, len(credentials))
	for _, credential := range credentials {
		metadata = append(metadata, catalogports.ClientSecretMetadata{
			CredentialID: credential.ID, CreatedAt: credential.CreatedAt,
			ExpiresAt: copyValue(credential.ExpiresAt), RevokedAt: copyValue(credential.RevokedAt), Status: string(credential.StatusAt(now)),
		})
	}
	return metadata
}

func (a *OIDCClients) Create(ctx context.Context, in catalogports.CreateOIDCClientInput) (*catalogports.CreatedOIDCClient, error) {
	r := in.Registration
	result, err := a.administration.Create(ctx, clientports.CreateAdminOAuth2ClientInput{
		ActorUserID: in.ActorUserID, Now: in.Now,
		Registration: clientports.RegisterClientInput{
			ClientName: r.ClientName, ClientType: r.ClientType, RedirectURIs: slices.Clone(r.RedirectURIs),
			GrantTypes: slices.Clone(r.GrantTypes), ResponseTypes: slices.Clone(r.ResponseTypes),
			TokenEndpointAuthMethod: oauthdomain.TokenEndpointAuthMethod(r.TokenEndpointAuthMethod), Scope: r.Scope,
			JwksURI: copyValue(r.JwksURI), TlsClientAuthSubjectDN: copyValue(r.TlsClientAuthSubjectDN),
		},
	})
	if err != nil {
		return nil, err
	}
	return &catalogports.CreatedOIDCClient{ClientID: result.Client.ClientID, ClientSecret: result.ClientSecret}, nil
}

func (a *OIDCClients) Update(ctx context.Context, in catalogports.UpdateOIDCClientInput) error {
	_, err := a.administration.Update(ctx, clientports.UpdateAdminOAuth2ClientInput{
		ActorUserID: in.ActorUserID, ClientID: in.ClientID, RedirectURIs: in.RedirectURIs,
		GrantTypes: in.GrantTypes, ResponseTypes: in.ResponseTypes, Scope: in.Scope,
		RequirePAR: in.RequirePAR, DpopBoundTokens: in.DpopBoundTokens, Now: in.Now,
	})
	return err
}

func (a *OIDCClients) BindApplication(ctx context.Context, tenantID, clientID, applicationID string) error {
	return a.administration.BindApplication(ctx, tenantID, clientID, applicationID)
}

func (a *OIDCClients) UpdateClaims(ctx context.Context, tenantID, clientID string, in catalogports.OIDCClaimUpdate) error {
	err := a.administration.UpdateClaims(ctx, tenantID, clientID, clientports.ClientClaimUpdate{
		SubSourceAttribute: in.SubSourceAttribute, Rules: in.Rules, Now: in.Now,
	})
	if errors.Is(err, clientports.ErrCatalogClientNotFound) {
		return catalogports.ErrOIDCClientNotFound
	}
	return err
}

func (a *OIDCClients) Delete(ctx context.Context, tenantID, clientID string) error {
	return a.administration.DeleteForApplication(ctx, tenantID, clientID)
}

func (a *OIDCClients) RotateSecret(ctx context.Context, in catalogports.RotateClientSecretInput) (*catalogports.RotateClientSecretResult, error) {
	result, err := a.administration.RotateSecret(ctx, clientports.RotateClientSecretInput{
		ActorUserID: in.ActorUserID, ClientID: in.ClientID, GraceDays: in.GraceDays, Now: in.Now,
	})
	if err != nil {
		return nil, secretError(err)
	}
	return &catalogports.RotateClientSecretResult{ClientSecret: result.ClientSecret, GraceUntil: copyValue(result.GraceUntil), Credentials: secretMetadata(result.Credentials, time.Now().UTC())}, nil
}

func (a *OIDCClients) IssueSecret(ctx context.Context, in catalogports.IssueClientSecretInput) (*catalogports.IssueClientSecretResult, error) {
	result, err := a.administration.IssueSecret(ctx, clientports.IssueClientSecretInput{
		ActorUserID: in.ActorUserID, ClientID: in.ClientID, ExpiresInDays: in.ExpiresInDays, Now: in.Now,
	})
	if err != nil {
		return nil, secretError(err)
	}
	return &catalogports.IssueClientSecretResult{ClientSecret: result.ClientSecret, Credential: secretMetadata([]oauthdomain.ClientSecretCredential{result.Credential}, in.Now)[0], Credentials: secretMetadata(result.Credentials, in.Now)}, nil
}

func (a *OIDCClients) RevokeSecret(ctx context.Context, in catalogports.RevokeClientSecretInput) (*catalogports.RevokeClientSecretResult, error) {
	result, err := a.administration.RevokeSecret(ctx, clientports.RevokeClientSecretInput{
		ActorUserID: in.ActorUserID, ClientID: in.ClientID, CredentialID: in.CredentialID, Now: in.Now,
	})
	if err != nil {
		return nil, secretError(err)
	}
	return &catalogports.RevokeClientSecretResult{Credentials: secretMetadata(result.Credentials, in.Now)}, nil
}

func secretError(err error) error {
	if errors.Is(err, clientports.ErrClientSecretLimitExceeded) {
		return catalogports.ErrClientSecretLimitExceeded
	}
	return err
}

type clientNames struct {
	repository clientports.OAuth2ClientRepository
}

func NewClientNames(repository clientports.OAuth2ClientRepository) appports.ClientNames {
	if repository == nil {
		return nil
	}
	return clientNames{repository: repository}
}

func (a clientNames) FindClientName(ctx context.Context, tenantID, clientID string) (*string, error) {
	client, err := a.repository.FindByID(ctx, tenantID, clientID)
	if err != nil || client == nil {
		return nil, err
	}
	return copyValue(client.ClientName), nil
}

func copyValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
