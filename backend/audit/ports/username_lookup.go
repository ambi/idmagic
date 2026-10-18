package ports

import "context"

// UsernameLookup は監査の経路が決めたテナント内で username を User ID へ解決する。
// 存在しない User と空の ID を区別し、提供側の保存型と書き込み操作を公開しない。
type UsernameLookup func(ctx context.Context, tenantID, username string) (userID string, found bool, err error)
