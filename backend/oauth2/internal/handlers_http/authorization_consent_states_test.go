package handlers_http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 保存値を共有する場合と独立した値を返す場合の両方で、状態と主体の保存を検査する。
type consentStateStore struct {
	ports.AuthorizationRequestStore
	id     string
	states []spec.AuthorizationCodeFlowState
	share  bool
}

//spec:covers REQ-OAUTH2-008: 同意待ち以外、別テナント、期限切れ、主体の不一致、未完了の認証では、同意とコードとイベントを作らない。
func TestConsentRejectsUnavailableOrMismatchedTransactions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		state   spec.AuthorizationCodeFlowState
		change  func(*domain.AuthorizationRequest, *authdomain.AuthenticationContext)
		problem string
	}{
		{name: "received", state: spec.AuthFlowReceived},
		{name: "authentication_pending", state: spec.AuthFlowAuthenticationPending},
		{name: "authenticated", state: spec.AuthFlowAuthenticated},
		{name: "consented", state: spec.AuthFlowConsented},
		{name: "code_issued", state: spec.AuthFlowCodeIssued},
		{name: "exchanged", state: spec.AuthFlowExchanged},
		{name: "rejected", state: spec.AuthFlowRejected},
		{name: "expired", state: spec.AuthFlowExpired},
		{name: "foreign_tenant", state: spec.AuthFlowConsentPending, change: func(req *domain.AuthorizationRequest, _ *authdomain.AuthenticationContext) { req.TenantID = "foreign" }},
		{name: "deadline", state: spec.AuthFlowConsentPending, change: func(req *domain.AuthorizationRequest, _ *authdomain.AuthenticationContext) {
			req.ExpiresAt = time.Now().Add(-time.Minute)
		}},
		{name: "different_subject", state: spec.AuthFlowConsentPending, change: func(_ *domain.AuthorizationRequest, authn *authdomain.AuthenticationContext) { authn.UserID = "bob" }, problem: "authentication_required"},
		{name: "pending_session", state: spec.AuthFlowConsentPending, change: func(_ *domain.AuthorizationRequest, authn *authdomain.AuthenticationContext) {
			authn.AuthenticationPending = true
		}, problem: "authentication_required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			requests := &consentStateStore{AuthorizationRequestStore: oauth2memory.NewAuthorizationRequestStore()}
			codes := &countingCodeStore{AuthorizationCodeStore: oauth2memory.NewAuthorizationCodeStore()}
			consents := oauth2memory.NewConsentRepository()
			authn := &authdomain.AuthenticationContext{UserID: "alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}
			e, events := newAuthorizeTestServer(t, authn, nil, func(d *httpadapter.Deps) {
				d.OAuth2.RequestStore, d.OAuth2.CodeStore, d.OAuth2.ConsentRepo = requests, codes, consents
			})
			response := runAuthorize(t, e, authorizeQuery(nil))
			req, err := requests.Find(ctx, requests.id)
			if err != nil || req == nil {
				t.Fatalf("要求=%v err=%v", req, err)
			}
			req.State = tc.state
			if tc.change != nil {
				tc.change(req, authn)
			}
			if err := requests.Save(ctx, req); err != nil {
				t.Fatal(err)
			}
			requests.states = nil
			post := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/consent", strings.NewReader(`{"action":"allow"}`))
			post.Header.Set("Content-Type", "application/json")
			post.Header.Set("Origin", "http://test")
			post.Header.Set("X-Csrf-Token", "csrf-val")
			post.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "csrf-val"})
			for _, cookie := range response.Result().Cookies() {
				post.AddCookie(cookie)
			}
			result := httptest.NewRecorder()
			e.ServeHTTP(result, post)
			want := tc.problem
			if want == "" {
				want = "transaction_unavailable"
			}
			if result.Code != http.StatusUnauthorized || !strings.Contains(result.Body.String(), want) {
				t.Fatalf("拒否 status=%d body=%s, want 401 %s", result.Code, result.Body, want)
			}
			consent, err := consents.Find(ctx, tenancydomain.DefaultTenantID, "alice", authClientID)
			if err != nil {
				t.Fatal(err)
			}
			if consent != nil || codes.saved != 0 || len(*events) != 0 || len(requests.states) != 0 {
				t.Fatalf("拒否後の作用 consent=%v codes=%d events=%v states=%v", consent, codes.saved, *events, requests.states)
			}
		})
	}
}

