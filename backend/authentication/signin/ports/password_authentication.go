package ports

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// PasswordRefusal は、パスワード認証が利用者を認証しなかった理由である。空なら認証した。
type PasswordRefusal string

const (
	PasswordRefusalNone PasswordRefusal = ""
	// PasswordRefusalThrottled は、アカウントか発信元の IP の試行の制限に達していることを表す。
	// パスワードは照合していない。
	PasswordRefusalThrottled PasswordRefusal = "throttled"
	// PasswordRefusalInvalidCredentials は、ユーザー名が未知か、パスワードが誤っていることを表す。
	PasswordRefusalInvalidCredentials PasswordRefusal = "invalid_credentials"
	// PasswordRefusalAccountDisabled は、正しいパスワードの User が Active でないことを表す。
	PasswordRefusalAccountDisabled PasswordRefusal = "account_disabled"
	// PasswordRefusalPasswordPolicy は、正しいパスワードが現在のパスワードのポリシーを満たさないことを表す。
	PasswordRefusalPasswordPolicy PasswordRefusal = "password_policy"
)

// PasswordAttempt は、ユーザー名とパスワードによる一回の認証の試みである。
type PasswordAttempt struct {
	Username string
	Password string
	// ClientIP は試行の制限と監査に使う発信元の IP である。空なら IP 単位の制限を数えない。
	ClientIP  string
	UserAgent string
}

// PasswordOutcome は、パスワード認証の結果である。
type PasswordOutcome struct {
	// User は、Refusal が空のときの認証した User である。
	User    *userdomain.User
	Refusal PasswordRefusal
	// RetryAfterSeconds は、Refusal が throttled のときの再試行までの秒数である。
	RetryAfterSeconds int
	// PasswordChangeRequired は、認証した User のパスワードが期限切れか、User が必須操作
	// update_password を持つことを表す。User は保存しない。
	PasswordChangeRequired bool
}

// PasswordAuthenticator は、セッションを作らずにユーザー名とパスワードで利用者を認証する。
// 試行の制限、失敗の記録と集約、認証の失敗のイベントはこの操作が行い、応答の形は呼び出し側が
// 拒否の理由から決める。Authentication が実装し、組み立て地点が結ぶ。
type PasswordAuthenticator interface {
	AuthenticatePassword(ctx context.Context, in PasswordAttempt) (PasswordOutcome, error)
}
