// Package domain は IdManagement bounded context の業務型を所有する
// (wi-178, ADR-089/ADR-093)。
package domain

// AgentStatus は Agent の運用状態 (ADR-048)。Active 通常稼働 / Disabled 可逆な運用停止 /
// Killed 緊急停止 (一方向終端)。Active 以外は新規トークンを発行しない (fail-closed)。
type AgentStatus string

const (
	AgentStatusActive   AgentStatus = "active"
	AgentStatusDisabled AgentStatus = "disabled"
	AgentStatusKilled   AgentStatus = "killed"
)

func (s AgentStatus) Valid() bool {
	switch s {
	case AgentStatusActive, AgentStatusDisabled, AgentStatusKilled:
		return true
	}
	return false
}

// AgentKind は Agent の自律性区分 (ADR-048)。Autonomous 自律実行 / Supervised 監督下実行。
type AgentKind string

const (
	AgentKindAutonomous AgentKind = "autonomous"
	AgentKindSupervised AgentKind = "supervised"
)

func (k AgentKind) Valid() bool {
	return k == AgentKindAutonomous || k == AgentKindSupervised
}

// ===============================================================
// ユーザー属性拡張 (wi-19 / ADR-039 / ADR-040)
// ===============================================================

// UserStatus は UserLifecycle.status。User の運用状態の **唯一の真実** で、
// 状態機械 UserLifecycle (states セクション) と一致する。Active / Disabled /
// Deleted が状態機械の状態、Locked / Staged / Suspended は Okta lifecycle_state /
// Keycloak 相当の追加状態。Active 以外は認証不可。Deleted は終端 (tombstone)。
// 「いつ遷移したか」は監査イベント (UserDisabled / UserDeleted 等) と
// UserLifecycle.status_changed_at が持つので、専用の disabled_at / deleted_at は持たない。
type UserStatus string

const (
	UserStatusActive          UserStatus = "active"
	UserStatusDisabled        UserStatus = "disabled"
	UserStatusPendingDeletion UserStatus = "pending_deletion"
	UserStatusDeleted         UserStatus = "deleted"
	UserStatusLocked          UserStatus = "locked"
	UserStatusStaged          UserStatus = "staged"
	UserStatusSuspended       UserStatus = "suspended"
)

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusActive, UserStatusDisabled, UserStatusPendingDeletion, UserStatusDeleted,
		UserStatusLocked, UserStatusStaged, UserStatusSuspended:
		return true
	}
	return false
}

// RequiredAction は次回ログイン時にユーザへ強制するアクション (Keycloak Required Actions 相当)。
type RequiredAction string

const (
	RequiredActionUpdatePassword     RequiredAction = "update_password"
	RequiredActionVerifyEmail        RequiredAction = "verify_email"
	RequiredActionConfigureTOTP      RequiredAction = "configure_totp"
	RequiredActionUpdateProfile      RequiredAction = "update_profile"
	RequiredActionTermsAndConditions RequiredAction = "terms_and_conditions"
)

func (a RequiredAction) Valid() bool {
	switch a {
	case RequiredActionUpdatePassword, RequiredActionVerifyEmail,
		RequiredActionConfigureTOTP, RequiredActionUpdateProfile,
		RequiredActionTermsAndConditions:
		return true
	}
	return false
}

// AttributeType は属性値の sum type discriminator (ADR-040)。OIDC 標準クレームの
// 組み込み属性と tenant 定義カスタム属性の両方で共通に使う。
type AttributeType string

const (
	AttributeTypeString      AttributeType = "string"
	AttributeTypeNumber      AttributeType = "number"
	AttributeTypeBoolean     AttributeType = "boolean"
	AttributeTypeDate        AttributeType = "date"
	AttributeTypeStringArray AttributeType = "string_array"
)

func (t AttributeType) Valid() bool {
	switch t {
	case AttributeTypeString, AttributeTypeNumber, AttributeTypeBoolean,
		AttributeTypeDate, AttributeTypeStringArray:
		return true
	}
	return false
}

// AttrVisibility は属性の開示範囲 (ADR-040)。claim_exposed のみ UserInfo / ID Token に出せる。
type AttrVisibility string

const (
	AttrVisibilityPrivate       AttrVisibility = "private"
	AttrVisibilitySelfReadable  AttrVisibility = "self_readable"
	AttrVisibilityAdminReadable AttrVisibility = "admin_readable"
	AttrVisibilityClaimExposed  AttrVisibility = "claim_exposed"
)

func (v AttrVisibility) Valid() bool {
	switch v {
	case AttrVisibilityPrivate, AttrVisibilitySelfReadable,
		AttrVisibilityAdminReadable, AttrVisibilityClaimExposed:
		return true
	}
	return false
}
