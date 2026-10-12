package handlers_http_test

import (
	"context"
	"net/http"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// groupMemberUserIDs は SCIM id が指す Group のメンバーを、保存先から内部 User の id で読む。
func groupMemberUserIDs(t *testing.T, h scimHarness, groupScimID string) []string {
	t.Helper()
	ctx := context.Background()
	ref, err := h.scimRepo.FindGroupRefByScimID(ctx, tenancydomain.DefaultTenantID, groupScimID)
	if err != nil || ref == nil {
		t.Fatalf("find group reference %s: %v (%v)", groupScimID, ref, err)
	}
	members, err := h.groupRepo.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, ref.GroupID)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}

//spec:covers EX-SOURCING-003-02: displayName のない Group の PUT を 400 invalidValue で拒否し、読み直した Group が拒否の前と同じである。
func TestScimReplaceGroup_RefusesAMissingDisplayNameAndLeavesTheGroupUnchanged(t *testing.T) {
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)
	member := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "kept-member@example.com"})
	groupPath := "/scim/v2/Groups/" + createScimGroupForTest(t, h, tokenStr, map[string]any{
		"displayName": "Kept",
		"members":     []map[string]any{{"value": member}},
	})
	_, before := doScimGet(t, h.echo, tokenStr, groupPath)

	// members だけを送る。displayName の欠落で拒否されなければ、完全置換が members を空にする。
	rec, body := doScimJSON(t, h.echo, http.MethodPut, tokenStr, groupPath, map[string]any{"members": []map[string]any{}})
	if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
		t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
	}
	assertScimResourceUnchanged(t, h, tokenStr, groupPath, before)
}

// 別テナントの User も SCIM id を持つので、id の形では区別できない。解決をテナント内に
// 限っていることを、拒否とメンバーシップの不在の対で読む。
//
//spec:covers EX-SOURCING-005-02: 別テナントに属する User を追加する PatchScimGroup を 400 の SCIM 誤り応答で拒否し、保存先のメンバーシップと読み直した Group が拒否の前と同じである。
func TestScimPatchGroup_RefusesAUserOfAnotherTenant(t *testing.T) {
	ctx := context.Background()
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)
	insider := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "insider@example.com"})
	outsider, err := h.usecases.CreateUser(ctx, "other-tenant", map[string]any{"userName": "outsider@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	groupID := createScimGroupForTest(t, h, tokenStr, map[string]any{
		"displayName": "Insiders",
		"members":     []map[string]any{{"value": insider}},
	})
	groupPath := "/scim/v2/Groups/" + groupID
	_, before := doScimGet(t, h.echo, tokenStr, groupPath)
	membersBefore := groupMemberUserIDs(t, h, groupID)

	rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, groupPath,
		patchOp("add", "members", []map[string]any{{"value": outsider["id"]}}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
	}
	schemas, _ := body["schemas"].([]any)
	if len(schemas) != 1 || schemas[0] != scimErrorSchemaURN {
		t.Errorf("schemas = %v, want [%s]", body["schemas"], scimErrorSchemaURN)
	}
	if after := groupMemberUserIDs(t, h, groupID); len(after) != len(membersBefore) || after[0] != membersBefore[0] {
		t.Errorf("stored members = %v, want %v", after, membersBefore)
	}
	assertScimResourceUnchanged(t, h, tokenStr, groupPath, before)
}

//spec:covers EX-SOURCING-005-03: type が Group と未知の値のメンバーを追加する PatchScimGroup を 400 invalidValue で拒否し、保存先のメンバーシップと読み直した Group が拒否の前と同じである。
func TestScimPatchGroup_RefusesAMemberTypeOtherThanUser(t *testing.T) {
	for _, memberType := range []string{"Group", "Device"} {
		t.Run(memberType, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			existing := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "existing@example.com"})
			candidate := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "candidate@example.com"})
			groupID := createScimGroupForTest(t, h, tokenStr, map[string]any{
				"displayName": "Typed",
				"members":     []map[string]any{{"value": existing}},
			})
			groupPath := "/scim/v2/Groups/" + groupID
			_, before := doScimGet(t, h.echo, tokenStr, groupPath)

			// value は実在する User を指すので、拒否の理由は type だけである。
			rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, groupPath,
				patchOp("add", "members", []map[string]any{{"value": candidate, "type": memberType}}))
			if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
				t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
			}
			if after := groupMemberUserIDs(t, h, groupID); len(after) != 1 {
				t.Errorf("stored members = %v, want only the existing member", after)
			}
			assertScimResourceUnchanged(t, h, tokenStr, groupPath, before)
		})
	}
}
