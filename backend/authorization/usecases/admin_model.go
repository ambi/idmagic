package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// PublishedModel は published した版と、その時点の整合トークン。
type PublishedModel struct {
	Model       *domain.AuthorizationModel
	Consistency string
}

// PutAuthorizationModel は検証を通ったモデルを新しい版として published する。
// 検証に落ちた版は保存しないので、実行時に初めて壊れるモデルは残らない。
func PutAuthorizationModel(
	ctx context.Context,
	d Deps,
	tenantID string,
	resourceTypes []domain.ResourceTypeDefinition,
	now time.Time,
) (PublishedModel, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return PublishedModel{}, err
	}
	candidate := &domain.AuthorizationModel{
		ID: id, TenantID: tenantID, ResourceTypes: resourceTypes, CreatedAt: now,
	}
	if err := candidate.Validate(); err != nil {
		return PublishedModel{}, err
	}
	stored, version, err := d.Models.Publish(ctx, candidate)
	if err != nil {
		return PublishedModel{}, err
	}
	d.emit(&domain.AuthorizationModelPublished{
		At: now, TenantID: tenantID, ModelID: stored.ID, Version: stored.Version,
		ResourceTypeCount: len(stored.ResourceTypes),
	})
	return PublishedModel{Model: stored, Consistency: domain.EncodeConsistencyToken(tenantID, version)}, nil
}

// GetAuthorizationModel は最新版、または version を指定した版を返す。
// 版が存在しない場合は ErrModelNotFound を返す。
func GetAuthorizationModel(ctx context.Context, d Deps, tenantID string, version int) (PublishedModel, error) {
	model, err := loadModel(ctx, d, tenantID, version)
	if err != nil {
		return PublishedModel{}, err
	}
	storeVersion, err := d.Tuples.Version(ctx, tenantID)
	if err != nil {
		return PublishedModel{}, err
	}
	return PublishedModel{Model: model, Consistency: domain.EncodeConsistencyToken(tenantID, storeVersion)}, nil
}

func loadModel(ctx context.Context, d Deps, tenantID string, version int) (*domain.AuthorizationModel, error) {
	var (
		model *domain.AuthorizationModel
		err   error
	)
	if version > 0 {
		model, err = d.Models.FindByVersion(ctx, tenantID, version)
	} else {
		model, err = d.Models.Latest(ctx, tenantID)
	}
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, fmt.Errorf("%w for tenant", domain.ErrModelNotFound)
	}
	return model, nil
}
