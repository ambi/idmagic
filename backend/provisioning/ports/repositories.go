// Package ports defines the Provisioning bounded context's repository
// abstractions (spec/contexts/provisioning.yaml). Implementations live in
// backend/provisioning/adapters/persistence/{memory,postgres}.
package ports

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// ErrConnectionAlreadyExists is returned by Register when the Application
// already has a ProvisioningConnection (spec/contexts/provisioning.yaml
// errors.ProvisioningConnectionAlreadyExistsError, "1 Application 1 connection").
var ErrConnectionAlreadyExists = errors.New("provisioning: connection already exists for this application")

// ProvisioningConnectionRepository persists ProvisioningConnection aggregates.
// CredentialSecret is a narrow accessor separate from Find: only the delivery
// engine (T006) may call it to authenticate outbound requests, so admin read
// paths (which use Find) never see the plaintext/opaque secret
// (spec/contexts/provisioning.yaml credential write-only 契約).
type ProvisioningConnectionRepository interface {
	// Register inserts a new connection together with its credential secret.
	// Returns ErrConnectionAlreadyExists if the Application already has one.
	Register(ctx context.Context, conn *domain.ProvisioningConnection, secret string) error
	// Update replaces the full connection record. secret is non-nil only when
	// the caller is rotating the credential (ProvisioningCredentialRotated).
	Update(ctx context.Context, conn *domain.ProvisioningConnection, secret *string) error
	Find(ctx context.Context, tenantID, applicationID string) (*domain.ProvisioningConnection, error)
	// CredentialSecret returns the plaintext/opaque secret for outbound calls.
	CredentialSecret(ctx context.Context, tenantID, applicationID string) (string, error)
	Delete(ctx context.Context, tenantID, applicationID string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.ProvisioningConnection, error)
}

// RemoteResourceLinkRepository persists the correlation between an idmagic
// User/Group and its downstream SCIM resource.
type RemoteResourceLinkRepository interface {
	Find(ctx context.Context, connectionID string, sourceType domain.ProvisioningSourceType, sourceID string) (*domain.RemoteResourceLink, error)
	// Upsert inserts the link on first sync, or updates it on a later sync. The
	// caller applies RemoteResourceLink.ApplySync's monotonicity check before
	// calling Upsert; Upsert itself does not re-derive ordering.
	Upsert(ctx context.Context, link *domain.RemoteResourceLink) error
}

// ProvisioningDeliveryRepository persists ProvisioningDelivery records.
type ProvisioningDeliveryRepository interface {
	// Save inserts a new delivery. It returns created=false without error when
	// an existing delivery already has the same idempotency key
	// (tenant_id, connection_id, source_type, source_id, source_version).
	Save(ctx context.Context, d *domain.ProvisioningDelivery) (created bool, err error)
	Find(ctx context.Context, tenantID, deliveryID string) (*domain.ProvisioningDelivery, error)
	// ListByConnection lists deliveries for a connection, most recent first.
	// status filters to a single status when non-nil.
	ListByConnection(ctx context.Context, tenantID, connectionID string, status *domain.ProvisioningDeliveryStatus, limit int) ([]*domain.ProvisioningDelivery, error)
	// ListUnenqueued returns pending deliveries with no Jobs.Job associated yet
	// (dispatcher recovery, LifecycleWorkflowRunLifecycle precedent).
	ListUnenqueued(ctx context.Context, limit int) ([]*domain.ProvisioningDelivery, error)
	// AttachJob associates a Jobs.Job with a pending, unattached delivery.
	AttachJob(ctx context.Context, tenantID, deliveryID, jobID string) (attached bool, err error)
	// UpdateStatus transitions a delivery's status and records the last error
	// (nil clears it). Callers are responsible for using a
	// domain.TransitionProvisioningDeliveryLifecycle-valid target status.
	UpdateStatus(ctx context.Context, tenantID, deliveryID string, status domain.ProvisioningDeliveryStatus, lastError *string) error
	// RetryDeadLetter resets a dead_letter delivery to pending and clears its
	// job_id so the dispatcher picks it up again. Returns false if the delivery
	// is not currently dead_letter.
	RetryDeadLetter(ctx context.Context, tenantID, deliveryID string) (bool, error)
}
