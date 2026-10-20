package ports

import "context"

// ClientLookup は、束縛先のクライアントが同じテナントに存在するかだけを調べる。
type ClientLookup interface {
	ClientExists(ctx context.Context, tenantID, clientID string) (bool, error)
}
