package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authorization/db_memory"
	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
	"github.com/ambi/idmagic/backend/authorization/usecases"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	tenantID = "tenant-a"
	// otherTenantID は同じ Store を共有する 2 つ目のテナント。境界の具体例は
	// 「別テナントに同じ識別子のタプルがある」状況でしか観測できないので、
	// 保管庫を分けずに同居させる。
	otherTenantID = "tenant-b"
)

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

// newBareHarness は参照モデルを published していない組み立てを返す。モデルの未登録が
// 観測点になる具体例は、newHarness が先に版を作ってしまうと作れない。
func newBareHarness() *harness {
	store := db_memory.NewStore()
	h := &harness{}
	h.deps = usecases.Deps{
		Tuples:     db_memory.NewRelationTupleRepository(store),
		Models:     db_memory.NewAuthorizationModelRepository(store),
		Principals: statusStub{},
		Authorizer: authorizationLocal.Local{},
		Emit:       func(e spec.DomainEvent) { h.events = append(h.events, e) },
	}
	return h
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := newBareHarness()
	h.publish(t, tenantID)
	return h
}

// publish は指定テナントへ参照モデルを published し、その版を返す。
func (h *harness) publish(t *testing.T, tenant string) usecases.PublishedModel {
	t.Helper()
	published, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenant, referenceTypes(), now)
	if err != nil {
		t.Fatalf("PutAuthorizationModel(%s): %v", tenant, err)
	}
	return published
}

func (h *harness) write(t *testing.T, tuples ...domain.RelationTuple) string {
	t.Helper()
	return h.writeFor(t, tenantID, tuples...)
}

func (h *harness) writeFor(t *testing.T, tenant string, tuples ...domain.RelationTuple) string {
	t.Helper()
	outcome, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenant, ports.TupleWrite{Writes: tuples}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples(%s): %v", tenant, err)
	}
	return outcome.Consistency
}

// countEvents は発行済みイベントのうち、与えた型のものを数える。
func (h *harness) countEvents(eventType string) int {
	count := 0
	for _, event := range h.events {
		if event.EventType() == eventType {
			count++
		}
	}
	return count
}

