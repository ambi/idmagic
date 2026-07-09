package ports

import "context"

// TenantSaltStore はテナントごとの相関 salt を扱う (wi-145 / ADR-046)。
//
// salt は username / IP の相関ハッシュ (SaltedHash) に使い、tenant salt により cross-tenant で
// 集約しない (ADR-041 / ADR-046)。tenant scope は ctx (tenancy.TenantID) から解決し、初回取得時に
// 生成する (generate-on-first-use)。生成した salt は secret 扱いで、平文として外部へ出さない。
type TenantSaltStore interface {
	// GetSalt は ctx のテナントの salt を返す。未生成なら生成して永続化する。
	GetSalt(ctx context.Context) ([]byte, error)
}
