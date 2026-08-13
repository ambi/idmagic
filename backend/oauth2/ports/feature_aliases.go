package ports

import (
	approvalports "github.com/ambi/idmagic/backend/oauth2/approval/ports"
	authorizationports "github.com/ambi/idmagic/backend/oauth2/authorization/ports"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	deviceports "github.com/ambi/idmagic/backend/oauth2/device/ports"
	tokenports "github.com/ambi/idmagic/backend/oauth2/token/ports"
)

// Feature ports are re-exported while external composition roots migrate.
type (
	OAuth2ClientRepository    = clientports.OAuth2ClientRepository
	ConsentRepository         = consentports.ConsentRepository
	AuthorizationRequestStore = authorizationports.AuthorizationRequestStore
	AuthorizationCodeStore    = authorizationports.AuthorizationCodeStore
	PARStore                  = authorizationports.PARStore
	DeviceCodeStore           = deviceports.DeviceCodeStore
	ApprovalRequestStore      = approvalports.ApprovalRequestStore
	AccessTokenDenylist       = tokenports.AccessTokenDenylist
	RefreshTokenStore         = tokenports.RefreshTokenStore
)

var ErrClientSecretCredentialLimitExceeded = clientports.ErrClientSecretCredentialLimitExceeded
