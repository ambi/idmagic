package handlers_http_test

// 実装にだけあった挙動を規則として書き起こした Tenancy の規則を、HTTP の入口から固定する。
// どのテストも、規則が定める応答と、規則が変えないと定める状態の両方を読む。

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

// rulesOptions は既定の配線から差し替える口である。空のフィールドはメモリーの実装を使う。
type rulesOptions struct {
	quotas tenantports.QuotaRepository
	sender sharednotification.EmailSender
	signer samltoken.SignerProvider
}

type rulesServer struct {
	e         *echo.Echo
	tenants   *memory.TenantRepository
	branding  *memory.TenantBrandingRepository
	templates *memory.NotificationTemplateRepository
	schemas   *usermemory.TenantUserAttributeSchemaRepository
	groups    *groupmemory.GroupRepository
	events    *[]spec.DomainEvent
}

// newRulesServer は default (制御面) と acme の 2 テナントを持つサーバーを production と同じ
// `httpadapter.Register` で組み立てる。actor が要求の主体になる。
func newRulesServer(t *testing.T, actor *userdomain.User, options rulesOptions) *rulesServer {
	t.Helper()
	ctx := context.Background()
	s := &rulesServer{
		tenants:   memory.NewTenantRepository(),
		branding:  memory.NewTenantBrandingRepository(),
		templates: memory.NewNotificationTemplateRepository(),
		schemas:   usermemory.NewTenantUserAttributeSchemaRepository(),
		groups:    groupmemory.NewGroupRepository(),
		e:         echo.New(),
	}
	for _, tenant := range []*domain.Tenant{activeTenant(domain.DefaultTenantID, "Default"), activeTenant("acme", "Acme")} {
		if err := s.tenants.Save(ctx, tenant); err != nil {
			t.Fatal(err)
		}
	}
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(actor)
	events := make([]spec.DomainEvent, 0)
	s.events = &events
	quotas := options.quotas
	if quotas == nil {
		quotas = memory.NewQuotaRepository()
	}
	sender := options.sender
	if sender == nil {
		sender = &recordingSender{}
	}
	notifier := &template.Notifier{
		Sender: sender,
		Tenant: tenantusecases.TenantNotificationSource{
			TenantRepo: s.tenants, BrandingRepo: s.branding, TemplateRepo: s.templates,
		},
	}
	httpadapter.Register(s.e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo:     s.tenants,
		Emit:           func(event spec.DomainEvent) { events = append(events, event) },
		UserRepo:       userRepo,
		GroupRepo:      s.groups,
		AttrSchemaRepo: s.schemas,
		EmailSender:    sender,
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}},
		FederationSigner: options.signer,
		Notification:     sharednotification.Module{EmailSender: sender, Notifier: notifier},
		Tenancy: tenancy.Module{
			TenantRepo:            s.tenants,
			BrandingRepo:          s.branding,
			BrandingAssetStore:    memory.NewTenantBrandingAssetStore(),
			NotificationTemplates: s.templates,
			QuotaRepo:             quotas,
		},
	})
	return s
}

func (s *rulesServer) get(t *testing.T, path string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}
	recorder := httptest.NewRecorder()
	s.e.ServeHTTP(recorder, request)
	return recorder
}

// send は Origin と CSRF トークンを揃えた状態変更要求を送る。拒否が CSRF の手前で起きたと
// 取り違えないよう、ブラウザーの資格情報は常に正しく揃える。
func (s *rulesServer) send(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return s.sendRaw(t, method, path, "application/json", bytes.NewReader(payload))
}

func (s *rulesServer) sendRaw(t *testing.T, method, path, contentType string, body *bytes.Reader) *httptest.ResponseRecorder {
	t.Helper()
	csrf, cookie := passwordResetContextCSRF(t, s.e, tenantPrefix(path)+"/api/auth/password-reset-context")
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	s.e.ServeHTTP(recorder, request)
	return recorder
}

func (s *rulesServer) upload(t *testing.T, kind string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "asset.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return s.sendRaw(t, http.MethodPost, "/realms/acme/api/admin/v1/tenant/branding/assets/"+kind,
		writer.FormDataContentType(), bytes.NewReader(body.Bytes()))
}

func (s *rulesServer) eventsOfType(eventType string) []spec.DomainEvent {
	var found []spec.DomainEvent
	for _, event := range *s.events {
		if event.EventType() == eventType {
			found = append(found, event)
		}
	}
	return found
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("body = %s: %v", recorder.Body.String(), err)
	}
	return body
}

func requireStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, body = %s, want %d", recorder.Code, recorder.Body.String(), want)
	}
}

func requireProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	requireStatus(t, recorder, status)
	if !strings.Contains(recorder.Body.String(), code) {
		t.Fatalf("body = %s, want %s", recorder.Body.String(), code)
	}
}

func systemAdmin() *userdomain.User {
	return settingsActor("sysadmin", domain.DefaultTenantID, []string{"system_admin"})
}

