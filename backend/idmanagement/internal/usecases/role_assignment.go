package usecases

// 予約ロールの割当て規則。字句の正規化 (NormalizeRoles) とは別の関心事として置く。
// 正規化は所属テナントを知らず、CSV の取り込みでも同じ字句の処理を共有する。

import (
	"errors"
	"slices"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// ReservedRoleSystemAdmin は制御面 User のための予約ロール名。
// 予約する文字列はこれだけで、他のロール名はテナントが自由に決める文字列として扱う。
const ReservedRoleSystemAdmin = "system_admin"

// ErrReservedRole は予約ロールを保持できない対象へ新しく割り当てようとした場合に返る。
var ErrReservedRole = errors.New("role is reserved for control plane users")

// ValidateRoleAssignment は正規化済みのロール集合が対象へ保存できるかを返す。
//
// tenantID は対象が所属するテナントであり、要求元のテナントではない。判定の根拠は
// 操作者が持つロールではなく対象側に置く。
//
// current は対象が現在保持しているロール、next は保存しようとするロールである。判定は
// next の絶対集合ではなく、この書き込みが新しく加える予約ロールだけを見る。絶対集合で
// 判定すると、`system_admin` をテナント固有の文字列として使ってきた環境で、無編集の
// エクスポートを再適用すると全行 unchanged になるという CSV の往復不変条件が壊れる。
func ValidateRoleAssignment(tenantID string, current, next []string) error {
	if !slices.Contains(next, ReservedRoleSystemAdmin) {
		return nil
	}
	if slices.Contains(current, ReservedRoleSystemAdmin) {
		return nil
	}
	if tenantID != tenancydomain.DefaultTenantID {
		return ErrReservedRole
	}
	return nil
}
