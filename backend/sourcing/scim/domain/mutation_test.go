package domain_test

import (
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/scim/domain"
)

// ParseUserWrite: userName は必須 (RFC7643-CORE-RESOURCES)。
// interfaces.CreateScimUser / UpdateScimUser
func TestParseUserWriteRequiresUserName(t *testing.T) {
	_, err := domain.ParseUserWrite(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing userName")
	}
	var mutErr *domain.MutationError
	if !isMutationError(err, &mutErr) {
		t.Fatalf("expected *domain.MutationError, got %T: %v", err, err)
	}
	if mutErr.ScimType != "invalidValue" {
		t.Errorf("ScimType = %q, want invalidValue", mutErr.ScimType)
	}
}

// 省略した mutable 属性は既定値にリセットされる (PUT full-replace semantics)。
func TestParseUserWriteDefaultsOmittedAttributes(t *testing.T) {
	w, err := domain.ParseUserWrite(map[string]any{"userName": "bjensen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.UserName != "bjensen" {
		t.Errorf("UserName = %q, want bjensen", w.UserName)
	}
	if w.GivenName != "" || w.FamilyName != "" || w.Formatted != "" || w.Email != "" {
		t.Errorf("expected omitted attributes to default empty, got %+v", w)
	}
	if !w.Active {
		t.Errorf("expected Active to default true, got false")
	}
}

// 明示された値は defaults を上書きする。
func TestParseUserWriteExplicitValues(t *testing.T) {
	body := map[string]any{
		"userName": "bjensen",
		"name": map[string]any{
			"givenName":  "Barbara",
			"familyName": "Jensen",
			"formatted":  "Barbara Jensen",
		},
		"emails": []any{
			map[string]any{"value": "bjensen@example.com", "primary": true},
		},
		"active": false,
	}
	w, err := domain.ParseUserWrite(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.GivenName != "Barbara" || w.FamilyName != "Jensen" || w.Formatted != "Barbara Jensen" {
		t.Errorf("unexpected name fields: %+v", w)
	}
	if w.Email != "bjensen@example.com" {
		t.Errorf("Email = %q, want bjensen@example.com", w.Email)
	}
	if w.Active {
		t.Error("expected Active=false to be honored")
	}
}

// REQ-SOURCING-006: SCIM multi-valued emails は primary、work、wire order の順で
// canonical email へ投影する。
func TestProjectCanonicalEmailPriority(t *testing.T) {
	tests := []struct {
		name   string
		emails []any
		want   string
	}{
		{
			name: "primary wins over earlier work",
			emails: []any{
				map[string]any{"value": "work@example.com", "type": "work"},
				map[string]any{"value": "primary@example.com", "type": "home", "primary": true},
			},
			want: "primary@example.com",
		},
		{
			name: "work is case insensitive fallback",
			emails: []any{
				map[string]any{"value": "home@example.com", "type": "home"},
				map[string]any{"value": "work@example.com", "type": "WoRk"},
			},
			want: "work@example.com",
		},
		{
			name: "wire order is final fallback",
			emails: []any{
				map[string]any{"value": "first@example.com", "type": "home"},
				map[string]any{"value": "second@example.com", "type": "other"},
			},
			want: "first@example.com",
		},
		{name: "empty list clears canonical email", emails: []any{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ProjectCanonicalEmail(tt.emails)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ProjectCanonicalEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

// REQ-SOURCING-006: 選ばれない element も含めて配列全体を先に検証する。
func TestProjectCanonicalEmailRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		emails []any
	}{
		{name: "multiple primary", emails: []any{
			map[string]any{"value": "one@example.com", "primary": true},
			map[string]any{"value": "two@example.com", "primary": true},
		}},
		{name: "non object element", emails: []any{"one@example.com"}},
		{name: "missing value", emails: []any{map[string]any{"type": "work"}}},
		{name: "blank value", emails: []any{map[string]any{"value": "  "}}},
		{name: "non string value", emails: []any{map[string]any{"value": 42}}},
		{name: "non string type", emails: []any{map[string]any{"value": "one@example.com", "type": 42}}},
		{name: "non boolean primary", emails: []any{map[string]any{"value": "one@example.com", "primary": "true"}}},
		{name: "invalid unselected element", emails: []any{
			map[string]any{"value": "primary@example.com", "primary": true},
			map[string]any{"value": 42},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.ProjectCanonicalEmail(tt.emails)
			assertMutationError(t, err, "invalidValue")
		})
	}
}

// REQ-SOURCING-006: POST/PUT body の未対応 complex core 属性は silent に捨てない。
func TestParseUserWriteRejectsUnsupportedComplexAttributes(t *testing.T) {
	for _, attr := range []string{"phoneNumbers", "addresses"} {
		t.Run(attr, func(t *testing.T) {
			_, err := domain.ParseUserWrite(map[string]any{"userName": "bjensen", attr: []any{}})
			assertMutationError(t, err, "invalidValue")
		})
	}
}

// REQ-SOURCING-007: enterprise extension の employeeNumber/department/manager を
// POST/PUT body から解析する。manager は value オブジェクトと文字列の両方を許可する。
func TestParseUserWriteEnterpriseExtension(t *testing.T) {
	t.Run("value object form", func(t *testing.T) {
		body := map[string]any{
			"userName": "bjensen",
			domain.EnterpriseUserSchemaURN: map[string]any{
				"employeeNumber": "701984",
				"department":     "Tour Operations",
				"manager":        map[string]any{"value": "scim_manager1"},
			},
		}
		w, err := domain.ParseUserWrite(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.EmployeeNumber != "701984" || w.Department != "Tour Operations" || w.ManagerValue != "scim_manager1" {
			t.Errorf("unexpected enterprise extension fields: %+v", w)
		}
	})

	t.Run("manager as bare string (Entra quirk)", func(t *testing.T) {
		body := map[string]any{
			"userName": "bjensen",
			domain.EnterpriseUserSchemaURN: map[string]any{
				"manager": "scim_manager1",
			},
		}
		w, err := domain.ParseUserWrite(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.ManagerValue != "scim_manager1" {
			t.Errorf("ManagerValue = %q, want scim_manager1", w.ManagerValue)
		}
	})

	t.Run("omitted extension defaults empty", func(t *testing.T) {
		w, err := domain.ParseUserWrite(map[string]any{"userName": "bjensen"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.EmployeeNumber != "" || w.Department != "" || w.ManagerValue != "" {
			t.Errorf("expected omitted enterprise extension to default empty, got %+v", w)
		}
	})
}

// REQ-SOURCING-007: 不正な enterprise extension は invalidValue で拒否する。
func TestParseUserWriteRejectsInvalidEnterpriseExtension(t *testing.T) {
	tests := []struct {
		name string
		ext  any
	}{
		{"extension not an object", "not-an-object"},
		{"employeeNumber not a string", map[string]any{"employeeNumber": 42}},
		{"department not a string", map[string]any{"department": 42}},
		{"manager missing value", map[string]any{"manager": map[string]any{}}},
		{"manager blank string", map[string]any{"manager": "  "}},
		{"manager wrong type", map[string]any{"manager": 42}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{
				"userName":                     "bjensen",
				domain.EnterpriseUserSchemaURN: tt.ext,
			}
			_, err := domain.ParseUserWrite(body)
			assertMutationError(t, err, "invalidValue")
		})
	}
}

// REQ-SOURCING-006: PATCH emails は domain validation 中に canonical value へ解決する。
func TestParseUserPatchOpsProjectsCanonicalEmail(t *testing.T) {
	body := map[string]any{
		"Operations": []any{map[string]any{
			"op":   "replace",
			"path": "emails",
			"value": []any{
				map[string]any{"value": "home@example.com", "type": "home"},
				map[string]any{"value": "work@example.com", "type": "work"},
			},
		}},
	}
	ops, err := domain.ParseUserPatchOps(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := ops[0].Value.(string); !ok || got != "work@example.com" {
		t.Fatalf("projected Value = %#v, want work@example.com", ops[0].Value)
	}
}

// PATCH は RFC7644-PATCH の allowlist に閉じた path だけを受け付ける。
func TestParseUserPatchOpsAllowedPath(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "replace", "path": "active", "value": false},
		},
	}
	ops, err := domain.ParseUserPatchOps(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) != 1 || ops[0].Attr != domain.UserAttrActive || ops[0].Op != "replace" {
		t.Fatalf("unexpected ops: %+v", ops)
	}
}

// 未対応 path は invalidPath。
func TestParseUserPatchOpsRejectsUnknownPath(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "replace", "path": "nickName", "value": "x"},
		},
	}
	_, err := domain.ParseUserPatchOps(body)
	assertMutationError(t, err, "invalidPath")
}

// readOnly 属性 (id/meta/schemas) への書込みは mutability。
func TestParseUserPatchOpsRejectsReadOnlyPath(t *testing.T) {
	for _, path := range []string{"id", "meta", "schemas"} {
		body := map[string]any{
			"Operations": []any{
				map[string]any{"op": "replace", "path": path, "value": "x"},
			},
		}
		_, err := domain.ParseUserPatchOps(body)
		assertMutationError(t, err, "mutability")
	}
}

// add/replace/remove 以外の op は invalidValue。
func TestParseUserPatchOpsRejectsUnknownOp(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "delete", "path": "active", "value": true},
		},
	}
	_, err := domain.ParseUserPatchOps(body)
	assertMutationError(t, err, "invalidValue")
}

