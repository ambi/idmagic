// Package domain はアカウントのセキュリティ通知の語彙を持つ (wi-90)。どのイベントが
// どの種別の通知になるか、どの種別を本人が止められるか、宛先の sub をイベントの
// どの項目から取るかは、すべてここのカタログが決める。
//
// カタログの鍵はイベントの Go の型ではなく `EventType()` の文字列である。ディスパッチャーは
// 監査の射影と同じワイヤ表現の上で動くため、この package は Authentication・IdManagement・
// 信頼済みデバイスのどの domain package にも依存しない。
package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// Category はセキュリティ通知の種別。SCL `SecurityNotificationCategory` の双子定義。
type Category string

const (
	CategoryNewDeviceSignIn  Category = "new_device_sign_in"
	CategoryCredentialChange Category = "credential_change"
	CategoryMfaChange        Category = "mfa_change"
	CategoryContactChange    Category = "contact_change"
	CategorySessionRevoked   Category = "session_revoked"
	CategoryImpersonation    Category = "impersonation"
)

// Categories は全種別を安定した並びで返す。アカウント API の一覧と設定の正規化が
// 同じ並びを使うので、UI は種別の一覧をコード側に持たなくてよい。
func Categories() []Category {
	return []Category{
		CategoryNewDeviceSignIn,
		CategoryCredentialChange,
		CategoryMfaChange,
		CategoryContactChange,
		CategorySessionRevoked,
		CategoryImpersonation,
	}
}

func (c Category) Valid() bool { return slices.Contains(Categories(), c) }

// mandatoryCategories は本人が受信を止められない種別。乗っ取りの直後に攻撃者が最初に
// 消すのは通知であり、通知を消せることは通知が無いことと変わらない。
var mandatoryCategories = []Category{
	CategoryCredentialChange,
	CategoryMfaChange,
	CategoryContactChange,
	CategoryImpersonation,
}

func (c Category) Mandatory() bool { return slices.Contains(mandatoryCategories, c) }

var (
	// ErrUnknownCategory は仕様に無い種別を指定された。
	ErrUnknownCategory = errors.New("unknown security notification category")
	// ErrMandatoryCategory は本人が止められない種別を止めようとした。
	ErrMandatoryCategory = errors.New("this security notification category cannot be disabled")
	// ErrEmptyUserID は設定の所有者が空。
	ErrEmptyUserID = errors.New("security notification preferences need an owner")
)

// Trigger はカタログの 1 行。1 つのイベント種別が、どの通知の種別になり、宛先の sub を
// payload のどの項目から取り、本文の説明を locale ごとに何と書くかを持つ。
type Trigger struct {
	// Category はこのイベントが属する通知の種別。
	Category Category
	// RecipientField は「通知を受け取る本人の sub」を持つ payload の項目名。ほとんどの
	// イベントは userId だが、なりすましだけは操作した admin ではなく対象本人を指す。
	RecipientField string
	// Descriptions は locale ごとの短い説明。本文の「内容」に入る唯一の可変部分であり、
	// 識別子や生の値ではなく人が読む語にする。
	Descriptions map[string]string
	// Guard は payload を見て通知するかどうかを決める追加条件。nil なら常に通知する。
	Guard func(payload map[string]any) bool
}

// Applies は payload がこの行の追加条件を満たすかを返す。
func (t Trigger) Applies(payload map[string]any) bool {
	if t.Guard == nil {
		return true
	}
	return t.Guard(payload)
}

// Description は locale の説明を返す。持たない locale では en へ落とす。
func (t Trigger) Description(locale string) string {
	if text, ok := t.Descriptions[locale]; ok && text != "" {
		return text
	}
	return t.Descriptions["en"]
}

// explicitSessionRevocations はセッションの終了のうち通知に値する理由。期限切れ・
// ログアウト・資格情報の変更に伴う自動失効まで通知すると、通知そのものが無視される。
// 資格情報の変更は credential_change / mfa_change 側で既に 1 通届いている。
var explicitSessionRevocations = []string{"self_revoke", "admin_revoke"}

