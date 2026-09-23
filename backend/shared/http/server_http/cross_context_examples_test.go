package server_http_test

// docs/domain/scenarios.feature.md が宣言する、複数の Context が協調して初めて成り立つ
// 具体例を、製品と同じ `Register` の組み立てで確かめる。
//
// 引き金は管理 API への要求だけにする。利用者の状態を保存先へ直接書くと、IdManagement の
// 状態は変わっても、そこから Authentication、OAuth2、SharedSignals へ届く経路が
// 通らない。結果は各 Context の製品入口 (ログイン、認証必須 API、`/introspect`) と、
// 入口に現れない効果 (失効エポック、発行イベント、セッション) の保存先から読む。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 無効化の前に、3 つの経路がすべて開いていることを観測しておく。無効化の後の拒否が
// 無効化の結果であると読めるのは、同じ入口が直前には通っていたときだけである。
//
//spec:covers REQ-PLATFORM-001, EX-PLATFORM-001-01: 管理 API による無効化 1 回で、利用者のステータスが disabled になり、既存セッションの `/api/auth/account` が 401 authentication_required、正しいパスワードの新規ログインが 401 invalid_credentials でセッションを作らず、所有 Agent 2 つの失効エポックが同じ時刻へ進んで発行済みトークンが `/introspect` で active=false になり、AgentAccessRevoked が Agent ごとに 1 件発行される。
func TestAdminDisablingAUserClosesEveryPathToItAtOnce(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithApiTokens(), stack.WithAgentRevocation())
	agentIDs := []string{"A1", "A2"}
	tokens := map[string]string{}
	for _, agentID := range agentIDs {
		clientID := seedOwnedAgent(t, s, agentID)
		tokens[agentID] = issueClientCredentialsToken(t, s, clientID)
		if active := s.Introspect(t, "default", tokens[agentID])["active"]; active != true {
			t.Fatalf("無効化の前に %s のトークンが active=%v、want true", agentID, active)
		}
	}
	session := signedInBrowser(t, s)
	if status, body := session.Get(t, "/api/auth/account"); status != http.StatusOK {
		t.Fatalf("無効化の前の既存セッション status=%d body=%v、want 200", status, body)
	}
	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeUsersWrite)

	before := time.Now().UTC()
	disabled := adminUserRequest(t, s, adminToken, http.MethodPost, "/api/admin/v1/users/"+stack.UserID+"/disable")
	after := time.Now().UTC()
	if disabled.Code != http.StatusNoContent {
		t.Fatalf("disable status=%d body=%s", disabled.Code, disabled.Body.String())
	}

	if status := storedUserStatus(t, s); status != idmdomain.UserStatusDisabled {
		t.Fatalf("無効化の後のステータス=%q、want disabled", status)
	}

	status, body := session.Get(t, "/api/auth/account")
	assertProblem(t, "無効化の後の既存セッション", status, body, "urn:idmagic:error:authentication_required")

	sessionsBefore := sessionCount(t, s)
	fresh := s.Browser(t, "default")
	_ = fresh.Authorize(t, stack.AuthorizationQuery("verifier-for-disabled-login-000000000000000000", nil)).Body.Close()
	status, body = fresh.SignInAttempt(t, "user", stack.UserPassword, "")
	assertProblem(t, "無効化の後の新規ログイン", status, body, "urn:idmagic:error:invalid_credentials")
	if cookie := fresh.SessionCookie(t); cookie != "" {
		t.Fatalf("拒否したログインがセッション Cookie を発行した: %q", cookie)
	}
	if got := sessionCount(t, s); got != sessionsBefore {
		t.Fatalf("拒否したログインの後のセッション数=%d、want %d のまま", got, sessionsBefore)
	}

	var epoch time.Time
	for i, agentID := range agentIDs {
		advanced, err := s.RevocationEpochs.FindByAgent(context.Background(), tenancydomain.DefaultTenantID, agentID)
		if err != nil || advanced == nil {
			t.Fatalf("%s の失効エポックが進んでいない: epoch=%v err=%v", agentID, advanced, err)
		}
		if advanced.Epoch.Before(before) || advanced.Epoch.After(after) {
			t.Fatalf("%s のエポック=%s、want [%s, %s] の範囲", agentID, advanced.Epoch, before, after)
		}
		if i == 0 {
			epoch = advanced.Epoch
		} else if !advanced.Epoch.Equal(epoch) {
			t.Fatalf("%s のエポック=%s、want %s と同一", agentID, advanced.Epoch, epoch)
		}
		if active := s.Introspect(t, "default", tokens[agentID])["active"]; active != false {
			t.Fatalf("無効化の後に %s のトークンが active=%v、want false", agentID, active)
		}
	}

	revoked := map[string]int{}
	for _, event := range s.Events.All() {
		if e, ok := event.(*ssdomain.AgentAccessRevoked); ok {
			revoked[e.AgentID]++
		}
	}
	if len(revoked) != len(agentIDs) || revoked["A1"] != 1 || revoked["A2"] != 1 {
		t.Fatalf("AgentAccessRevoked の Agent ごとの件数=%v、want A1 と A2 に 1 件ずつ", revoked)
	}
}

