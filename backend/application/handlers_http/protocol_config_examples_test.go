package handlers_http_test

// SAML プロトコル設定の更新 (REQ-APPLICATION-005) と、Application ごとのクレーム公開規則
// (REQ-APPLICATION-006) が宣言する具体例を、管理 API の入口で観測する。
//
// 拒否の側は応答だけでなく、保存された設定を読み直して「防いだ効果」を見る。
// 応答を書いてから保存へ進む実装は、応答コードだけを読むテストでは通ってしまう。

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"

	"github.com/labstack/echo/v5"
)

// seedAttributeSchema は、テナントが `employee_number` を `SelfReadable` で、内部属性を
// `Private` で定義した状態を置く。具体例 006-01 の `Given` はこの定義を前提にしており、
// 定義が無ければ規則の保存そのものが可視性 floor で拒否され、許可側を観測できない。
func seedAttributeSchema(t *testing.T) *usermemory.TenantUserAttributeSchemaRepository {
	t.Helper()
	repo := usermemory.NewTenantUserAttributeSchemaRepository()
	now := time.Now().UTC()
	if err := repo.Save(context.Background(), &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{
			{
				Key: "employee_number", Type: idmdomain.AttributeTypeString,
				Visibility: idmdomain.AttrVisibilitySelfReadable,
			},
			{
				Key: "password_reset_token", Type: idmdomain.AttributeTypeString,
				Visibility: idmdomain.AttrVisibilityPrivate,
			},
		},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed attribute schema: %v", err)
	}
	return repo
}

func createApplication(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, body map[string]any) string {
	t.Helper()
	created := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %v status=%d body=%s", body["name"], created.Code, created.Body.String())
	}
	var decoded struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode create: %v; body=%s", err, created.Body.String())
	}
	return decoded.Application.ID
}

type samlDetailView struct {
	Saml struct {
		IDPProfileID                      string                         `json:"idp_profile_id"`
		ACSURLs                           []string                       `json:"acs_urls"`
		SignResponse                      bool                           `json:"sign_response"`
		WantAuthnRequestsSigned           bool                           `json:"want_authn_requests_signed"`
		AuthnRequestSigningCertificatePEM string                         `json:"authn_request_signing_certificate_pem"`
		Rules                             []claimdomain.ClaimMappingRule `json:"rules"`
	} `json:"saml"`
}

func readSamlDetail(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, applicationID string) samlDetailView {
	t.Helper()
	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get application status=%d body=%s", detail.Code, detail.Body.String())
	}
	var view samlDetailView
	if err := json.Unmarshal(detail.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode detail: %v; body=%s", err, detail.Body.String())
	}
	return view
}

//spec:covers REQ-APPLICATION-005, EX-APPLICATION-005-01: ACS URL、署名方針、クレーム規則、IdP プロファイルの割り当てを 1 回の更新で保存し、同じテナントの別の SAML アプリケーションの設定はそのまま残ること。
func TestUpdateSamlConfigSavesOnlyTheTargetServiceProvider(t *testing.T) {
	e, _, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	target := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll SAML", "type": "saml",
		"entity_id": "https://payroll.example/sp", "acs_urls": []string{"https://payroll.example/acs"},
	})
	bystander := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Directory SAML", "type": "saml",
		"entity_id": "https://directory.example/sp", "acs_urls": []string{"https://directory.example/acs"},
	})

	updated := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+target+"/saml", csrf, cookie, map[string]any{
		"acs_urls":      []string{"https://payroll.example/acs/v2"},
		"sign_response": true,
		"rules": []map[string]any{
			{"claim_type": "employee_number", "source": "user_attribute", "source_key": "employee_number"},
		},
		"idp_profile_id": "default",
	})
	if updated.Code != http.StatusNoContent {
		t.Fatalf("update saml status=%d body=%s", updated.Code, updated.Body.String())
	}

	saved := readSamlDetail(t, e, csrf, cookie, target).Saml
	if len(saved.ACSURLs) != 1 || saved.ACSURLs[0] != "https://payroll.example/acs/v2" {
		t.Fatalf("acs_urls=%v, want the updated one", saved.ACSURLs)
	}
	if !saved.SignResponse {
		t.Fatalf("sign_response=%v, want the updated signing policy", saved.SignResponse)
	}
	if len(saved.Rules) != 1 || saved.Rules[0].ClaimType != "employee_number" {
		t.Fatalf("rules=%+v, want the updated claim rule", saved.Rules)
	}
	if saved.IDPProfileID != "default" {
		t.Fatalf("idp_profile_id=%q, want the assigned profile", saved.IDPProfileID)
	}

	// 「SAML サービスプロバイダー設定だけが」更新されることは、更新しなかった隣の SP を
	// 読み直さなければ観測できない。
	untouched := readSamlDetail(t, e, csrf, cookie, bystander).Saml
	if len(untouched.ACSURLs) != 1 || untouched.ACSURLs[0] != "https://directory.example/acs" {
		t.Fatalf("bystander acs_urls=%v, want the original one", untouched.ACSURLs)
	}
	if untouched.SignResponse || len(untouched.Rules) != 0 {
		t.Fatalf("bystander changed: sign_response=%v rules=%+v", untouched.SignResponse, untouched.Rules)
	}
}

