package ports

import "context"

// ClientLookup は、束縛先のクライアントが同じテナントに存在するかと、Agent に
// 束縛できる認証方式かを調べる。認証方式の列挙と許可集合は OAuth2 が持ち、
// IdManagement は判定の結果だけを受け取る。
type ClientLookup interface {
	// LookupBindableClient は、クライアントが存在するかと、要求ごとに鍵の所有を
	// 証明する認証方式か (provesKeyPossession) を返す。存在しなければ両方 false。
	LookupBindableClient(ctx context.Context, tenantID, clientID string) (exists, provesKeyPossession bool, err error)
}
