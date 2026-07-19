package usecases

import (
	"context"
	"errors"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// AdminDeps are the admin (Application-detail "provisioning" subroute)
// usecases' dependencies (spec/contexts/provisioning.yaml interfaces).
type AdminDeps struct {
	ConnectionRepo  ports.ProvisioningConnectionRepository
	DeliveryRepo    ports.ProvisioningDeliveryRepository
	AssignmentRepo  appports.AssignmentRepository
	UserRepo        idmports.UserRepository
	NewTargetClient func(conn *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error)
}

func (d AdminDeps) captureDeps() CaptureDeps {
	return CaptureDeps{ConnectionRepo: d.ConnectionRepo, DeliveryRepo: d.DeliveryRepo, AssignmentRepo: d.AssignmentRepo}
}

// defaultSCIMUserMapping seeds the SCIM core User mapping table
// (spec/contexts/provisioning.yaml §属性マッピングのセマンティクス 既定マッピング).
func defaultSCIMUserMapping() []domain.AttributeMappingRule {
	return []domain.AttributeMappingRule{
		{TargetPath: "externalId", SourceKind: domain.SourceKindAttribute, SourceKey: "id", ApplyOn: domain.ApplyCreateOnly, Required: true},
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "active", SourceKind: domain.SourceKindAttribute, SourceKey: "active", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "name.givenName", SourceKind: domain.SourceKindAttribute, SourceKey: "given_name", ApplyOn: domain.ApplyCreateAndUpdate},
		{TargetPath: "name.familyName", SourceKind: domain.SourceKindAttribute, SourceKey: "family_name", ApplyOn: domain.ApplyCreateAndUpdate},
		{TargetPath: "displayName", SourceKind: domain.SourceKindAttribute, SourceKey: "display_name", ApplyOn: domain.ApplyCreateAndUpdate},
		{TargetPath: `emails[type eq "work"].value`, SourceKind: domain.SourceKindAttribute, SourceKey: "email", ApplyOn: domain.ApplyCreateAndUpdate},
	}
}

// RegisterConnectionInput is RegisterConnection's input.
type RegisterConnectionInput struct {
	TenantID      string
	ApplicationID string
	BaseURL       string
	Credential    domain.ProvisioningCredentialInput
	Now           time.Time
}

// RegisterConnection creates a ProvisioningConnection with seeded defaults
// (spec/contexts/provisioning.yaml interfaces.RegisterProvisioningConnection).
func RegisterConnection(ctx context.Context, deps AdminDeps, in RegisterConnectionInput) (*domain.ProvisioningConnection, error) {
	if err := domain.ValidateOutboundBaseURL(in.BaseURL); err != nil {
		return nil, err
	}
	if err := validateCredential(in.Credential); err != nil {
		return nil, err
	}
	credentialID, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	now := in.Now.UTC()
	conn := &domain.ProvisioningConnection{
		ApplicationID:                     in.ApplicationID,
		TenantID:                          in.TenantID,
		Status:                            domain.ConnectionActive,
		BaseURL:                           in.BaseURL,
		Credential:                        domain.ProvisioningConnectionCredentialMetadata{CredentialID: credentialID, AuthMethod: in.Credential.AuthMethod, CreatedAt: now},
		FeatureFlags:                      domain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true},
		Scope:                             domain.ScopeAssignedOnly,
		AttributeMappings:                 defaultSCIMUserMapping(),
		Matching:                          domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy:                 domain.DeprovisionPolicy{OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate, OnGroupDeletedOrUnassigned: domain.DeprovisionNone},
		RateLimitPerMinute:                60,
		MaxAttempts:                       8,
		QuarantineAfterConsecutiveFailure: 10,
		Health:                            domain.HealthOK,
		CreatedAt:                         now,
		UpdatedAt:                         now,
	}
	if err := conn.Validate(); err != nil {
		return nil, err
	}
	if err := deps.ConnectionRepo.Register(ctx, conn, in.Credential.Secret()); err != nil {
		return nil, err
	}
	return conn, nil
}

// UpdateConnectionInput is UpdateConnection's input; nil fields are left
// unchanged (partial update).
type UpdateConnectionInput struct {
	TenantID                          string
	ApplicationID                     string
	BaseURL                           *string
	Status                            *domain.ProvisioningConnectionStatus
	Credential                        *domain.ProvisioningCredentialInput
	FeatureFlags                      *domain.ProvisioningFeatureFlags
	Scope                             *domain.ProvisioningScope
	GroupPush                         *domain.GroupPushConfig
	AttributeMappings                 *[]domain.AttributeMappingRule
	Matching                          *domain.MatchingRule
	DeprovisionPolicy                 *domain.DeprovisionPolicy
	RateLimitPerMinute                *int
	MaxAttempts                       *int
	NotificationEmail                 *string
	QuarantineAfterConsecutiveFailure *int
	Now                               time.Time
}

