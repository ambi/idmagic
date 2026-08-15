package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authorization/db_memory"
	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
	"github.com/ambi/idmagic/backend/authorization/usecases"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const tenantID = "tenant-a"

var now = time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)

// statusStub は代行チェーン上のプリンシパルの有効性を固定で返す。
type statusStub struct {
	inactive map[string]bool
	err      error
}

func (s statusStub) IsPrincipalActive(_ context.Context, _, principalType, principalID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return !s.inactive[principalType+":"+principalID], nil
}

// failingTuples は ListSubjects だけを失敗させ、ストア障害を模す。
type failingTuples struct {
	ports.RelationTupleRepository
}

var errStoreDown = errors.New("tuple store is down")

func (f failingTuples) ListSubjects(context.Context, string, domain.ObjectRef, string) ([]domain.SubjectRef, error) {
	return nil, errStoreDown
}

func referenceTypes() []domain.ResourceTypeDefinition {
	return []domain.ResourceTypeDefinition{
		{Name: "user"},
		{Name: "agent"},
		{Name: "group", Relations: []domain.RelationDefinition{
			{Name: "member", Rewrites: []domain.RelationRewrite{
				{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user", "agent"}},
			}},
		}},
		{Name: "document", Relations: []domain.RelationDefinition{
			{Name: "viewer", Rewrites: []domain.RelationRewrite{
				{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user", "agent", "group#member"}},
			}},
		}},
	}
}

type harness struct {
	deps   usecases.Deps
	events []spec.DomainEvent
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	store := db_memory.NewStore()
	h := &harness{}
	h.deps = usecases.Deps{
		Tuples:     db_memory.NewRelationTupleRepository(store),
		Models:     db_memory.NewAuthorizationModelRepository(store),
		Principals: statusStub{},
		Authorizer: authorizationLocal.Local{},
		Emit:       func(e spec.DomainEvent) { h.events = append(h.events, e) },
	}
	if _, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, referenceTypes(), now); err != nil {
		t.Fatalf("PutAuthorizationModel: %v", err)
	}
	return h
}

func (h *harness) write(t *testing.T, tuples ...domain.RelationTuple) string {
	t.Helper()
	outcome, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{Writes: tuples}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples: %v", err)
	}
	return outcome.Consistency
}

func viewerTuple(subjectType, subjectID string) domain.RelationTuple {
	return domain.RelationTuple{
		Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
		Subject: domain.SubjectRef{Type: subjectType, ID: subjectID},
	}
}

func baseInput() usecases.CheckAccessInput {
	return usecases.CheckAccessInput{
		TenantID: tenantID,
		Resource: domain.ObjectRef{Type: "document", ID: "d1"},
		Relation: "viewer",
		Subject:  domain.SubjectRef{Type: "user", ID: "alice"},
	}
}

