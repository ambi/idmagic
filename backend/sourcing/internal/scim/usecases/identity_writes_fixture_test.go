package usecases_test

import (
	"github.com/ambi/idmagic/backend/idmanagement"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

func scimIdentityWrites(refs scimports.ScimRepository, users userports.UserRepository, groups groupports.GroupRepository, emit func(spec.DomainEvent)) scimports.IdentityWrites {
	return sourcingdeps.NewWrites(idmanagement.Module{UserRepo: users, GroupRepo: groups},
		idmanagement.LifecycleDeps{QuotaRepo: tenancy.NewMemoryModule().QuotaRepo}, sourcing.Module{ScimRepo: refs}, emit)
}