func pngOfSize(size int) []byte {
	data := make([]byte, size)
	copy(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	return data
}

// recordingSender は送ったメールを記録し、ok が false なら送信の失敗を返す。
type recordingSender struct {
	fail bool
	sent []sharednotification.EmailMessage
}

func (s *recordingSender) SendEmail(_ context.Context, message sharednotification.EmailMessage) bool {
	s.sent = append(s.sent, message)
	return !s.fail
}

// failingQuotaRepository は上限と使用量の読み取りだけを失敗させる。
type failingQuotaRepository struct {
	tenantports.QuotaRepository
}

func (failingQuotaRepository) GetQuota(context.Context, string) (*domain.TenantQuota, error) {
	return nil, errors.New("quota store unavailable")
}

func (failingQuotaRepository) GetUsage(context.Context, string) (*domain.TenantUsage, error) {
	return nil, errors.New("quota store unavailable")
}

// ---- テナントのライフサイクル ----

const tenantsPath = "/realms/default/api/admin/v1/tenants"

//spec:covers EX-TENANCY-025-01: 前後の空白を除いた realm と表示名で、active かつ path のテナントを UUID の id で作成し、TenantCreated を一度だけ発行する。
func TestCreateTenantTrimsInputAndStartsActiveOnPath(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})

	response := s.send(t, http.MethodPost, tenantsPath, map[string]string{"realm": " globex ", "display_name": " Globex "})
	requireStatus(t, response, http.StatusCreated)
	body := decodeJSON(t, response)
	if body["realm"] != "globex" || body["display_name"] != "Globex" ||
		body["status"] != "active" || body["endpoint_style"] != "path" {
		t.Fatalf("created tenant = %v", body)
	}
	id, _ := body["id"].(string)
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("id = %q, want a server-assigned UUID", id)
	}
	stored, err := s.tenants.FindByRealm(context.Background(), "globex")
	if err != nil || stored == nil || stored.ID != id {
		t.Fatalf("stored = %+v, err = %v", stored, err)
	}
	if created := s.eventsOfType("TenantCreated"); len(created) != 1 {
		t.Fatalf("TenantCreated events = %d, want 1", len(created))
	}
}

//spec:covers EX-TENANCY-025-02, EX-TENANCY-025-03, EX-TENANCY-025-04: 予約語、ハイフンで終わる realm、xn-- で始まる realm、ほかの単一 DNS ラベルでない realm は 400 invalid_request になり、テナントもイベントも増えない。
func TestCreateTenantRefusesRealmsThatAreNotAnUnreservedDNSLabel(t *testing.T) {
	for _, realm := range []string{
		"login", "account", "acme-", "-acme", "xn--acme", "Acme", "a.b", "a_b", "", strings.Repeat("a", 64),
	} {
		t.Run(realm, func(t *testing.T) {
			s := newRulesServer(t, systemAdmin(), rulesOptions{})
			before, err := s.tenants.FindAll(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			response := s.send(t, http.MethodPost, tenantsPath, map[string]string{"realm": realm, "display_name": "X"})
			requireProblem(t, response, http.StatusBadRequest, "invalid_request")
			after, err := s.tenants.FindAll(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(after) != len(before) || len(*s.events) != 0 {
				t.Fatalf("tenants %d -> %d, events = %d", len(before), len(after), len(*s.events))
			}
		})
	}

	// 対照: 63 文字で内側にハイフンを持つ realm は受け付ける。
	s := newRulesServer(t, systemAdmin(), rulesOptions{})
	longest := "a" + strings.Repeat("-b", 31)
	requireStatus(t, s.send(t, http.MethodPost, tenantsPath, map[string]string{"realm": longest, "display_name": "X"}), http.StatusCreated)
}

//spec:covers EX-TENANCY-025-05: 既存の realm での作成は 409 tenant_conflict になり、テナントは増えず、イベントも発行されない。
func TestCreateTenantRefusesADuplicateRealm(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})

	response := s.send(t, http.MethodPost, tenantsPath, map[string]string{"realm": "acme", "display_name": "Other Acme"})
	requireProblem(t, response, http.StatusConflict, "tenant_conflict")
	all, err := s.tenants.FindAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || len(*s.events) != 0 {
		t.Fatalf("tenants = %d, events = %d", len(all), len(*s.events))
	}
	if kept, _ := s.tenants.FindByRealm(context.Background(), "acme"); kept == nil || kept.DisplayName != "Acme" {
		t.Fatalf("existing tenant = %+v", kept)
	}
}

//spec:covers EX-TENANCY-026-01: 一覧は id の昇順で、各テナントが quota と usage を持つ。
func TestListTenantsOrdersByIDWithQuotaAndUsage(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})
	if err := s.tenants.Save(context.Background(), activeTenant("0-first", "First")); err != nil {
		t.Fatal(err)
	}

	response := s.get(t, tenantsPath)
	requireStatus(t, response, http.StatusOK)
	var body struct {
		Tenants []map[string]any `json:"tenants"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(body.Tenants))
	for _, tenant := range body.Tenants {
		ids = append(ids, tenant["id"].(string))
		if _, ok := tenant["quota"]; !ok {
			t.Fatalf("tenant %v has no quota", tenant["id"])
		}
		if _, ok := tenant["usage"]; !ok {
			t.Fatalf("tenant %v has no usage", tenant["id"])
		}
	}
	if len(ids) != 3 || !slices.IsSorted(ids) {
		t.Fatalf("ids = %v, want three ids in ascending order", ids)
	}
}

//spec:covers EX-TENANCY-026-02: 上限と使用量を読み取れなくても一覧は 200 で返り、各テナントは quota と usage を持たない。
func TestListTenantsOmitsQuotaItCannotRead(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{quotas: failingQuotaRepository{memory.NewQuotaRepository()}})

	response := s.get(t, tenantsPath)
	requireStatus(t, response, http.StatusOK)
	var body struct {
		Tenants []map[string]any `json:"tenants"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tenants) != 2 {
		t.Fatalf("tenants = %d, want 2", len(body.Tenants))
	}
	for _, tenant := range body.Tenants {
		if _, ok := tenant["quota"]; ok {
			t.Fatalf("tenant %v carries a quota", tenant["id"])
		}
		if _, ok := tenant["usage"]; ok {
			t.Fatalf("tenant %v carries a usage", tenant["id"])
		}
	}
}

