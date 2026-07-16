package spec

import (
	"encoding/json"
	"time"
)

// DomainEvent は集計対象のドメインイベント。TS の z.discriminatedUnion('type', ...) に対応。
// Go では tagged interface + 各イベント型構造体で表現する。OccurredAt は各構造体が公開フィールド At を持つ。

type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// EmailSent is a cross-context notification delivery primitive used by
// authentication and identity self-service flows.
type EmailSent struct {
	At        time.Time `json:"-"`
	ToHash    string    `json:"toHash"`
	Purpose   string    `json:"purpose"`
	Delivered bool      `json:"delivered"`
}

func (e *EmailSent) EventType() string     { return "EmailSent" }
func (e *EmailSent) OccurredAt() time.Time { return e.At }

// AuthenticationEventAggregated は、攻撃 (クレデンシャル試行洪水) 時に個別の
// AuthenticationFailed を 1 行ずつ書かず、(tenant, kind, keyHash, 5 分窓) の bucket に
// 集約したことを表す (wi-20 スライス 3 / ADR-029 の throttle 判定と keyHash を共有する)。
// 1 つの窓につき最初の 1 件だけ emit し、以後の増分は bucket store の count に積む。
// よって payload の Count は「emit 時点の値」で、実体は bucket store 側で伸び続ける。

// --- wi-44 / ADR-041: 認証ステップ・MFA・session・federation・impersonation の語彙。
// use case / 実 IdP 連携は各専用 WI。本 WI はイベント型とストレージ列のみを用意する。

// MfaFactorEnrolled は本人が self-service で MFA factor (現状 TOTP) を登録した (wi-21)。
// secret は audit に流さず、種別だけを残す。

// MfaFactorRemoved は本人が self-service で MFA factor を解除した (wi-21)。
// 解除は所持証明 (有効な TOTP コード) を伴う。

// WebAuthnCredentialRegistered は本人が self-service で WebAuthn / Passkey credential を
// 登録した (wi-26)。公開鍵や credential_id は audit に流さず、種別だけを残す。

// WebAuthnCredentialRemoved は本人が self-service で WebAuthn / Passkey credential を
// 解除した (wi-26, step-up 済み)。

// RecoveryCodesGenerated は本人が backup recovery code を生成 / 再生成した (wi-26, step-up
// 済み)。平文や hash は audit に流さず、発行件数のみ残す。

// RecoveryCodesRevoked は本人が backup recovery code を明示的に失効した (wi-26, step-up 済み)。

// StepUpRequested は本人が高 sensitivity 操作のための step-up 再認証を開始した
// (ADR-043 / wi-43)。利用可能な factor の提示要求であり、まだ再認証は成立していない。

// StepUpCompleted は step-up 再認証が成立した (ADR-043 / wi-43)。method は再認証に
// 使った factor (password | totp)。これ以降 recency 窓内は sensitive 操作を許可する。

// SessionEnded は LoginSession が終了した (wi-20)。self / admin の明示的な失効では
// ActorUserID が操作者、reason が self_revoke / admin_revoke になる。

// UserRequiredActionSet は admin が次回ログイン時の強制アクションを付与した
// (Keycloak Required Actions 相当 / wi-19)。値は監査に平文で残しても安全な enum。

// UserRequiredActionCleared は強制アクションが解除された。admin の明示解除のほか、
// 本人がパスワードを変更した結果 update_password が自動解除される場合も発火する
// (その場合 ActorUserID は対象本人の sub)。

// UserSoftDeleted は admin がユーザーを soft-delete (削除予約) した。status は
// PendingDeletion に遷移し、PII / Consent / RefreshToken / Session は温存される。

// UserRestored は admin が PendingDeletion のユーザーを Restore した。status は
// Active に戻り、PII / credential は温存されたままログインが再開する。

// UserDeleted は admin または猶予期間経過後の自動 purge がユーザーを Purge した。
// PII は anonymize 済みで、関連 aggregate は cascade 削除されている。

// AuthorizationDetailsRequested はクライアントが authorization_details を要求し、
// 登録 type 適合検証を通過したことを表す (RFC 9396 / ADR-050)。

// AuthorizationDetailsConsented は ResourceOwner が提示された authorization_details に
// 同意したことを表す。発行・交換の部分集合判定の基準となる (ADR-050)。

// AuthorizationDetailsRejected は authorization_details の検証・同意・ダウンスコープ
// 違反により要求を拒否したことを表す (fail-closed, ADR-050)。

func MarshalDomainEvent(event DomainEvent) ([]byte, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	var wire map[string]any
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, err
	}
	wire["type"] = event.EventType()
	wire["occurredAt"] = event.OccurredAt().UTC().Format(time.RFC3339Nano)
	return json.Marshal(wire)
}
