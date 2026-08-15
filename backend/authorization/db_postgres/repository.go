// Package db_postgres は Authorization Context の PostgreSQL アダプター。
// 差分の適用と書き込み版の前進は 1 トランザクションで行い、部分適用を残さない。
package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// defaultListLimit は limit 未指定の一覧に適用する上限。無制限の走査を作らない。
const defaultListLimit = 1000

// RelationTupleRepository は関係タプルを PostgreSQL に永続化する。クエリは sqlc 生成。
type RelationTupleRepository struct{ Pool sharedpg.DB }

func (r *RelationTupleRepository) ListSubjects(ctx context.Context, tenantID string, resource domain.ObjectRef, relation string) ([]domain.SubjectRef, error) {
	rows, err := New(r.Pool).ListRelationTupleSubjects(ctx, ListRelationTupleSubjectsParams{
		TenantID: tenantID, ResourceType: resource.Type, ResourceID: resource.ID, Relation: relation,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SubjectRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.SubjectRef{Type: row.SubjectType, ID: row.SubjectID, Relation: row.SubjectRelation})
	}
	return out, nil
}

func (r *RelationTupleRepository) List(ctx context.Context, tenantID string, filter ports.RelationTupleFilter, limit int) ([]domain.RelationTuple, error) {
	rows, err := New(r.Pool).ListRelationTuples(ctx, ListRelationTuplesParams{
		TenantID: tenantID,
		Column2:  filter.ResourceType,
		Column3:  filter.ResourceID,
		Column4:  filter.Relation,
		Column5:  filter.SubjectType,
		Column6:  filter.SubjectID,
		Limit:    boundedLimit(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.RelationTuple, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RelationTuple{
			Resource: domain.ObjectRef{Type: row.ResourceType, ID: row.ResourceID},
			Relation: row.Relation,
			Subject:  domain.SubjectRef{Type: row.SubjectType, ID: row.SubjectID, Relation: row.SubjectRelation},
		})
	}
	return out, nil
}

func (r *RelationTupleRepository) ListResourceIDs(ctx context.Context, tenantID, resourceType string, limit int) ([]string, error) {
	ids, err := New(r.Pool).ListRelationTupleResourceIDs(ctx, ListRelationTupleResourceIDsParams{
		TenantID: tenantID, ResourceType: resourceType, Limit: boundedLimit(limit),
	})
	if err != nil {
		return nil, err
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// boundedLimit は呼び出し側の limit を上限つきの int32 に落とす。上限を超える値と
// 非正の値はどちらも既定に丸めるので、無制限の走査になる入力は存在しない。
func boundedLimit(limit int) int32 {
	if limit <= 0 || limit > defaultListLimit {
		return defaultListLimit
	}
	return int32(limit)
}

func (r *RelationTupleRepository) Write(ctx context.Context, tenantID string, write ports.TupleWrite) (ports.TupleWriteResult, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return ports.TupleWriteResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(r.Pool).WithTx(tx)

	result := ports.TupleWriteResult{}
	for _, t := range write.Deletes {
		affected, err := queries.DeleteRelationTuple(ctx, DeleteRelationTupleParams{
			TenantID: tenantID, ResourceType: t.Resource.Type, ResourceID: t.Resource.ID, Relation: t.Relation,
			SubjectType: t.Subject.Type, SubjectID: t.Subject.ID, SubjectRelation: t.Subject.Relation,
		})
		if err != nil {
			return ports.TupleWriteResult{}, err
		}
		result.DeletedCount += int(affected)
	}
	for _, object := range write.DeleteObjects {
		affected, err := queries.DeleteRelationTuplesForObject(ctx, DeleteRelationTuplesForObjectParams{
			TenantID: tenantID, ResourceType: object.Type, ResourceID: object.ID,
		})
		if err != nil {
			return ports.TupleWriteResult{}, err
		}
		result.DeletedCount += int(affected)
	}
	for _, t := range write.Writes {
		affected, err := queries.InsertRelationTuple(ctx, InsertRelationTupleParams{
			TenantID: tenantID, ResourceType: t.Resource.Type, ResourceID: t.Resource.ID, Relation: t.Relation,
			SubjectType: t.Subject.Type, SubjectID: t.Subject.ID, SubjectRelation: t.Subject.Relation,
		})
		if err != nil {
			return ports.TupleWriteResult{}, err
		}
		result.WrittenCount += int(affected)
	}
	version, err := queries.BumpAuthorizationWriteVersion(ctx, tenantID)
	if err != nil {
		return ports.TupleWriteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.TupleWriteResult{}, err
	}
	result.Version = version
	return result, nil
}

func (r *RelationTupleRepository) Version(ctx context.Context, tenantID string) (int64, error) {
	version, err := New(r.Pool).GetAuthorizationWriteVersion(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// AuthorizationModelRepository は認可モデルの版を PostgreSQL に永続化する。
// 版の採番は INSERT の中で行い、同時登録が同じ版を採らないようにする。
type AuthorizationModelRepository struct{ Pool sharedpg.DB }

func (r *AuthorizationModelRepository) Publish(ctx context.Context, model *domain.AuthorizationModel) (*domain.AuthorizationModel, int64, error) {
	definition, err := json.Marshal(model.ResourceTypes)
	if err != nil {
		return nil, 0, err
	}
	createdAt := model.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(r.Pool).WithTx(tx)

	row, err := queries.InsertAuthorizationModel(ctx, InsertAuthorizationModelParams{
		ID: model.ID, TenantID: model.TenantID, Definition: definition, CreatedAt: createdAt,
	})
	if err != nil {
		return nil, 0, err
	}
	version, err := queries.BumpAuthorizationWriteVersion(ctx, model.TenantID)
	if err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	stored, err := modelFromRow(row)
	if err != nil {
		return nil, 0, err
	}
	return stored, version, nil
}

func (r *AuthorizationModelRepository) Latest(ctx context.Context, tenantID string) (*domain.AuthorizationModel, error) {
	row, err := New(r.Pool).GetLatestAuthorizationModel(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return modelFromRow(row)
}

func (r *AuthorizationModelRepository) FindByVersion(ctx context.Context, tenantID string, version int) (*domain.AuthorizationModel, error) {
	// 版は列の型 (INTEGER) を超えられないので、範囲外の要求は存在しない版として扱う。
	if version <= 0 || version > math.MaxInt32 {
		return nil, nil
	}
	row, err := New(r.Pool).GetAuthorizationModelByVersion(ctx, GetAuthorizationModelByVersionParams{
		TenantID: tenantID, Version: int32(version),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return modelFromRow(row)
}

func modelFromRow(row *AuthorizationModel) (*domain.AuthorizationModel, error) {
	var resourceTypes []domain.ResourceTypeDefinition
	if err := json.Unmarshal(row.Definition, &resourceTypes); err != nil {
		return nil, err
	}
	return &domain.AuthorizationModel{
		ID: row.ID, TenantID: row.TenantID, Version: int(row.Version),
		ResourceTypes: resourceTypes, CreatedAt: row.CreatedAt,
	}, nil
}