// REQ-SOURCING-007: enterprise extension PATCH path は bare 名と URN 修飾済みの
// 完全パスの両方を受け付ける。
func TestParseUserPatchOpsEnterpriseExtensionPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
		attr domain.UserAttr
	}{
		{"bare employeeNumber", "employeeNumber", domain.UserAttrEmployeeNumber},
		{"urn-qualified employeeNumber", domain.EnterpriseUserSchemaURN + ":employeeNumber", domain.UserAttrEmployeeNumber},
		{"bare department", "department", domain.UserAttrDepartment},
		{"urn-qualified department", domain.EnterpriseUserSchemaURN + ":department", domain.UserAttrDepartment},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{
				"Operations": []any{
					map[string]any{"op": "replace", "path": tt.path, "value": "x"},
				},
			}
			ops, err := domain.ParseUserPatchOps(body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(ops) != 1 || ops[0].Attr != tt.attr {
				t.Fatalf("unexpected ops: %+v", ops)
			}
		})
	}
}

// REQ-SOURCING-007: PATCH manager は value オブジェクトと文字列の両方を canonical
// scim id 文字列へ投影する。
func TestParseUserPatchOpsProjectsManagerValue(t *testing.T) {
	t.Run("value object form", func(t *testing.T) {
		body := map[string]any{
			"Operations": []any{
				map[string]any{"op": "replace", "path": "manager", "value": map[string]any{"value": "scim_manager1"}},
			},
		}
		ops, err := domain.ParseUserPatchOps(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, ok := ops[0].Value.(string); !ok || got != "scim_manager1" || ops[0].Attr != domain.UserAttrManager {
			t.Fatalf("unexpected op: %+v", ops[0])
		}
	})

	t.Run("bare string form", func(t *testing.T) {
		body := map[string]any{
			"Operations": []any{
				map[string]any{"op": "replace", "path": domain.EnterpriseUserSchemaURN + ":manager", "value": "scim_manager1"},
			},
		}
		ops, err := domain.ParseUserPatchOps(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, ok := ops[0].Value.(string); !ok || got != "scim_manager1" {
			t.Fatalf("unexpected op: %+v", ops[0])
		}
	})

	t.Run("remove does not require a value", func(t *testing.T) {
		body := map[string]any{
			"Operations": []any{
				map[string]any{"op": "remove", "path": "manager"},
			},
		}
		ops, err := domain.ParseUserPatchOps(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ops) != 1 || ops[0].Attr != domain.UserAttrManager || ops[0].Op != "remove" {
			t.Fatalf("unexpected ops: %+v", ops)
		}
	})

	t.Run("invalid manager value is invalidValue", func(t *testing.T) {
		body := map[string]any{
			"Operations": []any{
				map[string]any{"op": "replace", "path": "manager", "value": map[string]any{}},
			},
		}
		_, err := domain.ParseUserPatchOps(body)
		assertMutationError(t, err, "invalidValue")
	})
}

// active の value は bool でなければならない。
func TestParseUserPatchOpsRejectsWrongValueType(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "replace", "path": "active", "value": "yes"},
		},
	}
	_, err := domain.ParseUserPatchOps(body)
	assertMutationError(t, err, "invalidValue")
}

