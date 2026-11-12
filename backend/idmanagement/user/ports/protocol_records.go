package ports

import (
	"context"
	"time"
)

// UserRecordsPurger は、完全削除で一つの保存先にある本人の記録をすべて消す。
// 保存先ごとの確定と、途中失敗後の再実行を維持する。
type UserRecordsPurger interface {
	DeleteAllForSub(ctx context.Context, sub string) error
}

// ConsentPurger は、完全削除で本人の同意と Agent への代行をすべて消し、消した行のうち撤回していない
// 行の件数を返す。その件数だけ consents の使用量を減らす（REQ-IDMANAGEMENT-050）。
type ConsentPurger interface {
	DeleteAllForSub(ctx context.Context, sub string) (int, error)
}

// AccountConsent は本人データのエクスポートに含める同意の表示値である。
// 同意の保存型と状態遷移は OAuth2 が所有する。
type AccountConsent struct {
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	State     string    `json:"state"`
	GrantedAt time.Time `json:"granted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
