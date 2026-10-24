package ports

import "context"

// SignInPolicyDeleter はアプリの削除が必要とする、ポリシーの削除だけの契約である。
type SignInPolicyDeleter interface {
	Delete(ctx context.Context, tenantID, applicationID string) error
}
