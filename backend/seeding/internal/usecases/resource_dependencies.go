package usecases

import (
	appports "github.com/ambi/idmagic/backend/application/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ResourceDependencies は seed が使う所有者の保存契約とハッシュ器を受け取る。
// 通常の更新ユースケースの検証やイベントを通らない制約は、Seeding のリスクで扱う。
type ResourceDependencies struct {
	ClientRepo          oauthports.OAuth2ClientRepository
	UserRepo            userports.UserRepository
	GroupRepo           groupports.GroupRepository
	MfaFactorRepo       totpports.MfaFactorRepository
	PasswordHistoryRepo passwordports.PasswordHistoryRepository
	AuthzDetailTypeRepo oauthports.AuthorizationDetailTypeRepository
	RPRepo              wsfederationports.WsFedRelyingPartyRepository
	SPRepo              samlports.SamlServiceProviderRepository
	Applications        appports.ApplicationRepository
	AssignmentRepo      appports.AssignmentRepository
	Hasher              passwordports.PasswordHasher
}
