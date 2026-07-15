// Package identitymanagement は IdentityManagement bounded context の DI 組立を所有する
// (ADR-091, wi-178)。User/Group/Agent の永続化 port を Module 1 個に束ね、中央
// server/routes.go と bootstrap の Dependencies が受け渡す。identitymanagement/adapters/http
// は oauth2/scim/authentication/tenancy 由来の port も必要とするため、HTTP route 登録は
// authentication.Module 同様に中央 routes.go が引き続き組み立てる (自己登録は行わない)。
package identitymanagement

import (
	"github.com/ambi/idmagic/backend/identitymanagement/ports"
)

// Module は identity-management context が所有する永続化 port の束。
type Module struct {
	UserRepo                 ports.UserRepository
	GroupRepo                ports.GroupRepository
	AgentRepo                ports.AgentRepository
	LifecycleWorkflowRepo    ports.LifecycleWorkflowRepository
	LifecycleWorkflowRunRepo ports.LifecycleWorkflowRunRepository
	UserWorkflowCapture      ports.UserWorkflowCapture
}
