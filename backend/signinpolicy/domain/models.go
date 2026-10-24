package domain

import "time"

// RequiredAuthnStrength はサインインポリシーが要求する認証強度。制約された選択肢を
// 内部の acr / amr へ写像する。Password は追加要求なし、Mfa は第二要素必須。
type RequiredAuthnStrength string

const (
	RequiredAuthnPassword RequiredAuthnStrength = "Password"
	RequiredAuthnMfa      RequiredAuthnStrength = "Mfa"
)

func (s RequiredAuthnStrength) Valid() bool {
	switch s {
	case RequiredAuthnPassword, RequiredAuthnMfa:
		return true
	}
	return false
}

// RequiredAuthnLevel は Application sign-in policy が要求する認証強度 (制約された選択肢)。
type RequiredAuthnLevel struct {
	Strength RequiredAuthnStrength `json:"strength"`
}

// AccessCondition は Application sign-in policy の静的条件。初期実装で実際に評価できる
// 構造化条件のみを持つ。旧 network / device の自由入力は廃止した。
type AccessCondition struct {
	// NetworkAllowCIDRs は許可するクライアント IP の CIDR 一覧。空なら制限なし。
	NetworkAllowCIDRs   []string `json:"network_allow_cidrs,omitempty"`
	ReauthMaxAgeSeconds *int     `json:"reauth_max_age_seconds,omitempty"`
}

// SignInRule は Application sign-in policy の 1 ルール。
type SignInRule struct {
	RuleID        string               `json:"rule_id"`
	Name          string               `json:"name"`
	Enabled       bool                 `json:"enabled"`
	RequiredAuthn RequiredAuthnLevel   `json:"required_authn"`
	Condition     AccessCondition      `json:"condition"`
	MfaEnrollment *MfaEnrollmentPolicy `json:"mfa_enrollment,omitempty"`
	// AllowTrustedDevice は required_authn=Mfa のとき、記憶済みの信頼済みデバイス
	// (amr に tdev) で MFA 要件を満たしてよいかどうか。nil は既定の true と読む:
	// この列を知らずに保存された既存ルールが、暗黙に「毎回 MFA」へ倒れないようにする。
	// テナントが信頼済みデバイスを有効にしていなければ、そもそも発行も評価もされない。
	AllowTrustedDevice *bool `json:"allow_trusted_device,omitempty"`
}

// TrustedDeviceAllowed は AllowTrustedDevice のゼロ値 (nil) を既定の true として読む。
func (r SignInRule) TrustedDeviceAllowed() bool {
	return r.AllowTrustedDevice == nil || *r.AllowTrustedDevice
}

type MfaEnrollmentPolicy struct {
	EnforcementStartAt *time.Time `json:"enforcement_start_at,omitempty"`
	GracePeriodSeconds *int       `json:"grace_period_seconds,omitempty"`
	AllowAdminBypass   bool       `json:"allow_admin_bypass"`
}

// AppSignInPolicy は Application ごとの federation 開始条件。
// 設定されるとテナントデフォルトを上書き (置換) する。
type AppSignInPolicy struct {
	TenantID      string       `json:"tenant_id"`
	ApplicationID string       `json:"application_id"`
	Rules         []SignInRule `json:"rules"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// TenantDefaultSignInPolicy はテナントデフォルトの federation 開始条件。
// 有効な上書きを持たない Application と IdMagic 自身のサインインに適用する。
type TenantDefaultSignInPolicy struct {
	TenantID  string       `json:"tenant_id"`
	Rules     []SignInRule `json:"rules"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// AppSignInPolicyUpdated は Application の sign-in policy を更新した event。
type AppSignInPolicyUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	ApplicationID string    `json:"applicationId"`
}

func (e *AppSignInPolicyUpdated) EventType() string     { return "AppSignInPolicyUpdated" }
func (e *AppSignInPolicyUpdated) OccurredAt() time.Time { return e.At }

// TenantDefaultSignInPolicyUpdated はテナント既定 sign-in policy を更新した event。
type TenantDefaultSignInPolicyUpdated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
}

func (e *TenantDefaultSignInPolicyUpdated) EventType() string {
	return "TenantDefaultSignInPolicyUpdated"
}
func (e *TenantDefaultSignInPolicyUpdated) OccurredAt() time.Time { return e.At }
