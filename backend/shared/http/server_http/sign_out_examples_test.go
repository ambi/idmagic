package server_http_test

// docs/contexts/authentication/scenarios.feature.md の REQ-AUTHENTICATION-035 が宣言する
// 具体例を、サインアウトの 2 つのプロトコル入口から観測する。
//
// この規則が言っているのは「テナントのエンドポイント形式によらず、サーバー側の
// セッションが失効する」ことである。形式の違いは、ブラウザーが送るセッション Cookie の
// 名前として現れる。サブドメイン形式では `__Host-` 接頭辞が付き、パス形式では付かない。
// 失効そのものは応答に現れないので、保存層と、同じ Cookie の再提示の双方から読み直す。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers EX-AUTHENTICATION-035-01, EX-AUTHENTICATION-035-02, EX-AUTHENTICATION-035-03: SAML シングルログアウトと WS-Federation のサインアウトのどちらでも、`__Host-` 接頭辞つきの Cookie でも接頭辞のない Cookie でも、サーバー側のセッションが失効し、同じセッション ID を再提示しても認証されないことを固定する。
func TestSignOutRevokesTheServerSideSessionForEveryEndpointShape(t *testing.T) {
	for _, signOut := range []struct {
		name string
		// target はサインアウトの入口。realm の下に置く。
		target string
	}{
		{"SAML シングルログアウト", "/saml/slo?SAMLRequest=" + url.QueryEscape(samlLogoutRequest(t))},
		{"WS-Federation のサインアウト", "/wsfed?wa=wsignout1.0&wtrealm=" + url.QueryEscape(stack.WsFedRealm)},
	} {
		for _, shape := range []struct {
			name, cookieName string
		}{
			{"サブドメイン形式 (__Host- 接頭辞つき)", "__Host-" + sessionusecases.SessionCookie},
			{"パス形式 (接頭辞なし)", sessionusecases.SessionCookie},
		} {
			t.Run(signOut.name+"/"+shape.name, func(t *testing.T) {
				s := stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithSaml(), stack.WithWsFederation())
				seedSamlLogoutServiceProvider(t, s)
				sessionID := seedSignOutSession(t, s)
				cookie := shape.cookieName + "=" + sessionID

				// 前提: サインアウトの前は、その Cookie で認証が解決できる。
				if resolved := resolveSignOutSession(t, s, cookie); resolved == nil {
					t.Fatal("前提が壊れている: サインアウトの前から認証されていない")
				}

				request := httptest.NewRequest(http.MethodGet,
					"/realms/"+tenancydomain.DefaultRealm+signOut.target, http.NoBody)
				request.Header.Set("Cookie", cookie)
				recorder := httptest.NewRecorder()
				s.Echo.ServeHTTP(recorder, request)
				if recorder.Code >= http.StatusBadRequest {
					t.Fatalf("サインアウト status=%d body=%s", recorder.Code, recorder.Body.String())
				}

				// サーバー側の失効。Cookie を消すだけの実装はここで落ちる。
				found, err := s.SessionStore.Find(context.Background(), sessionID)
				if err != nil {
					t.Fatal(err)
				}
				if found != nil {
					t.Fatalf("サインアウトのあともセッションが有効なまま残っている: %+v", found)
				}
				// ブラウザーの Cookie を復元して同じセッション ID を再提示しても認証されない。
				if resolved := resolveSignOutSession(t, s, cookie); resolved != nil {
					t.Fatalf("復元した Cookie で認証された: %+v", resolved)
				}
			})
		}
	}
}

// seedSignOutSession は失効の対象になる認証済みセッションを 1 本置き、その id を返す。
func seedSignOutSession(t *testing.T, s *stack.Stack) string {
	t.Helper()
	now := time.Now().UTC()
	session := &sessiondomain.LoginSession{
		ID: "sess-sign-out", TenantID: tenancydomain.DefaultTenantID, UserID: stack.UserID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ExpiresAt: now.Add(time.Hour),
	}
	if err := s.SessionStore.Save(s.RealmContext(t, tenancydomain.DefaultRealm), session); err != nil {
		t.Fatal(err)
	}
	return session.ID
}

// resolveSignOutSession は Cookie ヘッダ 1 本から認証文脈を解決する。
func resolveSignOutSession(
	t *testing.T, s *stack.Stack, cookie string,
) *authdomain.AuthenticationContext {
	t.Helper()
	resolved, err := s.Sessions.Resolve(
		s.RealmContext(t, tenancydomain.DefaultRealm),
		authdomain.HTTPHeadersAdapter{H: http.Header{"Cookie": []string{cookie}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

const signOutServiceProviderEntityID = "https://sp.example.com"

// seedSamlLogoutServiceProvider は SLO の返信先を持つ SP を 1 つ登録する。
func seedSamlLogoutServiceProvider(t *testing.T, s *stack.Stack) {
	t.Helper()
	s.SamlSPs.Seed(&samldomain.SamlServiceProvider{
		TenantID: tenancydomain.DefaultTenantID,
		EntityID: signOutServiceProviderEntityID,
		ACSURLs:  []string{"https://sp.example.com/acs"},
		SLOURL:   "https://sp.example.com/saml/slo",
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
			},
		},
	})
}

// samlLogoutRequest は Redirect 束縛で送れる LogoutRequest を組み立てる。
func samlLogoutRequest(t *testing.T) string {
	t.Helper()
	xml := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_logout-1" Version="2.0">` +
		`<saml:Issuer>` + signOutServiceProviderEntityID + `</saml:Issuer>` +
		`<saml:NameID>` + stack.UserID + `</saml:NameID></samlp:LogoutRequest>`
	encoded, err := samldomain.EncodeRedirect([]byte(xml))
	if err != nil {
		t.Fatalf("LogoutRequest の組み立て: %v", err)
	}
	return encoded
}
