package handlers_http_test

// docs/contexts/authorization/standards.md が宣言する 5 行を、製品の正式な入口である
// POST /api/admin/v1/authorization/check と
// POST /api/admin/v1/authorization/list-accessible-resources から観測する。
//
// 「判定 context に載せる」は応答からは読めない。Authorizer は差し替え可能なポートで、
// AUTHZEN=remote では AuthZRequest がそのまま AuthZEN の PDP へ渡るので、この 2 行が
// 固定しているのは判定器へ渡る AuthZRequest そのものの形である。したがって観測は、
// 製品の組み立てへ差した記録用 Authorizer が受け取った AuthZRequest を読む形にする。

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authorization"
	authorizationmemory "github.com/ambi/idmagic/backend/authorization/db_memory"
	authorizationdomain "github.com/ambi/idmagic/backend/authorization/domain"
	authorizationports "github.com/ambi/idmagic/backend/authorization/ports"
	"github.com/ambi/idmagic/backend/authorization/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// ---- 差し替えの口 ----

// recordingAuthorizer は判定器へ渡った AuthZRequest を記録する。合成そのものは
// 包んだ Authorizer が行うので、記録は製品の判定を変えない。
type recordingAuthorizer struct {
	inner    oauthports.Authorizer
	mu       sync.Mutex
	requests []spec.AuthZRequest
}

func (r *recordingAuthorizer) Authorize(ctx context.Context, request spec.AuthZRequest) (spec.AuthZResponse, error) {
	r.mu.Lock()
	r.requests = append(r.requests, request)
	r.mu.Unlock()
	return r.inner.Authorize(ctx, request)
}

// last は直前の判定が評価器へ渡した要求。1 度も渡っていなければ、載せるという
// 宣言そのものが破れているので落とす。
func (r *recordingAuthorizer) last(t *testing.T) spec.AuthZRequest {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		t.Fatal("the decision never reached the evaluator")
	}
	return r.requests[len(r.requests)-1]
}

// constantAuthorizer は関係の事実を見ずに固定の答えを返す PDP。判定が評価器の側に
// あることを観測するために使う。
type constantAuthorizer struct{ permit bool }

func (c constantAuthorizer) Authorize(context.Context, spec.AuthZRequest) (spec.AuthZResponse, error) {
	if c.permit {
		return spec.AuthZResponse{Permit: true}, nil
	}
	return spec.AuthZResponse{Permit: false, Reasons: []string{"remote_policy_denied"}}, nil
}

var errEvaluatorUnreachable = errors.New("authzen evaluator is unreachable")

// unreachableAuthorizer は判定を返せない評価器。
type unreachableAuthorizer struct{}

func (unreachableAuthorizer) Authorize(context.Context, spec.AuthZRequest) (spec.AuthZResponse, error) {
	return spec.AuthZResponse{}, errEvaluatorUnreachable
}

// unreachableTuples は関係タプルの読み出しだけを失敗させる。書き込みは通るので、
// 事実を組み立てる段だけがストアへ到達できない状況になる。
type unreachableTuples struct {
	authorizationports.RelationTupleRepository
}

func (unreachableTuples) ListSubjects(
	context.Context, string, authorizationdomain.ObjectRef, string,
) ([]authorizationdomain.SubjectRef, error) {
	return nil, errors.New("relation tuple store is unreachable")
}

// ---- 組み立て ----

// standardsSeams は製品の組み立てのうち、答えが出ない状況を作るために差し替える点。
// ゼロ値は製品と同じ組み立てになる。
type standardsSeams struct {
	// authorizer が nil でなければ、記録用 Authorizer の内側をこれに差し替える。
	authorizer oauthports.Authorizer
	// noAuthorizer は評価器を持たない組み立てを作る。
	noAuthorizer bool
	// unreachableStore は関係タプルの読み出しを失敗させる。
	unreachableStore bool
}

type standardsFixture struct {
	e        *echo.Echo
	recorded *recordingAuthorizer
}

// suspendedAgent は登録されているが有効ではない Agent。関係は持たせたうえで
// 有効性だけを落とせるようにする。
func suspendedAgent(id string) *agentdomain.Agent {
	agent := activeAgent(id)
	agent.Status = idmdomain.AgentStatusDisabled
	return agent
}

