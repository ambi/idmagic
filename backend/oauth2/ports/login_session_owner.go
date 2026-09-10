package ports

import "context"

// LoginSessionOwnerLookup は sid が指す LoginSession の主体 (User ID) を返す。
//
// RP-Initiated Logout は id_token_hint の `sub` と `sid` を一組として受け取るが、
// その 2 つが同じ主体を指しているかは token だけでは分からない。ログアウト対象を
// 解決する側が、名指しされた LoginSession の持ち主を読んで照合する。
//
// LoginSession は Authentication の Aggregate なので、実装は HTTP 層のアダプターが
// SessionManager から作る。found が false は「その sid のセッションが無い」ことを表し、
// 失効させる対象が無いという意味なので拒否の理由にはならない。
type LoginSessionOwnerLookup interface {
	LoginSessionOwner(ctx context.Context, sid string) (userID string, found bool, err error)
}
