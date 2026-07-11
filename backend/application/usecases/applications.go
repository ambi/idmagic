package usecases

// 管理者向け Application メタデータ操作 (Create / Update / Delete) と protocol binding
// の接続 / 解除。SCL Application bounded context の admin interface 群 (wi-69)。

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var ErrApplicationNotFound = errors.New("application not found")

var (
	ErrApplicationIconRequired = errors.New("application icon file is required")
	ErrApplicationIconTooLarge = errors.New("application icon exceeds 256KiB")
	ErrApplicationIconFormat   = errors.New("application icon must be PNG, JPEG, WebP, or GIF")
)

const MaxApplicationIconBytes = 256 * 1024

type ApplicationDeps struct {
	Repo              ports.ApplicationRepository
	IconStore         ports.ApplicationIconStore
	AssignmentRepo    ports.AssignmentRepository
	PolicyRepo        ports.SignInPolicyRepository
	Emit              func(spec.DomainEvent)
	TransactionalEmit func(spec.DomainEvent) error
}

type CreateApplicationInput struct {
	ActorUserID string
	Name        string
	Kind        domain.ApplicationKind
	LaunchURL   string
	Now         time.Time
}

func CreateApplication(ctx context.Context, deps ApplicationDeps, in CreateApplicationInput) (*domain.Application, error) {
	tenantID := tenancy.TenantID(ctx)
	now := adminNow(in.Now)
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	app := &domain.Application{
		TenantID:      tenantID,
		ApplicationID: id,
		Name:          strings.TrimSpace(in.Name),
		Kind:          in.Kind,
		Status:        domain.ApplicationActive,
		LaunchURL:     strings.TrimSpace(in.LaunchURL),
		Bindings:      []domain.ProtocolBinding{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := domain.ValidateApplication(app); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, app); err != nil {
		return nil, err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ApplicationCreated{At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: id}); err != nil {
		return nil, err
	}
	return app, nil
}

type UpdateApplicationInput struct {
	ActorUserID   string
	ApplicationID string
	Name          *string
	Status        *domain.ApplicationStatus
	LaunchURL     *string
	Now           time.Time
}

func UpdateApplication(ctx context.Context, deps ApplicationDeps, in UpdateApplicationInput) (*domain.Application, error) {
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	updated := *app
	updated.Bindings = slices.Clone(app.Bindings)
	changed := []string{}
	if in.Name != nil {
		if name := strings.TrimSpace(*in.Name); name != app.Name {
			updated.Name = name
			changed = append(changed, "name")
		}
	}
	if in.Status != nil && *in.Status != app.Status {
		updated.Status = *in.Status
		changed = append(changed, "status")
	}
	if in.LaunchURL != nil {
		if launch := strings.TrimSpace(*in.LaunchURL); launch != app.LaunchURL {
			updated.LaunchURL = launch
			changed = append(changed, "launch_url")
		}
	}
	if len(changed) == 0 {
		return &updated, nil
	}
	if err := domain.ValidateApplication(&updated); err != nil {
		return nil, err
	}
	updated.UpdatedAt = adminNow(in.Now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ApplicationUpdated{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: app.ApplicationID, ChangedFields: changed,
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DeleteApplication(ctx context.Context, deps ApplicationDeps, actorUserID, applicationID string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return ErrApplicationNotFound
	}
	if err := deps.AssignmentRepo.DeleteByApplication(ctx, tenantID, applicationID); err != nil {
		return err
	}
	if deps.PolicyRepo != nil {
		if err := deps.PolicyRepo.Delete(ctx, tenantID, applicationID); err != nil {
			return err
		}
	}
	if err := deps.Repo.Delete(ctx, tenantID, applicationID); err != nil {
		return err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ApplicationDeleted{At: adminNow(now), TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID}); err != nil {
		return err
	}
	return nil
}

type UploadApplicationIconInput struct {
	ActorUserID   string
	ApplicationID string
	ObjectKey     string
	Data          []byte
	IconURL       string
	Now           time.Time
}

func UploadApplicationIcon(ctx context.Context, deps ApplicationDeps, in UploadApplicationIconInput) (*domain.Application, error) {
	if deps.IconStore == nil {
		return nil, errors.New("application icon store is not configured")
	}
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	contentType, err := DetectApplicationIconContentType(in.Data)
	if err != nil {
		return nil, err
	}
	now := adminNow(in.Now)
	objectKey := strings.TrimSpace(in.ObjectKey)
	if objectKey == "" {
		var err error
		objectKey, err = spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
	}
	icon := &domain.ApplicationIcon{
		TenantID: tenantID, ApplicationID: app.ApplicationID, ObjectKey: objectKey,
		ContentType: contentType, SizeBytes: len(in.Data), Data: slices.Clone(in.Data), CreatedAt: now, UpdatedAt: now,
	}
	if err := deps.IconStore.Save(ctx, icon); err != nil {
		return nil, err
	}
	updated := *app
	updated.Bindings = slices.Clone(app.Bindings)
	updated.CategoryIDs = slices.Clone(app.CategoryIDs)
	updated.IconObjectKey = objectKey
	updated.IconURL = strings.TrimSpace(in.IconURL)
	updated.UpdatedAt = now
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ApplicationIconUpdated{
		At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: app.ApplicationID, Action: "uploaded",
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DeleteApplicationIcon(ctx context.Context, deps ApplicationDeps, actorUserID, applicationID string, now time.Time) (*domain.Application, error) {
	if deps.IconStore == nil {
		return nil, errors.New("application icon store is not configured")
	}
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	if err := deps.IconStore.DeleteByApplication(ctx, tenantID, applicationID); err != nil {
		return nil, err
	}
	updated := *app
	updated.Bindings = slices.Clone(app.Bindings)
	updated.CategoryIDs = slices.Clone(app.CategoryIDs)
	updated.IconObjectKey = ""
	updated.IconURL = ""
	updated.UpdatedAt = adminNow(now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ApplicationIconUpdated{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID, Action: "deleted",
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DetectApplicationIconContentType(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrApplicationIconRequired
	}
	if len(data) > MaxApplicationIconBytes {
		return "", ErrApplicationIconTooLarge
	}
	switch {
	case len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == '\r' && data[5] == '\n' && data[6] == 0x1a && data[7] == '\n':
		return "image/png", nil
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg", nil
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", nil
	case len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a"):
		return "image/gif", nil
	default:
		return "", fmt.Errorf("%w", ErrApplicationIconFormat)
	}
}

type AttachBindingInput struct {
	ActorUserID   string
	ApplicationID string
	Binding       domain.ProtocolBinding
	Now           time.Time
}

func AttachBinding(ctx context.Context, deps ApplicationDeps, in AttachBindingInput) (*domain.Application, error) {
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	if err := domain.ValidateBinding(in.Binding); err != nil {
		return nil, err
	}
	updated := *app
	updated.Bindings = slices.Clone(app.Bindings)
	// 同種別 binding は置き換える (1 application に 1 種別 1 binding)。
	updated.Bindings = slices.DeleteFunc(updated.Bindings, func(b domain.ProtocolBinding) bool {
		return b.Type == in.Binding.Type
	})
	updated.Bindings = append(updated.Bindings, in.Binding)
	if err := domain.ValidateApplication(&updated); err != nil {
		return nil, err
	}
	updated.UpdatedAt = adminNow(in.Now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ProtocolBindingAttached{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: app.ApplicationID, BindingType: string(in.Binding.Type),
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DetachBinding(ctx context.Context, deps ApplicationDeps, actorUserID, applicationID string, bindingType domain.ProtocolBindingType, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return ErrApplicationNotFound
	}
	updated := *app
	updated.Bindings = slices.DeleteFunc(slices.Clone(app.Bindings), func(b domain.ProtocolBinding) bool {
		return b.Type == bindingType
	})
	updated.UpdatedAt = adminNow(now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ProtocolBindingDetached{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID, BindingType: string(bindingType),
	}); err != nil {
		return err
	}
	return nil
}
