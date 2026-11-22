package usecases

// 管理者向け Application メタデータ操作 (Create / Update / Delete)。

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/mediavalidation"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var ErrApplicationNotFound = appports.ErrApplicationNotFound

var (
	ErrApplicationIconRequired = errors.New("application icon file is required")
	ErrApplicationIconTooLarge = errors.New("application icon exceeds 256KiB")
	ErrApplicationIconFormat   = errors.New("application icon must be PNG, JPEG, WebP, or GIF")
)

const MaxApplicationIconBytes = 256 * 1024

type ApplicationDeps struct {
	Repo           appports.ApplicationRepository
	IconStore      appports.ApplicationIconStore
	AssignmentRepo appports.AssignmentRepository
	PolicyRepo     appports.SignInPolicyDeleter
	Emit           func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's Hard Quota on applications (wi-160). nil skips enforcement (wiring gaps in tests/tools);
	// production bootstrap always sets it.
	QuotaRepo      tenantports.QuotaRepository
	DeleteProtocol func(context.Context, *appdomain.Application) (int, error)
}

// reserveApplicationQuota reserves one unit of the tenant's applications Hard
// Quota before a new Application is persisted. The caller releases it unless the
// Application is saved (REQ-TENANCY-050). A rejection also emits QuotaExceeded so
// quota pressure is auditable (SCL objective QuotaAudit).
func reserveApplicationQuota(ctx context.Context, deps ApplicationDeps, tenantID string, now time.Time) (*tenantports.QuotaReservation, error) {
	reservation, err := tenantports.ReserveQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceApplications, 1)
	if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
		emit(deps.Emit, &tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
	}
	return reservation, err
}

type CreateApplicationInput struct {
	ActorUserID string
	Name        string
	Kind        appdomain.ApplicationKind
	LaunchURL   string
	Protocol    *appdomain.ApplicationProtocol
	Now         time.Time
}

func CreateApplication(ctx context.Context, deps ApplicationDeps, in CreateApplicationInput) (*appdomain.Application, error) {
	tenantID := tenantports.TenantID(ctx)
	now := adminNow(in.Now)
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	return createApplication(ctx, deps, in, tenantID, id, now)
}

// CreateSeedApplication は固定 ID を共通の検証、使用量管理とイベントへ通す。
func CreateSeedApplication(ctx context.Context, deps ApplicationDeps, app *appdomain.Application) error {
	if app.ID == "" {
		return errors.New("seed application id is required")
	}
	_, err := createApplication(ctx, deps, CreateApplicationInput{
		ActorUserID: "seed", Name: app.Name, Kind: app.Kind, LaunchURL: app.LaunchURL, Protocol: app.Protocol,
	}, app.TenantID, app.ID, app.CreatedAt)
	return err
}

func createApplication(ctx context.Context, deps ApplicationDeps, in CreateApplicationInput, tenantID, id string, now time.Time) (*appdomain.Application, error) {
	app := &appdomain.Application{
		TenantID:  tenantID,
		ID:        id,
		Name:      strings.TrimSpace(in.Name),
		Kind:      in.Kind,
		Status:    appdomain.ApplicationActive,
		LaunchURL: strings.TrimSpace(in.LaunchURL),
		Protocol:  cloneProtocol(in.Protocol),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := appdomain.ValidateApplication(app); err != nil {
		return nil, err
	}
	reservation, err := reserveApplicationQuota(ctx, deps, tenantID, now)
	if err != nil {
		return nil, err
	}
	defer reservation.Release(ctx)
	if err := deps.Repo.Create(ctx, app); err != nil {
		return nil, err
	}
	reservation.Keep()
	emit(deps.Emit, &appdomain.ApplicationCreated{At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: id})
	return app, nil
}

type UpdateApplicationInput struct {
	ActorUserID   string
	ApplicationID string
	Name          *string
	Status        *appdomain.ApplicationStatus
	LaunchURL     *string
	Now           time.Time
}

func UpdateApplication(ctx context.Context, deps ApplicationDeps, in UpdateApplicationInput) (*appdomain.Application, error) {
	tenantID := tenantports.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrApplicationNotFound
	}
	updated := *app
	updated.Protocol = cloneProtocol(app.Protocol)
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
	if err := appdomain.ValidateApplication(&updated); err != nil {
		return nil, err
	}
	updated.UpdatedAt = adminNow(in.Now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &appdomain.ApplicationUpdated{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: app.ID, ChangedFields: changed,
	})
	return &updated, nil
}

