package usecases

import (
	"context"
	"errors"
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
	ErrInvalidSignOnPolicy = errors.New("invalid application sign-on policy")
	ErrPolicyDenied        = errors.New("application sign-on policy denied access")
	ErrPolicyStepUp        = errors.New("application sign-on policy requires step-up")
)

type SignOnPolicyDeps struct {
	AppRepo    ports.ApplicationRepository
	PolicyRepo ports.SignOnPolicyRepository
	Emit       func(spec.DomainEvent)
}

type UpdateSignOnPolicyInput struct {
	ActorSub      string
	ApplicationID string
	Rules         []spec.SignOnRule
	Now           time.Time
}

func EmptySignOnPolicy(tenantID, applicationID string, now time.Time) *spec.AppSignOnPolicy {
	return &spec.AppSignOnPolicy{
		TenantID: tenantID, ApplicationID: applicationID, Rules: []spec.SignOnRule{}, UpdatedAt: adminNow(now),
	}
}

func GetSignOnPolicy(ctx context.Context, deps SignOnPolicyDeps, applicationID string) (*spec.AppSignOnPolicy, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := ensureApplicationExists(ctx, deps.AppRepo, tenantID, applicationID); err != nil {
		return nil, err
	}
	if deps.PolicyRepo == nil {
		return EmptySignOnPolicy(tenantID, applicationID, time.Now().UTC()), nil
	}
	policy, err := deps.PolicyRepo.Get(ctx, tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return EmptySignOnPolicy(tenantID, applicationID, time.Now().UTC()), nil
	}
	if policy.Rules == nil {
		policy.Rules = []spec.SignOnRule{}
	}
	return policy, nil
}

func UpdateSignOnPolicy(ctx context.Context, deps SignOnPolicyDeps, in UpdateSignOnPolicyInput) (*spec.AppSignOnPolicy, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := ensureApplicationExists(ctx, deps.AppRepo, tenantID, in.ApplicationID); err != nil {
		return nil, err
	}
	rules := slices.Clone(in.Rules)
	if err := ValidateSignOnPolicyRules(rules); err != nil {
		return nil, err
	}
	policy := &spec.AppSignOnPolicy{
		TenantID: tenantID, ApplicationID: in.ApplicationID, Rules: rules, UpdatedAt: adminNow(in.Now),
	}
	if deps.PolicyRepo != nil {
		if err := deps.PolicyRepo.Save(ctx, policy); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &spec.AppSignOnPolicyUpdated{
		At: policy.UpdatedAt, TenantID: tenantID, ActorSub: in.ActorSub, ApplicationID: in.ApplicationID,
	})
	return policy, nil
}

func ValidateSignOnPolicyRules(rules []spec.SignOnRule) error {
	seen := map[string]struct{}{}
	for i := range rules {
		r := &rules[i]
		r.RuleID = strings.TrimSpace(r.RuleID)
		r.Name = strings.TrimSpace(r.Name)
		r.RequiredAuthn.ACR = strings.TrimSpace(r.RequiredAuthn.ACR)
		r.RequiredAuthn.Factor = strings.TrimSpace(r.RequiredAuthn.Factor)
		r.Condition.Network = strings.TrimSpace(r.Condition.Network)
		r.Condition.Device = strings.TrimSpace(r.Condition.Device)
		if r.RuleID == "" {
			id, err := spec.NewUUIDv4()
			if err != nil {
				return err
			}
			r.RuleID = id
		}
		if _, ok := seen[r.RuleID]; ok {
			return ErrInvalidSignOnPolicy
		}
		seen[r.RuleID] = struct{}{}
		if r.Name == "" {
			return ErrInvalidSignOnPolicy
		}
		if r.RequiredAuthn.Factor != "" && r.RequiredAuthn.Factor != "password" && r.RequiredAuthn.Factor != "totp" {
			return ErrInvalidSignOnPolicy
		}
		if r.Condition.ReauthMaxAgeSeconds != nil && *r.Condition.ReauthMaxAgeSeconds <= 0 {
			return ErrInvalidSignOnPolicy
		}
	}
	return nil
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

func EvaluateSignOnPolicy(policy *spec.AppSignOnPolicy, authn *authdomain.AuthenticationContext, now time.Time) PolicyEvaluation {
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
		if rule.Condition.Network != "" || rule.Condition.Device != "" {
			return PolicyEvaluation{Decision: PolicyDeny, Reason: "unsupported access condition"}
		}
		if rule.RequiredAuthn.ACR != "" && !authusecases.ACRSatisfies(authn.ACR, rule.RequiredAuthn.ACR) {
			return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "acr requirement is not satisfied"}
		}
		if rule.RequiredAuthn.Factor != "" && !factorSatisfied(authn.AMR, rule.RequiredAuthn.Factor) {
			return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "factor requirement is not satisfied"}
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

func factorSatisfied(amr []string, factor string) bool {
	switch factor {
	case "":
		return true
	case "password":
		return slices.Contains(amr, "pwd") || slices.Contains(amr, "password")
	case "totp":
		return slices.Contains(amr, "otp") || slices.Contains(amr, "totp")
	default:
		return false
	}
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
