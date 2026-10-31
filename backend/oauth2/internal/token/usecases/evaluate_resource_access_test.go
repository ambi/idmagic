package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/internal/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	evalTenant   = "tenant-1"
	evalResource = "https://mcp.example/"
	evalClient   = "rs-client"
)

type evalTokens struct {
	result *ports.IntrospectionResult
	err    error
}

func (e evalTokens) IntrospectAccessToken(context.Context, string) (*ports.IntrospectionResult, error) {
	return e.result, e.err
}

type evalRelations struct {
	decision policyports.RelationDecision
	err      error
	got      []policyports.RelationCheck
}

func (r *evalRelations) CheckRelation(_ context.Context, check policyports.RelationCheck) (policyports.RelationDecision, error) {
	r.got = append(r.got, check)
	return r.decision, r.err
}

func evalServer(t *testing.T, rules ...domain.ResourceAccessRule) ports.McpResourceServerRepository {
	t.Helper()
	repo := db_memory.NewMcpResourceServerRepository()
	if err := repo.Save(context.Background(), &domain.McpResourceServer{
		TenantID: evalTenant, ID: "rs-1", Resource: evalResource, Name: "rs", Scopes: []string{"docs:read"},
		ClientID: evalClient, AccessRules: rules, State: domain.McpResourceServerActive,
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

func evalRequest() ResourceAccessRequest {
	return ResourceAccessRequest{
		TenantID: evalTenant, ClientID: evalClient, Token: "token",
		ResourceType: "document", ResourceID: "d1", Relation: "viewer",
	}
}

func evalAgentToken() *ports.IntrospectionResult {
	return &ports.IntrospectionResult{
		Active: true, JTI: "jti", Sub: "agent-client", ClientID: "agent-client", Scope: "docs:read",
		Aud: []string{evalResource}, PrincipalType: domain.PrincipalTypeAgent, AgentID: "A1",
	}
}

var evalRule = domain.ResourceAccessRule{ResourceType: "document", Relation: "viewer", Scopes: []string{"docs:read"}}

//spec:covers REQ-OAUTH2-062: 不許可の理由と、関係の判定へ進まないこと
func TestEvaluateResourceAccessDenials(t *testing.T) {
	detailRule := evalRule
	detailRule.AuthorizationDetailType = "document_access"
	withToken := func(mutate func(*ports.IntrospectionResult)) *ports.IntrospectionResult {
		token := evalAgentToken()
		mutate(token)
		return token
	}
	cases := []struct {
		name   string
		rule   domain.ResourceAccessRule
		token  *ports.IntrospectionResult
		mutate func(*ResourceAccessRequest)
		reason string
	}{
		{name: "無効なトークン", rule: evalRule, token: &ports.IntrospectionResult{Active: false}, reason: domain.ReasonTokenRejected},
		{name: "別の資源宛て", rule: evalRule, token: withToken(func(r *ports.IntrospectionResult) { r.Aud = []string{"https://other.example/"} }), reason: domain.ReasonTokenRejected},
		{name: "未登録の関係", rule: evalRule, token: evalAgentToken(), mutate: func(r *ResourceAccessRequest) { r.Relation = "owner" }, reason: domain.ReasonOperationNotRegistered},
		{name: "未登録の型", rule: evalRule, token: evalAgentToken(), mutate: func(r *ResourceAccessRequest) { r.ResourceType = "folder" }, reason: domain.ReasonOperationNotRegistered},
		{name: "詳細がない", rule: detailRule, token: evalAgentToken(), reason: domain.ReasonAuthorizationDetailMismatch},
		{name: "詳細の action が違う", rule: detailRule, token: withToken(func(r *ports.IntrospectionResult) {
			r.AuthorizationDetails = []spec.AuthorizationDetail{{Type: "document_access", Locations: []string{evalResource}, Actions: []string{"editor"}}}
		}), reason: domain.ReasonAuthorizationDetailMismatch},
		{name: "詳細の identifier が違う", rule: detailRule, token: withToken(func(r *ports.IntrospectionResult) {
			r.AuthorizationDetails = []spec.AuthorizationDetail{{Type: "document_access", Locations: []string{evalResource}, Actions: []string{"viewer"}, Identifier: "d2"}}
		}), reason: domain.ReasonAuthorizationDetailMismatch},
		{name: "Agent へ解決できない act", rule: evalRule, token: withToken(func(r *ports.IntrospectionResult) {
			r.PrincipalType, r.AgentID, r.Sub = "", "", "alice"
			r.Act = map[string]any{"sub": "some-client"}
		}), reason: domain.ReasonPrincipalUnresolved},
		{name: "mTLS の拇印がない", rule: evalRule, token: withToken(func(r *ports.IntrospectionResult) {
			r.SenderConstraint = &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: "thumb"}
		}), reason: domain.ReasonSenderProofInvalid},
		{name: "mTLS の拇印が違う", rule: evalRule, token: withToken(func(r *ports.IntrospectionResult) {
			r.SenderConstraint = &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: "thumb"}
		}), mutate: func(r *ResourceAccessRequest) { r.CertificateThumbprint = "other" }, reason: domain.ReasonSenderProofInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			relations := &evalRelations{decision: policyports.RelationDecision{Permitted: true}}
			in := evalRequest()
			if tc.mutate != nil {
				tc.mutate(&in)
			}
			decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
				Servers: evalServer(t, tc.rule), Tokens: evalTokens{result: tc.token}, Relations: relations,
			}, in, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if decision.Permitted || !slices.Equal(decision.Reasons, []string{tc.reason}) {
				t.Fatalf("decision=%+v, want reason %s", decision, tc.reason)
			}
			if decision.Scope != "" || decision.AuthorizationDetails != nil {
				t.Fatalf("不許可でクレームを返した: %+v", decision)
			}
			if len(relations.got) != 0 {
				t.Fatalf("関係の判定へ進んだ: %+v", relations.got)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-062: 主体と行為者をトークンから導き、規則のスコープを求めること
func TestEvaluateResourceAccessDerivesPrincipalsFromToken(t *testing.T) {
	token := evalAgentToken()
	token.PrincipalType, token.AgentID, token.Sub = "", "A1", "alice"
	token.AgentChain = []string{"A1", "A2"}
	token.Act = map[string]any{"sub": "agent-client"}
	token.AuthorizationDetails = []spec.AuthorizationDetail{{Type: "t"}}
	relations := &evalRelations{decision: policyports.RelationDecision{Permitted: true}}
	var events []spec.DomainEvent

	decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
		Servers: evalServer(t, evalRule), Tokens: evalTokens{result: token}, Relations: relations,
		Emit: func(event spec.DomainEvent) { events = append(events, event) },
	}, evalRequest(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Permitted || decision.DelegationMode != domain.DelegationModeOnBehalfOf || len(decision.AuthorizationDetails) != 1 {
		t.Fatalf("decision=%+v", decision)
	}
	got := relations.got[0]
	want := []policyports.PrincipalRef{{Type: "agent", ID: "A1"}, {Type: "agent", ID: "A2"}}
	if got.Subject != (policyports.PrincipalRef{Type: "user", ID: "alice"}) || !slices.Equal(got.Actors, want) {
		t.Fatalf("subject=%+v actors=%+v", got.Subject, got.Actors)
	}
	if got.TenantID != evalTenant || !slices.Equal(got.RequiredScopes, []string{"docs:read"}) || !slices.Equal(got.GrantedScopes, []string{"docs:read"}) {
		t.Fatalf("check=%+v", got)
	}
	event, ok := events[0].(*domain.ResourceAccessEvaluated)
	if !ok || !event.Permitted || event.ResourceServerID != "rs-1" || event.TokenID != "jti" {
		t.Fatalf("event=%+v", events)
	}
}

//spec:covers REQ-OAUTH2-062, REQ-AUTHORIZATION-005: 評価不能を許可へ倒さないこと
func TestEvaluateResourceAccessFailsClosed(t *testing.T) {
	boom := errors.New("down")
	server := evalServer(t, evalRule)
	cases := map[string]ResourceAccessDeps{
		"トークンの検証の失敗": {Servers: server, Tokens: evalTokens{err: boom}, Relations: &evalRelations{}},
		"関係のストアの失敗":  {Servers: server, Tokens: evalTokens{result: evalAgentToken()}, Relations: &evalRelations{err: boom}},
		"未配線":        {Servers: server, Tokens: evalTokens{result: evalAgentToken()}},
	}
	for name, deps := range cases {
		t.Run(name, func(t *testing.T) {
			decision, err := EvaluateResourceAccess(context.Background(), deps, evalRequest(), time.Now())
			if err == nil || decision.Permitted {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-063: htu の範囲を文字列の前方一致で判定しないこと
func TestHTUWithinResource(t *testing.T) {
	cases := []struct {
		htu, resource string
		want          bool
	}{
		{"https://mcp.example/tools/a", "https://mcp.example/", true},
		{"https://mcp.example/mcp/a", "https://mcp.example/mcp", true},
		{"https://mcp.example/mcp", "https://mcp.example/mcp", true},
		{"https://mcp.example/mcpx", "https://mcp.example/mcp", false},
		{"https://mcp.example.evil/a", "https://mcp.example", false},
		{"http://mcp.example/a", "https://mcp.example/", false},
		{"://bad", "https://mcp.example/", false},
	}
	for _, tc := range cases {
		if got := htuWithinResource(tc.htu, tc.resource); got != tc.want {
			t.Errorf("htuWithinResource(%q, %q)=%v", tc.htu, tc.resource, got)
		}
	}
}

func approvedRule() domain.ResourceAccessRule {
	rule := evalRule
	rule.AuthorizationDetailType = "payment"
	rule.RequiresApprovedOperation = true
	return rule
}

func approvedDetail() spec.AuthorizationDetail {
	return spec.AuthorizationDetail{
		Type: "payment", Locations: []string{evalResource}, Actions: []string{"viewer"}, Identifier: "d1",
		Fields: map[string]any{"amount": float64(100)},
	}
}

func approvedToken() *ports.IntrospectionResult {
	token := evalAgentToken()
	token.ApprovalRequestID = "approval-1"
	token.AuthorizationDetails = []spec.AuthorizationDetail{approvedDetail()}
	return token
}

func approvedStore(t *testing.T, now time.Time) *approvalmemory.ApprovalRequestStore {
	t.Helper()
	store := approvalmemory.NewApprovalRequestStore()
	if err := store.Save(evalTenantContext(), &approvaldomain.ApprovalRequest{
		ID: "approval-1", ClientID: "agent-client", UserID: "agent-client", AuthReqIDHash: "hash",
		State: spec.ApprovalConsumed, AuthorizationDetails: []spec.AuthorizationDetail{approvedDetail()},
		RequestedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	return store
}

func evalTenantContext() context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: evalTenant}, "", "")
}

func approvedRequest(key, parameters string) ResourceAccessRequest {
	in := evalRequest()
	in.OperationKey, in.OperationParameters = key, parameters
	return in
}

// 拒否は承認を消費しない。消費してしまうと、拒否された後の正しい操作を実行できなくなる。
//
//spec:covers REQ-OAUTH2-065: 不許可の要求が使用権を記録しないこと
func TestApprovedOperationDenialsDoNotClaim(t *testing.T) {
	now := time.Now().UTC()
	allow := policyports.RelationDecision{Permitted: true}
	cases := []struct {
		name     string
		token    *ports.IntrospectionResult
		relation policyports.RelationDecision
		in       ResourceAccessRequest
		reason   string
	}{
		{name: "引数の不一致", token: approvedToken(), relation: allow, in: approvedRequest("k", `{"amount":1}`), reason: domain.ReasonApprovedOperationMismatch},
		{name: "比較できない引数", token: approvedToken(), relation: allow, in: approvedRequest("k", `[]`), reason: domain.ReasonApprovedOperationMismatch},
		{name: "関係の不許可", token: approvedToken(), relation: policyports.RelationDecision{Reasons: []string{"denied"}}, in: approvedRequest("k", `{"amount":100}`), reason: "denied"},
		{name: "承認の識別子がない", token: func() *ports.IntrospectionResult {
			token := approvedToken()
			token.ApprovalRequestID = ""
			return token
		}(), relation: allow, in: approvedRequest("k", `{"amount":100}`), reason: domain.ReasonApprovalRequired},
		{name: "別のクライアントのトークン", token: func() *ports.IntrospectionResult {
			token := approvedToken()
			token.ClientID = "other"
			return token
		}(), relation: allow, in: approvedRequest("k", `{"amount":100}`), reason: domain.ReasonApprovalExpired},
		{name: "別の主体のトークン", token: func() *ports.IntrospectionResult {
			token := approvedToken()
			token.Sub = "other"
			return token
		}(), relation: allow, in: approvedRequest("k", `{"amount":100}`), reason: domain.ReasonApprovalExpired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := evalTenantContext()
			store := approvedStore(t, now)
			decision, err := EvaluateResourceAccess(ctx, ResourceAccessDeps{
				Servers: evalServer(t, approvedRule()), Tokens: evalTokens{result: tc.token},
				Relations: &evalRelations{decision: tc.relation}, Approvals: store,
			}, tc.in, now)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Permitted || !slices.Equal(decision.Reasons, []string{tc.reason}) {
				t.Fatalf("decision=%+v, want %s", decision, tc.reason)
			}
			rec, err := store.FindByID(ctx, "approval-1")
			if err != nil || rec.Use != nil {
				t.Fatalf("使用権を記録した: %+v %v", rec, err)
			}
		})
	}
}

// 別の鍵で並行しても許可は一つだけで、勝者の鍵の再送は許可のままである。
//
//spec:covers REQ-OAUTH2-065: 並行する別の鍵の要求で一方だけを許可すること
func TestApprovedOperationConcurrentKeysHaveOneWinner(t *testing.T) {
	now := time.Now().UTC()
	ctx := evalTenantContext()
	store := approvedStore(t, now)
	deps := ResourceAccessDeps{
		Servers: evalServer(t, approvedRule()), Tokens: evalTokens{result: approvedToken()},
		Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}}, Approvals: store,
	}
	// evalRelations は呼び出しを記録するので、並行の主題は比較交換に絞って一つのストアへ向ける。
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := []string{}
	for _, key := range []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8"} {
		wg.Go(func() {
			local := deps
			local.Relations = &evalRelations{decision: policyports.RelationDecision{Permitted: true}}
			decision, err := EvaluateResourceAccess(ctx, local, approvedRequest(key, `{"amount":100}`), now)
			if err != nil {
				t.Error(err)
				return
			}
			if decision.Permitted {
				mu.Lock()
				winners = append(winners, key)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if len(winners) != 1 {
		t.Fatalf("winners=%v, want exactly one", winners)
	}
	replay, err := EvaluateResourceAccess(ctx, deps, approvedRequest(winners[0], `{"amount":100}`), now)
	if err != nil || !replay.Permitted || replay.ApprovedOperation == nil || !replay.ApprovedOperation.Replayed {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}

//spec:covers REQ-OAUTH2-065: 監査が引数と鍵の平文を持たず、相関を持つこと
func TestApprovedOperationEventCarriesNoSecretValues(t *testing.T) {
	now := time.Now().UTC()
	var events []spec.DomainEvent
	in := approvedRequest("secret-operation-key", `{"amount":100}`)
	decision, err := EvaluateResourceAccess(evalTenantContext(), ResourceAccessDeps{
		Servers: evalServer(t, approvedRule()), Tokens: evalTokens{result: approvedToken()},
		Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}}, Approvals: approvedStore(t, now),
		Emit: func(event spec.DomainEvent) { events = append(events, event) },
	}, in, now)
	if err != nil || !decision.Permitted || len(events) != 1 {
		t.Fatalf("decision=%+v err=%v events=%v", decision, err, events)
	}
	raw, err := json.Marshal(events[0])
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(raw)
	if strings.Contains(encoded, "secret-operation-key") || strings.Contains(encoded, `"amount"`) {
		t.Fatalf("event leaks arguments: %s", encoded)
	}
	if !strings.Contains(encoded, `"approvalRequestId":"approval-1"`) || !strings.Contains(encoded, decision.ApprovedOperation.OperationDigest) {
		t.Fatalf("event lacks correlation: %s", encoded)
	}
}

type failingClaimStore struct {
	*approvalmemory.ApprovalRequestStore
	err error
}

func (s failingClaimStore) ClaimUse(context.Context, string, approvaldomain.ApprovalUse, time.Time) (*approvaldomain.ApprovalRequest, bool, error) {
	return nil, false, s.err
}

//spec:covers REQ-OAUTH2-065: 使用権の記録の失敗と鍵の欠落と未配線を許可へ倒さないこと
func TestApprovedOperationFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	ctx := evalTenantContext()
	base := ResourceAccessDeps{
		Servers: evalServer(t, approvedRule()), Tokens: evalTokens{result: approvedToken()},
		Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}},
	}
	unwired := base
	if decision, err := EvaluateResourceAccess(ctx, unwired, approvedRequest("k", `{"amount":100}`), now); !errors.Is(err, ErrResourceAccessNotConfigured) || decision.Permitted {
		t.Fatalf("未配線 decision=%+v err=%v", decision, err)
	}
	boom := errors.New("down")
	failing := base
	failing.Approvals = failingClaimStore{ApprovalRequestStore: approvedStore(t, now), err: boom}
	if decision, err := EvaluateResourceAccess(ctx, failing, approvedRequest("k", `{"amount":100}`), now); !errors.Is(err, boom) || decision.Permitted {
		t.Fatalf("記録の失敗 decision=%+v err=%v", decision, err)
	}
	if decision, err := EvaluateResourceAccess(ctx, failing, approvedRequest("", `{"amount":100}`), now); !errors.Is(err, ErrOperationKeyRequired) || decision.Permitted {
		t.Fatalf("鍵の欠落 decision=%+v err=%v", decision, err)
	}
}

//spec:covers REQ-OAUTH2-062: 利用者の主体と代行の Agent、Agent の主体の記録の形
func TestEvaluationAuditNamesThePrincipals(t *testing.T) {
	agent := func(id string) policyports.PrincipalRef {
		return policyports.PrincipalRef{Type: domain.PrincipalTypeAgent, ID: id}
	}
	delegated := newEvaluationAudit("j", policyports.PrincipalRef{Type: "user", ID: "alice"}, []policyports.PrincipalRef{agent("A1"), agent("A2")})
	if delegated.userID != "alice" || delegated.agentID != "A1" || len(delegated.actorChain) != 2 {
		t.Fatalf("delegated = %+v", delegated)
	}
	autonomous := newEvaluationAudit("j", agent("A1"), nil)
	if autonomous.userID != "" || autonomous.agentID != "A1" || autonomous.actorChain != nil {
		t.Fatalf("autonomous = %+v", autonomous)
	}
}

//spec:covers REQ-OAUTH2-062: 条件は関係の判定の後に評価し、監査に版と結果と申告のダイジェストを残して平文を残さないこと
func TestEvaluateResourceAccessCondition(t *testing.T) {
	condition := &domain.ResourceAccessCondition{DenyWhen: `attributes.tool == "delete"`}
	rule := evalRule
	rule.Condition = condition
	const secretValue = "delete"
	run := func(t *testing.T, relations *evalRelations, token *ports.IntrospectionResult, attributes string) (ResourceAccessDecision, *domain.ResourceAccessEvaluated) {
		t.Helper()
		var event *domain.ResourceAccessEvaluated
		in := evalRequest()
		in.Attributes = attributes
		decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
			Servers: evalServer(t, rule), Tokens: evalTokens{result: token}, Relations: relations,
			Emit: func(recorded spec.DomainEvent) { event, _ = recorded.(*domain.ResourceAccessEvaluated) },
		}, in, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		return decision, event
	}
	permit := func() *evalRelations {
		return &evalRelations{decision: policyports.RelationDecision{Permitted: true}}
	}

	decision, event := run(t, permit(), evalAgentToken(), `{"tool":"`+secretValue+`"}`)
	if decision.Permitted || !slices.Equal(decision.Reasons, []string{domain.ReasonConditionDenied}) {
		t.Fatalf("拒否: %+v", decision)
	}
	if event.ConditionVersion != condition.Version() || event.ConditionOutcome != "deny" || len(event.DeclaredAttributesDigest) != 64 {
		t.Fatalf("監査: %+v", event)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secretValue) {
		t.Fatalf("監査に申告の平文がある: %s", raw)
	}

	decision, event = run(t, permit(), evalAgentToken(), `{`)
	if decision.Permitted || !slices.Equal(decision.Reasons, []string{domain.ReasonConditionIndeterminate}) || event.ConditionOutcome != "indeterminate" || len(event.DeclaredAttributesDigest) != 64 {
		t.Fatalf("読めない申告: %+v %+v", decision, event)
	}

	decision, event = run(t, permit(), evalAgentToken(), `{"tool":"read"}`)
	if !decision.Permitted || event.ConditionOutcome != "allow" {
		t.Fatalf("許可: %+v %+v", decision, event)
	}

	// 無効なトークンと関係の不許可では条件を評価しないので、申告で条件を探らせない。
	if decision, event = run(t, permit(), &ports.IntrospectionResult{Active: false}, `{"tool":"delete"}`); !slices.Equal(decision.Reasons, []string{domain.ReasonTokenRejected}) || event.ConditionOutcome != "" {
		t.Fatalf("無効なトークン: %+v %+v", decision, event)
	}
	denied := &evalRelations{decision: policyports.RelationDecision{Reasons: []string{"no_relation"}}}
	if decision, event = run(t, denied, evalAgentToken(), `{"tool":"delete"}`); !slices.Equal(decision.Reasons, []string{"no_relation"}) || event.ConditionVersion != "" || event.DeclaredAttributesDigest != "" {
		t.Fatalf("関係の不許可: %+v %+v", decision, event)
	}
}

//spec:covers REQ-OAUTH2-062: 条件のない規則は申告を読まず、読めない申告でも結果を変えないこと
func TestEvaluateResourceAccessIgnoresAttributesWithoutCondition(t *testing.T) {
	in := evalRequest()
	in.Attributes = `{`
	var event *domain.ResourceAccessEvaluated
	decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
		Servers: evalServer(t, evalRule), Tokens: evalTokens{result: evalAgentToken()},
		Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}},
		Emit:      func(recorded spec.DomainEvent) { event, _ = recorded.(*domain.ResourceAccessEvaluated) },
	}, in, time.Now().UTC())
	if err != nil || !decision.Permitted || event.ConditionVersion != "" || event.DeclaredAttributesDigest != "" {
		t.Fatalf("err=%v decision=%+v event=%+v", err, decision, event)
	}
}
