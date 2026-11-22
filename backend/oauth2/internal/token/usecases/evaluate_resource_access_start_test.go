package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/internal/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const startApprovalURL = "https://idp.example/realms/r1/account/approvals"

type recordedNotifications struct {
	sent []notificationports.Notification
}

func (r *recordedNotifications) Notify(_ context.Context, notification notificationports.Notification) bool {
	r.sent = append(r.sent, notification)
	return true
}

type startRecipients map[string]*userdomain.User

func (r startRecipients) FindBySub(_ context.Context, sub string) (*userdomain.User, error) {
	return r[sub], nil
}

// startFixture は、汎用の MCP クライアントの利用者のトークンで、承認を求める規則の操作を評価する。
type startFixture struct {
	t        *testing.T
	now      time.Time
	store    *approvalmemory.ApprovalRequestStore
	notified *recordedNotifications
	relation accessports.RelationDecision
	rule     oauthdomain.ResourceAccessRule
	servers  oauthports.McpResourceServerRepository
	events   []spec.DomainEvent
}

func newStartFixture(t *testing.T) *startFixture {
	t.Helper()
	f := &startFixture{
		t: t, now: time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC), store: approvalmemory.NewApprovalRequestStore(),
		notified: &recordedNotifications{}, relation: accessports.RelationDecision{Permitted: true}, rule: approvedRule(),
	}
	f.servers = evalServer(t, f.rule)
	return f
}

func startUserToken() *oauthports.IntrospectionResult {
	return &oauthports.IntrospectionResult{
		Active: true, JTI: "jti-user", Sub: "alice", ClientID: "generic-client", Scope: "docs:read", Aud: []string{evalResource},
	}
}

func startRequest(key, parameters string) ResourceAccessRequest {
	in := approvedRequest(key, parameters)
	in.ApprovalStart = true
	return in
}

func (f *startFixture) deps(token *oauthports.IntrospectionResult) ResourceAccessDeps {
	email := "alice@example.com"
	return ResourceAccessDeps{
		Servers: f.servers, Tokens: evalTokens{result: token}, Relations: &evalRelations{decision: f.relation},
		Approvals: f.store, Notifier: f.notified, ApprovalURL: startApprovalURL,
		Recipients: startRecipients{"alice": {ID: "alice", TenantID: evalTenant, PreferredUsername: "alice", Email: &email}},
		Emit:       func(event spec.DomainEvent) { f.events = append(f.events, event) },
	}
}

func (f *startFixture) evaluate(token *oauthports.IntrospectionResult, in ResourceAccessRequest) ResourceAccessDecision {
	f.t.Helper()
	decision, err := EvaluateResourceAccess(evalTenantContext(), f.deps(token), in, f.now)
	if err != nil {
		f.t.Fatalf("evaluate: %v", err)
	}
	return decision
}

func (f *startFixture) stored(id string) *approvaldomain.ApprovalRequest {
	f.t.Helper()
	rec, err := f.store.FindByID(evalTenantContext(), id)
	if err != nil || rec == nil {
		f.t.Fatalf("approval %s: %v %v", id, rec, err)
	}
	return rec
}

func (f *startFixture) decide(id string, event spec.ApprovalRequestEvent) {
	f.t.Helper()
	if decided, err := f.store.Decide(evalTenantContext(), id, "alice", event, f.now); err != nil || decided == nil {
		f.t.Fatalf("decide: %v %v", decided, err)
	}
}

func requireReason(t *testing.T, decision ResourceAccessDecision, reason string) {
	t.Helper()
	if decision.Permitted || !slices.Equal(decision.Reasons, []string{reason}) {
		t.Fatalf("decision=%+v, want the reason %s", decision, reason)
	}
}

