package usecases

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

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

// AccessGate はフェデレーション開始時のアプリ割当ゲートに必要な依存を保持する。
type AccessGate struct {
	ApplicationRepo             ports.ApplicationRepository
	ApplicationAssignmentRepo   ports.AssignmentRepository
	GroupMemberships            ports.GroupMemberships
	ApplicationSignInPolicyRepo ports.SignInPolicyRepository
	DefaultSignInPolicyRepo     ports.DefaultSignInPolicyRepository
	GateTrustedForwardedHops    int
}

// ClientIP は信頼済み転送ホップ数を考慮して X-Forwarded-For からクライアント IP を解決する。
// TRUSTED_FORWARDED_HOPS が 0 (直結/未設定) の場合は空を返し、CIDR 条件は fail-closed になる。
func (g *AccessGate) ClientIP(r *http.Request) string {
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

// EvaluateApplicationAccess は binding 経由のフェデレーション開始を許可してよいかを判定する。
// Application が見つからない (catalog 外) なら許可する。見つかった場合は active かつ
// subject が割当済みのときだけ、サインインポリシーの評価へ進む。判定不能・未割当・disabled は
// 拒否する (fail-closed)。
func (g *AccessGate) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType domain.ApplicationProtocolType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (domain.ApplicationAccessDecision, error) {
	if g.ApplicationRepo == nil {
		return domain.ApplicationAccessDecision{Allowed: true}, nil
	}
	app, err := g.ApplicationRepo.FindByProtocol(ctx, tenantID, bindingType, bindingKey)
	if err != nil {
		return domain.ApplicationAccessDecision{}, err
	}
	if app == nil {
		return domain.ApplicationAccessDecision{Allowed: true}, nil
	}
	if app.Status != domain.ApplicationActive {
		return domain.ApplicationAccessDecision{ApplicationID: app.ID, Reason: "application is disabled"}, nil
	}
	if g.ApplicationAssignmentRepo == nil {
		return domain.ApplicationAccessDecision{ApplicationID: app.ID, Reason: "application assignments are unavailable"}, nil
	}
	subjects := []ports.SubjectRef{{Type: domain.AssignmentSubjectUser, ID: sub}}
	if g.GroupMemberships != nil {
		groupIDs, err := g.GroupMemberships.GroupIDsOfUser(ctx, tenantID, sub)
		if err != nil {
			return domain.ApplicationAccessDecision{}, err
		}
		for _, groupID := range groupIDs {
			subjects = append(subjects, ports.SubjectRef{Type: domain.AssignmentSubjectGroup, ID: groupID})
		}
	}
	assignments, err := g.ApplicationAssignmentRepo.ListBySubjects(ctx, tenantID, subjects)
	if err != nil {
		return domain.ApplicationAccessDecision{}, err
	}
	assigned := false
	for _, a := range assignments {
		if a.ApplicationID == app.ID {
			assigned = true
			break
		}
	}
	if !assigned {
		return domain.ApplicationAccessDecision{ApplicationID: app.ID, Reason: "subject not assigned to application"}, nil
	}
	if g.ApplicationSignInPolicyRepo == nil {
		return domain.ApplicationAccessDecision{Allowed: true, ApplicationID: app.ID}, nil
	}
	policy, err := g.ApplicationSignInPolicyRepo.Get(ctx, tenantID, app.ID)
	if err != nil {
		return domain.ApplicationAccessDecision{}, err
	}
	// アプリ個別ポリシーがあればそれを、なければテナントデフォルトを適用する (上書きモデル)。
	var defaultPolicy *domain.TenantDefaultSignInPolicy
	if g.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err = g.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err != nil {
			return domain.ApplicationAccessDecision{}, err
		}
	}
	effective := EffectivePolicyForEvaluation(defaultPolicy, policy)
	evaluation := EvaluateSignInPolicy(effective, authn, clientIP, time.Now().UTC())
	switch evaluation.Decision {
	case PolicyAllow:
		return domain.ApplicationAccessDecision{Allowed: true, ApplicationID: app.ID}, nil
	case PolicyStepUpRequired:
		return domain.ApplicationAccessDecision{
			ApplicationID: app.ID, StepUpRequired: true, Reason: evaluation.Reason,
			TrustedDeviceAllowed: effective != nil && TrustedDeviceAllowedByRules(effective.Rules),
		}, nil
	default:
		return domain.ApplicationAccessDecision{ApplicationID: app.ID, Reason: evaluation.Reason}, nil
	}
}