// REQ-AUTHORIZATION-004: 主体と全 actor の双方が関係を持つときだけ許可する。
func TestCheckAccessRequiresSubjectAndActorChain(t *testing.T) {
	t.Run("the user alone is permitted without a delegation chain", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		result, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if !result.Permitted {
			t.Fatalf("expected permit, got reasons %v", result.Reasons)
		}
	})

	t.Run("an agent without its own relation cannot exceed the user", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		input := baseInput()
		input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
		result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if result.Permitted {
			t.Fatal("an agent that holds no relation must not be permitted")
		}
		if !slices.Contains(result.Reasons, "relationship_permits_actor_chain") {
			t.Fatalf("Reasons = %v, want relationship_permits_actor_chain", result.Reasons)
		}
	})

	t.Run("both the user and the agent holding the relation is permitted", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"), viewerTuple("agent", "researcher"))
		input := baseInput()
		input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
		result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if !result.Permitted {
			t.Fatalf("expected permit, got reasons %v", result.Reasons)
		}
	})

	t.Run("the agent alone cannot act without the delegating user", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("agent", "researcher"))
		input := baseInput()
		input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
		result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if result.Permitted {
			t.Fatal("the delegating user must hold the relation as well")
		}
		if !slices.Contains(result.Reasons, "relationship_permits_subject") {
			t.Fatalf("Reasons = %v, want relationship_permits_subject", result.Reasons)
		}
	})

	t.Run("a disabled or unresolvable actor is never permitted", func(t *testing.T) {
		for name, principals := range map[string]statusStub{
			"disabled":     {inactive: map[string]bool{"agent:researcher": true}},
			"unresolvable": {err: errors.New("agent lookup failed")},
			"no resolver":  {},
		} {
			t.Run(name, func(t *testing.T) {
				h := newHarness(t)
				h.write(t, viewerTuple("user", "alice"), viewerTuple("agent", "researcher"))
				if name == "no resolver" {
					h.deps.Principals = nil
				} else {
					h.deps.Principals = principals
				}
				input := baseInput()
				input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
				result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
				if err != nil {
					t.Fatalf("CheckAccess: %v", err)
				}
				if result.Permitted {
					t.Fatal("an actor whose status is not active must not be permitted")
				}
				if !slices.Contains(result.Reasons, "actor_chain_principals_active") {
					t.Fatalf("Reasons = %v, want actor_chain_principals_active", result.Reasons)
				}
			})
		}
	})

	t.Run("a relation outside the granted scopes is not permitted", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		input := baseInput()
		input.RequiredScopes = []string{"documents:read"}
		input.GrantedScopes = []string{"calendar:read"}
		result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if result.Permitted {
			t.Fatal("a relation the token does not carry a scope for must not be permitted")
		}
		if !slices.Contains(result.Reasons, "scope_subset_of_client_scope") {
			t.Fatalf("Reasons = %v, want scope_subset_of_client_scope", result.Reasons)
		}
	})
}

// REQ-AUTHORIZATION-005: 判定不能はいずれも許可へ退避しない。
func TestCheckAccessFailsClosed(t *testing.T) {
	t.Run("an unpublished model is an error, not a permit", func(t *testing.T) {
		store := db_memory.NewStore()
		deps := usecases.Deps{
			Tuples:     db_memory.NewRelationTupleRepository(store),
			Models:     db_memory.NewAuthorizationModelRepository(store),
			Authorizer: authorizationLocal.Local{},
		}
		result, err := usecases.CheckAccess(context.Background(), deps, baseInput(), now)
		if !errors.Is(err, domain.ErrModelNotFound) {
			t.Fatalf("err = %v, want %v", err, domain.ErrModelNotFound)
		}
		if result.Permitted {
			t.Fatal("a missing model must never permit")
		}
	})

	t.Run("a store failure is an error, not a permit", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		h.deps.Tuples = failingTuples{RelationTupleRepository: h.deps.Tuples}
		result, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now)
		if !errors.Is(err, errStoreDown) {
			t.Fatalf("err = %v, want %v", err, errStoreDown)
		}
		if result.Permitted {
			t.Fatal("a store failure must never permit")
		}
	})

	t.Run("a missing evaluator is an error, not a permit", func(t *testing.T) {
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		h.deps.Authorizer = nil
		if _, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now); !errors.Is(err, usecases.ErrAuthorizerUnavailable) {
			t.Fatalf("err = %v, want %v", err, usecases.ErrAuthorizerUnavailable)
		}
	})

	// 関係の事実を組み立てないまま評価器へ届いた要求は規則が落とす。
	t.Run("the rule table denies a request that carries no relationship facts", func(t *testing.T) {
		response := spec.Evaluate(spec.AuthZRequest{
			Subject:  spec.AuthZSubject{Type: "User", ID: "alice", Properties: spec.AuthZSubjectProps{TenantID: tenantID}},
			Action:   spec.ActionResourceAccess,
			Resource: spec.AuthZResource{Type: "document", ID: "d1", Properties: spec.AuthZResourceProps{TenantID: tenantID}},
		})
		if response.Permit {
			t.Fatal("resource:access without relationship facts must not permit")
		}
		if !slices.Contains(response.Reasons, "relationship_facts_present") {
			t.Fatalf("Reasons = %v, want relationship_facts_present", response.Reasons)
		}
	})
}

