package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

type failingUserInfoIntrospector struct{ failure error }

func (f failingUserInfoIntrospector) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return nil, f.failure
}

//spec:covers REQ-SYSTEM-034: GETとPOSTのUserInfoは依存先とクレーム生成の内部障害を公開せず、相関ID付きで診断し、保存状態を変えない。
func TestUserInfoInternalErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New("private-marker: SQL constraint users_pkey"),
		errors.New("private-marker: runtime error: index out of range"),
		errors.New("private-marker: cipher: message authentication failed"),
		errors.New("private-marker: goroutine 1 [running]:\n/internal/claims.go:42"),
		fmt.Errorf("read: %w", errors.New("private-marker: dial tcp 10.0.0.5:5432")),
		oauthusecases.NewOAuthError("server_error", "private-marker: internal library failure"),
		nil, // 保存済みポリシーのクレーム解決に失敗する経路。
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(fmt.Sprintf("%s/%v", method, failure), func(t *testing.T) {
				var logs bytes.Buffer
				previous := logging.Default()
				logging.SetDefault(logging.New(&logs, slog.LevelError, "test", ""))
				t.Cleanup(func() { logging.SetDefault(previous) })
				now := time.Now().UTC()
				users := idmanagement.NewMemoryModule().UserRepo
				user := &userdomain.User{
					ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
					Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
				}
				if err := users.Save(t.Context(), user); err != nil {
					t.Fatal(err)
				}
				clients := memory.NewClientRepository()
				client := &oauthdomain.OAuth2Client{
					TenantID: tenancydomain.DefaultTenantID, ClientID: "client-1", CreatedAt: now,
					ClaimPolicy: &claimdomain.ClaimMappingPolicy{
						NameID: claimdomain.NameIdConfiguration{Format: "sub", SourceAttribute: "private-marker"},
					},
				}
				clients.Seed(client)
				var intro oauthports.TokenIntrospector = &fakeIntrospector{result: &oauthports.IntrospectionResult{
					Active: true, Sub: user.ID, Scope: "openid", ClientID: client.ClientID,
				}}
				if failure != nil {
					intro = failingUserInfoIntrospector{failure: failure}
				}
				beforeUser, _ := users.FindBySubIncludingDeleted(t.Context(), user.ID)
				beforeClient, _ := clients.FindByID(t.Context(), client.TenantID, client.ClientID)
				e := echo.New()
				httpadapter.Register(e, httpadapter.Deps{
					Issuer: "http://test", UserRepo: users, TokenIntrospector: intro,
					OAuth2: oauth2.Module{ClientRepo: clients},
				})
				request := httptest.NewRequest(method, "/realms/default/userinfo", http.NoBody)
				request.Header.Set("Authorization", "Bearer access-token")
				request = request.WithContext(logging.ContextWithRequestID(request.Context(), "userinfo-request"))
				response := httptest.NewRecorder()
				e.ServeHTTP(response, request)
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusInternalServerError || body["error"] != "server_error" || body["error_description"] == "" {
					t.Fatalf("status=%d body=%s: %v", response.Code, response.Body.String(), err)
				}
				if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") || len(body) != 2 {
					t.Fatalf("disclosed error or claims: %s %v", response.Body.String(), response.Header())
				}
				if !strings.Contains(logs.String(), "private-marker") || !strings.Contains(logs.String(), "userinfo-request") {
					t.Fatalf("missing diagnostic: %s", logs.String())
				}
				afterUser, _ := users.FindBySubIncludingDeleted(t.Context(), user.ID)
				afterClient, _ := clients.FindByID(t.Context(), client.TenantID, client.ClientID)
				if !reflect.DeepEqual(beforeUser, afterUser) || !reflect.DeepEqual(beforeClient, afterClient) {
					t.Fatal("UserInfo failure changed persisted state")
				}
			})
		}
	}
}
