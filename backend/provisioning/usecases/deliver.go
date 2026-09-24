package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// DeliverDeps are ExecuteDelivery's dependencies.
type DeliverDeps struct {
	ConnectionRepo  ports.ProvisioningConnectionRepository
	DeliveryRepo    ports.ProvisioningDeliveryRepository
	LinkRepo        ports.RemoteResourceLinkRepository
	AttributeSource ports.AttributeSource
	// GroupMemberSource reads a Group's direct members when push_groups is on.
	// nil means membership is not pushed; the Group's own attributes still are.
	GroupMemberSource ports.GroupMemberSource
	// NewTargetClient builds the protocol client for conn using its (already
	// resolved) credential secret. Production wiring returns a *scim.Client;
	// tests inject a fake.
	NewTargetClient func(conn *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error)
}

var ErrDeliveryNotFound = errors.New("provisioning: delivery not found")

var ErrConnectionNotFound = errors.New("provisioning: connection not found")

// ExecuteDelivery performs one ProvisioningDelivery's downstream operation and
// updates RemoteResourceLink/ProvisioningDelivery on success. On success it
// returns the ProvisioningDeliveryLifecycle event of the in_flight → succeeded
// transition for the caller to emit once the rest of its bookkeeping is saved,
// or nil when nothing reached the downstream (the subject is absent there or at
// the source, or the delivery had already settled). It returns the
// error unchanged (without touching delivery status) when the downstream call
// fails: ProvisioningDeliveryLifecycle keeps status=in_flight for the whole
// Jobs-level attempt/retry loop (spec/contexts/provisioning.yaml
// states.ProvisioningDeliveryLifecycle), so the caller (the Jobs handler
// wrapper) decides dead_letter based on the Job's own attempts vs max_attempts.
func ExecuteDelivery(ctx context.Context, deps DeliverDeps, tenantID, deliveryID string, now time.Time) (spec.DomainEvent, error) {
	delivery, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, ErrDeliveryNotFound
	}
	if domain.IsProvisioningDeliveryTerminal(delivery.Status) {
		// Jobs が同じジョブを再実行しても、確定した配信を下流へ送り直さない。
		return nil, nil //nolint:nilnil // 送らなかった配信に遷移イベントは無い
	}
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, delivery.ConnectionID)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, ErrConnectionNotFound
	}
	secret, err := deps.ConnectionRepo.CredentialSecret(ctx, tenantID, conn.ApplicationID)
	if err != nil {
		return nil, err
	}
	client, err := deps.NewTargetClient(conn, secret)
	if err != nil {
		return nil, err
	}
	link, err := deps.LinkRepo.Find(ctx, conn.ApplicationID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return nil, err
	}

	var event spec.DomainEvent
	switch delivery.SourceType {
	case domain.SourceTypeUser:
		event, err = deliverUser(ctx, deps, client, conn, delivery, link, now)
	case domain.SourceTypeGroup:
		event, err = deliverGroup(ctx, deps, client, conn, delivery, link, now)
	default:
		err = fmt.Errorf("provisioning: unsupported source_type %q", delivery.SourceType)
	}
	if err != nil {
		return nil, err
	}
	if err := deps.DeliveryRepo.UpdateStatus(ctx, tenantID, deliveryID, domain.DeliverySucceeded, nil); err != nil {
		return nil, err
	}
	return event, nil
}

