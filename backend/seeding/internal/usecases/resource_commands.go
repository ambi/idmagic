package usecases

import (
	"context"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	wsfeddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

// ResourceCommands は所有者の規則と作用を伴う作成操作であり、構成入口で結ぶ。
type ResourceCommands interface {
	CreateClient(context.Context, *oauthdomain.OAuth2Client) error
	CreateUser(context.Context, *userdomain.User, string) error
	EnsurePasswordHistory(context.Context, string, string, time.Time) error
	CreateGroup(context.Context, *groupdomain.Group) error
	AddGroupMember(context.Context, *groupdomain.GroupMember) error
	CreateTOTP(context.Context, *totpdomain.MfaFactor) error
	CreateAuthorizationDetailType(context.Context, *oauthdomain.AuthorizationDetailType) error
	CreateRelyingParty(context.Context, *wsfeddomain.WsFedRelyingParty) error
	CreateServiceProvider(context.Context, *samldomain.SamlServiceProvider) error
	CreateApplication(context.Context, *appdomain.Application) error
	AssignApplication(context.Context, *appdomain.ApplicationAssignment) error
}