// listAll は保管庫に残っているタプルを API 越しではなく読み直す。拒否された書き込みが
// 何も残していないことは、応答だけでは確かめられない。
func (h *harness) listAll(t *testing.T) []domain.RelationTuple {
	t.Helper()
	listed, err := usecases.ListRelationTuples(context.Background(), h.deps, tenantID, ports.RelationTupleFilter{}, 0)
	if err != nil {
		t.Fatalf("ListRelationTuples(%s): %v", tenantID, err)
	}
	return listed.Tuples
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

//spec:covers REQ-AUTHORIZATION-004: 主体と全 actor の双方が関係を持つときだけ許可する。
//spec:covers EX-AUTHORIZATION-004-01: 表が許可する唯一の行。関係、プリンシパルの有効性、スコープの包含をすべて立てた入力だけが許可になる。
//spec:covers EX-AUTHORIZATION-004-02: 代行者が関係を持たない行を許可せず、relationship_permits_actor_chain を理由に残す。
//spec:covers EX-AUTHORIZATION-004-03: プリンシパルが有効でない、または状態を解決できない行を許可せず、actor_chain_principals_active を理由に残す。
//spec:covers EX-AUTHORIZATION-004-04: 要求した関係のスコープが提示トークンに含まれない行を許可せず、scope_subset_of_client_scope を理由に残す。
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
		// 表の scope 列は「含まれる」であって「制約しない」ではない。要求スコープを
		// 空のままにすると、スコープを一切見ない実装でもこの行が通ってしまう。
		input.RequiredScopes = []string{"documents:read"}
		input.GrantedScopes = []string{"documents:read", "calendar:read"}
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

//spec:covers REQ-AUTHORIZATION-005: 判定不能はいずれも許可へ退避しない。
//spec:covers EX-AUTHORIZATION-005-01: 答えの出ない 3 つの入口がいずれも許可にならないこと、および答えの出る拒否が、拒否した規則名を結果に残すこと。
//spec:covers EX-AUTHORIZATION-005-04: タプルストアへ到達できない判定が error になり、許可しない。
func TestCheckAccessFailsClosed(t *testing.T) {
	t.Run("a denial names the rule that refused it", func(t *testing.T) {
		// 規則名が残ることは、答えが出た拒否でしか観測できない。error を返す 3 つの
		// 入口では結果がゼロ値になり、「規則名が空」と区別がつかない。
		h := newHarness(t)
		h.write(t, viewerTuple("user", "alice"))
		input := baseInput()
		input.Subject = domain.SubjectRef{Type: "user", ID: "mallory"}
		result, err := usecases.CheckAccess(context.Background(), h.deps, input, now)
		if err != nil {
			t.Fatalf("CheckAccess: %v", err)
		}
		if result.Permitted {
			t.Fatal("mallory holds no relation and must not be permitted")
		}
		if !slices.Contains(result.Reasons, "relationship_permits_subject") {
			t.Fatalf("Reasons = %v, want the refusing rule relationship_permits_subject", result.Reasons)
		}
		if !slices.Contains(result.Reasons, domain.ReasonNoRelationPath) {
			t.Fatalf("Reasons = %v, want %s", result.Reasons, domain.ReasonNoRelationPath)
		}
	})

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

	//spec:covers AUTHZEN-FGA-FAIL-CLOSED: 事実が欠けたまま評価器へ届いた resource:access は、規則表の
	// 側でも許可にならない。行が挙げる 4 つの状況のうち「事実が欠けている」を、事実を供給する
	// 経路ではなく受け取る規則の側から固定する。
	//spec:covers EX-AUTHORIZATION-005-05: 関係の事実を組み立てないまま評価器へ届いた判定が、規則 relationship_facts_present により許可にならない。
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

//spec:covers REQ-AUTHORIZATION-006: 整合トークンは自テナントのものだけを受け付ける。
//spec:covers EX-AUTHORIZATION-006-03: 別テナントで発行された整合トークンの提示が ErrConsistencyNotSatisfied になる。自テナントのトークンが通る対と並べるので、拒否はトークンの発行元によるもので、トークンを一律に拒む実装ではない。
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

// lastCheckEvent は直近の FgaCheckEvaluated を返す。
func (h *harness) lastCheckEvent(t *testing.T) *domain.FgaCheckEvaluated {
	t.Helper()
	var evaluated *domain.FgaCheckEvaluated
	for _, event := range h.events {
		if typed, ok := event.(*domain.FgaCheckEvaluated); ok {
			evaluated = typed
		}
	}
	if evaluated == nil {
		t.Fatal("CheckAccess must emit FgaCheckEvaluated")
	}
	return evaluated
}

//spec:covers REQ-AUTHORIZATION-009: 判定の監査は非個人識別情報の要約だけを残す。
//spec:covers EX-AUTHORIZATION-009-01: 監査イベントが挙げられた 7 つの値をすべて持つこと、およびリソース識別子がダイジェストとしてだけ残り、主体識別子とタプルの内容がイベントのどこにも現れないことを、直列化した本文から観測する。
func TestCheckAccessAuditKeepsNoIdentifiers(t *testing.T) {
	h := newHarness(t)
	h.write(t, viewerTuple("user", "alice"))
	input := baseInput()
	input.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	evaluated := h.lastCheckEvent(t)
	if evaluated.ResourceIDDigest == "d1" || len(evaluated.ResourceIDDigest) != 16 {
		t.Fatalf("ResourceIDDigest = %q, want a 16-character digest", evaluated.ResourceIDDigest)
	}
	if evaluated.ActorChainDepth != 1 || evaluated.ModelVersion != 1 {
		t.Fatalf("unexpected audit summary: %+v", evaluated)
	}
	// 具体例が挙げる値のうち、要約の中身を成すもの。型と関係を落としたイベントは
	// 「何についての判定か」を残していない。
	if evaluated.ResourceType != "document" || evaluated.Relation != "viewer" {
		t.Fatalf("ResourceType/Relation = %q/%q, want document/viewer", evaluated.ResourceType, evaluated.Relation)
	}
	if evaluated.Permitted {
		t.Fatal("the agent holds no relation, so the audited decision must be a denial")
	}
	if len(evaluated.Reasons) == 0 {
		t.Fatal("a denial must leave the refusing rule names in the audit event")
	}
	for _, step := range evaluated.RelationPath {
		if step != "document#viewer" {
			t.Fatalf("relation path step %q leaks an identifier", step)
		}
	}
	// 許可の側も監査へ出す。Permitted を常に false にする実装はここで落ちる。
	h.write(t, viewerTuple("agent", "researcher"))
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	if permitted := h.lastCheckEvent(t); !permitted.Permitted {
		t.Fatalf("a permitted decision must be audited as permitted: %+v", permitted)
	}

	// 「複製されない」は field を 1 つずつ見ても確かめられない。直列化した本文に
	// 主体識別子もリソース識別子も現れないことを、イベント全体に対して読む。
	encoded, err := json.Marshal(evaluated)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	for _, identifier := range []string{"alice", "researcher", "d1"} {
		if strings.Contains(string(encoded), identifier) {
			t.Fatalf("the audit event copies the identifier %q: %s", identifier, encoded)
		}
	}

	// テナントが違えば同じリソースでもダイジェストは一致しない。
	if usecases.ResourceIDDigest(otherTenantID, input.Resource) == evaluated.ResourceIDDigest {
		t.Fatal("the digest must not correlate across tenants")
	}
}

//spec:covers REQ-AUTHORIZATION-007: 列挙は許可されたものだけを返し、打ち切りを隠さない。
//spec:covers EX-AUTHORIZATION-007-01: 許可されたリソースだけが返ること、判定が CheckAccess と同じ合成を通って代行チェーンも同じく評価されること、監査に残るのが 1 件ごとの判定ではなくまとめた 1 件だけであること。
//spec:covers EX-AUTHORIZATION-007-02: 走査が上限に達した結果が Truncated を立て、完全な一覧に見せかけない。
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

	//spec:covers REQ-AUTHORIZATION-007 / REQ-AUTHORIZATION-009: 走査は 1 件ごとの判定を監査へ
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

	// 代行チェーンが CheckAccess と同じく効くこと。関係を持たない代行者を載せると
	// 許可が 1 件も残らず、関係を持つ代行者を載せると主体だけのときと同じ一覧に戻る。
	// 前者だけでは、チェーンを見た結果ではなく列挙が壊れただけの実装と区別できない。
	delegated := input
	delegated.ActorChain = []usecases.Actor{{Type: "agent", ID: "researcher"}}
	listed, err = usecases.ListAccessibleResources(context.Background(), h.deps, delegated, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if len(listed.ResourceIDs) != 0 {
		t.Fatalf("ResourceIDs = %v, want none: the delegate holds no relation", listed.ResourceIDs)
	}

	h.write(t, viewerTuple("agent", "researcher"))
	listed, err = usecases.ListAccessibleResources(context.Background(), h.deps, delegated, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if !slices.Equal(listed.ResourceIDs, []string{"d1"}) {
		t.Fatalf("ResourceIDs = %v, want [d1]: the delegate holds viewer on d1 only", listed.ResourceIDs)
	}

	// 上限ちょうどは打ち切りではない。境界を片側からしか見ないと、候補が上限に
	// 達しただけで打ち切りを立てる実装と区別できない。
	h.deps.MaxEnumeratedResources = 3
	listed, err = usecases.ListAccessibleResources(context.Background(), h.deps, input, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if listed.Truncated {
		t.Fatal("exactly as many candidates as the bound is a complete list, not a truncated one")
	}

	h.deps.MaxEnumeratedResources = 2
	listed, err = usecases.ListAccessibleResources(context.Background(), h.deps, input, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources: %v", err)
	}
	if !listed.Truncated {
		t.Fatal("hitting the enumeration limit must be reported, not hidden")
	}
	if len(listed.ResourceIDs) > 2 {
		t.Fatalf("ResourceIDs = %v, want at most the bound of 2", listed.ResourceIDs)
	}

	// 候補が 1 件も無いテナントでも、版と整合トークンは返る。ここを通らないと、
	// 走査の中でしかトークンを組み立てない実装が空の一覧でトークンを落とす。
	h.publish(t, otherTenantID)
	empty, err := usecases.ListAccessibleResources(context.Background(), h.deps, usecases.ListAccessibleResourcesInput{
		TenantID: otherTenantID, ResourceType: "document", Relation: "viewer",
		Subject: domain.SubjectRef{Type: "user", ID: "alice"},
	}, now)
	if err != nil {
		t.Fatalf("ListAccessibleResources(%s): %v", otherTenantID, err)
	}
	if len(empty.ResourceIDs) != 0 || empty.Truncated {
		t.Fatalf("an empty tenant must list nothing and truncate nothing, got %+v", empty)
	}
	if empty.Consistency == "" || empty.ModelVersion != 1 {
		t.Fatalf("an empty listing must still carry the version it read: %+v", empty)
	}
}

//spec:covers REQ-AUTHORIZATION-002: モデルに適合しない差分は 1 件も適用しない。
//spec:covers EX-AUTHORIZATION-002-02: モデルが宣言していない型を含む差分が ErrTupleInvalid で拒否され、同じ差分に含まれる適合したタプルも 1 件も残らない。
//spec:covers EX-AUTHORIZATION-002-04: 同じ組が追加と削除の双方に現れる差分が ErrTupleInvalid で拒否され、その差分の追加も削除も保管庫へ効かない。
func TestWriteRelationTuplesRejectsTheWholeDiff(t *testing.T) {
	h := newHarness(t)
	_, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes: []domain.RelationTuple{viewerTuple("user", "alice"), viewerTuple("robot", "r2")},
	}, now)
	if !errors.Is(err, domain.ErrTupleInvalid) {
		t.Fatalf("err = %v, want %v", err, domain.ErrTupleInvalid)
	}
	if remaining := h.listAll(t); len(remaining) != 0 {
		t.Fatalf("a rejected diff must not be partially applied, found %v", remaining)
	}

	// 衝突する差分の拒否は、保管庫が空のままでは観測にならない。先に 1 件書いておき、
	// 拒否された差分が持っていた削除も追加も起きていないことを読む。
	h.write(t, viewerTuple("user", "alice"))
	_, err = usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes:  []domain.RelationTuple{viewerTuple("agent", "researcher")},
		Deletes: []domain.RelationTuple{viewerTuple("agent", "researcher"), viewerTuple("user", "alice")},
	}, now)
	if !errors.Is(err, domain.ErrTupleInvalid) {
		t.Fatalf("a tuple in both writes and deletes must be rejected, got %v", err)
	}
	remaining := h.listAll(t)
	if !slices.Equal(remaining, []domain.RelationTuple{viewerTuple("user", "alice")}) {
		t.Fatalf("a conflicting diff must apply neither its writes nor its deletes, store holds %v", remaining)
	}
}