//spec:covers REQ-APPLICATION-005, EX-APPLICATION-005-02: AuthnRequest 署名を必須にしながら検証できる証明書を伴わない更新が 400 invalid_request で拒否され、署名方針も証明書も同じ要求が運んだ ACS URL も保存されないこと。
func TestUpdateSamlConfigRefusesAuthnRequestSigningWithoutACertificate(t *testing.T) {
	e, _, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll SAML", "type": "saml",
		"entity_id": "https://payroll.example/sp", "acs_urls": []string{"https://payroll.example/acs"},
	})

	refused := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+applicationID+"/saml", csrf, cookie, map[string]any{
		"want_authn_requests_signed":            true,
		"authn_request_signing_certificate_pem": "not a certificate",
		"acs_urls":                              []string{"https://payroll.example/acs/v2"},
	})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(refused.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, refused.Body.String())
	}
	if problem.Type != "urn:idmagic:error:invalid_request" {
		t.Fatalf("problem type=%q, want the invalid_request refusal", problem.Type)
	}

	// 拒否が防いだ効果。署名必須も証明書も、同じ要求が運んだ ACS URL も保存されていない。
	saved := readSamlDetail(t, e, csrf, cookie, applicationID).Saml
	if saved.WantAuthnRequestsSigned {
		t.Fatalf("want_authn_requests_signed saved despite the refusal")
	}
	if saved.AuthnRequestSigningCertificatePEM != "" {
		t.Fatalf("certificate saved despite the refusal: %q", saved.AuthnRequestSigningCertificatePEM)
	}
	if len(saved.ACSURLs) != 1 || saved.ACSURLs[0] != "https://payroll.example/acs" {
		t.Fatalf("acs_urls=%v changed despite the refusal", saved.ACSURLs)
	}
}

type oidcRulesView struct {
	Oidc struct {
		Rules []claimdomain.ClaimMappingRule `json:"rules"`
	} `json:"oidc"`
}

func readOidcRules(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, applicationID string) []claimdomain.ClaimMappingRule {
	t.Helper()
	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get application status=%d body=%s", detail.Code, detail.Body.String())
	}
	var view oidcRulesView
	if err := json.Unmarshal(detail.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode detail: %v; body=%s", err, detail.Body.String())
	}
	return view.Oidc.Rules
}

// issuedClaimTypes は、保存された規則を製品が共有する 1 本のクレーム解決経路へ通し、
// 実際に発行されるクレーム型を返す。規則が保存されたことと、そのクレームが実際に
// 発行されることは別の主張であり、具体例は後者を言っている。
func issuedClaimTypes(t *testing.T, schema *usermemory.TenantUserAttributeSchemaRepository, rules []claimdomain.ClaimMappingRule) []string {
	t.Helper()
	defs, err := claimusecases.ResolveTenantAttributeDefs(
		context.Background(), tenancydomain.DefaultTenantID, schema)
	if err != nil {
		t.Fatalf("resolve attribute defs: %v", err)
	}
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: "persistent", SourceAttribute: "user_id"},
		Rules:  rules,
	}
	result, err := claimusecases.IssueClaimsWithFloor(policy, claimusecases.Attributes{
		"user_id":         {"alice"},
		"employee_number": {"E-1024"},
	}, defs)
	if err != nil {
		t.Fatalf("issue claims: %v", err)
	}
	types := make([]string, len(result.Claims))
	for i, claim := range result.Claims {
		types[i] = claim.ClaimType
	}
	return types
}

//spec:covers REQ-APPLICATION-006, EX-APPLICATION-006-01: `SelfReadable` な属性を指す規則の保存が ApplicationClaimMappingUpdated を発行し、その Application 向けにだけ `employee_number` が発行され、規則を更新していない隣の Application には発行されないこと。
func TestPerApplicationClaimRuleReleasesTheAttributeOnlyForThatApplication(t *testing.T) {
	e, events, schema := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	payroll := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "payroll", "type": "oidc", "redirect_uris": []string{"https://payroll.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	directory := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "directory", "type": "oidc", "redirect_uris": []string{"https://directory.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})

	updated := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+payroll+"/oidc", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "employee_number", "source": "user_attribute", "source_key": "employee_number"},
		},
	})
	if updated.Code != http.StatusNoContent {
		t.Fatalf("update oidc status=%d body=%s", updated.Code, updated.Body.String())
	}

	if !events.contains("ApplicationClaimMappingUpdated") {
		t.Fatalf("events=%v, want ApplicationClaimMappingUpdated", events.types())
	}
	if got := issuedClaimTypes(t, schema, readOidcRules(t, e, csrf, cookie, payroll)); !slices.Contains(got, "employee_number") {
		t.Fatalf("payroll claims=%v, want employee_number", got)
	}
	if got := issuedClaimTypes(t, schema, readOidcRules(t, e, csrf, cookie, directory)); slices.Contains(got, "employee_number") {
		t.Fatalf("directory claims=%v, want no employee_number", got)
	}
}