func newStandardsFixture(t *testing.T, seams standardsSeams, agents ...*agentdomain.Agent) *standardsFixture {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(actor("admin", []string{"admin"}))
	agentRepo := agentmemory.NewAgentRepository()
	for _, agent := range agents {
		if err := agentRepo.Save(context.Background(), agent); err != nil {
			t.Fatal(err)
		}
	}
	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: "acme", Realm: "acme", DisplayName: "acme",
		Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	store := authorizationmemory.NewStore()
	var tuples authorizationports.RelationTupleRepository = authorizationmemory.NewRelationTupleRepository(store)
	if seams.unreachableStore {
		tuples = unreachableTuples{RelationTupleRepository: tuples}
	}
	recorded := &recordingAuthorizer{inner: authorizationLocal.Local{}}
	if seams.authorizer != nil {
		recorded.inner = seams.authorizer
	}
	var authorizer oauthports.Authorizer = recorded
	if seams.noAuthorizer {
		authorizer = nil
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo:   tenantRepo,
		Emit:         func(spec.DomainEvent) {},
		IdManagement: idmanagement.Module{UserRepo: userRepo, AgentRepo: agentRepo},
		Authorization: authorization.Module{
			TupleRepo: tuples,
			ModelRepo: authorizationmemory.NewAuthorizationModelRepository(store),
		},
		Authorizer: authorizer,
		AuthnResolver: &fakeAuthnResolver{
			ctx: &authdomain.AuthenticationContext{UserID: "admin", AuthTime: time.Now().Unix()},
		},
	})
	return &standardsFixture{e: e, recorded: recorded}
}

// delegationModelRequest は代行と入れ子グループの両方を宣言するモデル。深さ上限の
// 観測には group#member が group#member を直接主体に取れることが要る。
func delegationModelRequest() map[string]any {
	directOn := func(types ...string) map[string]any {
		return map[string]any{"kind": "direct", "direct_subject_types": types}
	}
	return map[string]any{"resource_types": []map[string]any{
		{"name": "user"},
		{"name": "agent"},
		{"name": "group", "relations": []map[string]any{
			{"name": "member", "rewrites": []map[string]any{directOn("user", "agent", "group#member")}},
		}},
		{"name": "document", "relations": []map[string]any{
			{"name": "viewer", "rewrites": []map[string]any{directOn("user", "agent", "group#member")}},
		}},
	}}
}