//spec:covers REQ-AUTHORIZATION-008: オブジェクトの削除は、それに依存していた間接的な関係も止める。
//spec:covers EX-AUTHORIZATION-008-01: 削除したオブジェクトを参照するタプルがリソース側・主体側のいずれも残らないこと (DeletedCount が両側の 2 件)、およびそれに依存していた間接的な関係が以後成立せず、整合トークンが進むこと。
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

	// 何も参照していないオブジェクトの削除は、削除を 1 件も起こさない。件数を見ずに
	// 削除イベントを出す実装は、監査に存在しない削除を残す。
	deleted := h.countEvents("RelationTupleDeleted")
	missing, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		DeleteObjects: []domain.ObjectRef{{Type: "group", ID: "nonexistent"}},
	}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples: %v", err)
	}
	if missing.DeletedCount != 0 {
		t.Fatalf("DeletedCount = %d, want 0: nothing references that object", missing.DeletedCount)
	}
	if emitted := h.countEvents("RelationTupleDeleted"); emitted != deleted {
		t.Fatalf("RelationTupleDeleted emitted %d times for a delete that removed nothing", emitted-deleted)
	}
}

//spec:covers REQ-AUTHORIZATION-001: 整合しないモデルは版を作らない。
//spec:covers EX-AUTHORIZATION-001-02: 宣言されていない関係を参照するモデルが ErrModelInvalid で拒否され、published 済みの版が 1 つも増えない。
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