// 汎用のクライアントのトークンは承認を持たない。評価が起票し、本人の承認の後に、その識別子の参照で
// 一回だけ許可する。トークンは最後まで同じである。
//
//spec:covers REQ-OAUTH2-093: 起票の内容と通知、保留中の再起票の抑止、承認後の参照の一回の許可と Consumed、別の鍵の拒否と同じ鍵の再送
func TestEvaluationStartsAnApprovalAndPermitsItsReferenceOnce(t *testing.T) {
	f := newStartFixture(t)
	token := startUserToken()
	parameters := `{"amount":100}`

	started := f.evaluate(token, startRequest("k0", parameters))
	requireReason(t, started, oauthdomain.ReasonApprovalPending)
	approval := started.StartedApproval
	if approval == nil || approval.Status != "pending" || approval.URI != startApprovalURL+"?request="+approval.ID ||
		!approval.ExpiresAt.Equal(f.now.Add(600*time.Second)) || started.Remediation != nil {
		t.Fatalf("started approval=%+v remediation=%v", approval, started.Remediation)
	}
	rec := f.stored(approval.ID)
	wantDetail := spec.AuthorizationDetail{
		Type: "payment", Locations: []string{evalResource}, Actions: []string{"viewer"}, Identifier: "d1",
		Fields: map[string]any{"amount": float64(100)},
	}
	if rec.State != spec.ApprovalPending || rec.UserID != "alice" || rec.ClientID != "generic-client" || rec.AgentID != nil ||
		!rec.InitiatedByResourceServer() || rec.ResourceServerID == nil || *rec.ResourceServerID != "rs-1" ||
		len(rec.Scopes) != 0 || rec.BindingMessage != nil || rec.AuthReqIDHash != "" ||
		len(rec.AuthorizationDetails) != 1 || !equalDetail(rec.AuthorizationDetails[0], wantDetail) {
		t.Fatalf("stored approval=%+v", rec)
	}
	if len(f.notified.sent) != 1 || f.notified.sent[0].To != "alice@example.com" ||
		f.notified.sent[0].Key != notificationports.TemplateKeyAgentActionApprovalRequest ||
		f.notified.sent[0].Vars["approval_url"] != approval.URI {
		t.Fatalf("notifications=%+v", f.notified.sent)
	}
	evaluated, _ := f.events[len(f.events)-1].(*oauthdomain.ResourceAccessEvaluated)
	if evaluated == nil || evaluated.ApprovalRequestID != approval.ID || evaluated.Permitted {
		t.Fatalf("event=%+v", f.events[len(f.events)-1])
	}

	// 保留中の同じ操作は起票し直さず、通知も送らない。
	again := f.evaluate(token, startRequest("k1", `{ "amount": 100 }`))
	requireReason(t, again, oauthdomain.ReasonApprovalPending)
	if again.StartedApproval == nil || again.StartedApproval.ID != approval.ID || len(f.notified.sent) != 1 {
		t.Fatalf("repeated start: %+v notifications=%d", again.StartedApproval, len(f.notified.sent))
	}
	reference := func(key string) ResourceAccessRequest {
		in := startRequest(key, parameters)
		in.ApprovalRequestID = approval.ID
		return in
	}
	requireReason(t, f.evaluate(token, reference("k2")), oauthdomain.ReasonApprovalPending)

	f.decide(approval.ID, spec.ApprovalEventApprove)
	permitted := f.evaluate(token, reference("k3"))
	if !permitted.Permitted || permitted.ApprovedOperation == nil || permitted.ApprovedOperation.ApprovalRequestID != approval.ID ||
		permitted.ApprovedOperation.Replayed || permitted.StartedApproval != nil {
		t.Fatalf("approved reference: %+v", permitted)
	}
	if consumed := f.stored(approval.ID); consumed.State != spec.ApprovalConsumed || consumed.Use == nil {
		t.Fatalf("after the use: %+v", consumed)
	}
	requireReason(t, f.evaluate(token, reference("k4")), oauthdomain.ReasonApprovalAlreadyUsed)
	if replay := f.evaluate(token, reference("k3")); !replay.Permitted || !replay.ApprovedOperation.Replayed {
		t.Fatalf("same key: %+v", replay)
	}
	if len(f.notified.sent) != 1 {
		t.Fatalf("notifications=%d", len(f.notified.sent))
	}
}

func equalDetail(a, b spec.AuthorizationDetail) bool {
	return a.Type == b.Type && slices.Equal(a.Locations, b.Locations) && slices.Equal(a.Actions, b.Actions) &&
		a.Identifier == b.Identifier && len(a.Fields) == len(b.Fields) && a.Fields["amount"] == b.Fields["amount"]
}