func (f *standardsFixture) publishModel(t *testing.T) {
	t.Helper()
	rec := post(t, f.e, realmPrefix+"/api/admin/v1/authorization/model", delegationModelRequest())
	if rec.Code != http.StatusCreated {
		t.Fatalf("PutAuthorizationModel status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func (f *standardsFixture) writeTuples(t *testing.T, tuples ...map[string]any) {
	t.Helper()
	if len(tuples) == 0 {
		return
	}
	rec := post(t, f.e, realmPrefix+"/api/admin/v1/authorization/relation-tuples",
		map[string]any{"writes": tuples})
	if rec.Code != http.StatusOK {
		t.Fatalf("WriteRelationTuples status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// viewerGrant は document への viewer を 1 件与えるタプル。
func viewerGrant(resourceID, subjectType, subjectID string) map[string]any {
	return map[string]any{
		"resource_type": "document", "resource_id": resourceID, "relation": "viewer",
		"subject_type": subjectType, "subject_id": subjectID,
	}
}

// groupViewerGrant は document への viewer を group のメンバー集合へ与えるタプル。
func groupViewerGrant(resourceID, groupID string) map[string]any {
	return map[string]any{
		"resource_type": "document", "resource_id": resourceID, "relation": "viewer",
		"subject_type": "group", "subject_id": groupID, "subject_relation": "member",
	}
}

// nestedGroupGrant は document から user:alice まで depth 段の group を連ねる。
// 段数だけが違う 2 つの入力を作れるので、上限超過を普通の否認と区別できる。
func nestedGroupGrant(resourceID string, depth int) []map[string]any {
	tuples := []map[string]any{groupViewerGrant(resourceID, resourceID+"g0")}
	for i := range depth - 1 {
		tuples = append(tuples, map[string]any{
			"resource_type": "group", "resource_id": resourceID + "g" + strconv.Itoa(i), "relation": "member",
			"subject_type": "group", "subject_id": resourceID + "g" + strconv.Itoa(i+1),
			"subject_relation": "member",
		})
	}
	return append(tuples, map[string]any{
		"resource_type": "group", "resource_id": resourceID + "g" + strconv.Itoa(depth-1), "relation": "member",
		"subject_type": "user", "subject_id": "alice",
	})
}

func checkRequest(resourceID, subjectID string) map[string]any {
	return map[string]any{
		"resource_type": "document", "resource_id": resourceID, "relation": "viewer",
		"subject_type": "user", "subject_id": subjectID,
	}
}

// withChain は判定要求へ代行チェーンを外側から内側の順に足す。
func withChain(request map[string]any, actors ...map[string]any) map[string]any {
	request["actor_chain"] = actors
	return request
}

func agentActor(id string) map[string]any {
	return map[string]any{"type": "agent", "id": id}
}

type decisionResult struct {
	Permitted    bool     `json:"permitted"`
	ModelVersion int      `json:"model_version"`
	Consistency  string   `json:"consistency"`
	RelationPath []string `json:"relation_path"`
	Reasons      []string `json:"reasons"`
}

// check は判定を 1 回要求する。応答符号が 200 でないときは result をゼロ値のまま返し、
// 呼び出し側が退避していないことを応答そのもので確かめられるようにする。
func (f *standardsFixture) check(t *testing.T, body map[string]any) (*httptest.ResponseRecorder, decisionResult) {
	t.Helper()
	rec := post(t, f.e, realmPrefix+"/api/admin/v1/authorization/check", body)
	if rec.Code != http.StatusOK {
		return rec, decisionResult{}
	}
	var decoded struct {
		Result decisionResult `json:"result"`
	}
	decode(t, rec, &decoded)
	return rec, decoded.Result
}

type searchResult struct {
	ResourceIDs  []string `json:"resource_ids"`
	Truncated    bool     `json:"truncated"`
	ModelVersion int      `json:"model_version"`
	Consistency  string   `json:"consistency"`
}

func (f *standardsFixture) search(t *testing.T, body map[string]any) searchResult {
	t.Helper()
	rec := post(t, f.e, realmPrefix+"/api/admin/v1/authorization/list-accessible-resources", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListAccessibleResources status=%d body=%s", rec.Code, rec.Body.String())
	}
	var decoded struct {
		Result searchResult `json:"result"`
	}
	decode(t, rec, &decoded)
	return decoded.Result
}

// assertNoPermit は、答えの出なかった判定が許可として外へ出ていないことを応答本体で
// 確かめる。状態符号だけを読むと、5xx を返しながら許可を載せる実装を見逃す。
func assertNoPermit(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code == http.StatusOK {
		t.Fatalf("a decision that cannot be answered must not be a 200, got %s", rec.Body.String())
	}
	if strings.Contains(strings.ReplaceAll(rec.Body.String(), " ", ""), `"permitted":true`) {
		t.Fatalf("the response fell back to a permit: %s", rec.Body.String())
	}
}

// ---- AUTHZEN-FGA-EVALUATION ----

// 載ること、そして関係の成否が判定 context の事実として渡ること。応答だけを読む観測では、
// 関係を自分で判断して結果だけ返す実装と区別できないので、判定器へ渡った AuthZRequest を読む。
//
//spec:covers AUTHZEN-FGA-EVALUATION: 関係に基づく判定が {subject, action, resource, context} の評価に
func TestAuthorizationDecisionRidesOnTheSubjectActionResourceContextEvaluation(t *testing.T) {
	t.Run("the evaluation carries the subject, action, resource, and the relationship outcome", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{})
		f.publishModel(t)
		f.writeTuples(t, viewerGrant("d1", "user", "alice"))

		_, result := f.check(t, checkRequest("d1", "alice"))
		if !result.Permitted {
			t.Fatalf("alice holds viewer on d1, got reasons %v", result.Reasons)
		}

		request := f.recorded.last(t)
		if request.Action != spec.ActionResourceAccess {
			t.Fatalf("Action = %q, want %q", request.Action, spec.ActionResourceAccess)
		}
		if request.Subject.ID != "alice" || request.Subject.Properties.TenantID != "acme" {
			t.Fatalf("Subject = %+v, want alice in acme", request.Subject)
		}
		if request.Resource.Type != "document" || request.Resource.ID != "d1" ||
			request.Resource.Properties.TenantID != "acme" {
			t.Fatalf("Resource = %+v, want document d1 in acme", request.Resource)
		}
		if request.Context.Now.IsZero() {
			t.Fatal("the decision context must carry the evaluation time")
		}

		facts := request.Context.Relationship
		if facts == nil {
			t.Fatal("the relationship outcome must reach the evaluator as a context fact, not stay in the use case")
		}
		if !facts.Evaluated || !facts.SubjectPermitted {
			t.Fatalf("Relationship = %+v, want an evaluated fact permitting the subject", facts)
		}
		if facts.ModelVersion != 1 {
			t.Fatalf("Relationship.ModelVersion = %d, want the published version 1", facts.ModelVersion)
		}
		if !slices.Equal(facts.RelationPath, []string{"document#viewer"}) {
			t.Fatalf("Relationship.RelationPath = %v, want [document#viewer]", facts.RelationPath)
		}
	})

	t.Run("a subject holding no relation arrives as an evaluated negative, not as a gap", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{})
		f.publishModel(t)
		f.writeTuples(t, viewerGrant("d1", "user", "alice"))

		_, result := f.check(t, checkRequest("d1", "bob"))
		if result.Permitted {
			t.Fatal("bob was never granted viewer on d1")
		}

		facts := f.recorded.last(t).Context.Relationship
		if facts == nil || !facts.Evaluated {
			t.Fatalf("Relationship = %+v, want a fact saying the relation was evaluated", facts)
		}
		if facts.SubjectPermitted {
			t.Fatal("the fact must report the relation as not holding")
		}
		if !slices.Contains(facts.DenyReasons, authorizationdomain.ReasonNoRelationPath) {
			t.Fatalf("Relationship.DenyReasons = %v, want %s",
				facts.DenyReasons, authorizationdomain.ReasonNoRelationPath)
		}
	})

	t.Run("the evaluator's answer is the decision, not a second opinion", func(t *testing.T) {
		// 判定が評価に載っているなら、答えは評価器の側から来る。関係の事実をそのまま
		// 結果にする実装は、この 2 つの向きのどちらかで落ちる。
		permitting := newStandardsFixture(t, standardsSeams{authorizer: constantAuthorizer{permit: true}})
		permitting.publishModel(t)
		permitting.writeTuples(t, viewerGrant("d1", "user", "alice"))
		if _, result := permitting.check(t, checkRequest("d1", "bob")); !result.Permitted {
			t.Fatalf("the evaluator permitted; the use case must not overrule it, reasons %v", result.Reasons)
		}

		denying := newStandardsFixture(t, standardsSeams{authorizer: constantAuthorizer{permit: false}})
		denying.publishModel(t)
		denying.writeTuples(t, viewerGrant("d1", "user", "alice"))
		if _, result := denying.check(t, checkRequest("d1", "alice")); result.Permitted {
			t.Fatal("the evaluator denied; holding the relation must not permit on its own")
		}
	})
}

// ---- AUTHZEN-FGA-ACTOR-CHAIN ----

// 種別・識別子・有効性の 3 つに分かれて表れること。拒否理由だけを読む観測では、チェーンを
// 1 個の真偽値へ畳む実装や、種別と識別子を 1 本の文字列へ連結する実装を区別できない。
//
//spec:covers AUTHZEN-FGA-ACTOR-CHAIN: 代行チェーンが判定 context に明示的に載り、各段がプリンシパル
func TestAuthorizationDecisionCarriesEveryDelegationStageSeparately(t *testing.T) {
	newFixture := func(t *testing.T) *standardsFixture {
		t.Helper()
		f := newStandardsFixture(t, standardsSeams{},
			activeAgent("researcher"), suspendedAgent("retired"))
		f.publishModel(t)
		// 有効性だけを観測できるよう、3 つの代行者すべてに関係は持たせる。
		f.writeTuples(t,
			viewerGrant("d1", "user", "alice"),
			viewerGrant("d1", "agent", "researcher"),
			viewerGrant("d1", "agent", "retired"),
			viewerGrant("d1", "agent", "ghost"),
		)
		return f
	}

	t.Run("each stage keeps its own type, identifier, and validity", func(t *testing.T) {
		f := newFixture(t)
		_, result := f.check(t, withChain(checkRequest("d1", "alice"),
			agentActor("researcher"), agentActor("retired")))

		want := []spec.AuthZActor{
			{Type: "agent", ID: "researcher", Active: true},
			{Type: "agent", ID: "retired", Active: false},
		}
		if chain := f.recorded.last(t).Context.ActorChain; !slices.Equal(chain, want) {
			t.Fatalf("ActorChain = %+v, want %+v", chain, want)
		}
		if result.Permitted {
			t.Fatal("a suspended delegate must not be permitted")
		}
		if !slices.Contains(result.Reasons, "actor_chain_principals_active") {
			t.Fatalf("Reasons = %v, want actor_chain_principals_active", result.Reasons)
		}
	})

	t.Run("a delegate nothing registered resolves to inactive, not to unknown", func(t *testing.T) {
		f := newFixture(t)
		_, result := f.check(t, withChain(checkRequest("d1", "alice"), agentActor("ghost")))

		want := []spec.AuthZActor{{Type: "agent", ID: "ghost", Active: false}}
		if chain := f.recorded.last(t).Context.ActorChain; !slices.Equal(chain, want) {
			t.Fatalf("ActorChain = %+v, want %+v", chain, want)
		}
		if result.Permitted {
			t.Fatal("a delegate whose validity cannot be resolved must not be permitted")
		}
	})

	t.Run("direct access carries an empty chain rather than an invented stage", func(t *testing.T) {
		f := newFixture(t)
		_, result := f.check(t, checkRequest("d1", "alice"))
		if chain := f.recorded.last(t).Context.ActorChain; len(chain) != 0 {
			t.Fatalf("ActorChain = %+v, want empty for direct access", chain)
		}
		if !result.Permitted {
			t.Fatalf("alice holds viewer on d1, got reasons %v", result.Reasons)
		}
	})
}

// ---- AUTHZEN-FGA-FAIL-CLOSED ----

// ストアへ到達できない、のいずれでも許可へ退避しないこと。答えの出る入力で拒否が返ることは
// この行の観測にならないので、4 つとも「答えが出ない入力」で作る。
//
//spec:covers AUTHZEN-FGA-FAIL-CLOSED: 評価器が判定を返せない、事実が欠けている、深さ上限に達した、
func TestAuthorizationFailsClosedWhenTheDecisionCannotBeAnswered(t *testing.T) {
	t.Run("an evaluator that cannot answer is not a permit", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{authorizer: unreachableAuthorizer{}})
		f.publishModel(t)
		f.writeTuples(t, viewerGrant("d1", "user", "alice"))
		rec, _ := f.check(t, checkRequest("d1", "alice"))
		assertNoPermit(t, rec)
	})

	t.Run("a missing evaluator is not a permit", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{noAuthorizer: true})
		f.publishModel(t)
		f.writeTuples(t, viewerGrant("d1", "user", "alice"))
		rec, _ := f.check(t, checkRequest("d1", "alice"))
		assertNoPermit(t, rec)
	})

	t.Run("an unreachable relation store is not a permit", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{unreachableStore: true})
		f.publishModel(t)
		f.writeTuples(t, viewerGrant("d1", "user", "alice"))
		rec, _ := f.check(t, checkRequest("d1", "alice"))
		assertNoPermit(t, rec)
	})

	t.Run("facts that could never be built are not a permit", func(t *testing.T) {
		// モデルが 1 版も無いテナントでは関係の事実を組み立てられない。
		f := newStandardsFixture(t, standardsSeams{})
		rec, _ := f.check(t, checkRequest("d1", "alice"))
		assertNoPermit(t, rec)
	})

	t.Run("a graph deeper than the traversal bound denies instead of giving up open", func(t *testing.T) {
		// 段数だけが違う 2 つの入力を対にする。上限に収まる側が許可になるので、
		// 上限を超えた側の拒否は「関係が無い」ではなく「答えを出せなかった」である。
		within := newStandardsFixture(t, standardsSeams{})
		within.publishModel(t)
		within.writeTuples(t, nestedGroupGrant("shallow", 6)...)
		if _, result := within.check(t, checkRequest("shallow", "alice")); !result.Permitted {
			t.Fatalf("a chain inside the bound must permit, got reasons %v", result.Reasons)
		}

		beyond := newStandardsFixture(t, standardsSeams{})
		beyond.publishModel(t)
		beyond.writeTuples(t, nestedGroupGrant("deep", 12)...)
		_, result := beyond.check(t, checkRequest("deep", "alice"))
		if result.Permitted {
			t.Fatal("reaching the traversal bound must not fall back to permit")
		}
		if !slices.Contains(result.Reasons, authorizationdomain.ReasonDepthExceeded) {
			t.Fatalf("Reasons = %v, want %s", result.Reasons, authorizationdomain.ReasonDepthExceeded)
		}
	})
}