//spec:covers EX-AUTHORIZATION-001-01: 版がテナント内で単調増加し以前の版が書き換わらないこと、レスポンスが整合トークンを含み、GetAuthorizationModel が新しい版を最新として返すこと。
func TestPutAuthorizationModelAddsAVersionWithoutRewritingTheOldOne(t *testing.T) {
	h := newHarness(t)
	first, err := usecases.GetAuthorizationModel(context.Background(), h.deps, tenantID, 0)
	if err != nil {
		t.Fatalf("GetAuthorizationModel: %v", err)
	}
	if first.Model.Version != 1 {
		t.Fatalf("the first published version = %d, want 1", first.Model.Version)
	}

	// 2 つ目の版は 1 つ目と形が違う。以前の版を読み直したときに、上書きされたのか
	// 残っているのかを中身で見分けられるようにする。
	extended := append(referenceTypes(), domain.ResourceTypeDefinition{Name: "folder"})
	second, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, extended, now)
	if err != nil {
		t.Fatalf("PutAuthorizationModel: %v", err)
	}
	if second.Model.Version != first.Model.Version+1 {
		t.Fatalf("Version = %d, want %d: versions increase by one", second.Model.Version, first.Model.Version+1)
	}
	if second.Consistency == "" {
		t.Fatal("the publish response must carry a consistency token")
	}

	latest, err := usecases.GetAuthorizationModel(context.Background(), h.deps, tenantID, 0)
	if err != nil {
		t.Fatalf("GetAuthorizationModel: %v", err)
	}
	if latest.Model.Version != second.Model.Version {
		t.Fatalf("latest version = %d, want the just-published %d", latest.Model.Version, second.Model.Version)
	}
	if len(latest.Model.ResourceTypes) != len(extended) {
		t.Fatalf("the latest model must be the one just published, got %d resource types", len(latest.Model.ResourceTypes))
	}

	previous, err := usecases.GetAuthorizationModel(context.Background(), h.deps, tenantID, first.Model.Version)
	if err != nil {
		t.Fatalf("GetAuthorizationModel(version=1): %v", err)
	}
	if len(previous.Model.ResourceTypes) != len(referenceTypes()) {
		t.Fatalf("publishing must not rewrite version 1, it now has %d resource types",
			len(previous.Model.ResourceTypes))
	}

	// テナントごとの採番。別テナントの版数に引きずられる実装はここで落ちる。
	other := h.publish(t, otherTenantID)
	if other.Model.Version != 1 {
		t.Fatalf("the first version of %s = %d, want 1", otherTenantID, other.Model.Version)
	}
}

