package handlers_http_test

// 主要ユースケース追跡: REQ-OAUTH2-035。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

//spec:covers EX-OAUTH2-035-01: 作成の応答だけが client_secret を運び、更新した redirect_uris は保存され、削除まで通すと 3 つの Admin イベントが順に発行される。
//verify:operation CreateAdminOAuth2Client: client の作成、取得、更新、削除の HTTP 応答と保存結果
//verify:operation GetAdminOAuth2Client: client の作成、取得、更新、削除の HTTP 応答と保存結果
//verify:operation UpdateAdminOAuth2Client: client の作成、取得、更新、削除の HTTP 応答と保存結果
//verify:operation DeleteAdminOAuth2Client: client の作成、取得、更新、削除の HTTP 応答と保存結果
func TestAdminOAuth2ClientCRUD(t *testing.T) {
	e, clients, events := newAdminOAuth2ClientHandler(t)
	csrf, cookie := adminCSRF(t, e)

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/clients", csrf, cookie, map[string]any{
		"client_name":                          "Portal",
		"client_type":                          "confidential",
		"redirect_uris":                        []string{"https://portal.example/callback"},
		"grant_types":                          []string{"authorization_code"},
		"response_types":                       []string{"code"},
		"token_endpoint_auth_method":           "client_secret_basic",
		"backchannel_logout_uri":               "https://portal.example/backchannel-logout",
		"backchannel_logout_session_required":  true,
		"frontchannel_logout_uri":              "https://portal.example/frontchannel-logout",
		"frontchannel_logout_session_required": true,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		Client struct {
			ClientID string `json:"client_id"`
		} `json:"client"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Client.ClientID == "" || created.ClientSecret == "" {
		t.Fatalf("create response=%s", create.Body.String())
	}
	if strings.Contains(create.Body.String(), "client_secret_hash") {
		t.Fatalf("secret hash leaked: %s", create.Body.String())
	}
	createBody := oauthContractJSON(t, create, http.StatusCreated)
	clientBody := createBody["client"].(map[string]any)
	if clientBody["client_id"] != created.Client.ClientID || clientBody["tenant_id"] != tenancydomain.DefaultTenantID || clientBody["client_name"] != "Portal" || clientBody["client_type"] != "confidential" || clientBody["token_endpoint_auth_method"] != "client_secret_basic" || !reflect.DeepEqual(clientBody["redirect_uris"], []any{"https://portal.example/callback"}) || !reflect.DeepEqual(clientBody["grant_types"], []any{"authorization_code"}) || !reflect.DeepEqual(clientBody["response_types"], []any{"code"}) {
		t.Fatalf("created client=%v", clientBody)
	}
	if clientBody["id_token_signed_response_alg"] != "PS256" || clientBody["fapi_profile"] != "none" || clientBody["require_pushed_authorization_requests"] != false || clientBody["dpop_bound_access_tokens"] != false {
		t.Fatalf("created metadata=%v", clientBody)
	}
	if _, ok := clientBody["scope"].(string); !ok {
		t.Fatalf("scope=%v", clientBody)
	}
	oauthContractTime(t, clientBody, "created_at")

	get := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients/"+created.Client.ClientID, http.NoBody)
	get.Header.Set("X-Demo-Sub", "admin")
	getResponse := httptest.NewRecorder()
	e.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(getResponse.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if _, exists := got["client_secret"]; exists {
		t.Fatalf("secret leaked after create: %s", getResponse.Body.String())
	}
	if _, exists := got["client_secret_hash"]; exists {
		t.Fatalf("secret hash leaked after create: %s", getResponse.Body.String())
	}
	if got["backchannel_logout_uri"] != "https://portal.example/backchannel-logout" || got["backchannel_logout_session_required"] != true || got["frontchannel_logout_uri"] != "https://portal.example/frontchannel-logout" || got["frontchannel_logout_session_required"] != true {
		t.Fatalf("logout metadata not returned: %s", getResponse.Body.String())
	}
	if getBody := oauthContractJSON(t, getResponse, http.StatusOK); !reflect.DeepEqual(getBody, clientBody) {
		t.Fatalf("get=%v created=%v", getBody, clientBody)
	}

	update := adminJSONRequest(
		t, e, http.MethodPatch, "/api/admin/v1/clients/"+created.Client.ClientID, csrf, cookie,
		map[string]any{"redirect_uris": []string{"https://portal.example/new-callback"}, "backchannel_logout_uri": "https://portal.example/new-backchannel-logout", "backchannel_logout_session_required": false, "frontchannel_logout_uri": "https://portal.example/new-frontchannel-logout", "frontchannel_logout_session_required": false},
	)
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	updated := oauthContractJSON(t, update, http.StatusOK)
	if updated["client_id"] != created.Client.ClientID || updated["tenant_id"] != clientBody["tenant_id"] || updated["created_at"] != clientBody["created_at"] || !reflect.DeepEqual(updated["redirect_uris"], []any{"https://portal.example/new-callback"}) || updated["backchannel_logout_uri"] != "https://portal.example/new-backchannel-logout" || updated["backchannel_logout_session_required"] != false || updated["frontchannel_logout_uri"] != "https://portal.example/new-frontchannel-logout" || updated["frontchannel_logout_session_required"] != false {
		t.Fatalf("updated=%v", updated)
	}
	stored, err := clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, created.Client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.RedirectURIs) != 1 ||
		stored.RedirectURIs[0] != "https://portal.example/new-callback" || stored.BackChannelLogoutURI == nil || *stored.BackChannelLogoutURI != "https://portal.example/new-backchannel-logout" || stored.BackChannelLogoutSessionRequired || stored.FrontChannelLogoutURI == nil || *stored.FrontChannelLogoutURI != "https://portal.example/new-frontchannel-logout" || stored.FrontChannelLogoutSessionRequired {
		t.Fatalf("updated client=%+v", stored)
	}

	deleted := adminJSONRequest(
		t, e, http.MethodDelete, "/api/admin/v1/clients/"+created.Client.ClientID, csrf, cookie, nil,
	)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	if deleted.Body.Len() != 0 {
		t.Fatalf("204 body=%s", deleted.Body.String())
	}
	stored, err = clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, created.Client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		t.Fatalf("client still exists: %+v", stored)
	}
	gotEvents := make([]string, len(*events))
	for i, event := range *events {
		gotEvents[i] = event.EventType()
	}
	wantEvents := []string{"AdminOAuth2ClientCreated", "AdminOAuth2ClientUpdated", "AdminOAuth2ClientDeleted"}
	if strings.Join(gotEvents, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("events=%v want=%v", gotEvents, wantEvents)
	}
}

//spec:covers EX-OAUTH2-035-01: 管理 API が返すのは所属テナントのクライアントだけで、別テナントに同じ client_id があっても参照できない。
func TestAdminOAuth2ClientCannotCrossTenantBoundary(t *testing.T) {
	e, clients, _ := newAdminOAuth2ClientHandler(t)
	now := time.Now().UTC()
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: "acme", ClientID: "portal", ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://portal.example/callback"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now,
	})
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients/portal", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/clients", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	response = httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", response.Code, response.Body.String())
	}
}

// 別テナントのクライアントは存在しないものとして扱う。拒否の状態行だけでなく、応答本文が
// 存在しない client_id と区別できないことまで読まなければ、id の存在を推測させる実装を
// 見逃す。
//
//spec:covers EX-OAUTH2-035-02: 別テナントの管理者による参照、更新、削除は、存在しない client_id と同じ 404 client_not_found の本文で拒否され、acme の portal は変更も削除もされず、Admin イベントも発行されない。
func TestAdminOAuth2ClientOfAnotherTenantIsAnsweredAsNonexistent(t *testing.T) {
	e, clients, events := newAdminOAuth2ClientHandler(t)
	now := time.Now().UTC()
	portal := &oauthdomain.OAuth2Client{
		TenantID: "acme", ClientID: "portal", ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://portal.example/callback"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now, UpdatedAt: now,
	}
	clients.Seed(portal)
	csrf, cookie := adminCSRF(t, e)

	for _, operation := range []struct {
		method string
		body   any
	}{
		{http.MethodGet, nil},
		{http.MethodPatch, map[string]any{"redirect_uris": []string{"https://attacker.example/callback"}}},
		{http.MethodDelete, nil},
	} {
		crossTenant := adminJSONRequest(t, e, operation.method, "/api/admin/v1/clients/portal", csrf, cookie, operation.body)
		nonexistent := adminJSONRequest(t, e, operation.method, "/api/admin/v1/clients/no-such-client", csrf, cookie, operation.body)
		if crossTenant.Code != http.StatusNotFound || nonexistent.Code != http.StatusNotFound {
			t.Fatalf("%s: cross-tenant status=%d nonexistent status=%d, want 404 for both body=%s",
				operation.method, crossTenant.Code, nonexistent.Code, crossTenant.Body.String())
		}
		crossTenantProblem := problemWithoutInstance(t, crossTenant.Body.Bytes())
		if crossTenantProblem["type"] != "urn:idmagic:error:client_not_found" {
			t.Fatalf("%s: type=%v, want urn:idmagic:error:client_not_found body=%s",
				operation.method, crossTenantProblem["type"], crossTenant.Body.String())
		}
		nonexistentProblem := problemWithoutInstance(t, nonexistent.Body.Bytes())
		if !reflect.DeepEqual(crossTenantProblem, nonexistentProblem) {
			t.Fatalf("%s: cross-tenant body=%v differs from nonexistent body=%v",
				operation.method, crossTenantProblem, nonexistentProblem)
		}
		if strings.Contains(crossTenant.Body.String(), "portal.example") {
			t.Fatalf("%s: 拒否した応答が対象の設定を運んでいる body=%s", operation.method, crossTenant.Body.String())
		}
	}

	stored, err := clients.FindByID(context.Background(), "acme", "portal")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil {
		t.Fatal("別テナントの管理者の削除で acme の portal が消えた")
	}
	if !reflect.DeepEqual(stored.RedirectURIs, portal.RedirectURIs) {
		t.Fatalf("redirect_uris=%v, want %v (別テナントの管理者の更新が保存された)", stored.RedirectURIs, portal.RedirectURIs)
	}
	if len(*events) != 0 {
		t.Fatalf("拒否した操作がイベントを発行した: %v", *events)
	}
}

//spec:covers REQ-OAUTH2-035: デモの非対称認証と CIBA のグラントを管理 API から登録できる。
func TestAdminRegistersAgentDemoClient(t *testing.T) {
	e, clients, _ := newAdminOAuth2ClientHandler(t)
	csrf, cookie := adminCSRF(t, e)
	credential := testing_clientauth.NewPrivateKeyJWT(t, "agent-demo")
	grants := []string{"client_credentials", "urn:openid:params:grant-type:ciba"}
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/clients", csrf, cookie, map[string]any{
		"client_name": "文書閲覧エージェントのデモ", "client_type": "confidential",
		"redirect_uris": []string{}, "grant_types": grants,
		"token_endpoint_auth_method": "private_key_jwt", "scope": "openid profile tools:read",
		"jwks": credential.JWKS(), "fapi_profile": "none",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	body := oauthContractJSON(t, create, http.StatusCreated)
	clientBody := body["client"].(map[string]any)
	clientID, _ := clientBody["client_id"].(string)
	if clientID == "" || clientBody["token_endpoint_auth_method"] != "private_key_jwt" ||
		!reflect.DeepEqual(clientBody["grant_types"], []any{grants[0], grants[1]}) {
		t.Fatalf("created metadata=%v", clientBody)
	}
	if _, hasSecret := body["client_secret"]; hasSecret {
		t.Fatal("private_key_jwt client was issued a shared secret")
	}
	saved, err := clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, clientID)
	if err != nil || saved == nil {
		t.Fatalf("saved client missing: %v", err)
	}
	if saved.TokenEndpointAuthMethod != oauthdomain.AuthMethodPrivateKeyJwt ||
		!reflect.DeepEqual(saved.GrantTypes, []spec.GrantType{spec.GrantClientCredentials, spec.GrantCiba}) ||
		len(saved.RedirectURIs) != 0 || !reflect.DeepEqual(saved.JWKS, credential.JWKS()) {
		t.Fatal("saved metadata does not match the requested grants and credentials")
	}
}

// problemWithoutInstance は problem 本文から要求ごとに変わる instance を除いた残りを返す。
func problemWithoutInstance(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var problem map[string]any
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("problem 本文を JSON として読めない body=%s: %v", body, err)
	}
	delete(problem, "instance")
	return problem
}

func newAdminOAuth2ClientHandler(
	t *testing.T,
) (*echo.Echo, *oauth2memory.OAuth2ClientRepository, *[]spec.DomainEvent) {
	t.Helper()
	clients := oauth2memory.NewClientRepository()
	e, events := newAdminOAuth2ClientHandlerUsingRepository(t, clients)
	return e, clients, events
}

func newAdminOAuth2ClientHandlerUsingRepository(t *testing.T, clients clientports.OAuth2ClientRepository) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	return newAdminOAuth2ClientHandlerUsingModule(t, oauth2.Module{ClientRepo: clients}.WithMemoryClientDeletion(nil, nil), tenancy.Module{})
}

func newAdminOAuth2ClientHandlerUsingModule(t *testing.T, module oauth2.Module, tenancyModule tenancy.Module) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	users := idmanagement.NewMemoryModule().UserRepo
	now := time.Now().UTC()
	_ = users.Save(context.Background(), &userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	_ = users.Save(context.Background(), &userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	events := []spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",

		Emit: func(event spec.DomainEvent) {
			events = append(events, event)
		}, OAuth2: module, Tenancy: tenancyModule, UserRepo: users,
		AuthnResolver: authentication.DemoHeaderResolver{},
	})
	return e, &events
}

type failingClientDeletion struct {
	clientports.OAuth2ClientRepository
	err error
}

func (r failingClientDeletion) Delete(context.Context, string, string) error { return r.err }

//spec:covers REQ-SYSTEM-023: 保存先から返る内部障害を、クライアント削除の実経路が公開しない。
//verify:operation DeleteAdminOAuth2Client: 未分類の保存障害は 500 の Problem Details となり、クライアントとイベントを変えない。
func TestClientDeletionDoesNotDiscloseDependencyErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New("refresh_tokens_client_fkey SQLSTATE 23001"),
		errors.New("runtime: private-marker\ngoroutine 42\n/internal/library.go:123"),
		fmt.Errorf("dependency failed: %w", errors.New("private-marker: driver library error at 10.0.0.5:5432")),
	} {
		repo := oauth2memory.NewClientRepository()
		client := &oauthdomain.OAuth2Client{TenantID: tenancydomain.DefaultTenantID, ClientID: "client"}
		repo.Seed(client)
		e, events := newAdminOAuth2ClientHandlerUsingRepository(t, failingClientDeletion{OAuth2ClientRepository: repo, err: failure})
		csrf, cookie := adminCSRF(t, e)
		response := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/clients/client", csrf, cookie, nil)
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "urn:idmagic:error:internal_server_error") {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		for _, content := range []string{response.Body.String(), fmt.Sprint(response.Header())} {
			for _, secret := range []string{"refresh_tokens", "SQLSTATE", "private-marker", "goroutine", "/internal/", "10.0.0.5"} {
				if strings.Contains(content, secret) {
					t.Fatalf("internal detail was disclosed: %s", content)
				}
			}
		}
		stored, err := repo.FindByID(t.Context(), client.TenantID, client.ClientID)
		if err != nil || !reflect.DeepEqual(stored, client) || len(*events) != 0 {
			t.Fatalf("stored=%+v err=%v events=%v", stored, err, *events)
		}
	}
}

//spec:covers REQ-OAUTH2-080: 管理コンソールとポータルの名前を変えても、組み込みクライアントは削除できない。
//verify:operation DeleteAdminOAuth2Client: 組み込みクライアントの削除は 409 の Problem Details となり、保存状態とイベントは変わらない。
func TestFirstPartyClientDeletionIsRefused(t *testing.T) {
	for _, clientID := range []string{"console", "portal"} {
		t.Run(clientID, func(t *testing.T) {
			e, clients, events := newAdminOAuth2ClientHandler(t)
			client := &oauthdomain.OAuth2Client{
				TenantID: tenancydomain.DefaultTenantID, ClientID: clientID,
				ClientName: new("Renamed application"), FirstParty: true,
			}
			clients.Seed(client)
			csrf, cookie := adminCSRF(t, e)
			response := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/clients/"+clientID, csrf, cookie, nil)
			if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "urn:idmagic:error:first_party_client_protected") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			stored, err := clients.FindByID(t.Context(), client.TenantID, clientID)
			if err != nil || !reflect.DeepEqual(stored, client) || len(*events) != 0 {
				t.Fatalf("stored=%+v error=%v events=%v", stored, err, *events)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-035, REQ-TENANCY-036: 管理 API で利用された単独のクライアントを消すと、同意と束縛も消え、使用量が減る。
//verify:operation DeleteAdminOAuth2Client: 同意を持つ単独のクライアントの削除は 204 となり、同意、束縛、使用量を片付けて一度だけイベントを発行する。
func TestAdminClientDeletionRemovesConsentsAndReleasesQuota(t *testing.T) {
	ctx := context.Background()
	tenantID := tenancydomain.DefaultTenantID
	now := time.Now().UTC()
	clients := oauth2memory.NewClientRepository()
	consents := oauth2memory.NewConsentRepository()
	tenancyModule := tenancy.NewMemoryModule()
	quota := tenancyModule.QuotaRepo
	client := &oauthdomain.OAuth2Client{TenantID: tenantID, ClientID: "used-client"}
	clients.Seed(client)
	if err := consents.Save(ctx, tenantID, &oauthdomain.Consent{UserID: "user-1", ClientID: client.ClientID, Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{tenancydomain.ResourceOAuth2Clients, tenancydomain.ResourceConsents} {
		if err := quota.CheckAndIncrement(ctx, tenantID, resource, 1); err != nil {
			t.Fatal(err)
		}
	}
	before, err := quota.GetUsage(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	unbound := []string{}
	module := oauth2.Module{ClientRepo: clients, ConsentRepo: consents}.WithMemoryClientDeletion(
		func(_ context.Context, tenantID, clientID string) error {
			unbound = append(unbound, tenantID+"/"+clientID)
			return nil
		}, quota,
	)
	e, events := newAdminOAuth2ClientHandlerUsingModule(t, module, tenancyModule)
	csrf, cookie := adminCSRF(t, e)

	response := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/clients/"+client.ClientID, csrf, cookie, nil)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if stored, err := clients.FindByID(ctx, tenantID, client.ClientID); err != nil || stored != nil {
		t.Fatalf("client=%+v error=%v", stored, err)
	}
	if stored, err := consents.Find(ctx, tenantID, "user-1", client.ClientID); err != nil || stored != nil {
		t.Fatalf("consent=%+v error=%v", stored, err)
	}
	if !reflect.DeepEqual(unbound, []string{tenantID + "/" + client.ClientID}) {
		t.Fatalf("unbound=%v", unbound)
	}
	after, err := quota.GetUsage(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if after.OAuth2Clients != before.OAuth2Clients-1 || after.Consents != before.Consents-1 {
		t.Fatalf("usage before=%+v after=%+v", before, after)
	}
	if len(*events) != 1 {
		t.Fatalf("events=%+v", *events)
	}
	if _, ok := (*events)[0].(*oauthdomain.AdminOAuth2ClientDeleted); !ok {
		t.Fatalf("event=%T", (*events)[0])
	}
}

//spec:covers REQ-OAUTH2-035: Application が所有するクライアントの削除は、同意と使用量を変えずに 409 で拒否する。
func TestAdminClientDeletionOfApplicationOwnedClientKeepsItsState(t *testing.T) {
	ctx := context.Background()
	tenantID := tenancydomain.DefaultTenantID
	now := time.Now().UTC()
	clients := oauth2memory.NewClientRepository()
	consents := oauth2memory.NewConsentRepository()
	tenancyModule := tenancy.NewMemoryModule()
	client := &oauthdomain.OAuth2Client{TenantID: tenantID, ClientID: "owned-client", ApplicationID: "app-1"}
	clients.Seed(client)
	consent := &oauthdomain.Consent{UserID: "user-1", ClientID: client.ClientID, Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := consents.Save(ctx, tenantID, consent); err != nil {
		t.Fatal(err)
	}
	if err := tenancyModule.QuotaRepo.CheckAndIncrement(ctx, tenantID, tenancydomain.ResourceOAuth2Clients, 1); err != nil {
		t.Fatal(err)
	}
	before, err := tenancyModule.QuotaRepo.GetUsage(ctx, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	module := oauth2.Module{ClientRepo: clients, ConsentRepo: consents}.WithMemoryClientDeletion(nil, tenancyModule.QuotaRepo)
	e, events := newAdminOAuth2ClientHandlerUsingModule(t, module, tenancyModule)
	csrf, cookie := adminCSRF(t, e)

	response := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/clients/"+client.ClientID, csrf, cookie, nil)

	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "application_owned_protocol") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if stored, err := consents.Find(ctx, tenantID, "user-1", client.ClientID); err != nil || stored == nil {
		t.Fatalf("consent=%+v error=%v", stored, err)
	}
	if after, err := tenancyModule.QuotaRepo.GetUsage(ctx, tenantID); err != nil || *after != *before || len(*events) != 0 {
		t.Fatalf("usage before=%+v after=%+v error=%v events=%v", before, after, err, *events)
	}
}
