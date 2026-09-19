package usecases

// 予約ロールの割当て規則。字句の正規化 (NormalizeRoles) とは別の関心事として置く。
// 正規化は Agent を含む全対象で共有しており、対象種別も所属テナントも知らないため、
// 製品上の割当て規則をそこへ混ぜると Agent の規則が User の規則に飲まれる。

import (
	"errors"
	"slices"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// RoleAssignmentTarget は予約ロールの割当て可否を決める対象の種類。
type RoleAssignmentTarget string

const (
	RoleTargetUser  RoleAssignmentTarget = "user"
	RoleTargetGroup RoleAssignmentTarget = "group"
	RoleTargetAgent RoleAssignmentTarget = "agent"
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
func ValidateRoleAssignment(
	target RoleAssignmentTarget, tenantID string, current, next []string,
) error {
	if !slices.Contains(next, ReservedRoleSystemAdmin) {
		return nil
	}
	if slices.Contains(current, ReservedRoleSystemAdmin) {
		return nil
	}
	if target == RoleTargetAgent {
		return ErrReservedRole
	}
	if tenantID != tenancydomain.DefaultTenantID {
		return ErrReservedRole
	}
	return nil
}