//spec:covers EX-AUTHORIZATION-001-03: 書き換え規則が循環するモデルが ErrModelInvalid で拒否され、版が 1 つも増えない。
//spec:covers EX-AUTHORIZATION-001-04: 型名または関係名がフォーマットに反するモデルが ErrModelInvalid で拒否され、版が 1 つも増えない。
func TestPutAuthorizationModelLeavesNoVersionBehindWhenItRefuses(t *testing.T) {
	cases := map[string][]domain.ResourceTypeDefinition{
		"a cyclic rewrite": {
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "editor"},
				}},
				{Name: "editor", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "viewer"},
				}},
			}},
		},
		"a malformed type name": {
			{Name: "Document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"Document"}},
				}},
			}},
		},
		"a malformed relation name": {
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "Viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user"}},
				}},
			}},
		},
	}
	for name, types := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			if _, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, types, now); !errors.Is(err, domain.ErrModelInvalid) {
				t.Fatalf("err = %v, want %v", err, domain.ErrModelInvalid)
			}
			// 拒否が防いだ効果。採番してから検証する実装は、版を 2 へ進めてここで落ちる。
			published, err := usecases.GetAuthorizationModel(context.Background(), h.deps, tenantID, 0)
			if err != nil {
				t.Fatalf("GetAuthorizationModel: %v", err)
			}
			if published.Model.Version != 1 {
				t.Fatalf("latest version = %d, want the refused publish to have created none", published.Model.Version)
			}
		})
	}
}