//spec:covers REQ-TENANCY-045: realm で指定したテナントを上限と使用量とともに返し、読めない上限は省き、存在しない realm は 404 にする。
func TestGetTenantReturnsTheRealmWithQuotaAndUsage(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})
	found := s.get(t, tenantsPath+"/acme")
	requireStatus(t, found, http.StatusOK)
	body := decodeJSON(t, found)
	if body["realm"] != "acme" {
		t.Fatalf("realm = %v, want acme", body["realm"])
	}
	for _, field := range []string{"quota", "usage"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("tenant has no %s: %v", field, body)
		}
	}
	requireProblem(t, s.get(t, tenantsPath+"/ghost"), http.StatusNotFound, "tenant_not_found")

	unreadable := newRulesServer(t, systemAdmin(), rulesOptions{quotas: failingQuotaRepository{memory.NewQuotaRepository()}})
	omitted := unreadable.get(t, tenantsPath+"/acme")
	requireStatus(t, omitted, http.StatusOK)
	for _, field := range []string{"quota", "usage"} {
		if _, ok := decodeJSON(t, omitted)[field]; ok {
			t.Fatalf("tenant carries %s it could not read: %s", field, omitted.Body.String())
		}
	}
}

//spec:covers REQ-TENANCY-044: System 管理者の更新は要求の項目を TenantUpdated に記録し、一つの項目の拒否で何も保存せず、存在しない realm は 404 にする。
func TestSystemAdminUpdatesATenantUnderTheTenantAdminRules(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})

	updated := s.send(t, http.MethodPatch, tenantsPath+"/acme", map[string]any{"display_name": " Acme Corp "})
	requireStatus(t, updated, http.StatusOK)
	if name := decodeJSON(t, updated)["display_name"]; name != "Acme Corp" {
		t.Fatalf("display_name = %v, want the trimmed Acme Corp", name)
	}
	events := s.eventsOfType("TenantUpdated")
	if len(events) != 1 || !slices.Equal(events[0].(*domain.TenantUpdated).ChangedFields, []string{"display_name"}) {
		t.Fatalf("TenantUpdated = %+v, want one naming display_name", events)
	}

	refused := s.send(t, http.MethodPatch, tenantsPath+"/acme", map[string]any{
		"display_name": "Other", "max_delegation_depth": domain.DefaultMaxDelegationDepth + 1,
	})
	requireProblem(t, refused, http.StatusUnprocessableEntity, "policy_override_weaker")
	if tenant, err := s.tenants.FindByRealm(context.Background(), "acme"); err != nil || tenant.DisplayName != "Acme Corp" {
		t.Fatalf("after the refusal: %+v, %v; want display_name kept at Acme Corp", tenant, err)
	}
	if got := len(s.eventsOfType("TenantUpdated")); got != 1 {
		t.Fatalf("TenantUpdated events = %d after the refusal, want 1", got)
	}

	requireProblem(t, s.send(t, http.MethodPatch, tenantsPath+"/ghost", map[string]any{"display_name": "X"}),
		http.StatusNotFound, "tenant_not_found")
}

//spec:covers REQ-TENANCY-027, EX-TENANCY-027-01: 無効なテナントの無効化も 204 で成功し、disabled_at を一度目の時刻のまま保ち、TenantDisabled をもう一度発行する。
func TestDisablingADisabledTenantSucceedsAgain(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})
	path := tenantsPath + "/acme/disable"

	requireStatus(t, s.send(t, http.MethodPost, path, nil), http.StatusNoContent)
	first, _ := s.tenants.FindByRealm(context.Background(), "acme")
	if first == nil || first.DisabledAt == nil || first.Status != domain.TenantStatusDisabled {
		t.Fatalf("after the first disable: %+v", first)
	}
	firstAt := *first.DisabledAt

	requireStatus(t, s.send(t, http.MethodPost, path, nil), http.StatusNoContent)
	second, _ := s.tenants.FindByRealm(context.Background(), "acme")
	if second == nil || second.Status != domain.TenantStatusDisabled || second.DisabledAt == nil || !second.DisabledAt.Equal(firstAt) {
		t.Fatalf("disabled_at %v -> %+v, want the first request's time kept", firstAt, second)
	}
	if disabled := s.eventsOfType("TenantDisabled"); len(disabled) != 2 {
		t.Fatalf("TenantDisabled events = %d, want 2", len(disabled))
	}

	// 再開も、すでに有効なテナントに対して成功しイベントを発行する。
	requireStatus(t, s.send(t, http.MethodPost, tenantsPath+"/acme/enable", nil), http.StatusNoContent)
	requireStatus(t, s.send(t, http.MethodPost, tenantsPath+"/acme/enable", nil), http.StatusNoContent)
	if enabled := s.eventsOfType("TenantEnabled"); len(enabled) != 2 {
		t.Fatalf("TenantEnabled events = %d, want 2", len(enabled))
	}
}

