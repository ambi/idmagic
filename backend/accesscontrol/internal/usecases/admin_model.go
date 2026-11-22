package usecases

import (
	"context"
	"fmt"
	"time"

	accessdomain "github.com/ambi/idmagic/backend/accesscontrol/internal/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// PublishedModel は published したバージョンと、その時点の整合トークン。
type PublishedModel struct {
	Model       *accessdomain.AuthorizationModel
	Consistency string
}

// PutAuthorizationModel は検証を通ったモデルを新しいバージョンとして published する。
// 検証に落ちたバージョンは保存しないので、実行時に初めて壊れるモデルは残らない。
// 最新のバージョンが expectedVersion（未登録なら 0）でなければ domain.ErrModelVersionConflict
// を返し、先に確定した別の登録を上書きしない。
func PutAuthorizationModel(
	ctx context.Context,
	d Deps,
	tenantID string,
	expectedVersion int,
	resourceTypes []accessdomain.ResourceTypeDefinition,
	now time.Time,
) (PublishedModel, error) {
	for _, resourceType := range resourceTypes {
		if resourceType.Name == spec.ResourceTypeMcpOperation {
			return PublishedModel{}, fmt.Errorf("%w: reserved resource type", accessdomain.ErrModelInvalid)
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return PublishedModel{}, err
	}
	candidate := &accessdomain.AuthorizationModel{
		ID: id, TenantID: tenantID, ResourceTypes: resourceTypes, CreatedAt: now,
	}
	if err := candidate.Validate(); err != nil {
		return PublishedModel{}, err
	}
	stored, version, err := d.Models.Publish(ctx, candidate, expectedVersion)
	if err != nil {
		return PublishedModel{}, err
	}
	d.emit(&accessdomain.AuthorizationModelPublished{
		At: now, TenantID: tenantID, ModelID: stored.ID, Version: stored.Version,
		ResourceTypeCount: len(stored.ResourceTypes),
	})
	return PublishedModel{Model: stored, Consistency: accessdomain.EncodeConsistencyToken(tenantID, version)}, nil
}

// GetAuthorizationModel は最新バージョン、または version を指定したバージョンを返す。
// バージョンが存在しない場合は ErrModelNotFound を返す。
func GetAuthorizationModel(ctx context.Context, d Deps, tenantID string, version int) (PublishedModel, error) {
	model, err := loadModel(ctx, d, tenantID, version)
	if err != nil {
		return PublishedModel{}, err
	}
	storeVersion, err := d.Tuples.Version(ctx, tenantID)
	if err != nil {
		return PublishedModel{}, err
	}
	return PublishedModel{Model: model, Consistency: accessdomain.EncodeConsistencyToken(tenantID, storeVersion)}, nil
}

func loadModel(ctx context.Context, d Deps, tenantID string, version int) (*accessdomain.AuthorizationModel, error) {
	var (
		model *accessdomain.AuthorizationModel
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
		return nil, fmt.Errorf("%w for tenant", accessdomain.ErrModelNotFound)
	}
	return model, nil
}

func modelForResource(ctx context.Context, d Deps, tenantID, resourceType string) (*accessdomain.AuthorizationModel, error) {
	if resourceType == spec.ResourceTypeMcpOperation {
		return accessdomain.McpOperationModel(tenantID), nil
	}
	return loadModel(ctx, d, tenantID, 0)
}