func deliverUser(ctx context.Context, deps DeliverDeps, client ports.ProvisioningTargetClient, conn *domain.ProvisioningConnection, delivery *domain.ProvisioningDelivery, link *domain.RemoteResourceLink, now time.Time) (spec.DomainEvent, error) {
	remoteID := ""
	if link != nil {
		remoteID = link.RemoteID
	}
	deprovisioned := func(action domain.ProvisioningDeprovisionAction) spec.DomainEvent {
		return &domain.UserDeprovisioned{
			At: now, TenantID: delivery.TenantID, ConnectionID: delivery.ConnectionID, DeliveryID: delivery.ID,
			UserID: delivery.SourceID, Action: action,
		}
	}
	if delivery.Operation == domain.OperationDelete {
		if remoteID == "" {
			return nil, nil //nolint:nilnil // 下流に既に無いので冪等に成功し、遷移イベントは無い
		}
		if err := client.DeleteUser(ctx, remoteID); err != nil {
			return nil, err
		}
		return deprovisioned(domain.DeprovisionDelete), nil
	}
	deactivating := delivery.Operation == domain.OperationDeactivate
	if deactivating && remoteID == "" {
		// 下流に無い User を無効化するために作成はしない。
		return nil, nil //nolint:nilnil // 送らなかった配信に遷移イベントは無い
	}

	attrs, exists, err := deps.AttributeSource.ResolveAttributes(ctx, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil //nolint:nilnil // source aggregate is gone; nothing to provision
	}
	if deactivating {
		// 割り当て解除の無効化では User 自身は有効なままなので、属性源の active は
		// true を返す。下流での有効性は User の状態ではなく配信の操作が決める。
		attrs["active"] = false
	}

	if remoteID == "" {
		newID, _, err := client.CreateUser(ctx, conn.AttributeMappings, attrs)
		var conflict *ports.ConflictError
		if ports.AsConflictError(err, &conflict) {
			newID, err = adoptExistingUser(ctx, client, conn.Matching, attrs)
		}
		if err != nil {
			return nil, err
		}
		remoteID = newID
	} else {
		_, err := client.UpdateUser(ctx, remoteID, conn.AttributeMappings, attrs, conn.Capabilities != nil && conn.Capabilities.SupportsPatch)
		var notFound *ports.NotFoundError
		switch {
		case ports.AsNotFoundError(err, &notFound) && deactivating:
			return nil, nil //nolint:nilnil // 無効化する対象が下流から消えている
		case ports.AsNotFoundError(err, &notFound):
			newID, _, createErr := client.CreateUser(ctx, conn.AttributeMappings, attrs)
			if createErr != nil {
				return nil, createErr
			}
			remoteID = newID
		case err != nil:
			return nil, err
		}
	}

	var event spec.DomainEvent = &domain.UserProvisioned{
		At: now, TenantID: delivery.TenantID, ConnectionID: delivery.ConnectionID, DeliveryID: delivery.ID,
		UserID: delivery.SourceID, RemoteID: remoteID,
	}
	if deactivating {
		event = deprovisioned(domain.DeprovisionDeactivate)
	}
	if err := upsertLink(ctx, deps, conn, delivery, link, remoteID, now); err != nil {
		return nil, err
	}
	return event, nil
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

func deliverGroup(ctx context.Context, deps DeliverDeps, client ports.ProvisioningTargetClient, conn *domain.ProvisioningConnection, delivery *domain.ProvisioningDelivery, link *domain.RemoteResourceLink, now time.Time) (spec.DomainEvent, error) {
	pushed := func(remoteID string) spec.DomainEvent {
		return &domain.GroupPushed{
			At: now, TenantID: delivery.TenantID, ConnectionID: delivery.ConnectionID, DeliveryID: delivery.ID,
			GroupID: delivery.SourceID, RemoteID: remoteID,
		}
	}
	if delivery.Operation == domain.OperationDelete {
		if link == nil {
			return nil, nil //nolint:nilnil // 送らなかった配信に遷移イベントは無い
		}
		if err := client.DeleteGroup(ctx, link.RemoteID); err != nil {
			return nil, err
		}
		return pushed(link.RemoteID), nil
	}
	attrs, exists, err := deps.AttributeSource.ResolveAttributes(ctx, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil //nolint:nilnil // 送らなかった配信に遷移イベントは無い
	}
	attrs["display_name"] = groupDisplayName(attrs, conn.GroupPush)
	remoteID := ""
	if link != nil {
		remoteID = link.RemoteID
	}
	if remoteID == "" {
		newID, _, err := client.CreateGroup(ctx, conn.AttributeMappings, attrs)
		if err != nil {
			return nil, err
		}
		remoteID = newID
	} else if _, err := client.UpdateGroup(ctx, remoteID, conn.AttributeMappings, attrs, conn.Capabilities != nil && conn.Capabilities.SupportsPatch); err != nil {
		return nil, err
	}
	if err := pushGroupMembers(ctx, deps, client, conn, delivery, remoteID); err != nil {
		return nil, err
	}
	if err := upsertLink(ctx, deps, conn, delivery, link, remoteID, now); err != nil {
		return nil, err
	}
	return pushed(remoteID), nil
}

// upsertLink records remoteID as the subject's downstream resource. A delivery
// older than the one the link already reflects leaves it unchanged: the
// downstream call has still happened, so the delivery settles as succeeded.
func upsertLink(ctx context.Context, deps DeliverDeps, conn *domain.ProvisioningConnection, delivery *domain.ProvisioningDelivery, link *domain.RemoteResourceLink, remoteID string, now time.Time) error {
	newLink := domain.NewRemoteResourceLink(conn.ApplicationID, delivery.TenantID, delivery.SourceType, delivery.SourceID)
	if link != nil {
		*newLink = *link
	}
	if err := newLink.ApplySync(delivery.SourceVersion, remoteID, delivery.SourceID, nil, now); err != nil {
		if errors.Is(err, domain.ErrOutOfOrderSync) {
			return nil
		}
		return err
	}
	return deps.LinkRepo.Upsert(ctx, newLink)
}

// groupDisplayName resolves the attribute the connection picked as the Group's
// downstream `displayName` (docs/domain/provisioning/standards.md
// RFC7643-OUT-GROUP-RESOURCES).
//
// Which attribute that is belongs to the connection, not to the Group, which is
// why the attribute source resolves the Group's facts and this resolves the
// choice: ports.AttributeSource is not handed the connection, and the delivery
// engine already holds it.
//
// A Group that has not set the chosen attribute falls back to its name. Sending
// an empty displayName is not an option either — a downstream that validates its
// Group representation refuses it, which would turn one mistyped setting into a
// Group that never pushes.
func groupDisplayName(attrs map[string]any, config *domain.GroupPushConfig) any {
	if value, ok := attrs[config.DisplayNameSourceKey()].(string); ok && value != "" {
		return value
	}
	return attrs["name"]
}

// pushGroupMembers sends the Group's current direct members downstream as one
// incremental `add` (spec/contexts/provisioning.yaml events.GroupMembershipPushed).
//
// Only members IdMagic has already provisioned are sent: a member with no
// RemoteResourceLink has no downstream id to name, and inventing one would make
// the downstream create a resource this connection does not own. Sending `add`
// for a member the downstream already has is a no-op there, which is what lets
// this converge without IdMagic tracking downstream membership itself.
//
// Removal is deliberately not sent. Knowing which members to remove means
// knowing the downstream's current set, and IdMagic does not read it back; a
// wholesale replace would delete members this connection never added. See the
// work item's Out of Scope.
func pushGroupMembers(
	ctx context.Context,
	deps DeliverDeps,
	client ports.ProvisioningTargetClient,
	conn *domain.ProvisioningConnection,
	delivery *domain.ProvisioningDelivery,
	remoteGroupID string,
) error {
	if deps.GroupMemberSource == nil || remoteGroupID == "" {
		return nil
	}
	userIDs, err := deps.GroupMemberSource.ListMemberUserIDs(ctx, delivery.TenantID, delivery.SourceID)
	if err != nil {
		return err
	}
	remoteUserIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		memberLink, err := deps.LinkRepo.Find(ctx, conn.ApplicationID, domain.SourceTypeUser, userID)
		if err != nil {
			return err
		}
		if memberLink != nil && memberLink.RemoteID != "" {
			remoteUserIDs = append(remoteUserIDs, memberLink.RemoteID)
		}
	}
	if len(remoteUserIDs) == 0 {
		return nil
	}
	return client.PatchGroupMembers(ctx, remoteGroupID, "add", remoteUserIDs)
}