func DeleteApplication(ctx context.Context, deps ApplicationDeps, actorUserID, applicationID string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	app, err := deps.Repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return ErrApplicationNotFound
	}
	if err := deps.Repo.DeleteWithRelated(ctx, app, appports.DeletionParticipants{
		Assignments: deps.AssignmentRepo, Policy: deps.PolicyRepo, Icons: deps.IconStore,
		Quota: deps.QuotaRepo, DeleteProtocol: deps.DeleteProtocol,
	}); err != nil {
		return err
	}
	emit(deps.Emit, &appdomain.ApplicationDeleted{At: adminNow(now), TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID})
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

func UploadApplicationIcon(ctx context.Context, deps ApplicationDeps, in UploadApplicationIconInput) (*appdomain.Application, error) {
	if deps.IconStore == nil {
		return nil, errors.New("application icon store is not configured")
	}
	tenantID := tenantports.TenantID(ctx)
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
	icon := &appdomain.ApplicationIcon{
		TenantID: tenantID, ApplicationID: app.ID, ID: objectKey,
		ContentType: contentType, SizeBytes: len(in.Data), Data: slices.Clone(in.Data), CreatedAt: now, UpdatedAt: now,
	}
	if err := deps.IconStore.Save(ctx, icon); err != nil {
		return nil, err
	}
	updated := *app
	updated.Protocol = cloneProtocol(app.Protocol)
	updated.CategoryIDs = slices.Clone(app.CategoryIDs)
	updated.IconObjectKey = objectKey
	updated.IconURL = strings.TrimSpace(in.IconURL)
	updated.UpdatedAt = now
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &appdomain.ApplicationIconUpdated{
		At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: app.ID, Action: "uploaded",
	})
	return &updated, nil
}

func DeleteApplicationIcon(ctx context.Context, deps ApplicationDeps, actorUserID, applicationID string, now time.Time) (*appdomain.Application, error) {
	if deps.IconStore == nil {
		return nil, errors.New("application icon store is not configured")
	}
	tenantID := tenantports.TenantID(ctx)
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
	updated.Protocol = cloneProtocol(app.Protocol)
	updated.CategoryIDs = slices.Clone(app.CategoryIDs)
	updated.IconObjectKey = ""
	updated.IconURL = ""
	updated.UpdatedAt = adminNow(now)
	if err := deps.Repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &appdomain.ApplicationIconUpdated{
		At: updated.UpdatedAt, TenantID: tenantID, ActorUserID: actorUserID, ApplicationID: applicationID, Action: "deleted",
	})
	return &updated, nil
}

// DetectApplicationIconContentType は backend/shared/mediavalidation の magic byte
// 判定に委譲し、Application icon 固有のエラー値にマップする (wi-89, Tenant
// branding asset と検証ロジックを共有する)。
func DetectApplicationIconContentType(data []byte) (string, error) {
	contentType, err := mediavalidation.DetectImageContentType(data, MaxApplicationIconBytes)
	switch {
	case errors.Is(err, mediavalidation.ErrImageRequired):
		return "", ErrApplicationIconRequired
	case errors.Is(err, mediavalidation.ErrImageTooLarge):
		return "", ErrApplicationIconTooLarge
	case errors.Is(err, mediavalidation.ErrImageFormat):
		return "", fmt.Errorf("%w", ErrApplicationIconFormat)
	case err != nil:
		return "", err
	}
	return contentType, nil
}

func cloneProtocol(protocol *appdomain.ApplicationProtocol) *appdomain.ApplicationProtocol {
	if protocol == nil {
		return nil
	}
	cloned := *protocol
	return &cloned
}
