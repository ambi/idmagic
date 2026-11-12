package ports

import "context"

// PasswordDiagnosticGate は本番のアカウントと IP の認証試行の制限を診断でも使う。
type PasswordDiagnosticGate interface {
	Check(ctx context.Context, username, ip string) (retryAfterSeconds int, err error)
	Record(ctx context.Context, username, ip string, accepted bool) error
}