// REQ-AUTHORIZATION-006: 整合トークンは自テナントのものだけを受け付ける。
func TestCheckAccessRejectsUnsatisfiedConsistency(t *testing.T) {
	h := newHarness(t)
	consistency := h.write(t, viewerTuple("user", "alice"))

	input := baseInput()
	input.MinimumConsistency = consistency
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); err != nil {
		t.Fatalf("the token the write returned must be satisfiable: %v", err)
	}

	input.MinimumConsistency = domain.EncodeConsistencyToken(tenantID, 9999)
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); !errors.Is(err, domain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a token ahead of the store must be rejected, got %v", err)
	}

	input.MinimumConsistency = domain.EncodeConsistencyToken("tenant-b", 1)
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); !errors.Is(err, domain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a token from another tenant must be rejected, got %v", err)
	}
}

// REQ-AUTHORIZATION-009: 判定の監査は非個人識別情報の要約だけを残す。
func TestCheckAccessAuditKeepsNoIdentifiers(t *testing.T) {
	h := newHarness(t)
	h.write(t, viewerTuple("user", "alice"))
	input := baseInput()
	input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	var evaluated *domain.FgaCheckEvaluated
	for _, event := range h.events {
		if typed, ok := event.(*domain.FgaCheckEvaluated); ok {
			evaluated = typed
		}
	}
	if evaluated == nil {
		t.Fatal("CheckAccess must emit FgaCheckEvaluated")
	}
	if evaluated.ResourceIDDigest == "d1" || len(evaluated.ResourceIDDigest) != 16 {
		t.Fatalf("ResourceIDDigest = %q, want a 16-character digest", evaluated.ResourceIDDigest)
	}
	if evaluated.ActorChainDepth != 1 || evaluated.ModelVersion != 1 {
		t.Fatalf("unexpected audit summary: %+v", evaluated)
	}
	for _, step := range evaluated.RelationPath {
		if step != "document#viewer" {
			t.Fatalf("relation path step %q leaks an identifier", step)
		}
	}
	// テナントが違えば同じリソースでもダイジェストは一致しない。
	if usecases.ResourceIDDigest("tenant-b", input.Resource) == evaluated.ResourceIDDigest {
		t.Fatal("the digest must not correlate across tenants")
	}
}

