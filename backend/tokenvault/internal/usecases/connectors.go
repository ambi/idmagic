package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
)

// CreateConnector はコネクタを作る (REQ-TOKENVAULT-001)。
func CreateConnector(ctx context.Context, deps Deps, connector domain.Connector) (*domain.Connector, error) {
	now := deps.now()
	connector.ID = uuid.NewString()
	connector.Status = domain.ConnectorActive
	if connector.TokenEndpointAuthMethod == "" {
		connector.TokenEndpointAuthMethod = domain.AuthMethodClientSecretBasic
	}
	connector.Version, connector.CreatedAt, connector.UpdatedAt = 1, now, now
	if err := connector.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Connectors.Create(ctx, &connector); err != nil {
		return nil, err
	}
	deps.emit(&domain.TokenVaultConnectorCreated{At: now, TenantID: connector.TenantID, ConnectorID: connector.ID})
	return &connector, nil
}

// ConnectorChanges はコネクタの更新である。nil の項目は保存の値を維持する。
type ConnectorChanges struct {
	Name, AuthorizationEndpoint, TokenEndpoint, RevocationEndpoint, ClientID, ClientSecret *string
	Status                                                                                 *domain.ConnectorStatus
	TokenEndpointAuthMethod                                                                *domain.TokenEndpointAuthMethod
	AllowedScopes                                                                          *[]string
	Rules                                                                                  *[]domain.ExecutionRule
}

func (c ConnectorChanges) apply(connector *domain.Connector) {
	assign(&connector.Name, c.Name)
	assign(&connector.AuthorizationEndpoint, c.AuthorizationEndpoint)
	assign(&connector.TokenEndpoint, c.TokenEndpoint)
	assign(&connector.RevocationEndpoint, c.RevocationEndpoint)
	assign(&connector.ClientID, c.ClientID)
	assign(&connector.ClientSecret, c.ClientSecret)
	assign(&connector.Status, c.Status)
	assign(&connector.TokenEndpointAuthMethod, c.TokenEndpointAuthMethod)
	assign(&connector.AllowedScopes, c.AllowedScopes)
	assign(&connector.Rules, c.Rules)
}

func assign[T any](destination, value *T) {
	if value != nil {
		*destination = *value
	}
}

// UpdateConnector はコネクタを更新する (REQ-TOKENVAULT-001、REQ-TOKENVAULT-002)。
func UpdateConnector(ctx context.Context, deps Deps, tenantID, id string, changes ConnectorChanges) (*domain.Connector, error) {
	connector, err := GetConnector(ctx, deps, tenantID, id)
	if err != nil {
		return nil, err
	}
	changes.apply(connector)
	connector.UpdatedAt = deps.now()
	if err := connector.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Connectors.Update(ctx, connector); err != nil {
		return nil, err
	}
	deps.emit(&domain.TokenVaultConnectorUpdated{At: connector.UpdatedAt, TenantID: tenantID, ConnectorID: id, Status: connector.Status})
	return connector, nil
}

// GetConnector は自テナントのコネクタを返す。
func GetConnector(ctx context.Context, deps Deps, tenantID, id string) (*domain.Connector, error) {
	connector, err := deps.Connectors.Find(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, ErrConnectorNotFound
	}
	return connector, nil
}

// DeleteConnector はコネクタを消す。接続と進行中の連携も消える (REQ-TOKENVAULT-002)。
func DeleteConnector(ctx context.Context, deps Deps, tenantID, id string) error {
	deleted, err := deps.Connectors.Delete(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrConnectorNotFound
	}
	deps.emit(&domain.TokenVaultConnectorDeleted{At: deps.now(), TenantID: tenantID, ConnectorID: id})
	return nil
}

func oauthClient(connector *domain.Connector) oauthClientConfig {
	return oauthClientConfig{
		TokenEndpoint: connector.TokenEndpoint, RevocationEndpoint: connector.RevocationEndpoint,
		ClientID: connector.ClientID, ClientSecret: connector.ClientSecret, AuthMethod: connector.TokenEndpointAuthMethod,
	}
}
