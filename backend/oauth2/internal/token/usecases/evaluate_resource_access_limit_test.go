package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	ratelimitmemory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var limitNow = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

// recordingLimiter は数えた上限の窓を順に記録し、計数の有無と順序を観測する。
type recordingLimiter struct {
	*ratelimitmemory.RateLimiter
	mu      sync.Mutex
	windows []int
	err     error
}

func newRecordingLimiter() *recordingLimiter {
	return &recordingLimiter{RateLimiter: ratelimitmemory.NewRateLimiter(nil)}
}

func (l *recordingLimiter) AllowWithin(ctx context.Context, policyID, key string, config rlports.RateLimitPolicyConfig, now time.Time) (rlports.RateLimitResult, error) {
	l.mu.Lock()
	l.windows = append(l.windows, config.WindowSeconds)
	l.mu.Unlock()
	if l.err != nil {
		return rlports.RateLimitResult{}, l.err
	}
	return l.RateLimiter.AllowWithin(ctx, policyID, key, config, now)
}

func limitedRule(limits ...domain.ResourceAccessLimit) domain.ResourceAccessRule {
	rule := evalRule
	rule.Limits = limits
	return rule
}

func limitAgentToken(agentID string) *ports.IntrospectionResult {
	token := evalAgentToken()
	token.AgentID = agentID
	return token
}

func evaluateLimited(
	t *testing.T, limiter rlports.RateLimiter, rule domain.ResourceAccessRule, token *ports.IntrospectionResult, in ResourceAccessRequest, now time.Time,
) (ResourceAccessDecision, *domain.ResourceAccessEvaluated) {
	t.Helper()
	var event *domain.ResourceAccessEvaluated
	decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
		Servers: evalServer(t, rule), Tokens: evalTokens{result: token}, Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}},
		Limiter: limiter,
		Emit:    func(recorded spec.DomainEvent) { event, _ = recorded.(*domain.ResourceAccessEvaluated) },
	}, in, now)
	if err != nil {
		t.Fatal(err)
	}
	return decision, event
}

//spec:covers REQ-OAUTH2-062: 上限の値、その次、窓の後の境界と、単位ごとに別の回数を持つこと
func TestResourceAccessLimitBoundaryAndUnits(t *testing.T) {
	agentLimit := domain.ResourceAccessLimit{By: []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByAgent}, MaxRequests: 2, WindowSeconds: 60}
	rule := limitedRule(agentLimit)
	limiter := newRecordingLimiter()
	a1, a2 := limitAgentToken("a1"), limitAgentToken("a2")
	for i := range 2 {
		if decision, _ := evaluateLimited(t, limiter, rule, a1, evalRequest(), limitNow); !decision.Permitted {
			t.Fatalf("%d 回目: %+v", i+1, decision)
		}
	}
	decision, event := evaluateLimited(t, limiter, rule, a1, evalRequest(), limitNow.Add(59*time.Second))
	if decision.Permitted || !slices.Equal(decision.Reasons, []string{domain.ReasonLimitExceeded}) || decision.Scope != "" {
		t.Fatalf("超過: %+v", decision)
	}
	if !slices.Equal(event.ExceededLimitBy, agentLimit.By) || event.ExceededLimitWindowSeconds != 60 || event.AgentID != "a1" {
		t.Fatalf("超過の監査: %+v", event)
	}
	if decision, _ := evaluateLimited(t, limiter, rule, a2, evalRequest(), limitNow); !decision.Permitted {
		t.Fatalf("別の Agent は別の回数を持つ: %+v", decision)
	}
	if decision, _ := evaluateLimited(t, limiter, rule, a1, evalRequest(), limitNow.Add(60*time.Second)); !decision.Permitted {
		t.Fatalf("窓の後: %+v", decision)
	}

	perResource := limitedRule(domain.ResourceAccessLimit{By: []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByResource}, MaxRequests: 1, WindowSeconds: 60})
	limiter = newRecordingLimiter()
	first, second := evalRequest(), evalRequest()
	second.ResourceID = "d2"
	if decision, _ := evaluateLimited(t, limiter, perResource, a1, first, limitNow); !decision.Permitted {
		t.Fatalf("資源 1: %+v", decision)
	}
	if decision, _ := evaluateLimited(t, limiter, perResource, a1, second, limitNow); !decision.Permitted {
		t.Fatalf("資源 2 は別の回数を持つ: %+v", decision)
	}

	whole := limitedRule(domain.ResourceAccessLimit{MaxRequests: 1, WindowSeconds: 60})
	limiter = newRecordingLimiter()
	if decision, _ := evaluateLimited(t, limiter, whole, a1, first, limitNow); !decision.Permitted {
		t.Fatalf("全体 1 回目: %+v", decision)
	}
	if decision, _ := evaluateLimited(t, limiter, whole, a2, second, limitNow); decision.Permitted {
		t.Fatalf("by を省いた上限は Agent と資源をまたいで共有する: %+v", decision)
	}
}