// REQ-AUTHORIZATION-007: 列挙は許可されたものだけを返し、打ち切りを隠さない。
func TestListAccessibleResourcesIsBoundedAndFiltered(t *testing.T) {
	h := newHarness(t)
	h.write(t,
		viewerTuple("user", "alice"),
		domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d2"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "user", ID: "bob"},
		},
		domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d3"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "group", ID: "eng", Relation: "member"},
		},
		domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "group", ID: "eng"}, Relation: "member",
			Subject: domain.SubjectRef{Type: "user", ID: "alice"},
		},
	)
	input := usecases.ListAccessibleResourcesInput{
		TenantID: tenantID, ResourceType: "document", Relation: "viewer",
		Subject: domain.SubjectRef{Type: "user", ID: "alice"},
	}
	listed, err := usecases.ListAccessibleResources(context.Background(), h.deps, input, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if !slices.Equal(listed.ResourceIDs, []string{"d1", "d3"}) {
		t.Fatalf("ResourceIDs = %v, want [d1 d3]", listed.ResourceIDs)
	}
	if listed.Truncated {
		t.Fatal("three candidates must not truncate")
	}

	// REQ-AUTHORIZATION-007 / REQ-AUTHORIZATION-009: 走査は 1 件ごとの判定を監査へ
	// 展開せず、まとめた 1 件だけを残す。
	var perCheck, enumerated int
	for _, event := range h.events {
		switch typed := event.(type) {
		case *domain.FgaCheckEvaluated:
			perCheck++
		case *domain.FgaResourcesEnumerated:
			enumerated++
			if typed.CandidateCount != 3 || typed.PermittedCount != 2 || typed.Truncated {
				t.Fatalf("unexpected enumeration summary: %+v", typed)
			}
		}
	}
	if perCheck != 0 || enumerated != 1 {
		t.Fatalf("enumeration emitted %d per-check and %d summary events, want 0 and 1", perCheck, enumerated)
	}

	h.deps.MaxEnumeratedResources = 2
	listed, err = usecases.ListAccessibleResources(context.Background(), h.deps, input, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if !listed.Truncated {
		t.Fatal("hitting the enumeration limit must be reported, not hidden")
	}
}

// REQ-AUTHORIZATION-002: モデルに適合しない差分は 1 件も適用しない。
func TestWriteRelationTuplesRejectsTheWholeDiff(t *testing.T) {
	h := newHarness(t)
	_, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes: []domain.RelationTuple{viewerTuple("user", "alice"), viewerTuple("robot", "r2")},
	}, now)
	if !errors.Is(err, domain.ErrTupleInvalid) {
		t.Fatalf("err = %v, want %v", err, domain.ErrTupleInvalid)
	}
	listed, err := usecases.ListRelationTuples(context.Background(), h.deps, tenantID, ports.RelationTupleFilter{}, 0)
	if err != nil {
		t.Fatalf("ListRelationTuples: %v", err)
	}
	if len(listed.Tuples) != 0 {
		t.Fatalf("a rejected diff must not be partially applied, found %v", listed.Tuples)
	}

	_, err = usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes:  []domain.RelationTuple{viewerTuple("user", "alice")},
		Deletes: []domain.RelationTuple{viewerTuple("user", "alice")},
	}, now)
	if !errors.Is(err, domain.ErrTupleInvalid) {
		t.Fatalf("a tuple in both writes and deletes must be rejected, got %v", err)
	}
}

// REQ-AUTHORIZATION-008: オブジェクトの削除は、それに依存していた間接的な関係も止める。
func TestDeletingAnObjectStopsTheRelationsItCarried(t *testing.T) {
	h := newHarness(t)
	h.write(t,
		domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "group", ID: "eng", Relation: "member"},
		},
		domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "group", ID: "eng"}, Relation: "member",
			Subject: domain.SubjectRef{Type: "user", ID: "alice"},
		},
	)
	before, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now)
	if err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	if !before.Permitted {
		t.Fatalf("the group grant must permit first, got reasons %v", before.Reasons)
	}

	outcome, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		DeleteObjects: []domain.ObjectRef{{Type: "group", ID: "eng"}},
	}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples: %v", err)
	}
	if outcome.DeletedCount != 2 {
		t.Fatalf("DeletedCount = %d, want 2 (both sides of the group)", outcome.DeletedCount)
	}

	after, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now)
	if err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	if after.Permitted {
		t.Fatal("the grant that depended on the deleted object must stop holding")
	}
	if after.Consistency == before.Consistency {
		t.Fatal("the consistency token must advance after a delete")
	}
}

// REQ-AUTHORIZATION-001: 整合しないモデルは版を作らない。
func TestPutAuthorizationModelRejectsAnInconsistentModel(t *testing.T) {
	h := newHarness(t)
	_, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, []domain.ResourceTypeDefinition{
		{Name: "document", Relations: []domain.RelationDefinition{
			{Name: "viewer", Rewrites: []domain.RelationRewrite{
				{Kind: domain.RewriteComputedUserset, ComputedRelation: "editor"},
			}},
		}},
	}, now)
	if !errors.Is(err, domain.ErrModelInvalid) {
		t.Fatalf("err = %v, want %v", err, domain.ErrModelInvalid)
	}
	published, err := usecases.GetAuthorizationModel(context.Background(), h.deps, tenantID, 0)
	if err != nil {
		t.Fatalf("GetAuthorizationModel: %v", err)
	}
	if published.Model.Version != 1 {
		t.Fatalf("a rejected model must not advance the version, got %d", published.Model.Version)
	}
}
