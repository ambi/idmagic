package domain_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/authorization/domain"
)

// stubReader は (resource, relation) から直接主体を引く最小の TupleReader。
// failOn に一致する関係の読み出しでストア障害を模す。
type stubReader struct {
	tuples map[string][]domain.SubjectRef
	failOn string
	calls  int
}

var errStoreUnavailable = errors.New("tuple store unavailable")

func (r *stubReader) ListSubjects(_ context.Context, _ string, resource domain.ObjectRef, relation string) ([]domain.SubjectRef, error) {
	r.calls++
	key := resource.String() + "#" + relation
	if r.failOn != "" && r.failOn == key {
		return nil, errStoreUnavailable
	}
	return r.tuples[key], nil
}

func newReader(tuples map[string][]domain.SubjectRef) *stubReader {
	return &stubReader{tuples: tuples}
}

// documentModel は direct / computed_userset / tuple_to_userset をすべて含む
// 参照モデル。folder#viewer を document#viewer が親経由で継承する。
func documentModel() *domain.AuthorizationModel {
	return &domain.AuthorizationModel{
		ID: "model-1", TenantID: "tenant-a", Version: 1,
		ResourceTypes: []domain.ResourceTypeDefinition{
			{Name: "user", Relations: nil},
			{Name: "group", Relations: []domain.RelationDefinition{
				{Name: "member", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user", "group#member"}},
				}},
			}},
			{Name: "folder", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user", "group#member"}},
				}},
			}},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "parent", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"folder"}},
				}},
				{Name: "editor", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user"}},
				}},
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user", "user:*", "group#member"}},
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "editor"},
					{Kind: domain.RewriteTupleToUserset, TuplesetRelation: "parent", ComputedRelation: "viewer"},
				}},
			}},
		},
	}
}

func user(id string) domain.SubjectRef { return domain.SubjectRef{Type: "user", ID: id} }

// REQ-AUTHORIZATION-003: 直接・グループ・継承・親子のいずれの経路でも関係に到達する。
func TestCheckTraversesGroupAndParent(t *testing.T) {
	model := documentModel()
	reader := newReader(map[string][]domain.SubjectRef{
		"document:d1#viewer":    {user("alice")},
		"document:d2#editor":    {user("bob")},
		"document:d3#parent":    {{Type: "folder", ID: "f1"}},
		"folder:f1#viewer":      {{Type: "group", ID: "eng", Relation: "member"}},
		"group:eng#member":      {{Type: "group", ID: "platform", Relation: "member"}},
		"group:platform#member": {user("carol")},
		"document:d4#viewer":    {{Type: "user", ID: domain.Wildcard}},
	})

	cases := []struct {
		name     string
		resource domain.ObjectRef
		subject  domain.SubjectRef
		want     bool
		wantPath []string
	}{
		{"direct tuple", domain.ObjectRef{Type: "document", ID: "d1"}, user("alice"), true, []string{"document#viewer"}},
		{"computed userset", domain.ObjectRef{Type: "document", ID: "d2"}, user("bob"), true, []string{"document#viewer", "document#editor"}},
		{
			"nested group through parent folder",
			domain.ObjectRef{Type: "document", ID: "d3"},
			user("carol"), true,
			[]string{"document#viewer", "folder#viewer", "group#member", "group#member"},
		},
		{"wildcard", domain.ObjectRef{Type: "document", ID: "d4"}, user("dave"), true, []string{"document#viewer"}},
		{"unrelated subject", domain.ObjectRef{Type: "document", ID: "d1"}, user("mallory"), false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := model.Check(context.Background(), reader, "tenant-a", tc.resource, "viewer", tc.subject, domain.CheckOptions{})
			if err != nil {
				t.Fatalf("Check returned error: %v", err)
			}
			if decision.Permitted != tc.want {
				t.Fatalf("Permitted = %v, want %v (reasons %v)", decision.Permitted, tc.want, decision.Reasons)
			}
			if tc.want && !slices.Equal(decision.Path, tc.wantPath) {
				t.Fatalf("Path = %v, want %v", decision.Path, tc.wantPath)
			}
			if !tc.want && len(decision.Reasons) == 0 {
				t.Fatal("a denial must carry at least one reason")
			}
		})
	}
}

// REQ-AUTHORIZATION-003: 経路にオブジェクト識別子と主体識別子を含めない。
func TestCheckPathOmitsIdentifiers(t *testing.T) {
	model := documentModel()
	reader := newReader(map[string][]domain.SubjectRef{
		"document:secret-quarterly-plan#parent": {{Type: "folder", ID: "board-only"}},
		"folder:board-only#viewer":              {user("alice")},
	})
	decision, err := model.Check(context.Background(), reader, "tenant-a",
		domain.ObjectRef{Type: "document", ID: "secret-quarterly-plan"}, "viewer", user("alice"), domain.CheckOptions{})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !decision.Permitted {
		t.Fatalf("expected permit, got reasons %v", decision.Reasons)
	}
	for _, step := range decision.Path {
		if step != "document#viewer" && step != "folder#viewer" {
			t.Fatalf("path step %q leaks an identifier", step)
		}
	}
}