//spec:covers REQ-OAUTH2-062: 単位の値がない評価にその上限を適用しないこと
func TestResourceAccessLimitSkipsUnitWithoutValue(t *testing.T) {
	userLimit := domain.ResourceAccessLimit{By: []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByUser}, MaxRequests: 1, WindowSeconds: 60}
	limiter := newRecordingLimiter()
	for range 3 {
		if decision, _ := evaluateLimited(t, limiter, limitedRule(userLimit), limitAgentToken("a1"), evalRequest(), limitNow); !decision.Permitted {
			t.Fatalf("主体が Agent のトークンに user の上限を適用した: %+v", decision)
		}
	}
	if len(limiter.windows) != 0 {
		t.Fatalf("適用しない上限を数えた: %v", limiter.windows)
	}
}

// 他の理由の不許可の後に同じ Agent の予算が残っていれば、計数が論理積の最後にあると言える。
//
//spec:covers REQ-OAUTH2-062: ほかの理由で不許可の評価は上限の回数を数えないこと
func TestResourceAccessLimitCountsOnlyOtherwisePermitted(t *testing.T) {
	limits := []domain.ResourceAccessLimit{{By: []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByAgent}, MaxRequests: 1, WindowSeconds: 60}}
	withDetail := limitedRule(limits...)
	withDetail.AuthorizationDetailType = "payment"
	denied := limitedRule(limits...)
	denied.Condition = &domain.ResourceAccessCondition{DenyWhen: "true"}
	limiter := newRecordingLimiter()
	inactive := limitAgentToken("a1")
	inactive.Active = false
	for name, attempt := range map[string]struct {
		rule      domain.ResourceAccessRule
		token     *ports.IntrospectionResult
		relations policyports.RelationDecision
		reason    string
	}{
		"トークン": {limitedRule(limits...), inactive, policyports.RelationDecision{Permitted: true}, domain.ReasonTokenRejected},
		"明細":   {withDetail, limitAgentToken("a1"), policyports.RelationDecision{Permitted: true}, domain.ReasonAuthorizationDetailMismatch},
		"関係":   {limitedRule(limits...), limitAgentToken("a1"), policyports.RelationDecision{Reasons: []string{"no_relation"}}, "no_relation"},
		"条件":   {denied, limitAgentToken("a1"), policyports.RelationDecision{Permitted: true}, domain.ReasonConditionDenied},
	} {
		decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
			Servers: evalServer(t, attempt.rule), Tokens: evalTokens{result: attempt.token}, Relations: &evalRelations{decision: attempt.relations}, Limiter: limiter,
		}, evalRequest(), limitNow)
		if err != nil || decision.Permitted || !slices.Equal(decision.Reasons, []string{attempt.reason}) {
			t.Fatalf("%s: %+v %v", name, decision, err)
		}
	}
	if len(limiter.windows) != 0 {
		t.Fatalf("不許可の評価を数えた: %v", limiter.windows)
	}
	if decision, _ := evaluateLimited(t, limiter, limitedRule(limits...), limitAgentToken("a1"), evalRequest(), limitNow); !decision.Permitted {
		t.Fatalf("予算が残っていない: %+v", decision)
	}
}