//spec:covers EX-TENANCY-027-02: 存在しない realm の無効化は 404 tenant_not_found になり、イベントを発行しない。
func TestDisablingAnUnknownRealmIsNotFound(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{})

	requireProblem(t, s.send(t, http.MethodPost, tenantsPath+"/ghost/disable", nil), http.StatusNotFound, "tenant_not_found")
	if len(*s.events) != 0 {
		t.Fatalf("events = %d, want 0", len(*s.events))
	}
}

// ---- クォータ ----

//spec:covers REQ-TENANCY-037: 負の上限を含む更新は 400 invalid_request で拒否し、保存済みの上書きを変えない。0 の上限は保存する。
func TestQuotaUpdateRefusesNegativeLimits(t *testing.T) {
	quotas := memory.NewQuotaRepository()
	s := newRulesServer(t, systemAdmin(), rulesOptions{quotas: quotas})
	path := tenantsPath + "/acme/quota"
	storedUsers := func() *int {
		t.Helper()
		quota, err := quotas.GetQuota(context.Background(), "acme")
		if err != nil {
			t.Fatal(err)
		}
		return quota.Users
	}

	requireStatus(t, s.send(t, http.MethodPut, path, map[string]int{"users": 20000}), http.StatusOK)

	requireProblem(t, s.send(t, http.MethodPut, path, map[string]int{"users": 30000, "groups": -1}),
		http.StatusBadRequest, "invalid_request")
	if got := storedUsers(); got == nil || *got != 20000 {
		t.Fatalf("users = %v, want the stored 20000", got)
	}

	requireStatus(t, s.send(t, http.MethodPut, path, map[string]int{"users": 0}), http.StatusOK)
	if got := storedUsers(); got == nil || *got != 0 {
		t.Fatalf("users = %v, want 0", got)
	}
}

// ---- テナント設定 ----

const settingsPath = "/realms/acme/api/admin/v1/settings"

func (s *rulesServer) settings(t *testing.T) map[string]any {
	t.Helper()
	response := s.get(t, settingsPath)
	requireStatus(t, response, http.StatusOK)
	return decodeJSON(t, response)
}

//spec:covers EX-TENANCY-028-01, EX-TENANCY-028-02, EX-TENANCY-028-03: 正の値は上限とともに返り、上限を超える値は 422 で保存済みの値を変えず、0 は項目を消す。
func TestTrustedDeviceMaxAgeKeepsTheStoredValueOnRefusal(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	const thirtyDays = 2592000

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]int{"trusted_device_max_age_seconds": thirtyDays}), http.StatusOK)
	settings := s.settings(t)
	if settings["trusted_device_max_age_seconds"] != float64(thirtyDays) ||
		settings["trusted_device_max_age_seconds_ceiling"] != float64(7776000) {
		t.Fatalf("settings = %v", settings)
	}

	for _, refused := range []int{7776001, -1} {
		requireProblem(t, s.send(t, http.MethodPatch, settingsPath, map[string]int{"trusted_device_max_age_seconds": refused}),
			http.StatusUnprocessableEntity, "policy_override_weaker")
		if got := s.settings(t)["trusted_device_max_age_seconds"]; got != float64(thirtyDays) {
			t.Fatalf("after refusing %d: trusted_device_max_age_seconds = %v", refused, got)
		}
	}

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]int{"trusted_device_max_age_seconds": 0}), http.StatusOK)
	if _, ok := s.settings(t)["trusted_device_max_age_seconds"]; ok {
		t.Fatal("0 must disable the feature and drop the field")
	}
}

//spec:covers EX-TENANCY-029-01, EX-TENANCY-029-02, EX-TENANCY-029-03: 同梱翻訳のある言語は保存され、同梱翻訳のない言語は 400 で保存済みの値を変えず、空文字列は設定を消す。
func TestDefaultLocaleAcceptsBundledLanguagesOnly(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"default_locale": " ja "}), http.StatusOK)
	settings := s.settings(t)
	supported, _ := settings["supported_locales"].([]any)
	if settings["default_locale"] != "ja" || !slices.Contains(supported, any("ja")) || !slices.Contains(supported, any("en")) {
		t.Fatalf("settings = %v", settings)
	}

	requireProblem(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"default_locale": "fr"}),
		http.StatusBadRequest, "invalid_request")
	if got := s.settings(t)["default_locale"]; got != "ja" {
		t.Fatalf("after refusing fr: default_locale = %v", got)
	}

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"default_locale": ""}), http.StatusOK)
	if _, ok := s.settings(t)["default_locale"]; ok {
		t.Fatal("an empty default_locale must clear the setting")
	}
}

