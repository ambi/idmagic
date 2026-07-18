package domain_test

import (
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/scim/domain"
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

// 省略した mutable 属性は既定値にリセットされる (PUT full-replace semantics, ADR-122)。
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
