// Package provisioning is the Provisioning bounded context's top-level wiring
// point (ADR-091 Module pattern): Module aggregates the persistence
// dependencies bootstrap injects, and JobEnqueuer/Notifiers build the
// cross-context adapters (Jobs enqueue, IdManagement/Application capture
// notification) other composition-root code wires in.
package provisioning

import (
	"context"
	"encoding/json"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	provisioningscim "github.com/ambi/idmagic/backend/provisioning/client_scim"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	provisioninghttp "github.com/ambi/idmagic/backend/provisioning/handlers_http"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// NewTargetClient builds the outbound SCIM wire client for conn (ADR-128
// decision 2). Only bearer_token auth is wired end-to-end for now;
// oauth2_client_credentials connections authenticate with the stored secret as
// a bearer token, which is incorrect for that auth method and is a known gap
// left for a follow-up (wi-45 did not scope an OAuth2 token-fetch client).
func NewTargetClient(conn *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
	return provisioningscim.NewClient(conn.BaseURL, secret)
}

// Module holds the Provisioning bounded context's persistence dependencies.
type Module struct {
	ConnectionRepo ports.ProvisioningConnectionRepository
	RemoteLinkRepo ports.RemoteResourceLinkRepository
	DeliveryRepo   ports.ProvisioningDeliveryRepository
}

func (m Module) captureDeps(assignmentRepo appports.AssignmentRepository) usecases.CaptureDeps {
	return usecases.CaptureDeps{ConnectionRepo: m.ConnectionRepo, DeliveryRepo: m.DeliveryRepo, AssignmentRepo: assignmentRepo}
}

// UserNotifier builds the userports.ProvisioningNotifier IdManagement calls
// after committing a User mutation (ADR-128 decision 4).
func (m Module) UserNotifier(assignmentRepo appports.AssignmentRepository) userports.ProvisioningNotifier {
	return usecases.UserMutationNotifier{CaptureDeps: m.captureDeps(assignmentRepo)}
}

// AssignmentNotifier builds the appports.ProvisioningNotifier Application calls
// after committing an assignment change.
func (m Module) AssignmentNotifier(assignmentRepo appports.AssignmentRepository) appports.ProvisioningNotifier {
	return usecases.AssignmentMutationNotifier{CaptureDeps: m.captureDeps(assignmentRepo)}
}

// jobEnqueuer implements usecases.Enqueuer against the Jobs bounded context.
type jobEnqueuer struct {
	Repo      jobsports.JobRepository
	QuotaRepo tenantports.QuotaRepository
}

func (e jobEnqueuer) EnqueueProvisioningDelivery(ctx context.Context, tenantID, dedupKey, deliveryID string) (string, error) {
	params, err := json.Marshal(map[string]string{"delivery_id": deliveryID})
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	job, err := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: e.Repo, QuotaRepo: e.QuotaRepo}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: usecases.KindProvisioningDelivery, Params: params, DedupKey: &dedupKey, Now: now,
	}, now)
	if err != nil {
		return "", err
	}
	return job.ID, nil
}

// DispatcherDeps builds DispatchPendingDeliveries's dependencies.
func (m Module) DispatcherDeps(jobRepo jobsports.JobRepository, quotaRepo tenantports.QuotaRepository) usecases.DispatcherDeps {
	return usecases.DispatcherDeps{DeliveryRepo: m.DeliveryRepo, Enqueuer: jobEnqueuer{Repo: jobRepo, QuotaRepo: quotaRepo}}
}

// JobHandlerDeps builds ProvisioningDeliveryHandler's dependencies.
func (m Module) JobHandlerDeps(attrSource ports.AttributeSource, newTargetClient func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error)) usecases.JobHandlerDeps {
	return usecases.JobHandlerDeps{
		DeliverDeps: usecases.DeliverDeps{
			ConnectionRepo: m.ConnectionRepo, DeliveryRepo: m.DeliveryRepo, LinkRepo: m.RemoteLinkRepo,
			AttributeSource: attrSource, NewTargetClient: newTargetClient,
		},
		ConnectionRepo: m.ConnectionRepo, DeliveryRepo: m.DeliveryRepo,
	}
}

// Register registers the Application-detail "provisioning" subroute and the
// tenant-wide aggregate view (spec/contexts/provisioning.yaml §設定の置き場所).
func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator, assignmentRepo appports.AssignmentRepository, userRepo userports.UserRepository) {
	provisioninghttp.RegisterRoutes(g, provisioninghttp.Deps{
		Deps: deps, Authenticator: authenticator,
		ConnectionRepo: m.ConnectionRepo, DeliveryRepo: m.DeliveryRepo,
		AssignmentRepo: assignmentRepo, UserRepo: userRepo, NewTargetClient: NewTargetClient,
	})
}

// KindProvisioningDelivery re-exports usecases.KindProvisioningDelivery so
// bootstrap/worker code doesn't need to import backend/provisioning/usecases
// directly just for the Jobs handler registry key.
const KindProvisioningDelivery = usecases.KindProvisioningDelivery

// Handler re-exports usecases.ProvisioningDeliveryHandler for worker wiring.
func Handler(deps usecases.JobHandlerDeps) jobsusecases.Handler {
	return usecases.ProvisioningDeliveryHandler(deps)
}