// ParseGroupWrite: displayName は必須。
func TestParseGroupWriteRequiresDisplayName(t *testing.T) {
	_, err := domain.ParseGroupWrite(map[string]any{})
	assertMutationError(t, err, "invalidValue")
}

// members を省略すると空集合になる (PUT full-replace)。
func TestParseGroupWriteDefaultsMembersEmpty(t *testing.T) {
	w, err := domain.ParseGroupWrite(map[string]any{"displayName": "Engineering"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.DisplayName != "Engineering" {
		t.Errorf("DisplayName = %q, want Engineering", w.DisplayName)
	}
	if len(w.MemberScimIDs) != 0 {
		t.Errorf("expected empty members, got %v", w.MemberScimIDs)
	}
}

func TestParseGroupWriteMembers(t *testing.T) {
	body := map[string]any{
		"displayName": "Engineering",
		"members": []any{
			map[string]any{"value": "scim_u1"},
			map[string]any{"value": "scim_u2"},
		},
	}
	w, err := domain.ParseGroupWrite(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(w.MemberScimIDs) != 2 || w.MemberScimIDs[0] != "scim_u1" || w.MemberScimIDs[1] != "scim_u2" {
		t.Errorf("unexpected members: %v", w.MemberScimIDs)
	}
}

// REQ-SOURCING-005: Group member は type 省略または User だけを受け付ける。
func TestParseGroupWriteMemberType(t *testing.T) {
	t.Run("accepts case insensitive User", func(t *testing.T) {
		w, err := domain.ParseGroupWrite(map[string]any{
			"displayName": "Engineering",
			"members":     []any{map[string]any{"value": "scim_u1", "type": "uSeR"}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(w.MemberScimIDs) != 1 || w.MemberScimIDs[0] != "scim_u1" {
			t.Fatalf("unexpected members: %v", w.MemberScimIDs)
		}
	})

	for _, memberType := range []any{"Group", 42} {
		t.Run("rejects unsupported type", func(t *testing.T) {
			_, err := domain.ParseGroupWrite(map[string]any{
				"displayName": "Engineering",
				"members":     []any{map[string]any{"value": "scim_g1", "type": memberType}},
			})
			assertMutationError(t, err, "invalidValue")
		})
	}
}

// REQ-SOURCING-005: PATCH も mutation 適用前に全 member type を検証する。
func TestParseGroupPatchOpsRejectsGroupMemberType(t *testing.T) {
	body := map[string]any{
		"Operations": []any{map[string]any{
			"op":    "add",
			"path":  "members",
			"value": []any{map[string]any{"value": "scim_g1", "type": "Group"}},
		}},
	}
	_, err := domain.ParseGroupPatchOps(body)
	assertMutationError(t, err, "invalidValue")
}

// Group PATCH は displayName / members だけを受け付ける。
func TestParseGroupPatchOpsAllowedPath(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "add", "path": "members", "value": []any{
				map[string]any{"value": "scim_u1"},
			}},
		},
	}
	ops, err := domain.ParseGroupPatchOps(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops) != 1 || ops[0].Attr != domain.GroupAttrMembers || ops[0].Op != "add" {
		t.Fatalf("unexpected ops: %+v", ops)
	}
}

func TestParseGroupPatchOpsRejectsUnknownPath(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "replace", "path": "description", "value": "x"},
		},
	}
	_, err := domain.ParseGroupPatchOps(body)
	assertMutationError(t, err, "invalidPath")
}

func TestParseGroupPatchOpsRejectsReadOnlyPath(t *testing.T) {
	body := map[string]any{
		"Operations": []any{
			map[string]any{"op": "replace", "path": "meta", "value": "x"},
		},
	}
	_, err := domain.ParseGroupPatchOps(body)
	assertMutationError(t, err, "mutability")
}

func assertMutationError(t *testing.T, err error, wantScimType string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var mutErr *domain.MutationError
	if !isMutationError(err, &mutErr) {
		t.Fatalf("expected *domain.MutationError, got %T: %v", err, err)
	}
	if mutErr.ScimType != wantScimType {
		t.Errorf("ScimType = %q, want %q", mutErr.ScimType, wantScimType)
	}
}

func isMutationError(err error, target **domain.MutationError) bool {
	return errors.As(err, target)
}