//spec:covers EX-AUTHORIZATION-002-01: 追加と削除を含む差分が 1 回で適用されること、既に存在する組の再追加が冪等に扱われること、返った整合トークンをそのまま以後の判定へ渡せること。
func TestWriteRelationTuplesAppliesTheDiffAtOnceAndIsIdempotent(t *testing.T) {
	h := newHarness(t)
	h.write(t, viewerTuple("user", "alice"), viewerTuple("agent", "researcher"))

	// 追加と削除を 1 つの差分に混ぜる。両方が同じ呼び出しで効くことを件数で読む。
	outcome, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes: []domain.RelationTuple{
			// 既にある組の再追加。written には数えず、消えもしない。
			viewerTuple("user", "alice"),
			{
				Resource: domain.ObjectRef{Type: "document", ID: "d2"}, Relation: "viewer",
				Subject: domain.SubjectRef{Type: "user", ID: "alice"},
			},
		},
		Deletes: []domain.RelationTuple{viewerTuple("agent", "researcher")},
	}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples: %v", err)
	}
	if outcome.WrittenCount != 1 {
		t.Fatalf("WrittenCount = %d, want 1: re-adding an existing pair adds nothing", outcome.WrittenCount)
	}
	if outcome.DeletedCount != 1 {
		t.Fatalf("DeletedCount = %d, want 1", outcome.DeletedCount)
	}
	want := []domain.RelationTuple{
		{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "user", ID: "alice"},
		},
		{
			Resource: domain.ObjectRef{Type: "document", ID: "d2"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "user", ID: "alice"},
		},
	}
	if remaining := h.listAll(t); !slices.Equal(remaining, want) {
		t.Fatalf("store holds %v, want %v", remaining, want)
	}

	// 冪等は件数だけの話ではない。何も足さなかった書き込みが「書いた」と監査へ
	// 言えば、外から見た振る舞いは冪等でない。
	written := h.countEvents("RelationTupleWritten")
	readd, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes: []domain.RelationTuple{viewerTuple("user", "alice")},
	}, now)
	if err != nil {
		t.Fatalf("WriteRelationTuples: %v", err)
	}
	if readd.WrittenCount != 0 || readd.DeletedCount != 0 {
		t.Fatalf("re-adding an existing pair changed %+v, want no change", readd)
	}
	if after := h.countEvents("RelationTupleWritten"); after != written {
		t.Fatalf("RelationTupleWritten emitted %d times, want the idempotent re-add to emit none", after-written)
	}

	// 返ったトークンはそのまま判定へ渡せる。書き込みが進めた版より古いトークンを
	// 返す実装なら通ってしまうので、直後の判定で拒否されないことだけでなく、
	// トークンが実際に版を指していることを、1 つ先のトークンの拒否と対で読む。
	input := baseInput()
	input.MinimumConsistency = outcome.Consistency
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); err != nil {
		t.Fatalf("the token the write returned must be satisfiable: %v", err)
	}
	input.MinimumConsistency = domain.EncodeConsistencyToken(tenantID, 9999)
	if _, err := usecases.CheckAccess(context.Background(), h.deps, input, now); !errors.Is(err, domain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a token ahead of the store must still be rejected, got %v", err)
	}
}