// REQ-AUTHORIZATION-005: 循環・深さ超過・未知の関係はいずれも許可しない。
func TestCheckDeniesOnCycleAndDepth(t *testing.T) {
	// tuple_to_userset の循環はタプル側でしか作れないので、親をたがいに指すデータで作る。
	model := documentModel()
	reader := newReader(map[string][]domain.SubjectRef{
		"document:a#parent": {{Type: "folder", ID: "f1"}},
		"folder:f1#viewer":  {{Type: "group", ID: "loop", Relation: "member"}},
		"group:loop#member": {{Type: "group", ID: "loop", Relation: "member"}},
	})
	decision, err := model.Check(context.Background(), reader, "tenant-a",
		domain.ObjectRef{Type: "document", ID: "a"}, "viewer", user("alice"), domain.CheckOptions{})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Permitted {
		t.Fatal("a cyclic relation graph must not permit")
	}
	if !slices.Contains(decision.Reasons, domain.ReasonCycleDetected) {
		t.Fatalf("Reasons = %v, want %s", decision.Reasons, domain.ReasonCycleDetected)
	}

	// 深いグループの連なりを上限より長く作る。
	deep := map[string][]domain.SubjectRef{"document:a#viewer": {{Type: "group", ID: "g0", Relation: "member"}}}
	for i := range 10 {
		from := "group:g" + string(rune('0'+i)) + "#member"
		deep[from] = []domain.SubjectRef{{Type: "group", ID: "g" + string(rune('1'+i)), Relation: "member"}}
	}
	deep["group:g10#member"] = []domain.SubjectRef{user("alice")}
	decision, err = model.Check(context.Background(), newReader(deep), "tenant-a",
		domain.ObjectRef{Type: "document", ID: "a"}, "viewer", user("alice"), domain.CheckOptions{MaxDepth: 3})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Permitted {
		t.Fatal("exceeding the depth limit must not permit")
	}
	if !slices.Contains(decision.Reasons, domain.ReasonDepthExceeded) {
		t.Fatalf("Reasons = %v, want %s", decision.Reasons, domain.ReasonDepthExceeded)
	}

	// モデルが宣言していない関係。
	decision, err = model.Check(context.Background(), newReader(nil), "tenant-a",
		domain.ObjectRef{Type: "document", ID: "a"}, "shredder", user("alice"), domain.CheckOptions{})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Permitted || !slices.Contains(decision.Reasons, domain.ReasonUnknownRelation) {
		t.Fatalf("unknown relation must deny with %s, got %+v", domain.ReasonUnknownRelation, decision)
	}
}

// REQ-AUTHORIZATION-005: ストア障害は許可へ退避せず error として上へ返す。
func TestCheckPropagatesStoreFailure(t *testing.T) {
	model := documentModel()
	reader := newReader(map[string][]domain.SubjectRef{"document:a#viewer": {user("alice")}})
	reader.failOn = "document:a#viewer"
	decision, err := model.Check(context.Background(), reader, "tenant-a",
		domain.ObjectRef{Type: "document", ID: "a"}, "viewer", user("alice"), domain.CheckOptions{})
	if !errors.Is(err, errStoreUnavailable) {
		t.Fatalf("err = %v, want %v", err, errStoreUnavailable)
	}
	if decision.Permitted {
		t.Fatal("a store failure must never yield a permit")
	}
}

// REQ-AUTHORIZATION-005: モデルが受け入れなくなった形の既存タプルは数えない。
func TestCheckIgnoresSubjectFormsTheModelNoLongerAccepts(t *testing.T) {
	model := documentModel()
	reader := newReader(map[string][]domain.SubjectRef{
		// editor は user しか受け付けないので、group の subject set は無視される。
		"document:a#editor": {{Type: "group", ID: "eng", Relation: "member"}},
		"group:eng#member":  {user("alice")},
	})
	decision, err := model.Check(context.Background(), reader, "tenant-a",
		domain.ObjectRef{Type: "document", ID: "a"}, "editor", user("alice"), domain.CheckOptions{})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Permitted {
		t.Fatal("a subject form the model does not declare must not permit")
	}
	if !slices.Contains(decision.Reasons, domain.ReasonSubjectFormDenied) {
		t.Fatalf("Reasons = %v, want %s", decision.Reasons, domain.ReasonSubjectFormDenied)
	}
}