//spec:covers REQ-APPLICATION-006, EX-APPLICATION-006-02: `visibility=Private` の属性を `source_key` に指定した規則が 400 invalid_request で拒否され、規則は 1 つも保存されず、その属性が発行されるクレームにも現れないこと。
func TestClaimRuleNamingAPrivateAttributeIsRefusedAndReleasesNothing(t *testing.T) {
	e, events, schema := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	payroll := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "payroll", "type": "oidc", "redirect_uris": []string{"https://payroll.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})

	refused := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+payroll+"/oidc", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "employee_number", "source": "user_attribute", "source_key": "password_reset_token"},
		},
	})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}

	rules := readOidcRules(t, e, csrf, cookie, payroll)
	if len(rules) != 0 {
		t.Fatalf("rules=%+v saved despite the refusal", rules)
	}
	if got := issuedClaimTypes(t, schema, rules); slices.Contains(got, "employee_number") {
		t.Fatalf("claims=%v, want nothing released", got)
	}
	if events.contains("ApplicationClaimMappingUpdated") {
		t.Fatalf("events=%v, want no update event after the refusal", events.types())
	}
}

//spec:covers REQ-APPLICATION-006, EX-APPLICATION-006-03: 予約済みのクレーム型を `claim_type` に指定した規則が 400 invalid_request で拒否され、規則は保存されず、`sub` が規則の値へ置き換わらないこと。
func TestClaimRuleNamingAReservedClaimTypeIsRefusedAndSavesNothing(t *testing.T) {
	e, events, schema := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	payroll := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "payroll", "type": "oidc", "redirect_uris": []string{"https://payroll.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})

	refused := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+payroll+"/oidc", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "sub", "source": "fixed", "fixed_value": "attacker-controlled"},
		},
	})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}

	rules := readOidcRules(t, e, csrf, cookie, payroll)
	if len(rules) != 0 {
		t.Fatalf("rules=%+v saved despite the refusal", rules)
	}
	if got := issuedClaimTypes(t, schema, rules); slices.Contains(got, "sub") {
		t.Fatalf("claims=%v, want no engine-controlled sub from a rule", got)
	}
	if events.contains("ApplicationClaimMappingUpdated") {
		t.Fatalf("events=%v, want no update event after the refusal", events.types())
	}
}

// recordedEvents は発行されたドメインイベントの型名を発行順に覚える。具体例の `Then` は
// 「どのイベントが発行されるか」で書かれており、応答だけを読むテストは、設定は保存したが
// 通知を出さない実装を通してしまう。
type recordedEvents struct {
	mu    sync.Mutex
	names []string
}

func (r *recordedEvents) record(event spec.DomainEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, event.EventType())
}

func (r *recordedEvents) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.names)
}

func (r *recordedEvents) contains(name string) bool {
	return slices.Contains(r.types(), name)
}

// reset は、種を置く段で発行されたイベントを捨てる。拒否の後に「更新イベントが出ていない」
// ことを見るには、拒否より前の発行を記録から外さなければ区別できない。
func (r *recordedEvents) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = nil
}

// newApplicationHandlerRecordingEvents は、既定の fixture が持たない 2 つの配線だけを足す。
// 属性スキーマが無いと可視性 floor の許可側を観測できず、イベントの記録先が無いと
// 具体例の `Then` が言うイベントを観測できない。
func newApplicationHandlerRecordingEvents(
	t *testing.T,
) (*echo.Echo, *recordedEvents, *usermemory.TenantUserAttributeSchemaRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	// 具体例が名指す被割り当て者。割り当ての保存は主体の実在を確かめないが、
	// 実在しない id で観測すると具体例と違う世界を固定することになる。
	users.Seed(&userdomain.User{
		ID: "alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	// admin ロールを持たない認証済み利用者。ロール境界の拒否は、認証が通ったうえで
	// ロールだけが足りない要求でなければ観測できない。
	users.Seed(&userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	events := &recordedEvents{}
	schema := seedAttributeSchema(t)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:   "http://idp.test",
		Emit:     events.record,
		UserRepo: users, GroupRepo: groupmemory.NewGroupRepository(),
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			SignInPolicyRepo:        appmemory.NewSignInPolicyRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Tenancy:       tenancy.Module{AttrSchemaRepo: schema},
		Saml:          saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		OAuth2:        oauth2.Module{ClientRepo: oauth2memory.NewClientRepository()},
		WsFederation:  wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, events, schema
}
