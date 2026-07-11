package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	"github.com/labstack/echo/v5"

	scimhttp "github.com/ambi/idmagic/backend/scim/adapters/http"
	scimmemory "github.com/ambi/idmagic/backend/scim/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/scim/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestScimInboundProvisioning(t *testing.T) {
	ctx := context.Background()
	userRepo := idmmemory.NewUserRepository()
	groupRepo := idmmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()

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

	// 1. 未登録のアクセストークンでは 401 になること
	{
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
		req.Header.Set("Authorization", "Bearer unregistered-token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	}

	// 2. アクセストークン生成
	var tokenStr string
	{
		// トークン生成
		tokStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Okta Integration", 30)
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

		// authenticationSchemes は RFC 7643 §5 で REQUIRED。null や空を返さず、
		// Bearer トークン方式を申告していることを確認する。
		var config struct {
			AuthenticationSchemes []struct {
				Type string `json:"type"`
			} `json:"authenticationSchemes"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &config); err != nil {
			t.Fatalf("failed to decode ServiceProviderConfig: %v", err)
		}
		if len(config.AuthenticationSchemes) == 0 {
			t.Fatal("expected authenticationSchemes to be non-empty")
		}
		foundBearer := false
		for _, s := range config.AuthenticationSchemes {
			if s.Type == "oauthbearertoken" {
				foundBearer = true
			}
		}
		if !foundBearer {
			t.Errorf("expected an oauthbearertoken authentication scheme, got %+v", config.AuthenticationSchemes)
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
		ref, err := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
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
		if u.Lifecycle.Status != idmdomain.UserStatusActive {
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

		ref, _ := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u.Lifecycle.Status != idmdomain.UserStatusDisabled {
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

		ref, _ := scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimUserID)
		u, _ := userRepo.FindBySub(ctx, ref.UserID)
		if u == nil {
			t.Fatal("user should still be accessible by sub for admin purposes")
		}
		if u.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
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
	userRepo := idmmemory.NewUserRepository()
	groupRepo := idmmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()

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

	tokenStr, _, _ := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)

	// 1. テストユーザー作成
	user1Sub := "user_1"
	user2Sub := "user_2"
	_ = userRepo.Save(ctx, &idmdomain.User{ID: user1Sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "user1"})
	_ = userRepo.Save(ctx, &idmdomain.User{ID: user2Sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "user2"})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: tenancydomain.DefaultTenantID, ScimID: "scim_u1", UserID: user1Sub})
	_ = scimRepo.SaveUserRef(ctx, &ports.ScimUserRef{TenantID: tenancydomain.DefaultTenantID, ScimID: "scim_u2", UserID: user2Sub})

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

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, tenancydomain.DefaultTenantID, scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, ref.GroupID)
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

		ref, _ := scimRepo.FindGroupRefByScimID(ctx, tenancydomain.DefaultTenantID, scimGroupID)
		members, _ := groupRepo.ListMembersByGroup(ctx, tenancydomain.DefaultTenantID, ref.GroupID)
		if len(members) != 1 || members[0].UserID != user2Sub {
			t.Fatalf("expected member user2, got %v", members)
		}
	}
}