// ---- RFC8693-FGA-ACTOR-AND ----

// 許可すること。3 つの主体が関係を持つか持たないかの全 8 通りを通し、許可になるのは 3 つとも
// 持つ 1 通りだけであることを読む。連鎖の 1 主体だけが関係を持つ入力がこの表に 3 つ含まれるので、
// 和で効く実装、先頭の 1 段だけを見る実装、チェーンを無視する実装はいずれもここで落ちる。
//
//spec:covers RFC8693-FGA-ACTOR-AND: sub の主体と act チェーン上のすべての actor が同じ関係を持つときだけ
func TestDelegatedAccessRequiresTheRelationOnEveryActorInTheChain(t *testing.T) {
	for _, holders := range [][3]bool{
		{false, false, false},
		{false, false, true},
		{false, true, false},
		{false, true, true},
		{true, false, false},
		{true, false, true},
		{true, true, false},
		{true, true, true},
	} {
		subject, first, second := holders[0], holders[1], holders[2]
		name := fmt.Sprintf("subject=%t researcher=%t assistant=%t", subject, first, second)
		t.Run(name, func(t *testing.T) {
			f := newStandardsFixture(t, standardsSeams{}, activeAgent("researcher"), activeAgent("assistant"))
			f.publishModel(t)
			var tuples []map[string]any
			if subject {
				tuples = append(tuples, viewerGrant("d1", "user", "alice"))
			}
			if first {
				tuples = append(tuples, viewerGrant("d1", "agent", "researcher"))
			}
			if second {
				tuples = append(tuples, viewerGrant("d1", "agent", "assistant"))
			}
			f.writeTuples(t, tuples...)

			_, result := f.check(t, withChain(checkRequest("d1", "alice"),
				agentActor("researcher"), agentActor("assistant")))
			want := subject && first && second
			if result.Permitted != want {
				t.Fatalf("Permitted = %t, want %t (reasons %v)", result.Permitted, want, result.Reasons)
			}
		})
	}
}

