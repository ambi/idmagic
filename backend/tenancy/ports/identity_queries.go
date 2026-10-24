package ports

import "context"

type TestNotificationActor struct {
	Email       string
	DisplayName string
}

// TestNotificationActorReader は操作者本人の表示名と検証済みのメールを返す。存在しなければ nil を返す。
type TestNotificationActorReader func(ctx context.Context, sub string) (*TestNotificationActor, error)