//spec:covers REQ-OAUTH2-062: 宣言の順序によらず狭い単位と短い窓から数え、超えた上限で止めること
func TestResourceAccessLimitOrderProtectsSharedBudget(t *testing.T) {
	byAgent := []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByAgent}
	rule := limitedRule(
		domain.ResourceAccessLimit{MaxRequests: 100, WindowSeconds: 30},
		domain.ResourceAccessLimit{By: byAgent, MaxRequests: 100, WindowSeconds: 3600},
		domain.ResourceAccessLimit{By: byAgent, MaxRequests: 1, WindowSeconds: 10},
	)
	limiter := newRecordingLimiter()
	if decision, _ := evaluateLimited(t, limiter, rule, limitAgentToken("a1"), evalRequest(), limitNow); !decision.Permitted {
		t.Fatalf("許可: %+v", decision)
	}
	if want := []int{10, 3600, 30}; !slices.Equal(limiter.windows, want) {
		t.Fatalf("順序: got %v want %v", limiter.windows, want)
	}
	limiter.windows = nil
	decision, event := evaluateLimited(t, limiter, rule, limitAgentToken("a1"), evalRequest(), limitNow)
	if decision.Permitted || event.ExceededLimitWindowSeconds != 10 || !slices.Equal(limiter.windows, []int{10}) {
		t.Fatalf("超えた上限で止まらない: %+v %+v %v", decision, event, limiter.windows)
	}
}

//spec:covers REQ-OAUTH2-062: 上限の回数を読み書きできないか未配線の評価を許可しないこと
func TestResourceAccessLimitFailsClosed(t *testing.T) {
	rule := limitedRule(domain.ResourceAccessLimit{MaxRequests: 5, WindowSeconds: 60})
	broken := newRecordingLimiter()
	broken.err = errors.New("counter unavailable")
	for name, limiter := range map[string]rlports.RateLimiter{"障害": broken, "未配線": nil} {
		decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
			Servers: evalServer(t, rule), Tokens: evalTokens{result: limitAgentToken("a1")},
			Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}}, Limiter: limiter,
		}, evalRequest(), limitNow)
		if err == nil || decision.Permitted {
			t.Fatalf("%s: %+v %v", name, decision, err)
		}
	}
	if decision, _ := evaluateLimited(t, broken, evalRule, limitAgentToken("a1"), evalRequest(), limitNow); !decision.Permitted {
		t.Fatalf("上限のない規則が計数を読んだ: %+v", decision)
	}
}

//spec:covers REQ-OAUTH2-062: 並行した評価の許可の合計が上限と等しいこと
func TestResourceAccessLimitConcurrentEvaluations(t *testing.T) {
	const limit, attempts = 7, 64
	rule := limitedRule(domain.ResourceAccessLimit{By: []domain.ResourceAccessLimitUnit{domain.ResourceAccessLimitByAgent}, MaxRequests: limit, WindowSeconds: 60})
	servers := evalServer(t, rule)
	limiter := newRecordingLimiter()
	permitted := make(chan bool, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Go(func() {
			decision, err := EvaluateResourceAccess(context.Background(), ResourceAccessDeps{
				Servers: servers, Tokens: evalTokens{result: limitAgentToken("a1")},
				Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}}, Limiter: limiter,
			}, evalRequest(), limitNow)
			if err != nil {
				t.Error(err)
			}
			permitted <- decision.Permitted
		})
	}
	wg.Wait()
	close(permitted)
	count := 0
	for ok := range permitted {
		if ok {
			count++
		}
	}
	if count != limit {
		t.Fatalf("許可 %d, 上限 %d", count, limit)
	}
}

