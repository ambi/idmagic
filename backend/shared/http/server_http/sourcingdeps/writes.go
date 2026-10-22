package sourcingdeps

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// writes は、SCIM の対応表と所有者の作成を結び、通知を確定後へ遅らせる。
// 属性の投影と業務上の判断は含めない。
type writes struct {
	idmanagement.SourceCommands
	identity  idmanagement.Module
	lifecycle idmanagement.LifecycleDeps
	source    sourcing.Module
	emit      func(spec.DomainEvent)
}

func NewWrites(identity idmanagement.Module, lifecycle idmanagement.LifecycleDeps, source sourcing.Module, emit func(spec.DomainEvent)) scimports.IdentityWrites {
	if identity.UserRepo == nil || identity.GroupRepo == nil || source.ScimRepo == nil || lifecycle.QuotaRepo == nil || emit == nil {
		panic("SCIM writes require identity repositories, source references, quotas, and an event sink")
	}
	return &writes{
		identity: identity, lifecycle: lifecycle, source: source, emit: emit,
		SourceCommands: identity.SourceCommands(lifecycle, func(event spec.DomainEvent) error { emit(event); return nil }),
	}
}

func (w *writes) create(ctx context.Context, apply func(idmanagement.SourceCommands, scimports.ScimRepository) error) error {
	effects := &creationEffects{identity: w.identity, emit: w.emit}
	err := w.source.CreateInTransaction(ctx, func(tx pgx.Tx, refs scimports.ScimRepository) error {
		identity, lifecycle := w.identity, w.lifecycle
		if tx != nil {
			db := postgres.BindTransaction(tx)
			identity = idmanagement.NewPostgresModule(db)
			identity.UserMutationCommitter = idgovernance.NewPostgresModule(db, identity.UserRepo, idmanagement.UserWriterInTx).UserMutationCommitter
			lifecycle.QuotaRepo = tenancy.QuotaRepositoryInTx(tx)
			lifecycle.AttrSchemaRepo = identity.AttrSchemaRepo
		}
		identity.ProvisioningNotifier, identity.GroupProvisioningNotifier = effects, effects
		return apply(identity.SourceCommands(lifecycle, effects.record), refs)
	})
	if err != nil {
		// 上限による拒否は、作成のロールバック後も記録する。
		for _, event := range effects.events {
			if _, refused := event.(*tenancydomain.QuotaExceeded); refused {
				w.emit(event)
			}
		}
		return err
	}
	var notificationErrors error
	for _, notify := range effects.afterCommit {
		notificationErrors = errors.Join(notificationErrors, notify())
	}
	return notificationErrors
}

func (w *writes) CreateUser(ctx context.Context, scimID string, user *userdomain.User) (*userdomain.User, error) {
	var created *userdomain.User
	err := w.create(ctx, func(commands idmanagement.SourceCommands, refs scimports.ScimRepository) error {
		var err error
		created, err = commands.CreateUser(ctx, user)
		if err != nil {
			return err
		}
		return refs.SaveUserRef(ctx, &scimports.ScimUserRef{TenantID: created.TenantID, ScimID: scimID, UserID: created.ID})
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (w *writes) CreateGroup(ctx context.Context, scimID string, group *groupdomain.Group, members []string) (*groupdomain.Group, error) {
	var created *groupdomain.Group
	err := w.create(ctx, func(commands idmanagement.SourceCommands, refs scimports.ScimRepository) error {
		var err error
		created, err = commands.CreateGroup(ctx, group)
		if err != nil {
			return err
		}
		for _, member := range members {
			if _, err := commands.SetGroupMembership(ctx, created.TenantID, created.ID, member, true, created.CreatedAt); err != nil {
				return err
			}
		}
		return refs.SaveGroupRef(ctx, &scimports.ScimGroupRef{TenantID: created.TenantID, ScimID: scimID, GroupID: created.ID})
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

type creationEffects struct {
	identity    idmanagement.Module
	emit        func(spec.DomainEvent)
	events      []spec.DomainEvent
	afterCommit []func() error
}

func (e *creationEffects) record(event spec.DomainEvent) error {
	e.events = append(e.events, event)
	e.afterCommit = append(e.afterCommit, func() error { e.emit(event); return nil })
	return nil
}

func (e *creationEffects) NotifyUserMutation(ctx context.Context, tenantID, userID string, trigger userports.ProvisioningTrigger, now time.Time) error {
	if notifier := e.identity.ProvisioningNotifier; notifier != nil {
		e.afterCommit = append(e.afterCommit, func() error {
			if err := notifier.NotifyUserMutation(ctx, tenantID, userID, trigger, now); err != nil {
				logging.Error(ctx, "provisioning: capture notification failed", "error", err, "user_id", userID)
			}
			return nil
		})
	}
	return nil
}

func (e *creationEffects) NotifyGroupMutation(ctx context.Context, tenantID, groupID string, trigger groupports.ProvisioningTrigger, now time.Time) error {
	if notifier := e.identity.GroupProvisioningNotifier; notifier != nil {
		e.afterCommit = append(e.afterCommit, func() error { return notifier.NotifyGroupMutation(ctx, tenantID, groupID, trigger, now) })
	}
	return nil
}
