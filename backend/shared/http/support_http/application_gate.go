package support_http

// フェデレーション開始経路の割当ゲート (wi-69, invariant AssignmentGatesProtocol)。
//
// protocol row の application_id relation を持つ Application に対し、
// 解決された subject (本人 + 所属グループ) が割当済みかを fail-closed で判定する。
// catalog に属さない protocol record (application_id=NULL) は gating 対象外とする。

import (
	"context"
	"net/http"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

type ApplicationAccessDecision struct {
	Allowed        bool
	StepUpRequired bool
	ApplicationID  string
	Reason         string
	// TrustedDeviceAllowed は、実効ポリシーが記憶済みの信頼済みデバイスによる MFA の
	// 充足を認めるかどうか (wi-91)。StepUpRequired のときだけ意味を持ち、false なら
	// 呼び出し側は cookie を見ずに本物の第二要素を要求する。
	TrustedDeviceAllowed bool
}

// ApplicationAccessAllowed は binding 経由のフェデレーション開始を許可してよいかを返す。
// Application が見つからない (catalog 外) なら true。見つかった場合は active かつ
// subject が割当済みのときのみ true。判定不能・未割当・disabled は false (fail-closed)。
func (g *ApplicationGate) ApplicationAccessAllowed(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ApplicationProtocolType,
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
	bindingType appdomain.ApplicationProtocolType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (ApplicationAccessDecision, error) {
	if g.ApplicationRepo == nil {
		return ApplicationAccessDecision{Allowed: true}, nil
	}
	app, err := g.ApplicationRepo.FindByProtocol(ctx, tenantID, bindingType, bindingKey)
	if err != nil {
		return ApplicationAccessDecision{}, err
	}
	if app == nil {
		return ApplicationAccessDecision{Allowed: true}, nil
	}
	if app.Status != appdomain.ApplicationActive {
		return ApplicationAccessDecision{ApplicationID: app.ID, Reason: "application is disabled"}, nil
	}
	if g.ApplicationAssignmentRepo == nil {
		return ApplicationAccessDecision{ApplicationID: app.ID, Reason: "application assignments are unavailable"}, nil
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
		if a.ApplicationID == app.ID {
			assigned = true
			break
		}
	}
	if !assigned {
		return ApplicationAccessDecision{ApplicationID: app.ID, Reason: "subject not assigned to application"}, nil
	}
	if g.ApplicationSignInPolicyRepo == nil {
		return ApplicationAccessDecision{Allowed: true, ApplicationID: app.ID}, nil
	}
	policy, err := g.ApplicationSignInPolicyRepo.Get(ctx, tenantID, app.ID)
	if err != nil {
		return ApplicationAccessDecision{}, err
	}
	// アプリ個別ポリシーがあればそれを、なければテナントデフォルトを適用する (上書きモデル)。
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
		return ApplicationAccessDecision{Allowed: true, ApplicationID: app.ID}, nil
	case appusecases.PolicyStepUpRequired:
		return ApplicationAccessDecision{
			ApplicationID: app.ID, StepUpRequired: true, Reason: evaluation.Reason,
			TrustedDeviceAllowed: effective != nil && appusecases.TrustedDeviceAllowedByRules(effective.Rules),
		}, nil
	default:
		return ApplicationAccessDecision{ApplicationID: app.ID, Reason: evaluation.Reason}, nil
	}
}
