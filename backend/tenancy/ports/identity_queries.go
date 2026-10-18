package ports

import "context"

// DynamicGroupAttributeReferences は参照保護に必要な属性キーだけを返す。
type DynamicGroupAttributeReferences func(ctx context.Context, tenantID string) ([][]string, error)

type TestNotificationActor struct {
	Email       string
	DisplayName string
}

// TestNotificationActorReader は操作者本人の表示名と検証済みのメールを返す。存在しなければ nil を返す。
type TestNotificationActorReader func(ctx context.Context, sub string) (*TestNotificationActor, error)