// 予約と復元の往復で、ログインの可否がステータスに連動することを観測する。予約の前に
// 1 度ログインを通しておくので、予約中の拒否が資格情報ではなく予約によると読める。
//
//spec:covers REQ-PLATFORM-002, EX-PLATFORM-002-01: 管理 API による削除の予約でステータスが pending_deletion になり、正しいパスワードのログインが 401 invalid_credentials でセッションを作らず、猶予期間内の管理 API による復元でステータスが active に戻って同じパスワードのログインが通る。
func TestScheduledDeletionClosesLoginAndRestoreReopensIt(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithApiTokens())
	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeUsersWrite)
	if status, body := signInAttempt(t, s); status != http.StatusOK {
		t.Fatalf("予約の前のログイン status=%d body=%v、want 200", status, body)
	}

	scheduled := adminUserRequest(t, s, adminToken, http.MethodDelete, "/api/admin/v1/users/"+stack.UserID)
	if scheduled.Code != http.StatusNoContent {
		t.Fatalf("削除の予約 status=%d body=%s", scheduled.Code, scheduled.Body.String())
	}
	if status := storedUserStatus(t, s); status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("予約の後のステータス=%q、want pending_deletion", status)
	}
	sessionsBefore := sessionCount(t, s)
	status, body := signInAttempt(t, s)
	assertProblem(t, "予約中のログイン", status, body, "urn:idmagic:error:invalid_credentials")
	if got := sessionCount(t, s); got != sessionsBefore {
		t.Fatalf("拒否したログインの後のセッション数=%d、want %d のまま", got, sessionsBefore)
	}

	restored := adminUserRequest(t, s, adminToken, http.MethodPost, "/api/admin/v1/users/"+stack.UserID+"/restore")
	if restored.Code != http.StatusOK {
		t.Fatalf("復元 status=%d body=%s", restored.Code, restored.Body.String())
	}
	if status := storedUserStatus(t, s); status != idmdomain.UserStatusActive {
		t.Fatalf("復元の後のステータス=%q、want active", status)
	}
	if status, body := signInAttempt(t, s); status != http.StatusOK {
		t.Fatalf("復元の後のログイン status=%d body=%v、want 200", status, body)
	}
}

// seedOwnedAgent は stack.UserID が所有する Active の Agent を置き、client_credentials で
// トークンを取れるクライアントを束縛して、そのクライアント id を返す。
func seedOwnedAgent(t *testing.T, s *stack.Stack, agentID string) string {
	t.Helper()
	now := time.Now().UTC()
	clientID := "agent-client-" + strings.ToLower(agentID)
	secretHash := oauthdomain.HashClientSecret(clientID + "-secret")
	s.Clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: clientID, ClientSecretHash: &secretHash,
		ClientType:              spec.ClientConfidential,
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		Scope:                   "openid",
		FapiProfile:             oauthdomain.FapiNone,
		CreatedAt:               now,
	})
	if err := s.Agents.Save(context.Background(), &agentdomain.Agent{
		ID: agentID, TenantID: tenancydomain.DefaultTenantID, Name: agentID, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent %s: %v", agentID, err)
	}
	if _, err := s.Agents.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: agentID, ClientID: clientID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding %s: %v", agentID, err)
	}
	return clientID
}

func issueClientCredentialsToken(t *testing.T, s *stack.Stack, clientID string) string {
	t.Helper()
	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}
	request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, clientID+"-secret")
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &body) != nil || body.AccessToken == "" {
		t.Fatalf("%s の token status=%d body=%s", clientID, recorder.Code, recorder.Body.String())
	}
	return body.AccessToken
}

// signedInBrowser は stack.UserID としてログインを済ませ、セッション Cookie を持つ browser を返す。
func signedInBrowser(t *testing.T, s *stack.Stack) *stack.Browser {
	t.Helper()
	browser := s.Browser(t, "default")
	_ = browser.Authorize(t, stack.AuthorizationQuery("verifier-for-cross-context-session-0000000000000", nil)).Body.Close()
	browser.SignIn(t, "user", stack.UserPassword)
	if browser.SessionCookie(t) == "" {
		t.Fatal("ログインがセッション Cookie を発行しなかった")
	}
	return browser
}

// signInAttempt は新しい browser で正しいパスワードのログインを 1 回試みる。
func signInAttempt(t *testing.T, s *stack.Stack) (int, map[string]any) {
	t.Helper()
	browser := s.Browser(t, "default")
	_ = browser.Authorize(t, stack.AuthorizationQuery("verifier-for-cross-context-login-00000000000000000", nil)).Body.Close()
	return browser.SignInAttempt(t, "user", stack.UserPassword, "")
}

func adminUserRequest(t *testing.T, s *stack.Stack, token, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, stack.Issuer+"/realms/default"+path, bytes.NewReader(nil))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func storedUserStatus(t *testing.T, s *stack.Stack) idmdomain.UserStatus {
	t.Helper()
	user, err := s.Users.FindBySub(context.Background(), stack.UserID)
	if err != nil || user == nil {
		t.Fatalf("user %s: user=%v err=%v", stack.UserID, user, err)
	}
	return user.Lifecycle.Status
}

func sessionCount(t *testing.T, s *stack.Stack) int {
	t.Helper()
	sessions, err := s.SessionStore.ListBySub(s.RealmContext(t, "default"), stack.UserID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	return len(sessions)
}

func assertProblem(t *testing.T, what string, status int, body map[string]any, problemType string) {
	t.Helper()
	if status != http.StatusUnauthorized || body["type"] != problemType {
		t.Fatalf("%s: status=%d body=%v、want 401 %s", what, status, body, problemType)
	}
}