// 参照できるのは、同じ登録が同じ利用者、クライアント、Agent の評価で起票した承認だけである。
// 使えない参照は残っていない承認として扱い、起票を求めていれば新たに起票する。
//
//spec:covers REQ-OAUTH2-093: 他人、別のクライアント、Agent、登録、CIBA の承認、拒否、期限切れ、引数の変更の参照の扱い
func TestApprovalReferencesOnlyTheirOwnStartedApproval(t *testing.T) {
	parameters := `{"amount":100}`
	approved := func(t *testing.T, mutate func(*approvaldomain.ApprovalRequest)) (*startFixture, string) {
		t.Helper()
		f := newStartFixture(t)
		started := f.evaluate(startUserToken(), startRequest("k0", parameters))
		id := started.StartedApproval.ID
		f.decide(id, spec.ApprovalEventApprove)
		if mutate != nil {
			rec := f.stored(id)
			mutate(rec)
			if err := f.store.Save(evalTenantContext(), rec); err != nil {
				t.Fatal(err)
			}
		}
		return f, id
	}
	cases := []struct {
		name   string
		mutate func(*approvaldomain.ApprovalRequest)
		token  func(*oauthports.IntrospectionResult)
		in     func(*ResourceAccessRequest)
		reason string
		// restarts は、起票を求めた参照が新しい承認リクエストを起票することを表す。
		restarts bool
	}{
		{name: "他人のトークン", token: func(r *oauthports.IntrospectionResult) { r.Sub = "mallory" }, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "別のクライアント", token: func(r *oauthports.IntrospectionResult) { r.ClientID = "other-client" }, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "別の Agent", token: func(r *oauthports.IntrospectionResult) { r.AgentID = "A9" }, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "別の登録の起票", mutate: func(r *approvaldomain.ApprovalRequest) { other := "rs-2"; r.ResourceServerID = &other }, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "CIBA の承認", mutate: func(r *approvaldomain.ApprovalRequest) {
			r.Initiator, r.ResourceServerID, r.OperationDigest, r.AuthReqIDHash, r.IntervalSeconds = "", nil, nil, "hash", 5
		}, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "期限切れで起票を求めない", mutate: func(r *approvaldomain.ApprovalRequest) { r.ExpiresAt = r.RequestedAt }, in: func(r *ResourceAccessRequest) { r.ApprovalStart = false }, reason: oauthdomain.ReasonApprovalExpired},
		{name: "期限切れで起票を求める", mutate: func(r *approvaldomain.ApprovalRequest) { r.ExpiresAt = r.RequestedAt }, reason: oauthdomain.ReasonApprovalPending, restarts: true},
		{name: "引数の変更", in: func(r *ResourceAccessRequest) { r.OperationParameters = `{"amount":101}` }, reason: oauthdomain.ReasonApprovedOperationMismatch},
		{name: "対象の変更", in: func(r *ResourceAccessRequest) { r.ResourceID = "d2" }, reason: oauthdomain.ReasonApprovedOperationMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, id := approved(t, tc.mutate)
			token := startUserToken()
			if tc.token != nil {
				tc.token(token)
			}
			in := startRequest("k1", parameters)
			in.ApprovalRequestID = id
			if tc.in != nil {
				tc.in(&in)
			}
			decision := f.evaluate(token, in)
			requireReason(t, decision, tc.reason)
			if rec := f.stored(id); rec.Use != nil {
				t.Fatalf("the referenced approval was used: %+v", rec)
			}
			restarted := decision.StartedApproval != nil && decision.StartedApproval.ID != id
			if restarted != tc.restarts {
				t.Fatalf("started=%+v, want restart=%v", decision.StartedApproval, tc.restarts)
			}
		})
	}

	t.Run("拒否した承認", func(t *testing.T) {
		f := newStartFixture(t)
		id := f.evaluate(startUserToken(), startRequest("k0", parameters)).StartedApproval.ID
		f.decide(id, spec.ApprovalEventDeny)
		in := startRequest("k1", parameters)
		in.ApprovalRequestID = id
		decision := f.evaluate(startUserToken(), in)
		requireReason(t, decision, oauthdomain.ReasonApprovalDenied)
		if decision.StartedApproval != nil || len(f.notified.sent) != 1 {
			t.Fatalf("denied approval restarted: %+v notifications=%d", decision.StartedApproval, len(f.notified.sent))
		}
	})
}