// ---- AUTHZEN-FGA-SEARCH ----

// あること、打ち切りを結果に示すこと。上限を注入して観測すると既定の上限を持たない実装と
// 区別できないので、製品の既定のまま候補を上限より 1 件多く置く。
//
//spec:covers AUTHZEN-FGA-SEARCH: 主体を固定したリソースの探索を提供すること、その走査が上限つきで
func TestAccessibleResourceSearchIsSubjectFixedBoundedAndReportsTruncation(t *testing.T) {
	searchRequest := map[string]any{
		"resource_type": "document", "relation": "viewer",
		"subject_type": "user", "subject_id": "alice",
	}

	t.Run("the search returns exactly what the fixed subject reaches", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{})
		f.publishModel(t)
		f.writeTuples(t,
			viewerGrant("d1", "user", "alice"),
			viewerGrant("d2", "user", "bob"),
			groupViewerGrant("d3", "eng"),
			map[string]any{
				"resource_type": "group", "resource_id": "eng", "relation": "member",
				"subject_type": "user", "subject_id": "alice",
			},
		)
		result := f.search(t, searchRequest)
		if !slices.Equal(result.ResourceIDs, []string{"d1", "d3"}) {
			t.Fatalf("resource_ids = %v, want [d1 d3]", result.ResourceIDs)
		}
		if result.Truncated {
			t.Fatal("three candidates are inside the bound and must not report truncation")
		}
	})

	t.Run("the default traversal bound truncates and says so", func(t *testing.T) {
		f := newStandardsFixture(t, standardsSeams{})
		f.publishModel(t)
		limit := usecases.DefaultMaxEnumeratedResources
		tuples := make([]map[string]any, 0, limit+1)
		for i := range limit + 1 {
			tuples = append(tuples, viewerGrant(fmt.Sprintf("d%04d", i), "user", "alice"))
		}
		f.writeTuples(t, tuples...)

		result := f.search(t, searchRequest)
		if len(result.ResourceIDs) != limit {
			t.Fatalf("len(resource_ids) = %d, want the bound %d", len(result.ResourceIDs), limit)
		}
		if !result.Truncated {
			t.Fatal("a scan that stopped at the bound must say so rather than look complete")
		}
	})
}
