package handlers_http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// scimUserWriteScopes は作成を許すスコープである。拒否の理由をトークンの失効・期限・
// テナントに限るため、拒否されるトークンにもこのスコープを持たせる。
var scimUserWriteScopes = []string{string(apitokendomain.ScopeScimUsersRead), string(apitokendomain.ScopeScimUsersWrite)}

// 失効・期限切れ・別テナントの 3 通りを、同じスコープを持つトークンで作る。期限切れは
// 検証側の時計を進めて作るので、観測用のトークンは時計を進めた後に発行し直す。
//
//spec:covers EX-SOURCING-002-02: 失効済み、期限切れ、別テナントの Bearer トークンによる CreateScimUser を 401 の SCIM 誤り応答で拒否し、テナントの User 数が増えないことを保存先で観測する。
func TestScimCreateUser_RefusesAnUnusableTokenAndCreatesNoUser(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	h := newScimHarness(apitokenusecases.WithClock(func() time.Time { return now }))

	revoked, revokedMetadata, err := h.apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", "revoked", scimUserWriteScopes, 30, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.apiTokens.Revoke(ctx, tenancydomain.DefaultTenantID, revokedMetadata.ID); err != nil {
		t.Fatal(err)
	}
	expired, _, err := h.apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", "expired", scimUserWriteScopes, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	otherTenant, _, err := h.apiTokens.Issue(ctx, "other-tenant", "admin-test", "other tenant", scimUserWriteScopes, 30, "")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(48 * time.Hour)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "revoked", token: revoked},
		{name: "expired", token: expired},
		{name: "another tenant", token: otherTenant},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, body := doScimJSON(t, h.echo, http.MethodPost, tc.token, "/scim/v2/Users", map[string]any{
				"userName": "refused-" + tc.name + "@example.com",
			})
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d body=%v", rec.Code, body)
			}
			schemas, _ := body["schemas"].([]any)
			if len(schemas) != 1 || schemas[0] != scimErrorSchemaURN {
				t.Errorf("schemas = %v, want [%s]", body["schemas"], scimErrorSchemaURN)
			}
			// 拒否が防いだ効果。要求先テナントにも、トークンの発行先テナントにも User は増えない。
			for _, tenantID := range []string{tenancydomain.DefaultTenantID, "other-tenant"} {
				if count, err := h.userRepo.Count(ctx, tenantID); err != nil || count != 0 {
					t.Errorf("tenant %s user count = %d (%v), want 0", tenantID, count, err)
				}
			}
		})
	}
}

// PATCH の本文は検証が終わるまで適用しない。active=false を含む本文でも、操作の
// 要件を満たさなければ User は Active のまま残る。
//
//spec:covers EX-SOURCING-002-03: 空または欠けた Operations、オブジェクトでない操作、op の欠落、真偽値でない active を含む PatchScimUser を 400 invalidValue で拒否し、内部 User が Active のまま変わらないことを観測する。
func TestScimPatchUser_RefusesMalformedOperationsAndLeavesTheUserActive(t *testing.T) {
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)
	scimID := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "steady@example.com"})
	_, before := doScimGet(t, h.echo, tokenStr, "/scim/v2/Users/"+scimID)

	patchSchema := []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"}
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{name: "Operations missing", body: map[string]any{"schemas": patchSchema}},
		{name: "Operations empty", body: map[string]any{"schemas": patchSchema, "Operations": []any{}}},
		{name: "operation not an object", body: map[string]any{"schemas": patchSchema, "Operations": []any{"replace active false"}}},
		{name: "op missing", body: map[string]any{"schemas": patchSchema, "Operations": []any{map[string]any{"path": "active", "value": false}}}},
		{name: "active not a boolean", body: patchOp("replace", "active", "false")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, tc.body)
			if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
				t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
			}
			assertScimResourceUnchanged(t, h, tokenStr, "/scim/v2/Users/"+scimID, before)
			if status := internalUserStatus(t, h, scimID); status != idmdomain.UserStatusActive {
				t.Errorf("status = %s, want %s", status, idmdomain.UserStatusActive)
			}
		})
	}
}

// 前段で無効化した User を残しておき、存在しない id への削除がその User へ
// 波及しないことも読む。
//
//spec:covers EX-SOURCING-002-04: 存在しない id への DeleteScimUser が 404 の SCIM 誤り応答を返し、既存の Disabled な User を PendingDeletion へ進めない。
func TestScimDeleteUser_UnknownIDIsNotFound(t *testing.T) {
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)
	scimID := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "bystander@example.com"})
	patchRec, patched := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "active", false))
	if patchRec.Code != http.StatusOK {
		t.Fatalf("setup: disable: expected 200, got %d body=%v", patchRec.Code, patched)
	}

	rec := doScimRequest(t, h.echo, http.MethodDelete, "Bearer "+tokenStr, "/scim/v2/Users/does-not-exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeScimBody(t, rec.Body.Bytes())
	schemas, _ := body["schemas"].([]any)
	if len(schemas) != 1 || schemas[0] != scimErrorSchemaURN {
		t.Errorf("schemas = %v, want [%s]", body["schemas"], scimErrorSchemaURN)
	}
	if body["status"] != "404" {
		t.Errorf("status = %v, want \"404\"", body["status"])
	}
	if status := internalUserStatus(t, h, scimID); status != idmdomain.UserStatusDisabled {
		t.Errorf("bystander status = %s, want %s", status, idmdomain.UserStatusDisabled)
	}
}

// internalUserStatus は SCIM id が指す内部 User の lifecycle 状態を保存先から読む。
func internalUserStatus(t *testing.T, h scimHarness, scimID string) idmdomain.UserStatus {
	t.Helper()
	return storedScimUser(t, h, scimID).Lifecycle.Status
}
