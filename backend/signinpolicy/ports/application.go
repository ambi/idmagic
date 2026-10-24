package ports

import "context"

// ApplicationLookup は、アプリの保存表現を受け取らずに同じテナントの実在だけを確認する。
type ApplicationLookup interface {
	ApplicationExists(ctx context.Context, tenantID, applicationID string) (bool, error)
}