//spec:covers EX-TENANCY-030-01, EX-TENANCY-030-02, EX-TENANCY-030-03: 表示名だけの更新は基準時刻を動かさず、同じ上書きの再送は基準時刻を進め、すべて 0 の上書きは上書きを消す。
func TestPasswordPolicyOverrideNormalizationAndReferenceTime(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	storedAt := func() time.Time {
		t.Helper()
		tenant, err := s.tenants.FindByRealm(context.Background(), "acme")
		if err != nil || tenant == nil || tenant.PasswordPolicyUpdatedAt == nil {
			t.Fatalf("tenant = %+v, err = %v", tenant, err)
		}
		return *tenant.PasswordPolicyUpdatedAt
	}
	override := map[string]any{"password_policy_override": map[string]int{"min_length": 16}}

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, override), http.StatusOK)
	first := storedAt()

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"display_name": "Acme Corp"}), http.StatusOK)
	if got := storedAt(); !got.Equal(first) {
		t.Fatalf("display-name update moved password_policy_updated_at %v -> %v", first, got)
	}

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, override), http.StatusOK)
	if got := storedAt(); !got.After(first) {
		t.Fatalf("resending the same override left password_policy_updated_at at %v (was %v)", got, first)
	}

	zero := map[string]any{"password_policy_override": map[string]int{
		"min_length": 0, "max_length": 0, "history_depth": 0, "max_age_days": 0,
	}}
	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, zero), http.StatusOK)
	tenant, _ := s.tenants.FindByRealm(context.Background(), "acme")
	if tenant.PasswordPolicyOverride != nil {
		t.Fatalf("override = %+v, want cleared", tenant.PasswordPolicyOverride)
	}
	if _, ok := s.settings(t)["password_policy_override"]; ok {
		t.Fatal("settings still carry password_policy_override")
	}
}

//spec:covers EX-TENANCY-031-01, EX-TENANCY-031-02: 保存済みと同じ表示名の更新も changed_fields に display_name を載せ、拒否した更新は何も保存せずイベントも発行しない。
func TestSettingsUpdateEventListsTheFieldsInTheRequest(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"display_name": "Acme"}), http.StatusOK)
	updated := s.eventsOfType("TenantUpdated")
	if len(updated) != 1 || !slices.Contains(updated[0].(*domain.TenantUpdated).ChangedFields, "display_name") {
		t.Fatalf("TenantUpdated = %+v", updated)
	}

	response := s.send(t, http.MethodPatch, settingsPath, map[string]any{
		"display_name": "Acme Corp", "max_delegation_depth": domain.DefaultMaxDelegationDepth + 1,
	})
	requireProblem(t, response, http.StatusUnprocessableEntity, "policy_override_weaker")
	if tenant, _ := s.tenants.FindByRealm(context.Background(), "acme"); tenant.DisplayName != "Acme" {
		t.Fatalf("display_name = %q, want unchanged", tenant.DisplayName)
	}
	if got := len(s.eventsOfType("TenantUpdated")); got != 1 {
		t.Fatalf("TenantUpdated events = %d, want still 1", got)
	}
}

// ---- ブランド設定 ----

const brandingAdminPath = "/realms/acme/api/admin/v1/tenant/branding"

//spec:covers EX-TENANCY-032-01: 文字列は前後の空白を除いて保存し、空文字列は項目を未設定に戻す。
func TestBrandingTrimsTextAndEmptyStringUnsets(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	requireStatus(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]string{"product_name": " Acme ID "}), http.StatusOK)
	if got := decodeJSON(t, s.get(t, "/realms/acme/api/branding"))["product_name"]; got != "Acme ID" {
		t.Fatalf("product_name = %v", got)
	}
	requireStatus(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]string{"product_name": ""}), http.StatusOK)
	if _, ok := decodeJSON(t, s.get(t, "/realms/acme/api/branding"))["product_name"]; ok {
		t.Fatal("an empty product_name must unset the field")
	}
}

//spec:covers REQ-TENANCY-032, EX-TENANCY-032-02, EX-TENANCY-032-03: 81 文字の日本語のラベルと、大文字のスキームの URL は 400 invalid_branding になり、何も保存しない。80 文字の日本語のラベルは受け付ける。
func TestBrandingRefusesLongMultibyteLabelsAndUppercaseSchemes(t *testing.T) {
	label := func(n int) string { return strings.Repeat("ヘ", n) }
	for name, link := range map[string]map[string]string{
		"81 characters":            {"label": label(81), "url": "https://help.example.test"},
		"uppercase HTTPS scheme":   {"label": "Help", "url": "HTTPS://help.example.test"},
		"https in the wrong place": {"label": "Help", "url": "javascript:https://help.example.test"},
	} {
		t.Run(name, func(t *testing.T) {
			s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
			requireProblem(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]any{"footer_link_1": link}),
				http.StatusBadRequest, "invalid_branding")
			if saved, _ := s.branding.FindByTenant(context.Background(), "acme"); saved != nil {
				t.Fatalf("branding saved: %+v", saved)
			}
		})
	}

	// 対照: 長さは文字数で数えるので、80 文字 (240 バイト) のラベルと、2,048 文字で 2,048 バイトを
	// 超える URL は受け付ける。
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	url := "https://help.example.test/" + strings.Repeat("ヘ", 2048-len("https://help.example.test/"))
	requireStatus(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]any{
		"footer_link_1": map[string]string{"label": label(80), "url": url},
	}), http.StatusOK)
	if saved, _ := s.branding.FindByTenant(context.Background(), "acme"); saved == nil || saved.FooterLink1.Label != label(80) {
		t.Fatalf("branding = %+v, want the 80-character label saved", saved)
	}
}

