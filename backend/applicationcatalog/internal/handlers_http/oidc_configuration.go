package handlers_http

import (
	"context"
	"slices"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// oidcConfig と clientSecretMetadata は管理の HTTP が公開する項目だけを持つ。
// 所有者の保存型を直接シリアライズせず、資格情報の状態は所有者の判定を使う。
type oidcConfig struct {
	ClientID                string                         `json:"client_id"`
	ClientType              spec.ClientType                `json:"client_type"`
	RedirectURIs            []string                       `json:"redirect_uris"`
	GrantTypes              []spec.GrantType               `json:"grant_types"`
	ResponseTypes           []spec.ResponseType            `json:"response_types"`
	TokenEndpointAuthMethod string                         `json:"token_endpoint_auth_method"`
	Scope                   string                         `json:"scope"`
	RequirePAR              bool                           `json:"require_pushed_authorization_requests"`
	DpopBoundAccessTokens   bool                           `json:"dpop_bound_access_tokens"`
	FapiProfile             string                         `json:"fapi_profile"`
	ClientSecretRotatable   bool                           `json:"client_secret_rotatable"`
	SecretCredentials       []clientSecretMetadata         `json:"secret_credentials"`
	SubSourceAttribute      string                         `json:"sub_source_attribute"`
	Rules                   []claimdomain.ClaimMappingRule `json:"rules"`
}

type clientSecretMetadata struct {
	CredentialID string     `json:"credential_id"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	Status       string     `json:"status"`
}

func (d Deps) oidcConfiguration(ctx context.Context, tenantID, clientID string, now time.Time) (*oidcConfig, error) {
	if d.ClientRepo == nil {
		return nil, nil //nolint:nilnil // 未配線なら任意の設定表示を省略する。
	}
	client, err := d.ClientRepo.FindByID(ctx, tenantID, clientID)
	if err != nil || client == nil {
		return nil, err
	}
	// 資格情報の一覧だけのエラーは設定の表示を失敗させない。
	credentials, _ := d.ClientRepo.ListClientSecretCredentials(ctx, clientID)
	return configuration(client, credentials, now), nil
}

func configuration(client *oauthdomain.OAuth2Client, credentials []oauthdomain.ClientSecretCredential, now time.Time) *oidcConfig {
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
	return &oidcConfig{
		ClientID: client.ClientID, ClientType: client.ClientType,
		RedirectURIs: slices.Clone(client.RedirectURIs), GrantTypes: slices.Clone(client.GrantTypes), ResponseTypes: slices.Clone(client.ResponseTypes),
		TokenEndpointAuthMethod: string(client.TokenEndpointAuthMethod), Scope: client.Scope,
		RequirePAR: client.RequirePushedAuthorizationRequests, DpopBoundAccessTokens: client.DpopBoundAccessTokens,
		FapiProfile:           string(client.FapiProfile),
		ClientSecretRotatable: client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretBasic || client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretPost,
		SecretCredentials:     secretMetadata(credentials, now), SubSourceAttribute: sub, Rules: rules,
	}
}

func secretMetadata(credentials []oauthdomain.ClientSecretCredential, now time.Time) []clientSecretMetadata {
	metadata := make([]clientSecretMetadata, 0, len(credentials))
	for _, credential := range credentials {
		metadata = append(metadata, clientSecretMetadata{
			CredentialID: credential.ID, CreatedAt: credential.CreatedAt,
			ExpiresAt: copyValue(credential.ExpiresAt), RevokedAt: copyValue(credential.RevokedAt), Status: string(credential.StatusAt(now)),
		})
	}
	return metadata
}

func copyValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	return new(*value)
}
