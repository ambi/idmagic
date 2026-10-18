package usecases

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	ErrInvalidSignInPolicy = errors.New("invalid application sign-in policy")
	ErrPolicyDenied        = errors.New("application sign-in policy denied access")
	ErrPolicyStepUp        = errors.New("application sign-in policy requires step-up")
)

type SignInPolicyDeps struct {
	AppRepo     ports.ApplicationRepository
	PolicyRepo  ports.SignInPolicyRepository
	DefaultRepo ports.DefaultSignInPolicyRepository
	Emit        func(spec.DomainEvent)
}

type UpdateSignInPolicyInput struct {
	ActorUserID   string
	ApplicationID string
	Rules         []domain.SignInRule
	Now           time.Time
}

func EmptySignInPolicy(tenantID, applicationID string, now time.Time) *domain.AppSignInPolicy {
	at := adminNow(now)
	return &domain.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: applicationID, Rules: []domain.SignInRule{}, CreatedAt: at, UpdatedAt: at,
	}
}

func GetSignInPolicy(ctx context.Context, deps SignInPolicyDeps, applicationID string) (*domain.AppSignInPolicy, error) {
	tenantID := tenantports.TenantID(ctx)
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
		policy.Rules = []domain.SignInRule{}
	}
	return policy, nil
}

func UpdateSignInPolicy(ctx context.Context, deps SignInPolicyDeps, in UpdateSignInPolicyInput) (*domain.AppSignInPolicy, error) {
	tenantID := tenantports.TenantID(ctx)
	if err := ensureApplicationExists(ctx, deps.AppRepo, tenantID, in.ApplicationID); err != nil {
		return nil, err
	}
	rules := slices.Clone(in.Rules)
	if err := ValidateSignInPolicyRulesAt(rules, in.Now); err != nil {
		return nil, err
	}
	now := adminNow(in.Now)
	policy := &domain.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: in.ApplicationID, Rules: rules, CreatedAt: now, UpdatedAt: now,
	}
	if deps.PolicyRepo != nil {
		existing, err := deps.PolicyRepo.Get(ctx, tenantID, in.ApplicationID)
		if err != nil {
			return nil, err
		}
		if existing != nil && !existing.CreatedAt.IsZero() {
			policy.CreatedAt = existing.CreatedAt
		}
		if err := deps.PolicyRepo.Save(ctx, policy); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &domain.AppSignInPolicyUpdated{
		At: policy.UpdatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID, ApplicationID: in.ApplicationID,
	})
	return policy, nil
}

// EmptyDefaultSignInPolicy は未設定テナントの空のデフォルトポリシーを返す。
func EmptyDefaultSignInPolicy(tenantID string, now time.Time) *domain.TenantDefaultSignInPolicy {
	at := adminNow(now)
	return &domain.TenantDefaultSignInPolicy{
		TenantID: tenantID, Rules: []domain.SignInRule{}, CreatedAt: at, UpdatedAt: at,
	}
}

// GetDefaultSignInPolicy はテナントデフォルト sign-in policy を取得する。未設定なら空ルールを返す。
func GetDefaultSignInPolicy(ctx context.Context, deps SignInPolicyDeps) (*domain.TenantDefaultSignInPolicy, error) {
	tenantID := tenantports.TenantID(ctx)
	if deps.DefaultRepo == nil {
		return EmptyDefaultSignInPolicy(tenantID, time.Now().UTC()), nil
	}
	policy, err := deps.DefaultRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return EmptyDefaultSignInPolicy(tenantID, time.Now().UTC()), nil
	}
	if policy.Rules == nil {
		policy.Rules = []domain.SignInRule{}
	}
	return policy, nil
}

type UpdateDefaultSignInPolicyInput struct {
	ActorUserID string
	Rules       []domain.SignInRule
	Now         time.Time
}