//spec:covers EX-TENANCY-033-01, EX-TENANCY-033-02: 262,144 バイトの PNG は受け付け、262,145 バイトの PNG は 400 で拒否してロゴを保存しない。
func TestBrandingAssetSizeLimitIs256KiB(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	accepted := s.upload(t, "logo", pngOfSize(262144))
	requireStatus(t, accepted, http.StatusOK)
	if !strings.Contains(accepted.Body.String(), "logo_url") {
		t.Fatalf("body = %s, want logo_url", accepted.Body.String())
	}

	s = newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	requireProblem(t, s.upload(t, "logo", pngOfSize(262145)), http.StatusBadRequest, "invalid_request")
	if _, ok := decodeJSON(t, s.get(t, "/realms/acme/api/branding"))["logo_url"]; ok {
		t.Fatal("a refused upload stored a logo")
	}
}

//spec:covers EX-TENANCY-033-03: ブランド設定が未設定のテナントでロゴを削除しても 200 と空のブランド設定が返る。
func TestDeletingABrandingAssetOfAnUnconfiguredTenantSucceeds(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	response := s.send(t, http.MethodDelete, brandingAdminPath+"/assets/logo", nil)
	requireStatus(t, response, http.StatusOK)
	branding, _ := decodeJSON(t, response)["branding"].(map[string]any)
	if len(branding) != 0 {
		t.Fatalf("branding = %v, want empty", branding)
	}
}

//spec:covers EX-TENANCY-034-01, EX-TENANCY-034-02, EX-TENANCY-034-03: 未設定の ETag は "branding-default" で public, max-age=60 を付け、一致する If-None-Match には本文のない 304 を返し、更新のたびに ETag が変わる。
func TestPublicBrandingCarriesAVersionETag(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	unconfigured := s.get(t, "/realms/acme/api/branding")
	requireStatus(t, unconfigured, http.StatusOK)
	if got := unconfigured.Header().Get("ETag"); got != `"branding-default"` {
		t.Fatalf("ETag = %q", got)
	}
	if got := unconfigured.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if body := decodeJSON(t, unconfigured); len(body) != 0 {
		t.Fatalf("body = %v, want empty", body)
	}

	notModified := s.get(t, "/realms/acme/api/branding", "If-None-Match", `"branding-default"`)
	requireStatus(t, notModified, http.StatusNotModified)
	if notModified.Body.Len() != 0 {
		t.Fatalf("304 carries a body: %q", notModified.Body.String())
	}

	seen := map[string]bool{`"branding-default"`: true}
	for _, name := range []string{"Acme", "Acme ID"} {
		requireStatus(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]string{"product_name": name}), http.StatusOK)
		etag := s.get(t, "/realms/acme/api/branding").Header().Get("ETag")
		if seen[etag] {
			t.Fatalf("ETag %s repeats after an update", etag)
		}
		seen[etag] = true
		requireStatus(t, s.get(t, "/realms/acme/api/branding", "If-None-Match", `"branding-default"`), http.StatusOK)
	}
}

//spec:covers EX-TENANCY-035-01, EX-TENANCY-035-02: 配信するロゴは判定した形式、nosniff、private, max-age=3600 を持ち、未知の種別は 404 not_found になる。
func TestBrandingAssetDeliveryHeaders(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	requireStatus(t, s.upload(t, "logo", pngOfSize(64)), http.StatusOK)
	logoURL, _ := decodeJSON(t, s.get(t, "/realms/acme/api/branding"))["logo_url"].(string)

	asset := s.get(t, logoURL)
	requireStatus(t, asset, http.StatusOK)
	for header, want := range map[string]string{
		"Content-Type":           "image/png",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=3600",
	} {
		if got := asset.Header().Get(header); got != want {
			t.Fatalf("%s = %q, want %q", header, got, want)
		}
	}

	id := logoURL[strings.LastIndex(logoURL, "/")+1:]
	requireProblem(t, s.get(t, "/realms/acme/tenant-branding-assets/banner/"+id), http.StatusNotFound, "not_found")
}

// ---- 通知テンプレート ----

const passwordResetTemplate = "/realms/acme/api/admin/v1/tenant/notification-templates/password_reset/"

//spec:covers EX-TENANCY-038-01: 件名だけを指定したプレビューは、本文を組み込みの ja の文面で、テナントの表示名を使って描画する。
func TestPreviewFillsBlankFieldsFromTheEffectiveTemplate(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	builtin, err := template.Builtin(sharednotification.TemplateKeyPasswordReset, "ja")
	if err != nil {
		t.Fatal(err)
	}
	vars := template.SampleVars(sharednotification.TemplateKeyPasswordReset)
	vars["tenant_display_name"] = "Acme"
	want, err := template.Render(template.Definition{
		Subject: "件名だけ", BodyText: builtin.BodyText, BodyHTML: builtin.BodyHTML,
	}, vars)
	if err != nil {
		t.Fatal(err)
	}

	response := s.send(t, http.MethodPost, passwordResetTemplate+"ja/preview", map[string]string{"subject": "件名だけ", "body_text": "  "})
	requireStatus(t, response, http.StatusOK)
	body := decodeJSON(t, response)
	if body["subject"] != "件名だけ" || body["body_text"] != want.Text || body["body_html"] != want.HTML {
		t.Fatalf("preview = %v\nwant text = %q", body, want.Text)
	}
}

