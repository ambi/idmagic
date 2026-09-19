package usecases

import (
	"errors"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const otherTenant = "acme"

// 予約ロールの割当て規則を、対象種別と所属テナントの組で固定する。
//
// 表の行は規則そのものであり、代表 1 件では足りない。テナントを見ない実装、
// 対象種別を見ない実装、どちらも 1 行だけを見るテストは通してしまう。
//
//spec:covers REQ-IDMANAGEMENT-032: 対象種別と所属テナントの組で `system_admin` の新規割当てを許可または拒否すること。Agent はどのテナントでも拒否されること。
func TestValidateRoleAssignmentDecidesByTargetKindAndTenant(t *testing.T) {
	for _, tt := range []struct {
		name     string
		target   RoleAssignmentTarget
		tenantID string
		refused  bool
	}{
		{"制御面テナントの User", RoleTargetUser, tenancydomain.DefaultTenantID, false},
		{"制御面テナントの Group", RoleTargetGroup, tenancydomain.DefaultTenantID, false},
		{"制御面テナントの Agent", RoleTargetAgent, tenancydomain.DefaultTenantID, true},
		{"その他のテナントの User", RoleTargetUser, otherTenant, true},
		{"その他のテナントの Group", RoleTargetGroup, otherTenant, true},
		{"その他のテナントの Agent", RoleTargetAgent, otherTenant, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoleAssignment(tt.target, tt.tenantID, nil, []string{ReservedRoleSystemAdmin})
			if tt.refused && !errors.Is(err, ErrReservedRole) {
				t.Fatalf("err=%v, want ErrReservedRole", err)
			}
			if !tt.refused && err != nil {
				t.Fatalf("err=%v, want nil", err)
			}
		})
	}
}

// 予約ロール以外の名前は、どの対象でもどのテナントでも通る。
//
// 予約する文字列を `system_admin` だけに限る決定を固定する。ロール語彙全体を
// 閉じた列挙にした実装は、ここで落ちる。
func TestValidateRoleAssignmentLeavesTenantDefinedRolesAlone(t *testing.T) {
	for _, target := range []RoleAssignmentTarget{RoleTargetUser, RoleTargetGroup, RoleTargetAgent} {
		for _, tenantID := range []string{tenancydomain.DefaultTenantID, otherTenant} {
			roles := []string{"admin", "catalog:read", "system_admin_of_nothing", "systemadmin"}
			if err := ValidateRoleAssignment(target, tenantID, nil, roles); err != nil {
				t.Fatalf("target=%s tenant=%s err=%v, want nil", target, tenantID, err)
			}
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
			err := ValidateRoleAssignment(RoleTargetUser, otherTenant, tt.current, tt.next)
			if tt.refused && !errors.Is(err, ErrReservedRole) {
				t.Fatalf("err=%v, want ErrReservedRole", err)
			}
			if !tt.refused && err != nil {
				t.Fatalf("err=%v, want nil", err)
			}
		})
	}
}

// Agent は既存値を再送しても拒否する対象ではなく、差分の規則に従う。
//
// Agent をテナントに関わらず拒否する規則と、差分で判定する規則が独立していることを
// 固定する。Agent だけ絶対集合で見る実装は、既存データを持つ環境の Agent を
// 一切更新できなくする。
func TestValidateRoleAssignmentLetsAnAgentKeepAStoredReservedRole(t *testing.T) {
	stored := []string{ReservedRoleSystemAdmin}
	if err := ValidateRoleAssignment(RoleTargetAgent, tenancydomain.DefaultTenantID, stored, stored); err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if err := ValidateRoleAssignment(
		RoleTargetAgent, tenancydomain.DefaultTenantID, nil, stored,
	); !errors.Is(err, ErrReservedRole) {
		t.Fatalf("err=%v, want ErrReservedRole", err)
	}
}
