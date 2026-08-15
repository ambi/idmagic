// Package testing_contract は関係タプルと認可モデルの Repository が満たすべき
// 契約を 1 か所に置く。メモリと PostgreSQL の両アダプターが同じ本体を実行し、
// 「メモリだけ通る」振る舞いの差を残さない (backend/shared/storage/testing_postgres
// と同じ、テスト補助を非テストパッケージに置く構成)。
package testing_contract

import (
	"context"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Fixture は 1 回の契約テストが使う Repository の組。Tenants は契約が使う
// 記号名から実際のテナント識別子への対応で、PostgreSQL 版では実在する UUID を指す。
type Fixture struct {
	Tuples  ports.RelationTupleRepository
	Models  ports.AuthorizationModelRepository
	Tenants map[string]string
}

// Tenant は記号名に対応する実際のテナント識別子を返す。
func (f Fixture) Tenant(name string) string {
	if id, ok := f.Tenants[name]; ok {
		return id
	}
	return name
}

// NewFixture は指定した記号名のテナントが存在する状態の Repository を組み立てる。
// PostgreSQL 版は tenants 行を用意し、メモリ版は記号名をそのまま使う。
type NewFixture func(t *testing.T, tenantNames ...string) Fixture

func newModelID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

func tuple(resourceID, relation, subjectID string) domain.RelationTuple {
	return domain.RelationTuple{
		Resource: domain.ObjectRef{Type: "document", ID: resourceID},
		Relation: relation,
		Subject:  domain.SubjectRef{Type: "user", ID: subjectID},
	}
}

// RunRelationTupleRepositoryContract は関係タプルストアの契約を検証する。
func RunRelationTupleRepositoryContract(t *testing.T, newFixture NewFixture) {
	t.Helper()

	t.Run("write is idempotent and readable by subject", func(t *testing.T) {
		f := newFixture(t, "tenant-a")
		ctx := context.Background()
		tenantA := f.Tenant("tenant-a")
		first, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{Writes: []domain.RelationTuple{
			tuple("d1", "viewer", "alice"),
			tuple("d1", "viewer", "bob"),
		}})
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if first.WrittenCount != 2 {
			t.Fatalf("WrittenCount = %d, want 2", first.WrittenCount)
		}
		// 同じ組の再書き込みは冪等 (件数に数えないが、版は進む)。
		second, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{Writes: []domain.RelationTuple{
			tuple("d1", "viewer", "alice"),
		}})
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if second.WrittenCount != 0 {
			t.Fatalf("re-writing the same tuple must not count as a write, got %d", second.WrittenCount)
		}
		if second.Version <= first.Version {
			t.Fatalf("Version must advance on every write: %d -> %d", first.Version, second.Version)
		}

		subjects, err := f.Tuples.ListSubjects(ctx, tenantA, domain.ObjectRef{Type: "document", ID: "d1"}, "viewer")
		if err != nil {
			t.Fatalf("ListSubjects returned error: %v", err)
		}
		if len(subjects) != 2 {
			t.Fatalf("ListSubjects returned %d subjects, want 2", len(subjects))
		}
	})

	t.Run("delete removes exactly the named tuple", func(t *testing.T) {
		f := newFixture(t, "tenant-a")
		ctx := context.Background()
		tenantA := f.Tenant("tenant-a")
		if _, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{Writes: []domain.RelationTuple{
			tuple("d1", "viewer", "alice"), tuple("d1", "viewer", "bob"),
		}}); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		result, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{Deletes: []domain.RelationTuple{
			tuple("d1", "viewer", "alice"),
			// 存在しない組の削除は 0 件として扱い、失敗させない。
			tuple("d9", "viewer", "zoe"),
		}})
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if result.DeletedCount != 1 {
			t.Fatalf("DeletedCount = %d, want 1", result.DeletedCount)
		}
		subjects, err := f.Tuples.ListSubjects(ctx, tenantA, domain.ObjectRef{Type: "document", ID: "d1"}, "viewer")
		if err != nil {
			t.Fatalf("ListSubjects returned error: %v", err)
		}
		if len(subjects) != 1 || subjects[0].ID != "bob" {
			t.Fatalf("remaining subjects = %v, want only bob", subjects)
		}
	})

	// REQ-AUTHORIZATION-008: オブジェクトの削除は両側のタプルを取り除く。
	t.Run("deleting an object removes tuples on both sides", func(t *testing.T) {
		f := newFixture(t, "tenant-a")
		ctx := context.Background()
		tenantA := f.Tenant("tenant-a")
		asResource := domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "group", ID: "eng"}, Relation: "member",
			Subject: domain.SubjectRef{Type: "user", ID: "alice"},
		}
		asSubject := domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "group", ID: "eng", Relation: "member"},
		}
		unrelated := tuple("d2", "viewer", "bob")
		if _, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{
			Writes: []domain.RelationTuple{asResource, asSubject, unrelated},
		}); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		result, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{
			DeleteObjects: []domain.ObjectRef{{Type: "group", ID: "eng"}},
		})
		if err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		if result.DeletedCount != 2 {
			t.Fatalf("DeletedCount = %d, want 2 (resource side and subject side)", result.DeletedCount)
		}
		remaining, err := f.Tuples.List(ctx, tenantA, ports.RelationTupleFilter{}, 0)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if len(remaining) != 1 || remaining[0].Key() != unrelated.Key() {
			t.Fatalf("remaining tuples = %v, want only the unrelated one", remaining)
		}
	})

	// REQ-AUTHORIZATION-006: テナントをまたいでタプルが見えてはならない。
	t.Run("tuples never cross the tenant boundary", func(t *testing.T) {
		f := newFixture(t, "tenant-a", "tenant-b")
		ctx := context.Background()
		tenantA, tenantB := f.Tenant("tenant-a"), f.Tenant("tenant-b")
		if _, err := f.Tuples.Write(ctx, tenantB, ports.TupleWrite{
			Writes: []domain.RelationTuple{tuple("d1", "viewer", "alice")},
		}); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		subjects, err := f.Tuples.ListSubjects(ctx, tenantA, domain.ObjectRef{Type: "document", ID: "d1"}, "viewer")
		if err != nil {
			t.Fatalf("ListSubjects returned error: %v", err)
		}
		if len(subjects) != 0 {
			t.Fatalf("tenant-a must not see tenant-b tuples, got %v", subjects)
		}
		version, err := f.Tuples.Version(ctx, tenantA)
		if err != nil {
			t.Fatalf("Version returned error: %v", err)
		}
		if version != 0 {
			t.Fatalf("tenant-a write version = %d, want 0", version)
		}
	})

	t.Run("filters and resource enumeration are bounded", func(t *testing.T) {
		f := newFixture(t, "tenant-a")
		ctx := context.Background()
		tenantA := f.Tenant("tenant-a")
		writes := []domain.RelationTuple{
			tuple("d1", "viewer", "alice"), tuple("d1", "editor", "alice"),
			tuple("d2", "viewer", "alice"), tuple("d3", "viewer", "bob"),
		}
		if _, err := f.Tuples.Write(ctx, tenantA, ports.TupleWrite{Writes: writes}); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
		filtered, err := f.Tuples.List(ctx, tenantA, ports.RelationTupleFilter{Relation: "viewer", SubjectID: "alice"}, 0)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if len(filtered) != 2 {
			t.Fatalf("filtered tuples = %d, want 2", len(filtered))
		}
		ids, err := f.Tuples.ListResourceIDs(ctx, tenantA, "document", 0)
		if err != nil {
			t.Fatalf("ListResourceIDs returned error: %v", err)
		}
		if !slices.Equal(ids, []string{"d1", "d2", "d3"}) {
			t.Fatalf("ListResourceIDs = %v, want [d1 d2 d3] with duplicates collapsed", ids)
		}
		limited, err := f.Tuples.ListResourceIDs(ctx, tenantA, "document", 2)
		if err != nil {
			t.Fatalf("ListResourceIDs returned error: %v", err)
		}
		if len(limited) != 2 {
			t.Fatalf("ListResourceIDs honoured no limit: got %d ids", len(limited))
		}
	})
}

