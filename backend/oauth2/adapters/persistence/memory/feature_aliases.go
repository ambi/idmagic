package memory

import (
	authorizationmemory "github.com/ambi/idmagic/backend/oauth2/authorization/adapters/persistence/memory"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/adapters/persistence/memory"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/adapters/persistence/memory"
	devicememory "github.com/ambi/idmagic/backend/oauth2/device/adapters/persistence/memory"
	tokenmemory "github.com/ambi/idmagic/backend/oauth2/token/adapters/persistence/memory"
)

type (
	OAuth2ClientRepository    = clientmemory.OAuth2ClientRepository
	ConsentRepository         = consentmemory.ConsentRepository
	AuthorizationRequestStore = authorizationmemory.AuthorizationRequestStore
	AuthorizationCodeStore    = authorizationmemory.AuthorizationCodeStore
	PARStore                  = authorizationmemory.PARStore
	RefreshTokenStore         = tokenmemory.RefreshTokenStore
	AccessTokenDenylist       = tokenmemory.AccessTokenDenylist
	DeviceCodeStore           = devicememory.DeviceCodeStore
)

var (
	NewClientRepository          = clientmemory.NewClientRepository
	NewConsentRepository         = consentmemory.NewConsentRepository
	NewAuthorizationRequestStore = authorizationmemory.NewAuthorizationRequestStore
	NewAuthorizationCodeStore    = authorizationmemory.NewAuthorizationCodeStore
	NewPARStore                  = authorizationmemory.NewPARStore
	NewRefreshTokenStore         = tokenmemory.NewRefreshTokenStore
	NewAccessTokenDenylist       = tokenmemory.NewAccessTokenDenylist
	NewDeviceCodeStore           = devicememory.NewDeviceCodeStore
)