// UpdateDefaultSignInPolicy はテナントデフォルト sign-in policy を置き換える。
// 空 rules で保存すればデフォルトは allow-all に戻る。
func UpdateDefaultSignInPolicy(ctx context.Context, deps SignInPolicyDeps, in UpdateDefaultSignInPolicyInput) (*domain.TenantDefaultSignInPolicy, error) {
	tenantID := tenantports.TenantID(ctx)
	rules := slices.Clone(in.Rules)
	if err := ValidateSignInPolicyRulesAt(rules, in.Now); err != nil {
		return nil, err
	}
	now := adminNow(in.Now)
	policy := &domain.TenantDefaultSignInPolicy{
		TenantID: tenantID, Rules: rules, CreatedAt: now, UpdatedAt: now,
	}
	if deps.DefaultRepo != nil {
		existing, err := deps.DefaultRepo.Get(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		if existing != nil && !existing.CreatedAt.IsZero() {
			policy.CreatedAt = existing.CreatedAt
		}
		if err := deps.DefaultRepo.Save(ctx, policy); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &domain.TenantDefaultSignInPolicyUpdated{
		At: policy.UpdatedAt, TenantID: tenantID, ActorUserID: in.ActorUserID,
	})
	return policy, nil
}

// サインインポリシーの評価は入力だけで決まる計算なので、公開パッケージの domain に置く。
// ここではモジュールの内部の呼び出し側のために同じ名前で参照できるようにする。
var (
	EffectiveSignInRules         = domain.EffectiveSignInRules
	EffectivePolicyForEvaluation = domain.EffectivePolicyForEvaluation
	TrustedDeviceAllowedByRules  = domain.TrustedDeviceAllowedByRules
	MfaEnrollmentPolicyFromRules = domain.MfaEnrollmentPolicyFromRules
	UpcomingMfaEnforcementStart  = domain.UpcomingMfaEnforcementStart
)

// signInSettings は 1 ポリシー分の設定を強度比較のために平坦化した表現。
// UI は常に単一ルールを書き込むため、有効な最初のルールから読み取る。
type signInSettings struct {
	requireMfa         bool
	allowTrustedDevice bool
	reauth             *int
	cidrs              []string
}

func settingsFromRules(rules []domain.SignInRule) signInSettings {
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		return signInSettings{
			requireMfa:         rule.RequiredAuthn.Strength == domain.RequiredAuthnMfa,
			allowTrustedDevice: rule.TrustedDeviceAllowed(),
			reauth:             rule.Condition.ReauthMaxAgeSeconds,
			cidrs:              rule.Condition.NetworkAllowCIDRs,
		}
	}
	return signInSettings{}
}

// AppPolicyWeakerThanDefault はアプリ独自ポリシーがデフォルトより弱いかを返す (UI 警告用)。
// アプリが独自ポリシーを持たなければ (=デフォルトを適用) 常に false。弱さは認証強度・再認証時間・
// 許可ネットワークの 3 項目のいずれかで判定する。
func AppPolicyWeakerThanDefault(def *domain.TenantDefaultSignInPolicy, app *domain.AppSignInPolicy) bool {
	if !app.Configured() {
		return false
	}
	var defSettings signInSettings
	if def != nil {
		defSettings = settingsFromRules(def.Rules)
	}
	appSettings := settingsFromRules(app.Rules)

	// 認証強度: デフォルトが MFA 必須なのにアプリがそうでない。
	if defSettings.requireMfa && !appSettings.requireMfa {
		return true
	}
	// 記憶済みデバイスによる充足: デフォルトが毎回 MFA を求めるのにアプリが許している。
	if defSettings.requireMfa && appSettings.requireMfa &&
		!defSettings.allowTrustedDevice && appSettings.allowTrustedDevice {
		return true
	}
	// 再認証時間: デフォルトに上限があるのにアプリが未設定、または上限を延ばしている。
	if defSettings.reauth != nil {
		if appSettings.reauth == nil || *appSettings.reauth > *defSettings.reauth {
			return true
		}
	}
	// 許可ネットワーク: デフォルトが制限しているのにアプリが未制限、またはデフォルト外を許可する。
	if len(defSettings.cidrs) > 0 {
		if len(appSettings.cidrs) == 0 {
			return true
		}
		for _, entry := range appSettings.cidrs {
			if !slices.Contains(defSettings.cidrs, entry) {
				return true
			}
		}
	}
	return false
}

func ValidateSignInPolicyRules(rules []domain.SignInRule) error {
	return ValidateSignInPolicyRulesAt(rules, time.Time{})
}

func ValidateSignInPolicyRulesAt(rules []domain.SignInRule, now time.Time) error {
	seen := map[string]struct{}{}
	for i := range rules {
		r := &rules[i]
		r.RuleID = strings.TrimSpace(r.RuleID)
		r.Name = strings.TrimSpace(r.Name)
		if r.RequiredAuthn.Strength == "" {
			r.RequiredAuthn.Strength = domain.RequiredAuthnPassword
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
		if r.MfaEnrollment != nil {
			if r.RequiredAuthn.Strength != domain.RequiredAuthnMfa ||
				r.MfaEnrollment.EnforcementStartAt == nil ||
				r.MfaEnrollment.GracePeriodSeconds == nil ||
				*r.MfaEnrollment.GracePeriodSeconds <= 0 {
				return ErrInvalidSignInPolicy
			}
			if !now.IsZero() && r.MfaEnrollment.EnforcementStartAt.Before(now) {
				return ErrInvalidSignInPolicy
			}
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

type (
	PolicyDecision   = domain.PolicyDecision
	PolicyEvaluation = domain.PolicyEvaluation
)

const (
	PolicyAllow          = domain.PolicyAllow
	PolicyDeny           = domain.PolicyDeny
	PolicyStepUpRequired = domain.PolicyStepUpRequired
)

// EvaluateSignInPolicy は domain.EvaluateSignInPolicy を、零値の時刻を現在時刻で補って呼ぶ。
func EvaluateSignInPolicy(policy *domain.AppSignInPolicy, authn domain.SignInAuthentication, clientIP string, now time.Time) PolicyEvaluation {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return domain.EvaluateSignInPolicy(policy, authn, clientIP, now)
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
