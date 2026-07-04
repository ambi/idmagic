package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	scimhttp "github.com/ambi/idmagic/internal/scim/adapters/http"
	"github.com/ambi/idmagic/internal/scim/ports"
	"github.com/ambi/idmagic/internal/scim/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestScimInboundProvisioning(t *testing.T) {
	ctx := context.Background()
	userRepo := memory.NewUserRepository()
	groupRepo := memory.NewGroupRepository()
	scimRepo := memory.NewScimRepository()

	usecasesInst := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})

	sd := support.Deps{Emit: func(spec.DomainEvent) {}}
	authenticator := &support.Authenticator{
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
	}
	scimDeps := scimhttp.Deps{
		Deps:          sd,
		Authenticator: authenticator,
		Usecases:      usecasesInst,
	}

	e := echo.New()
	scimhttp.RegisterRoutes(e.Group(""), scimDeps)

	// 1. テナント設定で SCIM が無効なときはエラーになること
	{
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	}

	// 2. SCIM 有効化とトークン生成
	var tokenStr string
	{
		// 有効化
		_, err := usecasesInst.UpdateConfig(ctx, "default", true)
		if err != nil {
			t.Fatal(err)
		}

		// トークン生成
		tokStr, _, err := usecasesInst.GenerateToken(ctx, "default", "Okta Integration", 30)
		if err != nil {
			t.Fatal(err)
		}
		tokenStr = tokStr
	}

	// 3. 有効なトークンでの ServiceProviderConfig アクセス
	{
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}

	// 4. ユーザーのプロビジョニング (Create User)
	var scimUserID string
	{
		body := map[string]any{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": "bjensen@example.com",
			"name": map[string]any{
				"familyName": "Jensen",
				"givenName":  "Barbara",
				"formatted":  "Barbara Jensen",
			},
			"emails": []map[string]any{
				{
					"value":   "bjensen@example.com",
					"type":    "work",
					"primary": true,
				},
			},
			"active": true,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		var createdUser map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &createdUser)
		scimUserID = createdUser["id"].(string)

		// 内部ユーザーが作成されているか検証
		ref, err := scimRepo.FindUserRefByScimID(ctx, "default", scimUserID)
		if err != nil {
			t.Fatal(err)
		}
		if ref == nil {
			t.Fatal("expected user reference to be created")
		}

		u, err := userRepo.FindBySub(ctx, ref.UserID)
		if err != nil {
			t.Fatal(err)
		}
		if u == nil {
			t.Fatal("expected IdP user to be created")
		}
		if *u.Email != "bjensen@example.com" {
			t.Fatalf("expected email bjensen@example.com, got %s", *u.Email)
		}
		if u.Lifecycle.Status != spec.UserStatusActive {
			t.Fatalf("expected status active, got %s", u.Lifecycle.Status)
		}
	}

	// 5. ユーザーの無効化 (Patch User Active -> False)
	{
		body := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":    "replace",
					"path":  "active",
					"value": false,
				},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/"+scimUserID, bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ref, _ := scimRepo.FindUserRefByScimID(ctx, "default", scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u.Lifecycle.Status != spec.UserStatusDisabled {
			t.Fatalf("expected status disabled, got %s", u.Lifecycle.Status)
		}
	}

	// 6. ユーザーの削除 (Soft Delete)
	{
		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/"+scimUserID, http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}

		ref, _ := scimRepo.FindUserRefByScimID(ctx, "default", scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u == nil {
			t.Fatal("user should still be accessible by sub for admin purposes")
		}
		if u.Lifecycle.Status != spec.UserStatusPendingDeletion {
			t.Fatalf("expected status PendingDeletion, got %s", u.Lifecycle.Status)
		}

		// IncludingDeleted でも取得できるはず
		uFull, _ := userRepo.FindBySubIncludingDeleted(ctx, ref.UserID)
		if uFull == nil {
			t.Fatal("user should still exist in database")
		}
	}
}

func TestScimGroupSync(t *testing.T) {
	ctx := context.Background()
	userRepo := memory.NewUserRepository()
	groupRepo := memory.NewGroupRepository()
	scimRepo := memory.NewScimRepository()

	usecasesInst := usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {})

	sd := support.Deps{Emit: func(spec.DomainEvent) {}}
	authenticator := &support.Authenticator{
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
	}
	scimDeps := scimhttp.Deps{
		Deps:          sd,
		Authenticator: authenticator,
		Usecases:      usecasesInst,
	}

	e := echo.New()
	scimhttp.RegisterRoutes(e.Group(""), scimDeps)

	_, _ = usecasesInst.UpdateConfig(ctx, "default", true)
	tokenStr, _, _ := usecasesInst.GenerateToken(ctx, "default", "Integration", 30)

	// 1. テストユーザー作成
	user1Sub := "user_1"
	user2Sub := "user_2"
	_ = userRepo.Save(ctx, &spec.User{ID: user1Sub, TenantID: "default", PreferredUsername: "user1"})
	_ = userRepo.Save(ctx, &spec.User{ID: user2Sub, TenantID: "default", PreferredUsername: "user2"})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: "default", ScimID: "scim_u1", UserID: user1Sub})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: "default", ScimID: "scim_u2", UserID: user2Sub})

	// 2. グループ同期 (Create Group)
	var scimGroupID string
	{
		body := map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": "Engineering",
			"members": []map[string]any{
				{"value": "scim_u1"},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
		}

		var createdGroup map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &createdGroup)
		scimGroupID = createdGroup["id"].(string)

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, "default", scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, "default", ref.GroupID)
		if len(members) != 1 || members[0].UserID != user1Sub {
			t.Fatalf("expected member user1, got %v", members)
		}
	}

	// 3. グループメンバーシップ PATCH (Add user2, Remove user1)
	{
		body := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]any{
						{"value": "scim_u2"},
					},
				},
				{
					"op":   "remove",
					"path": "members",
					"value": []map[string]any{
						{"value": "scim_u1"},
					},
				},
			},
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/"+scimGroupID, bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		req.Header.Set("Content-Type", "application/scim+json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, "default", scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, "default", ref.GroupID)
		if len(members) != 1 || members[0].UserID != user2Sub {
			t.Fatalf("expected member user2, got %v", members)
		}
	}
}