//spec:covers EX-TENANCY-038-02: プレビューの product_name には、サンプル値ではなくテナントのブランド設定の値を使う。
func TestPreviewUsesTheTenantProductName(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	requireStatus(t, s.send(t, http.MethodPut, brandingAdminPath, map[string]string{"product_name": "Acme ID"}), http.StatusOK)

	response := s.send(t, http.MethodPost, passwordResetTemplate+"ja/preview", map[string]string{"subject": "{{product_name}} のお知らせ"})
	requireStatus(t, response, http.StatusOK)
	if got := decodeJSON(t, response)["subject"]; got != "Acme ID のお知らせ" {
		t.Fatalf("subject = %v", got)
	}
}

//spec:covers EX-TENANCY-039-01: 上書きがないテンプレートのリセットも 200 と組み込みの文面を返し、NotificationTemplateReset を発行する。
func TestResetWithoutAnOverrideStillEmitsTheEvent(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	response := s.send(t, http.MethodDelete, passwordResetTemplate+"ja", nil)
	requireStatus(t, response, http.StatusOK)
	if customized := decodeJSON(t, response)["customized"]; customized != false {
		t.Fatalf("customized = %v", customized)
	}
	if reset := s.eventsOfType("NotificationTemplateReset"); len(reset) != 1 {
		t.Fatalf("NotificationTemplateReset events = %d, want 1", len(reset))
	}
}

func (s *rulesServer) overrideTemplate(t *testing.T, locale, subject string) {
	t.Helper()
	requireStatus(t, s.send(t, http.MethodPut, passwordResetTemplate+locale, map[string]string{
		"subject": subject, "body_text": "{{reset_url}}", "body_html": "<p>{{reset_url}}</p>",
	}), http.StatusOK)
}

//spec:covers EX-TENANCY-040-01: テナントの既定言語が en でも、ja の試し送りは ja の文面を操作者へ送る。
func TestTestSendUsesTheEditedLocale(t *testing.T) {
	sender := &recordingSender{}
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{sender: sender})
	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]string{"default_locale": "en"}), http.StatusOK)
	s.overrideTemplate(t, "ja", "JA-OVERRIDE")
	s.overrideTemplate(t, "en", "EN-OVERRIDE")

	response := s.send(t, http.MethodPost, passwordResetTemplate+"ja/test", nil)
	requireStatus(t, response, http.StatusOK)
	if len(sender.sent) != 1 || sender.sent[0].Subject != "JA-OVERRIDE" || sender.sent[0].To != "operator@example.test" {
		t.Fatalf("sent = %+v", sender.sent)
	}
}

//spec:covers EX-TENANCY-040-02: 送信に失敗しても 200 を返し、delivered は false、to は操作者のアドレスである。
func TestTestSendReportsAFailedDelivery(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{sender: &recordingSender{fail: true}})

	response := s.send(t, http.MethodPost, passwordResetTemplate+"ja/test", nil)
	requireStatus(t, response, http.StatusOK)
	body := decodeJSON(t, response)
	if body["delivered"] != false || body["to"] != "operator@example.test" {
		t.Fatalf("body = %v", body)
	}
}

// ---- 連携エンドポイント ----

const integrationPath = "/realms/acme/api/admin/v1/integration-endpoints"

func federationSigner(t *testing.T) (*samltoken.Signer, *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	// JST の境界で作る。応答が UTC へそろえることを、入力の時刻帯と区別して読む。
	jst := time.FixedZone("JST", 9*60*60)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "Acme federation signing"},
		NotBefore:    time.Date(2026, 1, 1, 9, 0, 0, 0, jst),
		NotAfter:     time.Date(2036, 1, 1, 9, 0, 0, 0, jst),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := samltoken.NewSigner(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	return signer, cert
}

