// Package spec: SCL → Go バインディング。
//
// 仕様本体（language-agnostic）は spec/scl.yaml。
// 本ファイルはランタイム検証のための Go バインディング。SCL を変更したら本ファイル
// も合わせて更新する。乖離は coherence test で検出する。
package spec

// ===============================================================
// SCL `models` セクションの enum を Go の typed string で表す。
// ワイヤ形式（snake_case）は vocabulary[].aliases[0] と同じ。
// ===============================================================

type ClientType string

const (
	ClientPublic       ClientType = "public"
	ClientConfidential ClientType = "confidential"
)

func (c ClientType) Valid() bool { return c == ClientPublic || c == ClientConfidential }

type GrantType string

const (
	GrantAuthorizationCode GrantType = "authorization_code"
	GrantRefreshToken      GrantType = "refresh_token"
	GrantClientCredentials GrantType = "client_credentials"
	GrantDeviceCode        GrantType = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTokenExchange     GrantType = "urn:ietf:params:oauth:grant-type:token-exchange"
)

func (g GrantType) Valid() bool {
	switch g {
	case GrantAuthorizationCode, GrantRefreshToken, GrantClientCredentials, GrantDeviceCode, GrantTokenExchange:
		return true
	}
	return false
}

type ResponseType string

const ResponseTypeCode ResponseType = "code"

func (r ResponseType) Valid() bool { return r == ResponseTypeCode }

type SignatureAlgorithm string

const (
	SigAlgPS256 SignatureAlgorithm = "PS256"
	SigAlgES256 SignatureAlgorithm = "ES256"
)

func (s SignatureAlgorithm) Valid() bool { return s == SigAlgPS256 || s == SigAlgES256 }

// KeyProvider は署名鍵マテリアルの保管場所と署名の実行主体。
// Local / Postgres は private key をプロセス内に保持する dev/test 用、
// VaultTransit は private key を Vault 内に保持し署名を Vault へ委譲する本番用。
type KeyProvider string

const (
	KeyProviderLocal        KeyProvider = "Local"
	KeyProviderPostgres     KeyProvider = "Postgres"
	KeyProviderVaultTransit KeyProvider = "VaultTransit"
)

func (p KeyProvider) Valid() bool {
	return p == KeyProviderLocal || p == KeyProviderPostgres || p == KeyProviderVaultTransit
}

// KeyUsage は鍵の用途。現状は OAuth2/OIDC の JWT 署名のみ。
type KeyUsage string

const KeyUsageSigning KeyUsage = "Signing"

func (u KeyUsage) Valid() bool { return u == KeyUsageSigning }

type CodeChallengeMethod string

const CodeChallengeMethodS256 CodeChallengeMethod = "S256"

func (c CodeChallengeMethod) Valid() bool { return c == CodeChallengeMethodS256 }

type MfaFactorType string

const (
	MfaFactorTOTP     MfaFactorType = "totp"
	MfaFactorWebAuthn MfaFactorType = "webauthn"
	MfaFactorHWK      MfaFactorType = "hwk"
	MfaFactorSWK      MfaFactorType = "swk"
)

func (m MfaFactorType) Valid() bool {
	switch m {
	case MfaFactorTOTP, MfaFactorWebAuthn, MfaFactorHWK, MfaFactorSWK:
		return true
	}
	return false
}

// WebAuthnTransport は authenticator の接続方式のヒント (WebAuthn Level 3 §5.8.4)。
type WebAuthnTransport string

const (
	WebAuthnTransportUSB      WebAuthnTransport = "usb"
	WebAuthnTransportNFC      WebAuthnTransport = "nfc"
	WebAuthnTransportBLE      WebAuthnTransport = "ble"
	WebAuthnTransportInternal WebAuthnTransport = "internal"
	WebAuthnTransportHybrid   WebAuthnTransport = "hybrid"
)

func (t WebAuthnTransport) Valid() bool {
	switch t {
	case WebAuthnTransportUSB, WebAuthnTransportNFC, WebAuthnTransportBLE,
		WebAuthnTransportInternal, WebAuthnTransportHybrid:
		return true
	}
	return false
}

// ===============================================================
// 状態機械 (SCL state_machines)
// ===============================================================

type AuthorizationCodeFlowState string

