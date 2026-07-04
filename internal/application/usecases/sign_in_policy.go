package usecases

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/internal/application/ports"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
)

var (
	ErrInvalidSignInPolicy = errors.New("invalid application sign-in policy")
	ErrPolicyDenied        = errors.New("application sign-in policy denied access")
	ErrPolicyStepUp        = errors.New("application sign-in policy requires step-up")
)

type SignInPolicyDeps struct {
	AppRepo    ports.ApplicationRepository
	PolicyRepo ports.SignInPolicyRepository
	Emit       func(spec.DomainEvent)
}

type UpdateSignInPolicyInput struct {
	ActorSub      string
	ApplicationID string
	Rules         []spec.SignInRule
	Now           time.Time
}

func EmptySignInPolicy(tenantID, applicationID string, now time.Time) *spec.AppSignInPolicy {
	return &spec.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: applicationID, Rules: []spec.SignInRule{}, UpdatedAt: adminNow(now),
	}
}

func GetSignInPolicy(ctx context.Context, deps SignInPolicyDeps, applicationID string) (*spec.AppSignInPolicy, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := ensureApplicationExists(ctx, deps.AppRepo, tenantID, applicationID); err != nil {
		return nil, err
	}
	if deps.PolicyRepo == nil {
		return EmptySignInPolicy(tenantID, applicationID, time.Now().UTC()), nil
	}
	policy, err := deps.PolicyRepo.Get(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return EmptySignInPolicy(tenantID, applicationID, time.Now().UTC()), nil
	}
	if policy.Rules == nil {
		policy.Rules = []spec.SignInRule{}
	}
	return policy, nil
}

func UpdateSignInPolicy(ctx context.Context, deps SignInPolicyDeps, in UpdateSignInPolicyInput) (*spec.AppSignInPolicy, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := ensureApplicationExists(ctx, deps.AppRepo, tenantID, in.ApplicationID); err != nil {
		return nil, err
	}
	rules := slices.Clone(in.Rules)
	if err := ValidateSignInPolicyRules(rules); err != nil {
		return nil, err
	}
	policy := &spec.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: in.ApplicationID, Rules: rules, UpdatedAt: adminNow(in.Now),
	}
	if deps.PolicyRepo != nil {
		if err := deps.PolicyRepo.Save(ctx, policy); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &spec.AppSignInPolicyUpdated{
		At: policy.UpdatedAt, TenantID: tenantID, ActorSub: in.ActorSub, ApplicationID: in.ApplicationID,
	})
	return policy, nil
}

func ValidateSignInPolicyRules(rules []spec.SignInRule) error {
	seen := map[string]struct{}{}
	for i := range rules {
		r := &rules[i]
		r.RuleID = strings.TrimSpace(r.RuleID)
		r.Name = strings.TrimSpace(r.Name)
		if r.RequiredAuthn.Strength == "" {
			r.RequiredAuthn.Strength = spec.RequiredAuthnPassword
		}
		if r.RuleID == "" {
			id, err := spec.NewUUIDv4()
			if err != nil {
				return err
			}
			r.RuleID = id
		}
		if _, ok := seen[r.RuleID]; ok {
			return ErrInvalidSignInPolicy
		}
		seen[r.RuleID] = struct{}{}
		if r.Name == "" {
			return ErrInvalidSignInPolicy
		}
		if !r.RequiredAuthn.Strength.Valid() {
			return ErrInvalidSignInPolicy
		}
		cidrs, err := normalizeCIDRs(r.Condition.NetworkAllowCIDRs)
		if err != nil {
			return err
		}
		r.Condition.NetworkAllowCIDRs = cidrs
		if r.Condition.ReauthMaxAgeSeconds != nil && *r.Condition.ReauthMaxAgeSeconds <= 0 {
			return ErrInvalidSignInPolicy
		}
	}
	return nil
}

// normalizeCIDRs は許可 CIDR を検証・正規化する。空要素は無視し、パースできない値は拒否する。
func normalizeCIDRs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, ErrInvalidSignInPolicy
		}
		out = append(out, prefix.Masked().String())
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

type PolicyDecision string

const (
	PolicyAllow          PolicyDecision = "allow"
	PolicyDeny           PolicyDecision = "deny"
	PolicyStepUpRequired PolicyDecision = "step_up_required"
)

type PolicyEvaluation struct {
	Decision PolicyDecision
	Reason   string
}

// EvaluateSignInPolicy はサインインポリシーを fail-closed で評価する。
// clientIP は許可 CIDR 条件の評価に使う。IP が空で許可 CIDR が設定されていれば拒否する。
func EvaluateSignInPolicy(policy *spec.AppSignInPolicy, authn *authdomain.AuthenticationContext, clientIP string, now time.Time) PolicyEvaluation {
	if policy == nil || len(policy.Rules) == 0 {
		return PolicyEvaluation{Decision: PolicyAllow}
	}
	if authn == nil || authn.AuthenticationPending {
		return PolicyEvaluation{Decision: PolicyDeny, Reason: "authentication context is unavailable"}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, rule := range policy.Rules {
		if !rule.Enabled {
			continue
		}
		if len(rule.Condition.NetworkAllowCIDRs) > 0 && !clientIPAllowed(clientIP, rule.Condition.NetworkAllowCIDRs) {
			return PolicyEvaluation{Decision: PolicyDeny, Reason: "client network is not allowed"}
		}
		if rule.RequiredAuthn.Strength == spec.RequiredAuthnMfa && !authusecases.ACRSatisfies(authn.ACR, authusecases.ACRMFA) {
			return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "mfa requirement is not satisfied"}
		}
		if rule.Condition.ReauthMaxAgeSeconds != nil {
			recent := max(authn.AuthTime, authn.StepUpAt)
			if recent <= 0 || now.Unix()-recent > int64(*rule.Condition.ReauthMaxAgeSeconds) {
				return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "reauth max age exceeded"}
			}
		}
	}
	return PolicyEvaluation{Decision: PolicyAllow}
}

// clientIPAllowed は clientIP が許可 CIDR のいずれかに含まれるかを返す。
// IP が空・パース不能なら fail-closed で false。
func clientIPAllowed(clientIP string, cidrs []string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, entry := range cidrs {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func ensureApplicationExists(ctx context.Context, repo ports.ApplicationRepository, tenantID, applicationID string) error {
	if repo == nil {
		return ErrApplicationNotFound
	}
	app, err := repo.FindByID(ctx, tenantID, applicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return ErrApplicationNotFound
	}
	return nil
}
