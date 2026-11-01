package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

const personalTokensPath = "/realms/default/api/account/v1/api-tokens"

//spec:covers REQ-APITOKENS-005: 一般ユーザーの発行、所有者の分離、利用と失効を正式な HTTP 経路で確認する。
//verify:operation IssuePersonalApiToken: 本人向け発行の JSON、主体と利用可能なトークン。
//verify:operation ListPersonalApiTokens: 本人の記録だけを返す一覧。
//verify:operation RevokePersonalApiToken: 所有者を照合した失効と拒否後の有効性。
func TestPersonalApiTokenLifecycle(t *testing.T) {
	s := newExampleStack(t)
	user := signedIn(t, s, stack.UserID)
	admin := signedIn(t, s, stack.AdminUserID)
	body := map[string]any{"description": "personal", "scopes": []string{"account:read"}, "expiry_days": 7}
	issued := user.do(t, http.MethodPost, personalTokensPath, body)
	if issued.Code != http.StatusCreated {
		t.Fatalf("issue status=%d body=%s", issued.Code, issued.Body.String())
	}
	var result struct {
		Token string `json:"token"`
		Meta  struct {
			ID     string `json:"id"`
			UserID string `json:"user_id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Token == "" || result.Meta.UserID != stack.UserID {
		t.Fatalf("issue=%+v", result)
	}
	if got := bearer(t, s, http.MethodGet, "/realms/default/api/account/v1/summary", result.Token, nil); got.Code != http.StatusOK {
		t.Fatalf("use status=%d body=%s", got.Code, got.Body.String())
	}
	if got := bearer(t, s, http.MethodGet, personalTokensPath, result.Token, nil); got.Code != http.StatusForbidden {
		t.Fatalf("token can manage tokens: %d %s", got.Code, got.Body.String())
	}
	other := admin.do(t, http.MethodGet, personalTokensPath, nil)
	if other.Code != http.StatusOK {
		t.Fatalf("other list=%d %s", other.Code, other.Body.String())
	}
	var list struct {
		Tokens []json.RawMessage `json:"tokens"`
	}
	if err := json.Unmarshal(other.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Tokens) != 0 {
		t.Fatalf("other user sees token: %s", other.Body.String())
	}
	if got := admin.do(t, http.MethodDelete, personalTokensPath+"/"+result.Meta.ID, nil); got.Code != http.StatusNoContent {
		t.Fatalf("foreign revoke=%d", got.Code)
	}
	if got := bearer(t, s, http.MethodGet, "/realms/default/api/account/v1/summary", result.Token, nil); got.Code != http.StatusOK {
		t.Fatalf("foreign revoke changed token: %d", got.Code)
	}
	if got := user.do(t, http.MethodDelete, personalTokensPath+"/"+result.Meta.ID, nil); got.Code != http.StatusNoContent {
		t.Fatalf("revoke=%d", got.Code)
	}
	if got := bearer(t, s, http.MethodGet, "/realms/default/api/account/v1/summary", result.Token, nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token usable: %d", got.Code)
	}
	for _, input := range []map[string]any{
		{"scopes": []string{"users:write"}, "expiry_days": 7},
		{"scopes": []string{"account:read"}, "expiry_days": 0},
		{"scopes": []string{"account:read"}, "expiry_days": 366},
		{"scopes": []string{}, "expiry_days": 7},
	} {
		if got := user.do(t, http.MethodPost, personalTokensPath, input); got.Code != http.StatusBadRequest {
			t.Fatalf("invalid input accepted: %d %s", got.Code, got.Body.String())
		}
	}
	if got := user.do(t, http.MethodGet, personalTokensPath, nil); got.Code != http.StatusOK {
		t.Fatalf("own list=%d", got.Code)
	}
	if len(storedApiTokens(t, s)) != 1 {
		t.Fatal("refused requests mutated token records")
	}
	user.csrfHeader = ""
	if got := user.do(t, http.MethodPost, personalTokensPath, body); got.Code != http.StatusForbidden {
		t.Fatalf("missing csrf=%d", got.Code)
	}
}

//spec:covers REQ-APITOKENS-005, EX-APITOKENS-005-01: 未認証、古い認証、別テナントと非対話の要求は記録を増減させない。
//verify:operation IssuePersonalApiToken: 再認証と CSRF の拒否、記録の維持。
//verify:operation ListPersonalApiTokens: 認証の拒否とテナント境界。
//verify:operation RevokePersonalApiToken: 別テナントと CSRF の拒否、記録の維持。
func TestPersonalApiTokenRefusals(t *testing.T) {
	s := newExampleStack(t)
	body := map[string]any{"description": "personal", "scopes": []string{"account:read"}, "expiry_days": 7}
	stale, err := s.Sessions.Create(s.RealmContext(t, realm), stack.UserID, []string{"pwd"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := s.Sessions.Store.Find(s.RealmContext(t, realm), stale.SessionID)
	if err != nil || stored == nil {
		t.Fatalf("session=%v error=%v", stored, err)
	}
	stored.AuthTime = time.Now().Add(-time.Hour).Unix()
	if err := s.Sessions.Store.Save(s.RealmContext(t, realm), stored); err != nil {
		t.Fatal(err)
	}
	client := sessionClient{s: s, sessionID: stale.SessionID, csrfHeader: csrfTokenValue}
	denied := client.do(t, http.MethodPost, personalTokensPath, body)
	if denied.Code != http.StatusForbidden || problemType(t, denied) != "urn:idmagic:error:step_up_required" {
		t.Fatalf("stale auth: %d %s", denied.Code, denied.Body.String())
	}
	if len(storedApiTokens(t, s)) != 0 {
		t.Fatal("stale authentication issued a token")
	}
	if got := bearer(t, s, http.MethodGet, personalTokensPath, portalAccessToken(t, s), nil); got.Code != http.StatusForbidden {
		t.Fatalf("admin portal crossed account boundary: %d", got.Code)
	}
	unauthenticated := sessionClient{s: s, csrfHeader: csrfTokenValue}
	if got := unauthenticated.do(t, http.MethodGet, personalTokensPath, nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated=%d", got.Code)
	}
	user := signedIn(t, s, stack.UserID)
	issued := user.do(t, http.MethodPost, personalTokensPath, body)
	if issued.Code != http.StatusCreated {
		t.Fatalf("issue=%d %s", issued.Code, issued.Body.String())
	}
	tokens := storedApiTokens(t, s)
	otherPath := "/realms/" + stack.OtherRealm + "/api/account/v1/api-tokens"
	foreign := s.AdminSessionCookie(t, stack.OtherRealm)
	other := sessionClient{s: s, sessionID: foreign.Value, csrfHeader: csrfTokenValue}
	if got := other.do(t, http.MethodDelete, otherPath+"/"+tokens[0].ID, nil); got.Code != http.StatusNoContent {
		t.Fatalf("foreign revoke=%d", got.Code)
	}
	if storedApiTokens(t, s)[0].RevokedAt != nil {
		t.Fatal("foreign tenant revoked personal token")
	}
	if got := user.do(t, http.MethodGet, otherPath, nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("foreign session=%d", got.Code)
	}
	user.csrfHeader = ""
	if got := user.do(t, http.MethodDelete, personalTokensPath+"/"+tokens[0].ID, nil); got.Code != http.StatusForbidden {
		t.Fatalf("missing csrf=%d", got.Code)
	}
	if storedApiTokens(t, s)[0].RevokedAt != nil {
		t.Fatal("CSRF refusal revoked token")
	}
}
