package domain

// パスワードの製品共通の既定値。設定 API の表示と上書きの検証、Authentication が
// 解決するポリシーが同じ値を使う。有効期限は既定で無効にし、テナントが明示して有効にする。
const (
	PasswordPolicyMinLength    = 12
	PasswordPolicyMaxLength    = 128
	PasswordPolicyHistoryDepth = 5
	PasswordPolicyMaxAgeDays   = 0
)