// RunAuthorizationModelRepositoryContract は認可モデルの版ストアの契約を検証する。
func RunAuthorizationModelRepositoryContract(t *testing.T, newFixture NewFixture) {
	t.Helper()

	minimalTypes := []domain.ResourceTypeDefinition{
		{Name: "user"},
		{Name: "document", Relations: []domain.RelationDefinition{
			{Name: "viewer", Rewrites: []domain.RelationRewrite{
				{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user"}},
			}},
		}},
	}

	t.Run("versions are append-only and monotonic", func(t *testing.T) {
		f := newFixture(t, "tenant-a")
		ctx := context.Background()
		tenantA := f.Tenant("tenant-a")
		if latest, err := f.Models.Latest(ctx, tenantA); err != nil || latest != nil {
			t.Fatalf("Latest on an empty tenant = (%v, %v), want (nil, nil)", latest, err)
		}
		first, version, err := f.Models.Publish(ctx, &domain.AuthorizationModel{
			ID: newModelID(t), TenantID: tenantA, ResourceTypes: minimalTypes,
		})
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if first.Version != 1 || version == 0 {
			t.Fatalf("Publish = (version %d, consistency %d), want (1, > 0)", first.Version, version)
		}
		second, _, err := f.Models.Publish(ctx, &domain.AuthorizationModel{
			ID: newModelID(t), TenantID: tenantA, ResourceTypes: minimalTypes,
		})
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if second.Version != 2 {
			t.Fatalf("second Version = %d, want 2", second.Version)
		}
		latest, err := f.Models.Latest(ctx, tenantA)
		if err != nil || latest == nil || latest.Version != 2 {
			t.Fatalf("Latest = (%v, %v), want version 2", latest, err)
		}
		restored, err := f.Models.FindByVersion(ctx, tenantA, 1)
		if err != nil || restored == nil {
			t.Fatalf("FindByVersion(1) = (%v, %v), want the first version", restored, err)
		}
		if len(restored.ResourceTypes) != len(minimalTypes) {
			t.Fatalf("stored definition lost resource types: %+v", restored.ResourceTypes)
		}
		if missing, err := f.Models.FindByVersion(ctx, tenantA, 99); err != nil || missing != nil {
			t.Fatalf("FindByVersion(99) = (%v, %v), want (nil, nil)", missing, err)
		}
	})

	// REQ-AUTHORIZATION-006: 版の採番はテナントごとに独立している。
	t.Run("version numbering is per tenant", func(t *testing.T) {
		f := newFixture(t, "tenant-a", "tenant-b")
		ctx := context.Background()
		tenantA, tenantB := f.Tenant("tenant-a"), f.Tenant("tenant-b")
		if _, _, err := f.Models.Publish(ctx, &domain.AuthorizationModel{
			ID: newModelID(t), TenantID: tenantA, ResourceTypes: minimalTypes,
		}); err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		other, _, err := f.Models.Publish(ctx, &domain.AuthorizationModel{
			ID: newModelID(t), TenantID: tenantB, ResourceTypes: minimalTypes,
		})
		if err != nil {
			t.Fatalf("Publish returned error: %v", err)
		}
		if other.Version != 1 {
			t.Fatalf("tenant-b first Version = %d, want 1", other.Version)
		}
		if latest, err := f.Models.Latest(ctx, tenantB); err != nil || latest.ID != other.ID {
			t.Fatalf("tenant-b Latest = (%v, %v), want its own model", latest, err)
		}
	})
}