//spec:covers EX-TENANCY-041-01: 署名証明書のフィンガープリントは DER の SHA-256 をコロンで区切った大文字の 16 進で、有効期間は UTC で返る。
func TestIntegrationEndpointsFingerprintFormat(t *testing.T) {
	signer, cert := federationSigner(t)
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{signer: signer})

	response := s.get(t, integrationPath)
	requireStatus(t, response, http.StatusOK)
	var body struct {
		SAML struct {
			SigningCertificate struct {
				Fingerprint string `json:"fingerprint_sha256"`
				NotBefore   string `json:"not_before"`
				NotAfter    string `json:"not_after"`
			} `json:"signing_certificate"`
		} `json:"saml"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(cert.Raw)
	parts := make([]string, 0, len(digest))
	for _, b := range digest {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	certificate := body.SAML.SigningCertificate
	if certificate.Fingerprint != strings.Join(parts, ":") {
		t.Fatalf("fingerprint = %q, want %q", certificate.Fingerprint, strings.Join(parts, ":"))
	}
	if certificate.NotBefore != "2026-01-01T00:00:00Z" || certificate.NotAfter != "2036-01-01T00:00:00Z" {
		t.Fatalf("validity = %s .. %s, want UTC", certificate.NotBefore, certificate.NotAfter)
	}
}

type unavailableSigner struct{ samltoken.SignerProvider }

func (unavailableSigner) Resolve(context.Context) (*samltoken.Signer, error) {
	return nil, errors.New("signing key unavailable")
}

//spec:covers EX-TENANCY-042-01: 署名用の資格情報を解決できないときは 503 federation_credentials_unavailable を返し、どの URL も返さない。
func TestIntegrationEndpointsWithoutCredentialsReturnNothing(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{signer: unavailableSigner{}})

	response := s.get(t, integrationPath)
	requireProblem(t, response, http.StatusServiceUnavailable, "federation_credentials_unavailable")
	if strings.Contains(response.Body.String(), "/authorize") || strings.Contains(response.Body.String(), "issuer") {
		t.Fatalf("refusal carries endpoints: %s", response.Body.String())
	}
}

// ---- 属性スキーマ ----

const userSchemaPath = "/realms/acme/api/admin/v1/tenant/user-attribute-schema"

func attribute(key, attributeType string) map[string]any {
	return map[string]any{"key": key, "type": attributeType, "visibility": "admin_readable"}
}

func (s *rulesServer) seedReferencedSchema(t *testing.T) {
	t.Helper()
	requireStatus(t, s.send(t, http.MethodPut, userSchemaPath, map[string]any{
		"attributes": []any{attribute("region", "string"), attribute("shift", "string")},
	}), http.StatusOK)
	if err := s.groups.SaveDynamicRule(context.Background(), &groupdomain.DynamicGroupRule{
		GroupID: "engineering", TenantID: "acme", Expression: `user.region == "emea"`, Enabled: true,
		Version: 1, ReferencedAttributes: []string{"region"}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	*s.events = (*s.events)[:0]
}

func (s *rulesServer) storedAttributeTypes(t *testing.T) map[string]string {
	t.Helper()
	schema, err := s.schemas.FindByTenant(context.Background(), "acme")
	if err != nil || schema == nil {
		t.Fatalf("schema = %+v, err = %v", schema, err)
	}
	types := map[string]string{}
	for _, def := range schema.Attributes {
		types[def.Key] = string(def.Type)
	}
	return types
}

//spec:covers EX-TENANCY-043-01, EX-TENANCY-043-02: 動的グループが参照する属性の削除と型の変更は 409 attribute_referenced_by_dynamic_group になり、スキーマを変えずイベントも発行しない。
func TestUserAttributeSchemaRefusesChangingAReferencedAttribute(t *testing.T) {
	for name, attributes := range map[string][]any{
		"delete":      {attribute("shift", "string")},
		"change type": {attribute("region", "boolean"), attribute("shift", "string")},
	} {
		t.Run(name, func(t *testing.T) {
			s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
			s.seedReferencedSchema(t)

			response := s.send(t, http.MethodPut, userSchemaPath, map[string]any{"attributes": attributes})
			requireProblem(t, response, http.StatusConflict, "attribute_referenced_by_dynamic_group")
			if got := s.storedAttributeTypes(t); got["region"] != "string" || got["shift"] != "string" {
				t.Fatalf("schema = %v, want unchanged", got)
			}
			if len(*s.events) != 0 {
				t.Fatalf("events = %d, want 0", len(*s.events))
			}
		})
	}
}

//spec:covers EX-TENANCY-043-03: 参照されていない属性だけを外す更新は成功し、要求になかった属性は削除される。
func TestUserAttributeSchemaReplacesTheWholeDefinition(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
	s.seedReferencedSchema(t)

	requireStatus(t, s.send(t, http.MethodPut, userSchemaPath, map[string]any{
		"attributes": []any{attribute("region", "string")},
	}), http.StatusOK)
	if got := s.storedAttributeTypes(t); len(got) != 1 || got["region"] != "string" {
		t.Fatalf("schema = %v, want only region", got)
	}
}

//spec:covers EX-TENANCY-028-01: 上限ちょうどの 7,776,000 は保存され、その 1 秒上は拒否される。
func TestTrustedDeviceMaxAgeAcceptsExactlyTheCeiling(t *testing.T) {
	s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})

	requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]int{"trusted_device_max_age_seconds": 7776000}), http.StatusOK)
	if got := s.settings(t)["trusted_device_max_age_seconds"]; got != float64(7776000) {
		t.Fatalf("trusted_device_max_age_seconds = %v, want the ceiling", got)
	}
}

//spec:covers EX-TENANCY-019-02, EX-TENANCY-019-03: 基準値ちょうどの上書きと、有効期限の範囲の両端は保存され、そこから一つ弱めた値は policy_override_weaker で拒否される。
func TestPasswordPolicyOverrideBoundaries(t *testing.T) {
	for _, tc := range []struct {
		field    string
		accepted int
		refused  int
	}{
		{"min_length", 12, 11},
		{"max_length", 128, 129},
		{"history_depth", 5, 4},
		{"max_age_days", 30, 29},
		{"max_age_days", 3650, 3651},
	} {
		t.Run(fmt.Sprintf("%s=%d", tc.field, tc.accepted), func(t *testing.T) {
			s := newRulesServer(t, refusalAdmin("acme"), rulesOptions{})
			requireStatus(t, s.send(t, http.MethodPatch, settingsPath, map[string]any{
				"password_policy_override": map[string]int{tc.field: tc.accepted},
			}), http.StatusOK)
			requireProblem(t, s.send(t, http.MethodPatch, settingsPath, map[string]any{
				"password_policy_override": map[string]int{tc.field: tc.refused},
			}), http.StatusUnprocessableEntity, "policy_override_weaker")
			override, _ := s.settings(t)["password_policy_override"].(map[string]any)
			if override[tc.field] != float64(tc.accepted) {
				t.Fatalf("override = %v, want %s kept at %d", override, tc.field, tc.accepted)
			}
		})
	}
}
