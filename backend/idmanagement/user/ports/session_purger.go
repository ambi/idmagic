package ports

import "context"

// SessionPurger は、User を匿名化するときに、その User のログインセッションをすべて消す。
// ログインセッションは Authentication が所有し、組み立て地点が結ぶ。
type SessionPurger interface {
	DeleteAllForSub(ctx context.Context, sub string) error
}