//spec:covers EX-AUTHORIZATION-002-03: direct 規則が許していない主体型またはワイルドカードを含む差分が ErrTupleInvalid で拒否され、同じ差分の適合したタプルも 1 件も残らない。
func TestWriteRelationTuplesRejectsSubjectFormsTheDirectRuleDoesNotAllow(t *testing.T) {
	// document#viewer の direct は user / agent / group#member だけを許す。
	cases := map[string]domain.RelationTuple{
		"a subject type the direct rule does not list": {
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "group", ID: "eng"},
		},
		"a wildcard the direct rule does not declare": {
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer",
			Subject: domain.SubjectRef{Type: "user", ID: domain.Wildcard},
		},
	}
	for name, offending := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			_, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
				Writes: []domain.RelationTuple{viewerTuple("user", "alice"), offending},
			}, now)
			if !errors.Is(err, domain.ErrTupleInvalid) {
				t.Fatalf("err = %v, want %v", err, domain.ErrTupleInvalid)
			}
			if remaining := h.listAll(t); len(remaining) != 0 {
				t.Fatalf("a rejected diff must not apply its valid half, found %v", remaining)
			}
		})
	}
}

//spec:covers EX-AUTHORIZATION-002-05: 認可モデルが未登録のテナントへの書き込みが ErrModelNotFound で拒否され、タプルを 1 件も残さない。
func TestWriteRelationTuplesRefusesATenantWithNoModel(t *testing.T) {
	h := newBareHarness()
	_, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{
		Writes: []domain.RelationTuple{viewerTuple("user", "alice")},
	}, now)
	if !errors.Is(err, domain.ErrModelNotFound) {
		t.Fatalf("err = %v, want %v", err, domain.ErrModelNotFound)
	}
	if remaining := h.listAll(t); len(remaining) != 0 {
		t.Fatalf("a write with no model to validate against must leave nothing behind, found %v", remaining)
	}
}

//spec:covers EX-AUTHORIZATION-006-01: 別テナントに同じリソース識別子・関係・主体識別子のタプルがあっても読み出されず、自テナントの判定は不許可になる。
//spec:covers EX-AUTHORIZATION-006-02: 同じ識別子を渡しても、対象テナントは呼び出し元で解決したものから変わらない。
func TestCheckAccessReadsOnlyTheCallersTenant(t *testing.T) {
	h := newHarness(t)
	h.publish(t, otherTenantID)
	// 同じリソース識別子・関係・主体識別子を、別テナントにだけ置く。
	h.writeFor(t, otherTenantID, viewerTuple("user", "alice"))

	denied, err := usecases.CheckAccess(context.Background(), h.deps, baseInput(), now)
	if err != nil {
		t.Fatalf("CheckAccess: %v", err)
	}
	if denied.Permitted {
		t.Fatal("the grant lives in another tenant and must not reach this decision")
	}
	if !slices.Contains(denied.Reasons, domain.ReasonNoRelationPath) {
		t.Fatalf("Reasons = %v, want %s", denied.Reasons, domain.ReasonNoRelationPath)
	}
	// 判定が別テナントへ滑っていないことは、監査に残るテナントで読む。不許可という
	// 答えだけでは、対象テナントを取り違えたうえで別の理由で拒否した実装と区別できない。
	if evaluated := h.lastCheckEvent(t); evaluated.TenantID != tenantID {
		t.Fatalf("the audited decision belongs to %q, want the caller's tenant %q", evaluated.TenantID, tenantID)
	}

	// 同じ入力を別テナントの呼び出しとして通すと許可になる。入力ではなく呼び出し元の
	// テナントだけが答えを分けていることを、この対が示す。
	sameInput := baseInput()
	sameInput.TenantID = otherTenantID
	permitted, err := usecases.CheckAccess(context.Background(), h.deps, sameInput, now)
	if err != nil {
		t.Fatalf("CheckAccess(%s): %v", otherTenantID, err)
	}
	if !permitted.Permitted {
		t.Fatalf("the grant lives in %s and must be seen there, reasons %v", otherTenantID, permitted.Reasons)
	}
}
