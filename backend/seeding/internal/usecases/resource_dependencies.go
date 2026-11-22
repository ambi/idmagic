package usecases

import (
	appports "github.com/ambi/idmagic/backend/application/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ResourceDependencies は計画に使う所有者の照会とハッシュの照合を受け取る。
// 書き込みは ResourceCommands を通す。
type ResourceDependencies struct {
	ClientRepo          oauthports.OAuth2ClientRepository
	UserRepo            userports.UserRepository
	GroupRepo           groupports.GroupRepository
	MfaFactorRepo       totpports.MfaFactorRepository
	PasswordHistoryRepo passwordports.PasswordHistoryRepository
	AuthzDetailTypeRepo oauthports.AuthorizationDetailTypeRepository
	RPRepo              wsfedports.WsFedRelyingPartyRepository
	SPRepo              samlports.SamlServiceProviderRepository
	Applications        appports.ApplicationRepository
	AssignmentRepo      appports.AssignmentRepository
	Hasher              passwordports.PasswordHasher
}