// 計数を使用権の前に置くので、超過で承認が失われない。照合を計数の前に置くので、不一致は予算を減らさない。
//
//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-065: 超過で使用権を記録せず、承認の不一致で回数を数えないこと
func TestResourceAccessLimitKeepsApprovalUnclaimed(t *testing.T) {
	rule := approvedRule()
	rule.Limits = []domain.ResourceAccessLimit{{MaxRequests: 1, WindowSeconds: 10}}
	now := time.Now().UTC()
	store := approvedStore(t, now)
	limiter := newRecordingLimiter()
	evaluate := func(in ResourceAccessRequest, at time.Time) ResourceAccessDecision {
		t.Helper()
		decision, err := EvaluateResourceAccess(evalTenantContext(), ResourceAccessDeps{
			Servers: evalServer(t, rule), Tokens: evalTokens{result: approvedToken()},
			Relations: &evalRelations{decision: policyports.RelationDecision{Permitted: true}},
			Approvals: store, Limiter: limiter,
		}, in, at)
		if err != nil {
			t.Fatal(err)
		}
		return decision
	}
	matching := approvedRequest("k1", `{"amount":100}`)
	if d := evaluate(approvedRequest("k1", `{"amount":999}`), now); !slices.Equal(d.Reasons, []string{domain.ReasonApprovedOperationMismatch}) {
		t.Fatalf("不一致: %+v", d)
	}
	if len(limiter.windows) != 0 {
		t.Fatalf("不一致を数えた: %v", limiter.windows)
	}

	// 別の操作で予算を使い切ってから、承認した操作を評価する。
	if _, err := limiter.AllowWithin(evalTenantContext(), resourceAccessLimitPolicyID, limitKeyForTest(t, rule), rlports.RateLimitPolicyConfig{MaxRequests: 1, WindowSeconds: 10}, now); err != nil {
		t.Fatal(err)
	}
	if d := evaluate(matching, now); !slices.Equal(d.Reasons, []string{domain.ReasonLimitExceeded}) {
		t.Fatalf("超過: %+v", d)
	}
	stored, err := store.FindByID(evalTenantContext(), approvedToken().ApprovalRequestID)
	if err != nil || stored == nil || stored.Use != nil {
		t.Fatalf("超過が使用権を記録した: %+v %v", stored, err)
	}

	// 窓の後に、別の鍵でも同じ承認を使える。
	later := evaluate(approvedRequest("k2", `{"amount":100}`), now.Add(11*time.Second))
	if !later.Permitted || later.ApprovedOperation == nil || later.ApprovedOperation.Replayed {
		t.Fatalf("窓の後: %+v", later)
	}
	// 記録済みの別の鍵は窓の予算を減らさずに拒否する。
	limiter.windows = nil
	if d := evaluate(approvedRequest("k3", `{"amount":100}`), now.Add(11*time.Second)); !slices.Equal(d.Reasons, []string{domain.ReasonApprovalAlreadyUsed}) || len(limiter.windows) != 0 {
		t.Fatalf("記録済みの別の鍵: %+v %v", d, limiter.windows)
	}
}

// 監査と応答に計数の鍵の材料を増やさない。
//
//spec:covers REQ-OAUTH2-062: 超過の不許可が上限の値を応答へ語らないこと
func TestResourceAccessLimitDecisionCarriesNoLimitValues(t *testing.T) {
	rule := limitedRule(domain.ResourceAccessLimit{MaxRequests: 1, WindowSeconds: 77})
	limiter := newRecordingLimiter()
	evaluateLimited(t, limiter, rule, limitAgentToken("a1"), evalRequest(), limitNow)
	decision, _ := evaluateLimited(t, limiter, rule, limitAgentToken("a1"), evalRequest(), limitNow)
	if strings.Contains(strings.Join(decision.Reasons, " "), "77") || decision.AuthorizationDetails != nil || decision.DelegationMode != "" {
		t.Fatalf("超過の結果: %+v", decision)
	}
}

func limitKeyForTest(t *testing.T, rule domain.ResourceAccessRule) string {
	t.Helper()
	servers := evalServer(t, rule)
	server, err := servers.FindByClientID(evalTenantContext(), evalTenant, evalClient)
	if err != nil || server == nil {
		t.Fatalf("server: %v %v", server, err)
	}
	key, ok := resourceAccessLimitKey(server, rule, rule.Limits[0], domain.ResourceAccessLimitSubject{}, evalTenant)
	if !ok {
		t.Fatal("key")
	}
	return key
}
