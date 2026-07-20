package db_memory

import (
	authorizationmemory "github.com/ambi/idmagic/backend/oauth2/authorization/db_memory"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	devicememory "github.com/ambi/idmagic/backend/oauth2/device/db_memory"
	tokenmemory "github.com/ambi/idmagic/backend/oauth2/token/db_memory"
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