func (s *consentStateStore) Save(ctx context.Context, req *domain.AuthorizationRequest) error {
	s.id = req.ID
	return s.AuthorizationRequestStore.Save(ctx, req)
}

func (s *consentStateStore) Find(ctx context.Context, id string) (*domain.AuthorizationRequest, error) {
	req, err := s.AuthorizationRequestStore.Find(ctx, id)
	if err != nil || req == nil || s.share {
		return req, err
	}
	detached := *req
	return &detached, nil
}

func (s *consentStateStore) UpdateState(ctx context.Context, id string, state spec.AuthorizationCodeFlowState) error {
	if err := s.AuthorizationRequestStore.UpdateState(ctx, id, state); err != nil {
		return err
	}
	s.states = append(s.states, state)
	return nil
}

//spec:covers REQ-OAUTH2-008, REQ-OAUTH2-005: 同意画面の前に認証情報と ConsentPending を保存し、許可では Consented を経てコードを発行し、拒否では同意とコードを作らず Rejected にする。
func TestAuthorizationConsentStateProgression(t *testing.T) {
	for _, action := range []string{"allow", "deny"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			requests := &consentStateStore{AuthorizationRequestStore: oauth2memory.NewAuthorizationRequestStore()}
			codes := &countingCodeStore{AuthorizationCodeStore: oauth2memory.NewAuthorizationCodeStore()}
			consents := oauth2memory.NewConsentRepository()
			authn := &authdomain.AuthenticationContext{
				UserID: "alice", SessionID: "session-alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}, ACR: "urn:pwd",
			}
			e, events := newAuthorizeTestServer(t, authn, nil, func(d *httpadapter.Deps) {
				d.OAuth2.RequestStore, d.OAuth2.CodeStore, d.OAuth2.ConsentRepo = requests, codes, consents
			})
			response := runAuthorize(t, e, authorizeQuery(url.Values{"state": {"consent-state"}}))
			if response.Code != http.StatusSeeOther || !strings.HasSuffix(response.Header().Get("Location"), "/consent") {
				t.Fatalf("同意への遷移 status=%d location=%q", response.Code, response.Header().Get("Location"))
			}
			req, err := requests.Find(ctx, requests.id)
			if err != nil || req == nil {
				t.Fatalf("認可要求=%v err=%v", req, err)
			}
			if req.State != spec.AuthFlowConsentPending || req.UserID == nil || *req.UserID != authn.UserID || req.AuthTime == nil || *req.AuthTime != authn.AuthTime {
				t.Fatalf("同意待ちの状態または認証情報が未保存: %+v", req)
			}
			if codes.saved != 0 {
				t.Fatalf("同意前のコード保存=%d", codes.saved)
			}
			transaction := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/transaction", http.NoBody)
			for _, cookie := range response.Result().Cookies() {
				transaction.AddCookie(cookie)
			}
			transactionResponse := httptest.NewRecorder()
			e.ServeHTTP(transactionResponse, transaction)
			var body struct {
				Kind string `json:"kind"`
				CSRF string `json:"csrf_token"`
			}
			if err := json.Unmarshal(transactionResponse.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if transactionResponse.Code != http.StatusOK || body.Kind != "consent" || body.CSRF == "" {
				t.Fatalf("同意待ちの継続 status=%d body=%s", transactionResponse.Code, transactionResponse.Body)
			}
			post := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/consent", strings.NewReader(`{"action":"`+action+`"}`))
			post.Header.Set("Content-Type", "application/json")
			post.Header.Set("Origin", "http://test")
			post.Header.Set("X-Csrf-Token", body.CSRF)
			for _, source := range []*httptest.ResponseRecorder{response, transactionResponse} {
				for _, cookie := range source.Result().Cookies() {
					post.AddCookie(cookie)
				}
			}
			result := httptest.NewRecorder()
			e.ServeHTTP(result, post)
			var output struct {
				Redirect string `json:"redirect_to"`
			}
			if err := json.Unmarshal(result.Body.Bytes(), &output); err != nil {
				t.Fatal(err)
			}
			if result.Code != http.StatusOK {
				t.Fatalf("同意結果 status=%d body=%s", result.Code, result.Body)
			}
			redirect, err := url.Parse(output.Redirect)
			if err != nil {
				t.Fatal(err)
			}
			if redirect.Query().Get("state") != "consent-state" || redirect.Query().Get("iss") == "" {
				t.Fatalf("state または iss が欠落: %s", redirect)
			}
			want := []spec.AuthorizationCodeFlowState{spec.AuthFlowAuthenticationPending, spec.AuthFlowAuthenticated, spec.AuthFlowConsentPending}
			consent, err := consents.Find(ctx, req.TenantID, authn.UserID, authClientID)
			if err != nil {
				t.Fatal(err)
			}
			if action == "allow" {
				want = append(want, spec.AuthFlowConsented, spec.AuthFlowCodeIssued)
				if codes.saved != 1 || redirect.Query().Get("code") == "" || consent == nil || consent.State != domain.ConsentGranted {
					t.Fatalf("許可後の保存 code=%d consent=%+v redirect=%s", codes.saved, consent, redirect)
				}
				if len(*events) != 2 || (*events)[0].EventType() != "ConsentGranted" || (*events)[1].EventType() != "AuthorizationCodeIssued" {
					t.Fatalf("許可のイベント=%v", *events)
				}
			} else {
				want = append(want, spec.AuthFlowRejected)
				if codes.saved != 0 || consent != nil || len(*events) != 0 || redirect.Query().Get("error") != "access_denied" || redirect.Query().Get("code") != "" {
					t.Fatalf("拒否後の作用 code=%d consent=%+v events=%v redirect=%s", codes.saved, consent, *events, redirect)
				}
			}
			if !slices.Equal(requests.states, want) {
				t.Fatalf("保存した遷移=%v, want %v", requests.states, want)
			}
			stored, err := requests.Find(ctx, req.ID)
			if err != nil || stored.State != want[len(want)-1] {
				t.Fatalf("最終状態=%+v err=%v", stored, err)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-008, REQ-OAUTH2-005: 同意済みの継続では prompt=consent を再適用せず、同意した主体と同じセッションにだけコードを発行する。
func TestConsentedAuthorizationResumesWithoutRequestingConsentAgain(t *testing.T) {
	for _, tc := range []struct {
		subject string
		share   bool
	}{
		{subject: "alice"}, {subject: "bob"}, {subject: "alice", share: true}, {subject: "bob", share: true},
	} {
		t.Run(fmt.Sprintf("%s/shared=%t", tc.subject, tc.share), func(t *testing.T) {
			ctx := context.Background()
			requests := &consentStateStore{AuthorizationRequestStore: oauth2memory.NewAuthorizationRequestStore(), share: tc.share}
			authn := &authdomain.AuthenticationContext{UserID: "alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}
			e, _ := newAuthorizeTestServer(t, authn, nil, func(d *httpadapter.Deps) { d.OAuth2.RequestStore = requests })
			response := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"consent"}}))
			if err := requests.UpdateState(ctx, requests.id, spec.AuthFlowConsented); err != nil {
				t.Fatal(err)
			}
			if err := requests.AttachAuthentication(ctx, requests.id, tc.subject, authn.AuthTime, authn.AMR, "", ""); err != nil {
				t.Fatal(err)
			}
			requests.states = nil
			resume := httptest.NewRequest(http.MethodGet, "/realms/default/authorize/resume", http.NoBody)
			for _, cookie := range response.Result().Cookies() {
				resume.AddCookie(cookie)
			}
			result := httptest.NewRecorder()
			e.ServeHTTP(result, resume)
			redirect, err := url.Parse(result.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if result.Code != http.StatusFound {
				t.Fatalf("同意済みの再開 status=%d redirect=%s body=%s", result.Code, redirect, result.Body)
			}
			if tc.subject != authn.UserID {
				if redirect.Query().Get("error") != "invalid_request" || redirect.Query().Get("code") != "" || len(requests.states) != 0 {
					t.Fatalf("他人の同意で発行した: redirect=%s states=%v", redirect, requests.states)
				}
				stored, err := requests.Find(ctx, requests.id)
				if err != nil || stored.UserID == nil || *stored.UserID != tc.subject {
					t.Fatalf("拒否後の主体=%+v err=%v", stored, err)
				}
				return
			}
			if redirect.Query().Get("code") == "" {
				t.Fatalf("コードが欠落: %s", redirect)
			}
			if !slices.Equal(requests.states, []spec.AuthorizationCodeFlowState{spec.AuthFlowCodeIssued}) {
				t.Fatalf("同意済みからの遷移=%v", requests.states)
			}
		})
	}
}
