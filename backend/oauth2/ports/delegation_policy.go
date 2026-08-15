package ports

import "context"

// DelegationPolicyResolver は Token Exchange の act チェーンに許す深さの上限を
// テナントから解決する。OAuth2 は Tenancy の Repository を直接は知らず、この
// 1 メソッドの抽象だけを通して問い合わせる。
//
// 解決に失敗したときは error を返す。呼び出し側はシステム既定へ退避せず交換を
// 拒否する: 上書きは厳しい方向にのみ働くので、読めなかった上書きへ既定で退避すると、
// テナントが下げたはずの上限が黙って戻ってしまう。
type DelegationPolicyResolver interface {
	MaxDelegationDepth(ctx context.Context, tenantID string) (int, error)
}
