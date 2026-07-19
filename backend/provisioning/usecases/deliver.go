package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// DeliverDeps are ExecuteDelivery's dependencies.
type DeliverDeps struct {
	ConnectionRepo  ports.ProvisioningConnectionRepository
	DeliveryRepo    ports.ProvisioningDeliveryRepository
	LinkRepo        ports.RemoteResourceLinkRepository
	AttributeSource ports.AttributeSource
	// NewTargetClient builds the protocol client for conn using its (already
	// resolved) credential secret. Production wiring returns a *scim.Client;
	// tests inject a fake.
	NewTargetClient func(conn *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error)
}

var ErrDeliveryNotFound = errors.New("provisioning: delivery not found")

var ErrConnectionNotFound = errors.New("provisioning: connection not found")

// ExecuteDelivery performs one ProvisioningDelivery's downstream operation and
// updates RemoteResourceLink/ProvisioningDelivery on success. It returns the
// error unchanged (without touching delivery status) when the downstream call
// fails: ProvisioningDeliveryLifecycle keeps status=in_flight for the whole
// Jobs-level attempt/retry loop (spec/contexts/provisioning.yaml
// states.ProvisioningDeliveryLifecycle), so the caller (the Jobs handler
// wrapper) decides dead_letter based on the Job's own attempts vs max_attempts.
func ExecuteDelivery(ctx context.Context, deps DeliverDeps, tenantID, deliveryID string, now time.Time) error {
	delivery, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil {
		return err
	}
	if delivery == nil {
		return ErrDeliveryNotFound
	}
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, delivery.ConnectionID)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrConnectionNotFound
	}
	secret, err := deps.ConnectionRepo.CredentialSecret(ctx, tenantID, conn.ApplicationID)
	if err != nil {
		return err
	}
	client, err := deps.NewTargetClient(conn, secret)
	if err != nil {
		return err
	}
	link, err := deps.LinkRepo.Find(ctx, conn.ApplicationID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return err
	}

	switch delivery.SourceType {
	case domain.SourceTypeUser:
		err = deliverUser(ctx, deps, client, conn, delivery, link, now)
	case domain.SourceTypeGroup:
		err = deliverGroup(ctx, deps, client, conn, delivery, link, now)
	default:
		err = fmt.Errorf("provisioning: unsupported source_type %q", delivery.SourceType)
	}
	if err != nil {
		return err
	}
	return deps.DeliveryRepo.UpdateStatus(ctx, tenantID, deliveryID, domain.DeliverySucceeded, nil)
}

func deliverUser(ctx context.Context, deps DeliverDeps, client ports.ProvisioningTargetClient, conn *domain.ProvisioningConnection, delivery *domain.ProvisioningDelivery, link *domain.RemoteResourceLink, now time.Time) error {
	if delivery.Operation == domain.OperationDelete {
		if link == nil {
			return nil // already absent downstream: idempotent success
		}
		if err := client.DeleteUser(ctx, link.RemoteID); err != nil {
			return err
		}
		return nil
	}

	attrs, exists, err := deps.AttributeSource.ResolveAttributes(ctx, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return err
	}
	if !exists {
		return nil // source aggregate is gone; nothing to provision
	}

	remoteID := ""
	if link != nil {
		remoteID = link.RemoteID
	}
	if remoteID == "" {
		newID, _, err := client.CreateUser(ctx, conn.AttributeMappings, attrs)
		var conflict *ports.ConflictError
		if ports.AsConflictError(err, &conflict) {
			newID, err = adoptExistingUser(ctx, client, conn.Matching, attrs)
		}
		if err != nil {
			return err
		}
		remoteID = newID
	} else {
		_, err := client.UpdateUser(ctx, remoteID, conn.AttributeMappings, attrs, conn.Capabilities != nil && conn.Capabilities.SupportsPatch)
		var notFound *ports.NotFoundError
		if ports.AsNotFoundError(err, &notFound) {
			newID, _, createErr := client.CreateUser(ctx, conn.AttributeMappings, attrs)
			if createErr != nil {
				return createErr
			}
			remoteID = newID
		} else if err != nil {
			return err
		}
	}

	newLink := domain.NewRemoteResourceLink(conn.ApplicationID, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if link != nil {
		*newLink = *link
	}
	if err := newLink.ApplySync(delivery.SourceVersion, remoteID, delivery.SourceID, nil, now); err != nil {
		if !errors.Is(err, domain.ErrOutOfOrderSync) {
			return err
		}
		return nil // an even newer delivery already applied; this one is stale, treat as success
	}
	return deps.LinkRepo.Upsert(ctx, newLink)
}

func adoptExistingUser(ctx context.Context, client ports.ProvisioningTargetClient, matching domain.MatchingRule, attrs map[string]any) (string, error) {
	attribute := matching.ConflictMatchAttribute
	if attribute == "" {
		attribute = "userName"
	}
	value, _ := attrs["preferred_username"].(string)
	remoteID, found, err := client.SearchUserByAttribute(ctx, attribute, value)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("provisioning: 409 conflict but no existing resource found by %s=%q", attribute, value)
	}
	return remoteID, nil
}

func deliverGroup(ctx context.Context, deps DeliverDeps, client ports.ProvisioningTargetClient, conn *domain.ProvisioningConnection, delivery *domain.ProvisioningDelivery, link *domain.RemoteResourceLink, now time.Time) error {
	if delivery.Operation == domain.OperationDelete {
		if link == nil {
			return nil
		}
		return client.DeleteGroup(ctx, link.RemoteID)
	}
	attrs, exists, err := deps.AttributeSource.ResolveAttributes(ctx, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	remoteID := ""
	if link != nil {
		remoteID = link.RemoteID
	}
	if remoteID == "" {
		newID, _, err := client.CreateGroup(ctx, conn.AttributeMappings, attrs)
		if err != nil {
			return err
		}
		remoteID = newID
	} else if _, err := client.UpdateGroup(ctx, remoteID, conn.AttributeMappings, attrs, conn.Capabilities != nil && conn.Capabilities.SupportsPatch); err != nil {
		return err
	}
	newLink := domain.NewRemoteResourceLink(conn.ApplicationID, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if link != nil {
		*newLink = *link
	}
	if err := newLink.ApplySync(delivery.SourceVersion, remoteID, delivery.SourceID, nil, now); err != nil {
		if !errors.Is(err, domain.ErrOutOfOrderSync) {
			return err
		}
		return nil
	}
	return deps.LinkRepo.Upsert(ctx, newLink)
}
