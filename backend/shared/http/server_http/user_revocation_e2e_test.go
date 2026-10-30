package server_http_test

// User の失効を、製品と同じ組み立てで確かめる。イベントの発行から組み立て地点の変換、失効判定と
// refresh の判定までの配線は、下位の境界では観測できない。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const userRevocationRefreshToken = "user-revocation-refresh-token"

func newUserRevocationStack(t *testing.T) *stack.Stack {
	t.Helper()
	return stack.New(t, stack.WithBrowserFlow(), stack.WithApiTokens(), stack.WithAgentRevocation())
}

// userToken は Web アプリへ発行した User のアクセストークンである。失効の判定の主題は発行の
// 経路ではないので、製品の署名器で直接発行する。
func userToken(t *testing.T, s *stack.Stack) string {
	t.Helper()
	ctx := s.RealmContext(t, "default")
	client, err := s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, stack.BrowserClientID)
	if err != nil || client == nil {
		t.Fatalf("client: %v %v", client, err)
	}
	token, _, err := s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{Client: client, Sub: stack.UserID, Scopes: []string{"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func seedUserRefreshToken(t *testing.T, s *stack.Stack, issuedAt time.Time) {
	t.Helper()
	if err := s.Refresh.Save(s.RealmContext(t, "default"), &oauthdomain.RefreshTokenRecord{
		ID: "refresh-1", TenantID: tenancydomain.DefaultTenantID, Hash: oauthdomain.HashRefreshToken(userRevocationRefreshToken),
		FamilyID: "family-1", ClientID: stack.BrowserClientID, UserID: stack.UserID, Scopes: []string{"openid"},
		IssuedAt: issuedAt, ExpiresAt: issuedAt.Add(time.Hour), AbsoluteExpiresAt: issuedAt.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func refreshStatus(t *testing.T, s *stack.Stack) (int, string) {
	t.Helper()
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {userRevocationRefreshToken}}
	request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(stack.BrowserClientID, stack.BrowserClientSecret)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder.Code, body.Error
}

// 管理者のセッションの失効の後は、発行済みのアクセストークンもリフレッシュトークンも使えない。
//
//spec:covers REQ-SHAREDSIGNALS-012: 管理者の失効で発行済みのトークンが無効になり refresh でも取り直せないこと
//spec:covers REQ-OAUTH2-047: User の失効が Bearer と同じ判定に入ること
func TestAdminSessionRevocationStopsIssuedUserTokens(t *testing.T) {
	s := newUserRevocationStack(t)
	seedUserRefreshToken(t, s, time.Now().UTC().Add(-time.Minute))
	token := userToken(t, s)
	if active := s.Introspect(t, "default", token)["active"]; active != true {
		t.Fatalf("失効の前 active=%v", active)
	}
	signedInBrowser(t, s)
	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeSessionsWrite)

	recorder := adminUserRequest(t, s, adminToken, http.MethodPost, "/api/admin/v1/users/"+stack.UserID+"/sessions/revoke-all")
	if recorder.Code/100 != 2 {
		t.Fatalf("revoke-all status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	if active := s.Introspect(t, "default", token)["active"]; active != false {
		t.Fatalf("失効の後 active=%v", active)
	}
	if status, code := refreshStatus(t, s); status != http.StatusBadRequest || code != "invalid_grant" {
		t.Fatalf("refresh status=%d error=%s", status, code)
	}
}

// 無効化の後は失効し、再有効化しても以前のトークンは戻らない。
//
//spec:covers REQ-SHAREDSIGNALS-012: 無効化の前後と再有効化で判定が分かれること
func TestUserDisableAndEnableKeepsOldTokensRevoked(t *testing.T) {
	s := newUserRevocationStack(t)
	token := userToken(t, s)
	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeUsersWrite)

	if recorder := adminUserRequest(t, s, adminToken, http.MethodPost, "/api/admin/v1/users/"+stack.UserID+"/disable"); recorder.Code != http.StatusNoContent {
		t.Fatalf("disable status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if active := s.Introspect(t, "default", token)["active"]; active != false {
		t.Fatalf("無効化の後 active=%v", active)
	}
	if recorder := adminUserRequest(t, s, adminToken, http.MethodPost, "/api/admin/v1/users/"+stack.UserID+"/enable"); recorder.Code != http.StatusNoContent {
		t.Fatalf("enable status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if active := s.Introspect(t, "default", token)["active"]; active != false {
		t.Fatalf("再有効化の後に以前のトークンが active=%v", active)
	}
}

// User がいないトークンは、エポックの行がなくても有効にならない。
//
//spec:covers REQ-SHAREDSIGNALS-012: 物理削除の後のトークンを無効とすること
func TestDeletedUserTokenIsInactive(t *testing.T) {
	s := newUserRevocationStack(t)
	ctx := s.RealmContext(t, "default")
	client, err := s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, stack.BrowserClientID)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{Client: client, Sub: "user-that-never-existed", Scopes: []string{"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	if active := s.Introspect(t, "default", token)["active"]; active != false {
		t.Fatalf("active=%v", active)
	}
}
