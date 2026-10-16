package ports

import (
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	authorizationports "github.com/ambi/idmagic/backend/oauth2/internal/authorization/ports"
	deviceports "github.com/ambi/idmagic/backend/oauth2/internal/device/ports"
	tokenports "github.com/ambi/idmagic/backend/oauth2/internal/token/ports"
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