// 起票は承認だけが足りない評価に限る。権限のない呼び出し元と Agent の主体には本人へのメールを送らせない。
//
//spec:covers REQ-OAUTH2-093: 関係の不許可、主体が Agent、承認が不要な操作、上限で起票と通知をしないこと
func TestApprovalStartIsLimitedToMissingApprovals(t *testing.T) {
	parameters := `{"amount":100}`
	t.Run("関係の不許可", func(t *testing.T) {
		f := newStartFixture(t)
		f.relation = accessports.RelationDecision{Permitted: false, Reasons: []string{"relation_denied"}}
		decision := f.evaluate(startUserToken(), startRequest("k", parameters))
		if decision.Permitted || decision.StartedApproval != nil || len(f.notified.sent) != 0 {
			t.Fatalf("decision=%+v notifications=%d", decision, len(f.notified.sent))
		}
	})
	t.Run("主体が Agent", func(t *testing.T) {
		f := newStartFixture(t)
		decision := f.evaluate(evalAgentToken(), startRequest("k", parameters))
		requireReason(t, decision, oauthdomain.ReasonApprovalRequired)
		if decision.StartedApproval != nil || decision.Remediation == nil || len(f.notified.sent) != 0 {
			t.Fatalf("decision=%+v notifications=%d", decision, len(f.notified.sent))
		}
	})
	t.Run("承認が不要な操作", func(t *testing.T) {
		f := newStartFixture(t)
		f.servers = evalServer(t, evalRule)
		in := startRequest("k", parameters)
		in.ApprovalRequestID = "0b6f2c8e-6a59-4f1e-9a0f-3f1d6c2b7a10"
		if decision := f.evaluate(startUserToken(), in); !decision.Permitted || decision.ApprovedOperation != nil || len(f.notified.sent) != 0 {
			t.Fatalf("decision=%+v notifications=%d", decision, len(f.notified.sent))
		}
	})
	t.Run("保留中の上限", func(t *testing.T) {
		f := newStartFixture(t)
		for i := range approvaldomain.MaxPendingResourceServerApprovals {
			requireReason(t, f.evaluate(startUserToken(), startRequest("k", `{"amount":`+strings.Repeat("1", i+1)+`}`)), oauthdomain.ReasonApprovalPending)
		}
		decision := f.evaluate(startUserToken(), startRequest("k", `{"amount":0}`))
		requireReason(t, decision, oauthdomain.ReasonApprovalRequestLimitExceeded)
		if decision.StartedApproval != nil || len(f.notified.sent) != approvaldomain.MaxPendingResourceServerApprovals {
			t.Fatalf("decision=%+v notifications=%d", decision, len(f.notified.sent))
		}
	})
}

type failingStartStore struct {
	*approvalmemory.ApprovalRequestStore
}

func (failingStartStore) StartForResourceServer(context.Context, *approvaldomain.ApprovalRequest, int, time.Time) (*approvaldomain.ApprovalRequest, approvalports.StartOutcome, error) {
	return nil, approvalports.StartCreated, errors.New("store down")
}

//spec:covers REQ-OAUTH2-093: 承認ストアへ書けないときは評価不能にし、許可も通知もしないこと
func TestApprovalStartFailsClosedWhenTheStoreFails(t *testing.T) {
	f := newStartFixture(t)
	deps := f.deps(startUserToken())
	deps.Approvals = failingStartStore{f.store}
	if decision, err := EvaluateResourceAccess(evalTenantContext(), deps, startRequest("k", `{"amount":100}`), f.now); err == nil || decision.Permitted {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if len(f.notified.sent) != 0 {
		t.Fatalf("notified on a failed start: %d", len(f.notified.sent))
	}
}
