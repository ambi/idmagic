package handlers_http_test

// Application ごとのサインインポリシー (REQ-APPLICATION-009) と、テナントデフォルトの
// サインインポリシー (REQ-APPLICATION-010) の強制点は OAuth2.Authorize である。
// 保存の側は backend/application が持ち、ここは「保存した規則がフェデレーションの可否を
// 実際に変えるか」だけを観測する。

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const signInPolicyApplicationID = "policy-app"

// signInPolicyStack は、割り当て済みの Application 1 つと、同意済みのグラントを持つ
// 認可経路を建てる。サインインポリシーの保存先は呼び出し側が書き換えられるように返す。
func signInPolicyStack(t *testing.T) (
	*echo.Echo, *[]spec.DomainEvent,
	*appmemory.SignInPolicyRepository, *appmemory.DefaultSignInPolicyRepository,
) {
	t.Helper()
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	applications := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	policies := appmemory.NewSignInPolicyRepository()
	defaults := appmemory.NewDefaultSignInPolicyRepository()
	if err := applications.Save(context.Background(), &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: signInPolicyApplicationID, Name: "Policy application",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: authClientID},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed application: %v", err)
	}
	if err := assignments.Save(context.Background(), &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: signInPolicyApplicationID,
		SubjectType: appdomain.AssignmentSubjectUser, SubjectID: authn.UserID,
		Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	granted := &domain.Consent{
		UserID: authn.UserID, ClientID: authClientID,
		Scopes:    []string{"openid", "profile"},
		State:     domain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	e, emitted := newAuthorizeTestServer(t, authn, granted, func(deps *httpadapter.Deps) {
		deps.Application = application.Module{
			Repo: applications, AssignmentRepo: assignments,
			SignInPolicyRepo: policies, DefaultSignInPolicyRepo: defaults,
		}
	})
	return e, emitted, policies, defaults
}

func mfaRequiredRule(condition appdomain.AccessCondition) appdomain.SignInRule {
	return appdomain.SignInRule{
		RuleID: "mfa", Name: "MFA", Enabled: true,
		RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
		Condition:     condition,
	}
}

func emittedTypes(events *[]spec.DomainEvent) []string {
	types := make([]string, 0, len(*events))
	for _, event := range *events {
		types = append(types, event.EventType())
	}
	return types
}

func hasEvent(events *[]spec.DomainEvent, want string) bool {
	return strings.Contains(strings.Join(emittedTypes(events), " "), want)
}

//spec:covers REQ-APPLICATION-009, EX-APPLICATION-009-01: MFA 必須の Application ポリシーがトークン発行の前に評価され、単要素セッションには認可コードが出ず AppStepUpRequired が発行されること。ポリシーの保存と AppSignInPolicyUpdated は TestUpdateAppSignInPolicyEmitsItsEvent が固定する。
func TestAppSignInPolicyIsEvaluatedBeforeTheAuthorizationCodeIsIssued(t *testing.T) {
	e, emitted, policies, _ := signInPolicyStack(t)

	// ポリシーが無い間は同じ要求でコードが出る。強制点が効いていることは、
	// 「出ていたものが出なくなる」でしか観測できない。
	allowed := runAuthorize(t, e, authorizeQuery(url.Values{}))
	if !strings.Contains(allowed.Header().Get("Location"), "code=") {
		t.Fatalf("baseline must issue a code: status=%d location=%q",
			allowed.Code, allowed.Header().Get("Location"))
	}

	now := time.Now().UTC()
	if err := policies.Save(context.Background(), &appdomain.AppSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: signInPolicyApplicationID,
		Rules:     []appdomain.SignInRule{mfaRequiredRule(appdomain.AccessCondition{})},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	refused := runAuthorize(t, e, authorizeQuery(url.Values{}))
	location := refused.Header().Get("Location")
	if strings.Contains(location, "code=") {
		t.Fatalf("single-factor session must not receive a code: location=%q", location)
	}
	if !hasEvent(emitted, "AppStepUpRequired") {
		t.Fatalf("events=%v, want AppStepUpRequired", emittedTypes(emitted))
	}
}

//spec:covers REQ-APPLICATION-009, EX-APPLICATION-009-03: 許可 CIDR に含まれないクライアント IP のフェデレーションを拒否し、認可コードを出さず AppAccessDeniedByPolicy を発行すること。
func TestSignInPolicyRefusesAClientIPOutsideTheAllowedCIDR(t *testing.T) {
	e, emitted, policies, _ := signInPolicyStack(t)

	now := time.Now().UTC()
	if err := policies.Save(context.Background(), &appdomain.AppSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: signInPolicyApplicationID,
		Rules: []appdomain.SignInRule{{
			RuleID: "network", Name: "Office only", Enabled: true,
			RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnPassword},
			Condition:     appdomain.AccessCondition{NetworkAllowCIDRs: []string{"10.10.0.0/16"}},
		}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save policy: %v", err)
	}

	refused := runAuthorize(t, e, authorizeQuery(url.Values{}))
	location := refused.Header().Get("Location")
	if strings.Contains(location, "code=") {
		t.Fatalf("a client outside the allowed CIDR must not receive a code: location=%q", location)
	}
	if !strings.Contains(location, "error=access_denied") {
		t.Fatalf("location=%q, want access_denied", location)
	}
	if !hasEvent(emitted, "AppAccessDeniedByPolicy") {
		t.Fatalf("events=%v, want AppAccessDeniedByPolicy", emittedTypes(emitted))
	}
}

//spec:covers REQ-APPLICATION-010, EX-APPLICATION-010-01: 個別ポリシーを持たない Application にテナントデフォルトの MFA 必須が適用され、単要素セッションが認可コードを受け取れずステップアップを求められること。保存、警告、3 段の区別表示、未登録人数の表示は同じ id を名指す他のテストが固定する。
func TestTenantDefaultSignInPolicyAppliesToAnApplicationWithoutItsOwn(t *testing.T) {
	e, emitted, _, defaults := signInPolicyStack(t)

	allowed := runAuthorize(t, e, authorizeQuery(url.Values{}))
	if !strings.Contains(allowed.Header().Get("Location"), "code=") {
		t.Fatalf("baseline must issue a code: location=%q", allowed.Header().Get("Location"))
	}

	now := time.Now().UTC()
	if err := defaults.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID:  tenancydomain.DefaultTenantID,
		Rules:     []appdomain.SignInRule{mfaRequiredRule(appdomain.AccessCondition{})},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save default policy: %v", err)
	}

	refused := runAuthorize(t, e, authorizeQuery(url.Values{}))
	if strings.Contains(refused.Header().Get("Location"), "code=") {
		t.Fatalf("the tenant default must gate an application without its own policy: location=%q",
			refused.Header().Get("Location"))
	}
	if !hasEvent(emitted, "AppStepUpRequired") {
		t.Fatalf("events=%v, want AppStepUpRequired", emittedTypes(emitted))
	}
}

//spec:covers REQ-APPLICATION-010, EX-APPLICATION-010-02: 規則を空にしたテナントデフォルトが、独自ポリシーを持たない Application のフェデレーションに追加要件を課さないこと。TenantDefaultSignInPolicyUpdated の発行は TestEmptyTenantDefaultSignInPolicyEmitsItsEventAndImposesNothing が固定する。
func TestEmptyTenantDefaultSignInPolicyImposesNoExtraRequirement(t *testing.T) {
	e, _, _, defaults := signInPolicyStack(t)

	now := time.Now().UTC()
	if err := defaults.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, Rules: []appdomain.SignInRule{},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save default policy: %v", err)
	}

	allowed := runAuthorize(t, e, authorizeQuery(url.Values{}))
	if !strings.Contains(allowed.Header().Get("Location"), "code=") {
		t.Fatalf("an empty default must impose nothing: status=%d location=%q body=%s",
			allowed.Code, allowed.Header().Get("Location"), allowed.Body.String())
	}
	if allowed.Code != http.StatusFound {
		t.Fatalf("status=%d, want a 302 to the redirect_uri", allowed.Code)
	}
}
