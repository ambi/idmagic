package usecases

import (
	"errors"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const otherTenant = "acme"

// 予約ロールの割当て規則を、対象の所属テナントで固定する。
//
//spec:covers REQ-IDMANAGEMENT-032: 所属テナントで `system_admin` の新規割当てを許可または拒否すること。
func TestValidateRoleAssignmentDecidesByTenant(t *testing.T) {
	for _, tt := range []struct {
		name     string
		tenantID string
		refused  bool
	}{
		{"制御面テナント", tenancydomain.DefaultTenantID, false},
		{"その他のテナント", otherTenant, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleAssignment(tt.tenantID, nil, []string{ReservedRoleSystemAdmin})
			if tt.refused && !errors.Is(err, ErrReservedRole) {
				t.Fatalf("err=%v, want ErrReservedRole", err)
			}
			if !tt.refused && err != nil {
				t.Fatalf("err=%v, want nil", err)
			}
		})
	}
}

// 予約ロール以外の名前は、どのテナントでも通る。
//
// 予約する文字列を `system_admin` だけに限る決定を固定する。ロール語彙全体を
// 閉じた列挙にした実装は、ここで落ちる。
func TestValidateRoleAssignmentLeavesTenantDefinedRolesAlone(t *testing.T) {
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, otherTenant} {
		roles := []string{"admin", "catalog:read", "system_admin_of_nothing", "systemadmin"}
		if err := ValidateRoleAssignment(tenantID, nil, roles); err != nil {
			t.Fatalf("tenant=%s err=%v, want nil", tenantID, err)
		}
	}
}

// 判定は保存後の集合ではなく、書き込みが新しく加える分だけを見る。
//
// これが無いと、絶対集合で判定する実装が拒否のテストをすべて通したまま、
// 無編集の CSV エクスポートの再適用と明示的な除去経路を壊せる。
//
//spec:covers EX-IDMANAGEMENT-032-08: 対象が既に保持している `system_admin` を同じ書き込みで再送しても拒否されず、`roles` から外す書き込みも拒否されないこと。
func TestValidateRoleAssignmentJudgesWhatTheWriteAdds(t *testing.T) {
	stored := []string{ReservedRoleSystemAdmin, "support"}

	for _, tt := range []struct {
		name    string
		current []string
		next    []string
		refused bool
	}{
		{"既存値の再送", stored, []string{ReservedRoleSystemAdmin, "support"}, false},
		{"既存値を保ったまま別のロールを足す", stored, []string{ReservedRoleSystemAdmin, "support", "catalog:read"}, false},
		{"明示的な除去", stored, []string{"support"}, false},
		{"何も持たない対象への新規付与", []string{"support"}, []string{ReservedRoleSystemAdmin, "support"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleAssignment(otherTenant, tt.current, tt.next)
			if tt.refused && !errors.Is(err, ErrReservedRole) {
				t.Fatalf("err=%v, want ErrReservedRole", err)
			}
			if !tt.refused && err != nil {
				t.Fatalf("err=%v, want nil", err)
			}
		})
	}
}