// UpdateConnection applies a partial update to an existing connection
// (spec/contexts/provisioning.yaml interfaces.UpdateProvisioningConnection).
func UpdateConnection(ctx context.Context, deps AdminDeps, in UpdateConnectionInput) (*domain.ProvisioningConnection, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, in.TenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	if in.BaseURL != nil {
		if err := domain.ValidateOutboundBaseURL(*in.BaseURL); err != nil {
			return nil, err
		}
		conn.BaseURL = *in.BaseURL
	}
	if in.Status != nil {
		conn.Status = *in.Status
	}
	if in.FeatureFlags != nil {
		conn.FeatureFlags = *in.FeatureFlags
	}
	if in.Scope != nil {
		conn.Scope = *in.Scope
	}
	if in.GroupPush != nil {
		conn.GroupPush = in.GroupPush
	}
	if in.AttributeMappings != nil {
		conn.AttributeMappings = *in.AttributeMappings
	}
	if in.Matching != nil {
		conn.Matching = *in.Matching
	}
	if in.DeprovisionPolicy != nil {
		conn.DeprovisionPolicy = *in.DeprovisionPolicy
	}
	if in.RateLimitPerMinute != nil {
		conn.RateLimitPerMinute = *in.RateLimitPerMinute
	}
	if in.MaxAttempts != nil {
		conn.MaxAttempts = *in.MaxAttempts
	}
	if in.NotificationEmail != nil {
		conn.NotificationEmail = in.NotificationEmail
	}
	if in.QuarantineAfterConsecutiveFailure != nil {
		conn.QuarantineAfterConsecutiveFailure = *in.QuarantineAfterConsecutiveFailure
	}
	conn.UpdatedAt = in.Now.UTC()
	if err := conn.Validate(); err != nil {
		return nil, err
	}
	var secret *string
	if in.Credential != nil {
		if err := validateCredential(*in.Credential); err != nil {
			return nil, err
		}
		credentialID, err := spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
		conn.Credential = domain.ProvisioningConnectionCredentialMetadata{CredentialID: credentialID, AuthMethod: in.Credential.AuthMethod, CreatedAt: conn.Credential.CreatedAt, RotatedAt: &conn.UpdatedAt}
		s := in.Credential.Secret()
		secret = &s
	}
	if err := deps.ConnectionRepo.Update(ctx, conn, secret); err != nil {
		return nil, err
	}
	return conn, nil
}

func validateCredential(c domain.ProvisioningCredentialInput) error {
	if !c.AuthMethod.Valid() {
		return errors.New("provisioning: invalid auth_method")
	}
	if c.AuthMethod == domain.AuthBearerToken && c.BearerToken == "" {
		return errors.New("provisioning: bearer_token is required for auth_method=bearer_token")
	}
	if c.AuthMethod == domain.AuthOAuth2ClientCredentials && (c.OAuth2TokenURL == "" || c.OAuth2ClientID == "" || c.OAuth2ClientSecret == "") {
		return errors.New("provisioning: oauth2 credential fields are required for auth_method=oauth2_client_credentials")
	}
	return nil
}

// GetConnection returns the connection for applicationID, or nil if none exists.
func GetConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string) (*domain.ProvisioningConnection, error) {
	return deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
}

// DeleteConnection removes the connection (hard delete, Application precedent).
func DeleteConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string) error {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrConnectionNotFound
	}
	return deps.ConnectionRepo.Delete(ctx, tenantID, applicationID)
}

// TestConnectionResult is TestConnection's output.
type TestConnectionResult struct {
	Reachable    bool                             `json:"reachable"`
	Capabilities *domain.ProvisioningCapabilities `json:"capabilities,omitempty"`
	Error        string                           `json:"error,omitempty"`
}

// TestConnection calls the downstream /ServiceProviderConfig and persists the
// discovered capabilities (spec/contexts/provisioning.yaml
// interfaces.TestProvisioningConnection).
func TestConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (TestConnectionResult, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return TestConnectionResult{}, err
	}
	if conn == nil {
		return TestConnectionResult{}, ErrConnectionNotFound
	}
	secret, err := deps.ConnectionRepo.CredentialSecret(ctx, tenantID, applicationID)
	if err != nil {
		return TestConnectionResult{}, err
	}
	client, err := deps.NewTargetClient(conn, secret)
	if err != nil {
		return TestConnectionResult{}, err
	}
	caps, discoverErr := client.Discover(ctx)
	if discoverErr != nil {
		return TestConnectionResult{Reachable: false, Error: discoverErr.Error()}, nil //nolint:nilerr // downstream unreachable is a business result (Reachable=false), not a usecase-level error
	}
	caps.DiscoveredAt = now.UTC()
	conn.Capabilities = &caps
	conn.UpdatedAt = now.UTC()
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return TestConnectionResult{}, err
	}
	return TestConnectionResult{Reachable: true, Capabilities: &caps}, nil
}

var ErrSubjectNotInScope = errors.New("provisioning: subject is not in this connection's scope")

