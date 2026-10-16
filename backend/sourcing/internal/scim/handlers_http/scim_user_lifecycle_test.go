package handlers_http_test

import (
	"github.com/ambi/idmagic/backend/idmanagement"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

// scimUserLifecycle は、組み立ての地点と同じく IdManagement の操作を SCIM のポートへ渡す。
func scimUserLifecycle(users userports.UserRepository) scimports.UserLifecycle {
	return idmanagement.Module{UserRepo: users}.UserLifecycleCommands(idmanagement.LifecycleDeps{}, nil, "scim")
}
