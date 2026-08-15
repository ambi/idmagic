package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
)

// TupleWriteOutcome は適用件数と適用後の整合トークン。
type TupleWriteOutcome struct {
	WrittenCount int
	DeletedCount int
	Consistency  string
}

// WriteRelationTuples は差分を検証してから 1 トランザクションで適用する。
// 1 件でもモデルに適合しなければ何も適用しない。部分適用は、書いたつもりの
// 権限が半分だけ有効という最も分かりにくい状態を作るからである。
func WriteRelationTuples(
	ctx context.Context,
	d Deps,
	tenantID string,
	write ports.TupleWrite,
	now time.Time,
) (TupleWriteOutcome, error) {
	model, err := loadModel(ctx, d, tenantID, 0)
	if err != nil {
		return TupleWriteOutcome{}, err
	}
	seen := map[string]struct{}{}
	for _, t := range write.Writes {
		if err := model.ValidateTuple(t); err != nil {
			return TupleWriteOutcome{}, err
		}
		seen[t.Key()] = struct{}{}
	}
	for _, t := range write.Deletes {
		// 削除はモデルから外れた古いタプルにも効く必要があるので、書式だけを見る。
		if err := t.Validate(); err != nil {
			return TupleWriteOutcome{}, err
		}
		if _, conflict := seen[t.Key()]; conflict {
			return TupleWriteOutcome{}, fmt.Errorf("%w: %s appears in both writes and deletes", domain.ErrTupleInvalid, t)
		}
	}
	for _, object := range write.DeleteObjects {
		if object.Type == "" || object.ID == "" {
			return TupleWriteOutcome{}, fmt.Errorf("%w: delete_objects entries require a type and an id", domain.ErrTupleInvalid)
		}
	}

	result, err := d.Tuples.Write(ctx, tenantID, write)
	if err != nil {
		return TupleWriteOutcome{}, err
	}
	consistency := domain.EncodeConsistencyToken(tenantID, result.Version)
	if result.WrittenCount > 0 {
		d.emit(&domain.RelationTupleWritten{
			At: now, TenantID: tenantID, WrittenCount: result.WrittenCount, Consistency: consistency,
		})
	}
	if result.DeletedCount > 0 {
		d.emit(&domain.RelationTupleDeleted{
			At: now, TenantID: tenantID, DeletedCount: result.DeletedCount, Consistency: consistency,
		})
	}
	return TupleWriteOutcome{
		WrittenCount: result.WrittenCount, DeletedCount: result.DeletedCount, Consistency: consistency,
	}, nil
}

// ListedTuples は絞り込んだタプルと、その時点の整合トークン。
type ListedTuples struct {
	Tuples      []domain.RelationTuple
	Consistency string
}

// ListRelationTuples は所属テナントのタプルを絞り込んで返す。
func ListRelationTuples(
	ctx context.Context,
	d Deps,
	tenantID string,
	filter ports.RelationTupleFilter,
	limit int,
) (ListedTuples, error) {
	tuples, err := d.Tuples.List(ctx, tenantID, filter, limit)
	if err != nil {
		return ListedTuples{}, err
	}
	version, err := d.Tuples.Version(ctx, tenantID)
	if err != nil {
		return ListedTuples{}, err
	}
	return ListedTuples{Tuples: tuples, Consistency: domain.EncodeConsistencyToken(tenantID, version)}, nil
}