// ProvisionOnDemand creates an immediate pending delivery for a single subject
// (spec/contexts/provisioning.yaml interfaces.ProvisionOnDemand).
func ProvisionOnDemand(ctx context.Context, deps AdminDeps, tenantID, applicationID string, sourceType domain.ProvisioningSourceType, sourceID string, now time.Time) (*domain.ProvisioningDelivery, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	if sourceType == domain.SourceTypeUser {
		ok, err := inScope(ctx, deps.captureDeps(), *conn, tenantID, sourceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrSubjectNotInScope
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	delivery := &domain.ProvisioningDelivery{
		ID: id, TenantID: tenantID, ConnectionID: applicationID, SourceType: sourceType, SourceID: sourceID,
		SourceVersion: now.UnixNano(), Operation: domain.OperationUpdate, Status: domain.DeliveryPending,
		CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if _, err := deps.DeliveryRepo.Save(ctx, delivery); err != nil {
		return nil, err
	}
	return delivery, nil
}

// StartFullResync enqueues an update delivery for every subject in the
// connection's scope (spec/contexts/provisioning.yaml interfaces.StartFullResync).
// FullResyncCompleted is not emitted by this scoped implementation: tracking a
// resync batch's completion across many asynchronous deliveries is left for a
// follow-up (wi-45 T007 known gap).
func StartFullResync(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (int, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectionNotFound
	}
	var subjectIDs []string
	if conn.Scope == domain.ScopeAllUsers {
		if deps.UserRepo == nil {
			return 0, errors.New("provisioning: all_users resync requires UserRepo")
		}
		users, err := deps.UserRepo.FindAll(ctx, tenantID)
		if err != nil {
			return 0, err
		}
		for _, u := range users {
			subjectIDs = append(subjectIDs, u.ID)
		}
	} else if deps.AssignmentRepo != nil {
		assignments, err := deps.AssignmentRepo.ListByApplication(ctx, tenantID, applicationID)
		if err != nil {
			return 0, err
		}
		for _, a := range assignments {
			if a.SubjectType == "user" {
				subjectIDs = append(subjectIDs, a.SubjectID)
			}
		}
	}
	enqueued := 0
	for _, subjectID := range subjectIDs {
		id, err := spec.NewUUIDv4()
		if err != nil {
			return enqueued, err
		}
		delivery := &domain.ProvisioningDelivery{
			ID: id, TenantID: tenantID, ConnectionID: applicationID, SourceType: domain.SourceTypeUser, SourceID: subjectID,
			SourceVersion: now.UnixNano(), Operation: domain.OperationUpdate, Status: domain.DeliveryPending,
			CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
		}
		created, err := deps.DeliveryRepo.Save(ctx, delivery)
		if err != nil {
			return enqueued, err
		}
		if created {
			enqueued++
		}
	}
	conn.LastFullSyncAt = new(now.UTC())
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return enqueued, err
	}
	return enqueued, nil
}

// ListDeliveries lists a connection's deliveries, most recent first.
func ListDeliveries(ctx context.Context, deps AdminDeps, tenantID, applicationID string, status *domain.ProvisioningDeliveryStatus, limit int) ([]*domain.ProvisioningDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	return deps.DeliveryRepo.ListByConnection(ctx, tenantID, applicationID, status, limit)
}

// GetDelivery returns one delivery, verifying it belongs to applicationID.
func GetDelivery(ctx context.Context, deps AdminDeps, tenantID, applicationID, deliveryID string) (*domain.ProvisioningDelivery, error) {
	d, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, err
	}
	if d == nil || d.ConnectionID != applicationID {
		return nil, ErrDeliveryNotFound
	}
	return d, nil
}

var ErrDeliveryNotRetryable = errors.New("provisioning: delivery is not dead_letter")

// RetryDelivery resets a dead_letter delivery to pending
// (spec/contexts/provisioning.yaml interfaces.RetryProvisioningDelivery).
func RetryDelivery(ctx context.Context, deps AdminDeps, tenantID, applicationID, deliveryID string) (*domain.ProvisioningDelivery, error) {
	d, err := GetDelivery(ctx, deps, tenantID, applicationID, deliveryID)
	if err != nil {
		return nil, err
	}
	if d.Status != domain.DeliveryDeadLetter {
		return nil, ErrDeliveryNotRetryable
	}
	retried, err := deps.DeliveryRepo.RetryDeadLetter(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, err
	}
	if !retried {
		return nil, ErrDeliveryNotRetryable
	}
	return deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
}

// ResumeConnection clears quarantine (spec/contexts/provisioning.yaml
// interfaces.ResumeProvisioningConnection).
func ResumeConnection(ctx context.Context, deps AdminDeps, tenantID, applicationID string, now time.Time) (*domain.ProvisioningConnection, error) {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	if err := conn.Resume(now); err != nil {
		return nil, err
	}
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return nil, err
	}
	return conn, nil
}

// ListTenantConnections lists every connection in the tenant
// (spec/contexts/provisioning.yaml interfaces.ListTenantProvisioningConnections).
func ListTenantConnections(ctx context.Context, deps AdminDeps, tenantID string) ([]*domain.ProvisioningConnection, error) {
	return deps.ConnectionRepo.ListByTenant(ctx, tenantID)
}