// triggers はイベント種別から通知への対応表。ここに無いイベント種別は通知を生まない。
// ディスパッチャー自身が発行する AccountSecurityNotificationSent も無いので、通知が
// 通知を呼ぶことはない。認証の失敗を含めないのは、攻撃者が量を制御できる事象を
// メールの送信量に直結させないためである。
var triggers = map[string]Trigger{
	"UserAuthenticated": {
		Category:       CategoryNewDeviceSignIn,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "新しいデバイスからのサインイン",
			"en": "Sign-in from a new device",
		},
	},
	"PasswordChanged": {
		Category:       CategoryCredentialChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "パスワードの変更",
			"en": "Password changed",
		},
	},
	"MfaFactorEnrolled": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "認証アプリの登録",
			"en": "Authenticator app added",
		},
	},
	"MfaFactorRemoved": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "認証アプリの解除",
			"en": "Authenticator app removed",
		},
	},
	"WebAuthnCredentialRegistered": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "パスキーの登録",
			"en": "Passkey added",
		},
	},
	"WebAuthnCredentialRemoved": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "パスキーの解除",
			"en": "Passkey removed",
		},
	},
	"RecoveryCodesGenerated": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "復旧コードの生成",
			"en": "Recovery codes generated",
		},
	},
	"RecoveryCodesRevoked": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "復旧コードの失効",
			"en": "Recovery codes revoked",
		},
	},
	"AuthenticatorResetCompleted": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "管理者による認証器のリセット",
			"en": "Authenticators reset by an administrator",
		},
	},
	"TrustedDeviceRegistered": {
		Category:       CategoryMfaChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "サインイン時に第二要素を省略するデバイスの記憶",
			"en": "A device was remembered to skip the second factor",
		},
	},
	"EmailChangeRequested": {
		Category:       CategoryContactChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "メールアドレスの変更の要求",
			"en": "Email address change requested",
		},
	},
	"EmailChanged": {
		Category:       CategoryContactChange,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "メールアドレスの変更",
			"en": "Email address changed",
		},
	},
	"SessionEnded": {
		Category:       CategorySessionRevoked,
		RecipientField: "userId",
		Descriptions: map[string]string{
			"ja": "サインイン中のセッションの失効",
			"en": "A signed-in session was revoked",
		},
		Guard: func(payload map[string]any) bool {
			reason, _ := payload["reason"].(string)
			return slices.Contains(explicitSessionRevocations, reason)
		},
	},
	"SessionImpersonationStarted": {
		Category:       CategoryImpersonation,
		RecipientField: "targetUserId",
		Descriptions: map[string]string{
			"ja": "管理者があなたとして操作を開始",
			"en": "An administrator started acting as you",
		},
	},
}

// TriggerFor はイベント種別に対応するカタログの行を返す。
func TriggerFor(eventType string) (Trigger, bool) {
	trigger, ok := triggers[eventType]
	return trigger, ok
}

// TriggerEventTypes はカタログに載る全イベント種別を並びを固定して返す (テストと
// 監査用)。
func TriggerEventTypes() []string {
	types := make([]string, 0, len(triggers))
	for eventType := range triggers {
		types = append(types, eventType)
	}
	slices.Sort(types)
	return types
}

// Preferences は本人によるセキュリティ通知の受信設定。SCL `NotificationPreference` の
// 双子定義。保持するのは「無効にした種別」だけなので、後から種別が増えても既存の設定は
// 有効のまま引き継がれる。ゼロ値は「すべて有効」であり、行が無いことと同じ意味になる。
type Preferences struct {
	UserID    string
	Disabled  []Category
	UpdatedAt time.Time
}

// Allows はその種別の通知を送ってよいかを返す。必須の種別は Disabled に入り得ないが、
// 保存済みの行が (DB を直接書かれるなどで) 壊れていても必須の種別は止まらないよう、
// ここでも必須かどうかを見る。
func (p Preferences) Allows(category Category) bool {
	if category.Mandatory() {
		return true
	}
	return !slices.Contains(p.Disabled, category)
}

// NewPreferences は受信設定を検証して組み立てる。未知の種別と必須の種別は拒否し、
// 一部だけを適用することはしない。重複はカタログの並びに正規化して畳む。
func NewPreferences(userID string, disabled []Category, now time.Time) (Preferences, error) {
	if userID == "" {
		return Preferences{}, ErrEmptyUserID
	}
	for _, category := range disabled {
		if !category.Valid() {
			return Preferences{}, fmt.Errorf("%w: %q", ErrUnknownCategory, category)
		}
		if category.Mandatory() {
			return Preferences{}, fmt.Errorf("%w: %q", ErrMandatoryCategory, category)
		}
	}
	normalized := make([]Category, 0, len(disabled))
	for _, category := range Categories() {
		if slices.Contains(disabled, category) {
			normalized = append(normalized, category)
		}
	}
	return Preferences{UserID: userID, Disabled: normalized, UpdatedAt: now.UTC()}, nil
}
