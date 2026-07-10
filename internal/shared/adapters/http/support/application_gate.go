package support

// フェデレーション開始経路の割当ゲート (wi-69, invariant AssignmentGatesProtocol)。
//
// protocol binding (OIDC client_id / WS-Fed wtrealm) を所有する Application に対し、
// 解決された subject (本人 + 所属グループ) が割当済みかを fail-closed で判定する。
// catalog に属さない client (binding 未登録) は gating 対象外とし、既存挙動を保つ。

import (
	"context"
	"net/http"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/internal/application/domain"
	appports "github.com/ambi/idmagic/internal/application/ports"
	appusecases "github.com/ambi/idmagic/internal/application/usecases"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
)

type ApplicationAccessDecision struct {
	Allowed        bool
	StepUpRequired bool
	ApplicationID  string
	Reason         string
}

// ApplicationAccessAllowed は binding 経由のフェデレーション開始を許可してよいかを返す。
// Application が見つからない (catalog 外) なら true。見つかった場合は active かつ
// subject が割当済みのときのみ true。判定不能・未割当・disabled は false (fail-closed)。
func (g *ApplicationGate) ApplicationAccessAllowed(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ProtocolBindingType,
	bindingKey, sub string,
) (bool, error) {
	decision, err := g.EvaluateApplicationAccess(ctx, tenantID, bindingType, bindingKey, sub, nil, "")
	if err != nil {
		return false, err
	}
	return decision.Allowed, nil
}

// ClientIP は信頼済み転送ホップ数を考慮して X-Forwarded-For からクライアント IP を解決する。
// TRUSTED_FORWARDED_HOPS が 0 (直結/未設定) の場合は空を返し、CIDR 条件は fail-closed になる。
func (g *ApplicationGate) ClientIP(r *http.Request) string {
	if r == nil || g.GateTrustedForwardedHops <= 0 {
		return ""
	}
	parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	ips := make([]string, 0, len(parts))
	for _, part := range parts {
		if ip := strings.TrimSpace(part); ip != "" {
			ips = append(ips, ip)
		}
	}
	index := len(ips) - 1 - g.GateTrustedForwardedHops
	if index < 0 || index >= len(ips) {
		return ""
	}
	return ips[index]
}

func (g *ApplicationGate) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ProtocolBindingType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (ApplicationAccessDecision, error) {
	if g.ApplicationRepo == nil {
		return ApplicationAccessDecision{Allowed: true}, nil
	}
	app, err := g.ApplicationRepo.FindByBinding(ctx, tenantID, bindingType, bindingKey)
	if err != nil {
		return ApplicationAccessDecision{}, err
	}
	if app == nil {
		return ApplicationAccessDecision{Allowed: true}, nil
	}
	if app.Status != appdomain.ApplicationActive {
		return ApplicationAccessDecision{ApplicationID: app.ApplicationID, Reason: "application is disabled"}, nil
	}
	if g.ApplicationAssignmentRepo == nil {
		return ApplicationAccessDecision{ApplicationID: app.ApplicationID, Reason: "application assignments are unavailable"}, nil
	}
	subjects := []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: sub}}
	if g.GroupRepo != nil {
		groups, err := g.GroupRepo.ListGroupsByUser(ctx, tenantID, sub)
		if err != nil {
			return ApplicationAccessDecision{}, err
		}
		for _, grp := range groups {
			subjects = append(subjects, appports.SubjectRef{Type: appdomain.AssignmentSubjectGroup, ID: grp.ID})
		}
	}
	assignments, err := g.ApplicationAssignmentRepo.ListBySubjects(ctx, tenantID, subjects)
	if err != nil {
		return ApplicationAccessDecision{}, err
	}
	assigned := false
	for _, a := range assignments {
		if a.ApplicationID == app.ApplicationID {
			assigned = true
			break
		}
	}
	if !assigned {
		return ApplicationAccessDecision{ApplicationID: app.ApplicationID, Reason: "subject not assigned to application"}, nil
	}
	if g.ApplicationSignInPolicyRepo == nil {
		return ApplicationAccessDecision{Allowed: true, ApplicationID: app.ApplicationID}, nil
	}
	policy, err := g.ApplicationSignInPolicyRepo.Get(ctx, tenantID, app.ApplicationID)
	if err != nil {
		return ApplicationAccessDecision{}, err
	}
	// アプリ個別ポリシーがあればそれを、なければテナントデフォルトを適用する (上書きモデル, ADR-081)。
	var defaultPolicy *appdomain.TenantDefaultSignInPolicy
	if g.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err = g.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err != nil {
			return ApplicationAccessDecision{}, err
		}
	}
	effective := appusecases.EffectivePolicyForEvaluation(defaultPolicy, policy)
	evaluation := appusecases.EvaluateSignInPolicy(effective, authn, clientIP, time.Now().UTC())
	switch evaluation.Decision {
	case appusecases.PolicyAllow:
		return ApplicationAccessDecision{Allowed: true, ApplicationID: app.ApplicationID}, nil
	case appusecases.PolicyStepUpRequired:
		return ApplicationAccessDecision{ApplicationID: app.ApplicationID, StepUpRequired: true, Reason: evaluation.Reason}, nil
	default:
		return ApplicationAccessDecision{ApplicationID: app.ApplicationID, Reason: evaluation.Reason}, nil
	}
}
