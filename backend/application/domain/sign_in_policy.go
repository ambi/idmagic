package domain

import (
	"net/netip"
	"slices"
	"strings"
	"time"
)

// SignInAuthentication は、ポリシーが読む認証事実だけを表す。
// ACR と AMR の解釈は供給元に閉じ、認証方式を評価器へ持ち込まない。
// 各メソッドは作用を持たず、Pending は認証結果がない場合も true を返す。
type SignInAuthentication interface {
	Pending() bool
	MFAAuthenticated() bool
	SecondFactorAuthenticated() bool
	AuthenticatedAt() int64
	SteppedUpAt() int64
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

// Configured は、アプリが独自のポリシーを持つか (有効なルールが 1 つ以上あるか) を返す。
// 持たなければテナントのデフォルトが適用される。
func (p *AppSignInPolicy) Configured() bool {
	if p == nil {
		return false
	}
	for _, rule := range p.Rules {
		if rule.Enabled {
			return true
		}
	}
	return false
}

// EffectiveSignInRules は上書きモデルで実際に適用されるルール列を返す。
// アプリが独自ポリシーを持てばそれがデフォルトを完全に置換し、持たなければデフォルトを適用する。
func EffectiveSignInRules(def *TenantDefaultSignInPolicy, app *AppSignInPolicy) []SignInRule {
	if app.Configured() {
		return slices.Clone(app.Rules)
	}
	if def != nil {
		return slices.Clone(def.Rules)
	}
	return nil
}

// EffectivePolicyForEvaluation は上書き後の実効ルールで評価用 policy を組み立てる。ルールが無ければ nil。
func EffectivePolicyForEvaluation(def *TenantDefaultSignInPolicy, app *AppSignInPolicy) *AppSignInPolicy {
	rules := EffectiveSignInRules(def, app)
	if len(rules) == 0 {
		return nil
	}
	out := &AppSignInPolicy{Rules: rules}
	if app != nil {
		out.TenantID = app.TenantID
		out.ApplicationID = app.ApplicationID
	}
	return out
}

// TrustedDeviceAllowedByRules は、記憶済みの信頼済みデバイスで MFA 要件を満たしてよいかを
// 実効ルールから返す。MFA を要求する有効なルールが 1 つでもこれを禁じていれば false
// (「毎回 MFA」が優先される)。MFA を要求するルールが無い場合は判断の対象自体が無いので true。
func TrustedDeviceAllowedByRules(rules []SignInRule) bool {
	for i := range rules {
		rule := &rules[i]
		if rule.Enabled && rule.RequiredAuthn.Strength == RequiredAuthnMfa && !rule.TrustedDeviceAllowed() {
			return false
		}
	}
	return true
}

func MfaEnrollmentPolicyFromRules(rules []SignInRule) *MfaEnrollmentPolicy {
	for i := range rules {
		if rules[i].Enabled && rules[i].RequiredAuthn.Strength == RequiredAuthnMfa {
			return rules[i].MfaEnrollment
		}
	}
	return nil
}

// UpcomingMfaEnforcementStart は、実効ルールが now より後に MFA の強制を始めるとき、その開始日時を返す。
// 強制が既に始まっているか予定がなければ nil を返す。
func UpcomingMfaEnforcementStart(rules []SignInRule, now time.Time) *time.Time {
	policy := MfaEnrollmentPolicyFromRules(rules)
	if policy == nil || policy.EnforcementStartAt == nil || !now.Before(*policy.EnforcementStartAt) {
		return nil
	}
	start := *policy.EnforcementStartAt
	return &start
}

// EvaluateSignInPolicy はサインインポリシーを fail-closed で評価する。
// clientIP は許可 CIDR 条件の評価に使う。IP が空で許可 CIDR が設定されていれば拒否する。
func EvaluateSignInPolicy(policy *AppSignInPolicy, authn SignInAuthentication, clientIP string, now time.Time) PolicyEvaluation {
	if policy == nil || len(policy.Rules) == 0 {
		return PolicyEvaluation{Decision: PolicyAllow}
	}
	if authn == nil || authn.Pending() {
		return PolicyEvaluation{Decision: PolicyDeny, Reason: "authentication context is unavailable"}
	}
	for _, rule := range policy.Rules {
		if !rule.Enabled {
			continue
		}
		if len(rule.Condition.NetworkAllowCIDRs) > 0 && !ClientIPAllowed(clientIP, rule.Condition.NetworkAllowCIDRs) {
			return PolicyEvaluation{Decision: PolicyDeny, Reason: "client network is not allowed"}
		}
		mfaEnforced := rule.MfaEnrollment == nil || rule.MfaEnrollment.EnforcementStartAt == nil || !now.Before(*rule.MfaEnrollment.EnforcementStartAt)
		if rule.RequiredAuthn.Strength == RequiredAuthnMfa && mfaEnforced && !mfaSatisfied(rule, authn) {
			return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "mfa requirement is not satisfied"}
		}
		if rule.Condition.ReauthMaxAgeSeconds != nil {
			recent := max(authn.AuthenticatedAt(), authn.SteppedUpAt())
			if recent <= 0 || now.Unix()-recent > int64(*rule.Condition.ReauthMaxAgeSeconds) {
				return PolicyEvaluation{Decision: PolicyStepUpRequired, Reason: "reauth max age exceeded"}
			}
		}
	}
	return PolicyEvaluation{Decision: PolicyAllow}
}

// mfaSatisfied は、毎回の MFA を求めるルールでは実際の第二要素も要求する。
func mfaSatisfied(rule SignInRule, authn SignInAuthentication) bool {
	if !authn.MFAAuthenticated() {
		return false
	}
	if rule.TrustedDeviceAllowed() {
		return true
	}
	return authn.SecondFactorAuthenticated()
}

// ClientIPAllowed は clientIP が許可 CIDR のいずれかに含まれるかを返す。
// IP が空・パース不能・ゾーン付きなら fail-closed で false。IPv4 射影の IPv6 アドレスは
// IPv4 として照合する。パースできない CIDR は読み飛ばす。
func ClientIPAllowed(clientIP string, cidrs []string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(clientIP))
	if err != nil || addr.Zone() != "" {
		return false
	}
	addr = addr.Unmap()
	for _, entry := range cidrs {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			continue
		}
		prefix = prefix.Masked()
		// IPv4 射影の接頭辞 (::ffff:a.b.c.d/96 以上) は、その IPv4 部分の接頭辞として照合する。
		if prefix.Addr().Is4In6() {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
