package applicationdeps

import (
	"context"

	appports "github.com/ambi/idmagic/backend/application/ports"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
)

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
	if client.ClientName == nil {
		return nil, err
	}
	return new(*client.ClientName), nil
}