const (
	AuthFlowReceived              AuthorizationCodeFlowState = "received"
	AuthFlowAuthenticationPending AuthorizationCodeFlowState = "authentication_pending"
	AuthFlowAuthenticated         AuthorizationCodeFlowState = "authenticated"
	AuthFlowConsentPending        AuthorizationCodeFlowState = "consent_pending"
	AuthFlowConsented             AuthorizationCodeFlowState = "consented"
	AuthFlowCodeIssued            AuthorizationCodeFlowState = "code_issued"
	AuthFlowExchanged             AuthorizationCodeFlowState = "exchanged"
	AuthFlowRejected              AuthorizationCodeFlowState = "rejected"
	AuthFlowExpired               AuthorizationCodeFlowState = "expired"
)

func (s AuthorizationCodeFlowState) Valid() bool {
	switch s {
	case AuthFlowReceived, AuthFlowAuthenticationPending, AuthFlowAuthenticated,
		AuthFlowConsentPending, AuthFlowConsented, AuthFlowCodeIssued,
		AuthFlowExchanged, AuthFlowRejected, AuthFlowExpired:
		return true
	}
	return false
}

type AuthorizationCodeRecordState string

const (
	AuthCodeRecordIssued   AuthorizationCodeRecordState = "issued"
	AuthCodeRecordRedeemed AuthorizationCodeRecordState = "redeemed"
	AuthCodeRecordExpired  AuthorizationCodeRecordState = "expired"
)

func (s AuthorizationCodeRecordState) Valid() bool {
	switch s {
	case AuthCodeRecordIssued, AuthCodeRecordRedeemed, AuthCodeRecordExpired:
		return true
	}
	return false
}

// SessionEndReason は LoginSession 終了の理由 (wi-20)。self_revoke / admin_revoke は
// 明示的なセッション失効、それ以外は自動失効・ライフサイクル起因。
type SessionEndReason string

const (
	SessionEndLogout         SessionEndReason = "logout"
	SessionEndIdle           SessionEndReason = "idle"
	SessionEndAbsolute       SessionEndReason = "absolute"
	SessionEndSelfRevoke     SessionEndReason = "self_revoke"
	SessionEndAdminRevoke    SessionEndReason = "admin_revoke"
	SessionEndPasswordChange SessionEndReason = "password_change"
	SessionEndMfaChange      SessionEndReason = "mfa_change"
	SessionEndOther          SessionEndReason = "other"
)

func (r SessionEndReason) Valid() bool {
	switch r {
	case SessionEndLogout, SessionEndIdle, SessionEndAbsolute, SessionEndSelfRevoke,
		SessionEndAdminRevoke, SessionEndPasswordChange, SessionEndMfaChange, SessionEndOther:
		return true
	}
	return false
}

type DeviceCodeFlowState string

const (
	DeviceFlowIssued          DeviceCodeFlowState = "issued"
	DeviceFlowUserCodeEntered DeviceCodeFlowState = "user_code_entered"
	DeviceFlowApproved        DeviceCodeFlowState = "approved"
	DeviceFlowDenied          DeviceCodeFlowState = "denied"
	DeviceFlowExchanged       DeviceCodeFlowState = "exchanged"
	DeviceFlowExpired         DeviceCodeFlowState = "expired"
)

func (s DeviceCodeFlowState) Valid() bool {
	switch s {
	case DeviceFlowIssued, DeviceFlowUserCodeEntered,
		DeviceFlowApproved, DeviceFlowDenied, DeviceFlowExchanged, DeviceFlowExpired:
		return true
	}
	return false
}

// レスポンスモード（authorize エンドポイントから redirect_uri に code を運ぶ方式）
type ResponseMode string

const (
	ResponseModeQuery    ResponseMode = "query"
	ResponseModeFormPost ResponseMode = "form_post"
)

func (r ResponseMode) Valid() bool { return r == ResponseModeQuery || r == ResponseModeFormPost }

// SenderConstraint は DPoP / mTLS による proof-of-possession トークン拘束。
type SenderConstraintType string

const (
	SenderConstraintDPoP SenderConstraintType = "dpop"
	SenderConstraintMTLS SenderConstraintType = "mtls"
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusDisabled TenantStatus = "disabled"
)

func (s TenantStatus) Valid() bool {
	return s == TenantStatusActive || s == TenantStatusDisabled
}

// AgentStatus / AgentKind / UserStatus / RequiredAction / AttributeType / AttrVisibility は
// identitymanagement/domain へ移設した (wi-178, ADR-089/ADR-093)。
