package handlers_http_test

// Tenancy が宣言する拒否について、応答と「拒否が変えなかった状態」の両方を確かめる。
//
// この Context の拒否はテナント境界そのものである。連携エンドポイントの参照が越境すれば
// 他テナントの構成が読め、システムコンソールの一覧が越境すれば他テナントの存在そのものが
// 漏れる。Quota の拒否は作成を止めるだけでなく使用量も動かしてはならず、テスト送信の
// 拒否は管理者権限が任意宛先メールの踏み台になる経路を塞ぐ。
//
// どの拒否にも、同じ入口で同じ操作が通る対照を 1 つ置く。「拒否されたので何も起きて
// いない」と「そもそも何も起こせない構成だった」は、効果の側だけを読むと同じに見える。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

// refusalServer は、テナント境界と拒否の効果を読むのに必要なポートだけを張った合成サーバー。
// 経路は production と同じ `httpadapter.Register` が組み立てる。
type refusalServer struct {
	e        *echo.Echo
	branding *memory.TenantBrandingRepository
	quotas   *memory.QuotaRepository
	groups   *groupmemory.GroupRepository
	sender   *email_memory.NoopEmailSender
	events   *[]spec.DomainEvent
}

// newRefusalServer は default (制御面) と acme の 2 テナントを持つサーバーを組み立てる。
// 越境の拒否は「行き先が存在しない」ことではなく「到達させない」ことなので、
// 越えようとする先が実在していなければ何も確かめられない。
func newRefusalServer(t *testing.T, actors ...*userdomain.User) *refusalServer {
	t.Helper()
	ctx := context.Background()

	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range []*domain.Tenant{
		activeTenant(domain.DefaultTenantID, "Default"),
		activeTenant("acme", "Acme"),
		activeTenant("beta", "Beta"),
	} {
		if err := tenantRepo.Save(ctx, tenant); err != nil {
			t.Fatal(err)
		}
	}

	userRepo := usermemory.NewUserRepository()
	resolver := &fakeAuthnResolver{}
	for index, actor := range actors {
		userRepo.Seed(actor)
		if index == 0 {
			resolver.ctx = &authdomain.AuthenticationContext{
				UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
			}
		}
	}

	server := &refusalServer{
		e:        echo.New(),
		branding: memory.NewTenantBrandingRepository(),
		quotas:   memory.NewQuotaRepository(),
		groups:   groupmemory.NewGroupRepository(),
		sender:   &email_memory.NoopEmailSender{},
	}
	events := make([]spec.DomainEvent, 0)
	server.events = &events

	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}

	templates := memory.NewNotificationTemplateRepository()
	// Notifier は production と同じ形で組む。テナント上書きは Tenancy 由来なので、
	// TenantNotificationSource を張らない既定構成では上書きが送信に反映されず、
	// 「上書きが保存されていない」の効果を読めない。
	notifier := &template.Notifier{
		Sender: server.sender,
		Tenant: tenantusecases.TenantNotificationSource{
			TenantRepo: tenantRepo, BrandingRepo: server.branding, TemplateRepo: templates,
		},
	}

	httpadapter.Register(server.e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		PaginationCodec:  support.NewCursorCodec([]byte("test-pagination-secret")),
		TenantRepo:       tenantRepo,
		Emit:             func(event spec.DomainEvent) { events = append(events, event) },
		UserRepo:         userRepo,
		GroupRepo:        server.groups,
		EmailSender:      server.sender,
		AuthnResolver:    resolver,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		Notification:     sharednotification.Module{EmailSender: server.sender, Notifier: notifier},
		Tenancy: tenancy.Module{
			TenantRepo:            tenantRepo,
			BrandingRepo:          server.branding,
			BrandingAssetStore:    memory.NewTenantBrandingAssetStore(),
			NotificationTemplates: templates,
			QuotaRepo:             server.quotas,
		},
	})
	return server
}

// get は認証済みセッションの GET を送る。CSRF は状態を変えない要求には要らない。
func (s *refusalServer) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	s.e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return recorder
}

// send は Origin と double-submit の CSRF トークンを備えた状態変更要求を送る。
// 拒否が CSRF の手前で起きたのか宣言された判定で起きたのかを取り違えないよう、
// ブラウザーの資格情報は常に正しく揃える。
func (s *refusalServer) send(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload := []byte(nil)
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	csrf, cookie := passwordResetContextCSRF(t, s.e, tenantPrefix(path)+"/api/auth/password_reset_context")
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	s.e.ServeHTTP(recorder, request)
	return recorder
}

func refusalAdmin(tenantID string) *userdomain.User {
	const sub = "operator"
	actor := settingsActor(sub, tenantID, []string{"admin"})
	email := sub + "@example.test"
	actor.Email = &email
	actor.EmailVerified = true
	return actor
}

// EX-TENANCY-001-02: 連携エンドポイント画面には対象テナントの指定手段が無く、
// 別テナントの realm を URL に混ぜても解決済みテナント以外の情報は返らない。
// 応答にクライアントシークレット、API トークン、秘密鍵は含まれない。
//
// この画面は 1 リクエストでテナントの連携構成を丸ごと返す。越境を許せば、
// 他テナントの発行者、エンドポイント、署名証明書の指紋がまとめて読める。
func TestAdminIntegrationEndpointsRefuseCrossTenantTargetingAndLeakNoSecret(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))

	// 1. クエリで他テナントを指す。対象指定パラメータは存在しないので、
	//    どの名前で渡しても解決済みテナントの情報しか返らない。
	targeted := server.get(t, "/realms/acme/api/admin/v1/integration-endpoints"+
		"?tenant_id=beta&realm=beta&target_tenant_id=beta&tenant=beta")
	if targeted.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", targeted.Code, targeted.Body.String())
	}
	var catalog map[string]any
	if err := json.Unmarshal(targeted.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if issuer, _ := catalog["issuer"].(string); issuer != "http://idp.test/realms/acme" {
		t.Fatalf("issuer=%q, want the resolved tenant's issuer", issuer)
	}
	if strings.Contains(targeted.Body.String(), "beta") {
		t.Fatalf("応答が指定された別テナントの realm を運んでいる: %s", targeted.Body.String())
	}

	// 2. 秘密は 1 つも出ない。この画面が返すのは URL と証明書の指紋だけである。
	for _, forbidden := range []string{
		"client_secret", "api_token", "private_key", "BEGIN PRIVATE KEY", "BEGIN RSA PRIVATE KEY",
	} {
		if strings.Contains(targeted.Body.String(), forbidden) {
			t.Fatalf("応答が %q を含む: %s", forbidden, targeted.Body.String())
		}
	}

	// 3. 別テナントの正規ロケーションから同じ画面を開くことも通らない。
	//    acme の管理者の権限は acme の realm URL の外では効かない。
	//    ここが 403 ではなく 401 なのは、acme のセッションが beta の realm では
	//    そもそもセッションとして成立しないためである。拒否の位置は認可ではなく認証にある。
	crossed := server.get(t, "/realms/beta/api/admin/v1/integration-endpoints")
	if crossed.Code != http.StatusUnauthorized {
		t.Fatalf("cross-tenant status=%d body=%s, want 401", crossed.Code, crossed.Body.String())
	}
	if strings.Contains(crossed.Body.String(), "idp.test/realms/beta") {
		t.Fatalf("拒否の応答が beta のエンドポイントを運んでいる: %s", crossed.Body.String())
	}
}

// EX-TENANCY-014-01: `admin` ロールだけを持つテナント管理者は、システムコンソールの
// テナント一覧に到達できない。
//
// この拒否が素通りすれば、他テナントの存在そのものが漏れる。テナント名は多くの場合
// 顧客名なので、一覧が読めることは顧客名簿が読めることに等しい。
func TestListTenantsRefusesTenantAdminAndReturnsNoOtherTenant(t *testing.T) {
	server := newRefusalServer(t,
		refusalAdmin(domain.DefaultTenantID),
		settingsActor("sysadmin", domain.DefaultTenantID, []string{"system_admin"}),
	)

	refused := server.get(t, "/realms/default/api/admin/v1/tenants")
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	// 拒否の応答が一覧の断片も運んでいないこと。件数だけでなく realm 名で読む。
	for _, realm := range []string{"acme", "beta"} {
		if strings.Contains(refused.Body.String(), realm) {
			t.Fatalf("拒否の応答が %q を運んでいる: %s", realm, refused.Body.String())
		}
	}
	var refusedBody struct {
		Tenants []any `json:"tenants"`
	}
	if err := json.Unmarshal(refused.Body.Bytes(), &refusedBody); err == nil && len(refusedBody.Tenants) != 0 {
		t.Fatalf("拒否の応答にテナントが %d 件含まれる", len(refusedBody.Tenants))
	}

	// 対照: 制御面テナントの system_admin なら同じ経路で一覧が返る。
	// 拒否がロールに由来し、一覧そのものが空だったのではないことを示す。
	allowed := newRefusalServer(t, settingsActor("sysadmin", domain.DefaultTenantID, []string{"system_admin"}))
	listed := allowed.get(t, "/realms/default/api/admin/v1/tenants")
	if listed.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", listed.Code, listed.Body.String())
	}
	var body struct {
		Tenants []struct {
			Realm string `json:"realm"`
		} `json:"tenants"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tenants) < 3 {
		t.Fatalf("対照のテナント数 = %d, want the three seeded tenants", len(body.Tenants))
	}
}

// brandingIsSystemDefault は公開 branding が未設定のままであることを確かめる。
// ログイン画面はこの応答を読んで組込みデフォルトへ落ちるので、「保存されていない」の
// 効果は保存層ではなく利用者が見る側の応答で読む。
func (s *refusalServer) brandingIsSystemDefault(t *testing.T) {
	t.Helper()
	recorder := s.get(t, "/realms/acme/api/branding")
	if recorder.Code != http.StatusOK {
		t.Fatalf("branding status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		ProductName  string `json:"product_name"`
		PrimaryColor string `json:"primary_color"`
		FooterLink1  *struct {
			Label string `json:"label"`
			URL   string `json:"url"`
		} `json:"footer_link_1"`
		LogoURL   string  `json:"logo_url"`
		UpdatedAt *string `json:"updated_at"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ProductName != "" || body.PrimaryColor != "" || body.FooterLink1 != nil ||
		body.LogoURL != "" || body.UpdatedAt != nil {
		t.Fatalf("拒否された入力が branding に残っている: %+v", body)
	}
}

// EX-TENANCY-005-01: `javascript:` スキームの footer リンクと SVG のロゴは
// `InvalidRequestError` で拒否され、保存されない。branding は組込みデフォルトのままになる。
//
// 保存されてしまえば、ログイン画面のフッターが `javascript:` を実行するリンクになる。
// ログイン画面は資格情報を入力する画面なので、そこでの任意スクリプト実行は
// そのままパスワードの窃取になる。SVG も同じで、画像として配られる SVG は
// スクリプトを運べる。
func TestUpdateBrandingRefusesUnsafeInputAndKeepsTheSystemDefault(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	const path = "/realms/acme/api/admin/v1/tenant/branding"

	// 未設定のテナントは組込みデフォルトを使う。拒否の後もここへ戻ることを確かめる。
	server.brandingIsSystemDefault(t)

	refusedLink := server.send(t, http.MethodPut, path, map[string]any{
		"footer_link_1": map[string]any{"label": "ヘルプ", "url": "javascript:alert(1)"},
	})
	if refusedLink.Code != http.StatusBadRequest {
		t.Fatalf("javascript: status=%d body=%s, want 400", refusedLink.Code, refusedLink.Body.String())
	}
	if !bytes.Contains(refusedLink.Body.Bytes(), []byte("invalid_branding")) {
		t.Fatalf("body=%s, want invalid_branding", refusedLink.Body.String())
	}
	server.brandingIsSystemDefault(t)

	refusedLogo := uploadBrandingAsset(t, server.e,
		"/realms/acme/api/admin/v1/tenant/branding/assets/logo", []byte("<svg onload=alert(1)></svg>"))
	if refusedLogo.Code != http.StatusBadRequest {
		t.Fatalf("svg status=%d body=%s, want 400", refusedLogo.Code, refusedLogo.Body.String())
	}
	server.brandingIsSystemDefault(t)
	if stored, err := server.branding.FindByTenant(context.Background(), "acme"); err != nil || stored != nil {
		t.Fatalf("拒否されたのに branding が保存されている: %+v, %v", stored, err)
	}

	// 対照: 同じ入口で、低コントラストという理由では拒否しない。branding の検証は
	// 安全でない入力だけを止め、好みの問題には踏み込まない。
	accepted := server.send(t, http.MethodPut, path, map[string]any{"primary_color": "#eeeeee"})
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
	public := server.get(t, "/realms/acme/api/branding")
	if !strings.Contains(public.Body.String(), "#eeeeee") {
		t.Fatalf("対照の色が公開 branding に出ていない: %s", public.Body.String())
	}
}

// EX-TENANCY-005-02: label だけを指定した footer リンクは `InvalidRequestError` で拒否され、
// 片方だけの上書きは保存されない。
//
// ラベルだけのリンクが保存されると、ログイン画面には押せる見た目のリンクが出て
// どこへも行かない。フッターは問い合わせ先を出す場所なので、押せない問い合わせ先は
// 利用者を締め出す。
func TestUpdateBrandingRefusesIncompleteFooterLinkAndSavesNothing(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	const path = "/realms/acme/api/admin/v1/tenant/branding"

	refused := server.send(t, http.MethodPut, path, map[string]any{
		"footer_link_1": map[string]any{"label": "ヘルプ"},
	})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	server.brandingIsSystemDefault(t)

	// 対照: label と url が揃えば同じ入口で保存され、公開 branding にも出る。
	accepted := server.send(t, http.MethodPut, path, map[string]any{
		"footer_link_1": map[string]any{"label": "ヘルプ", "url": "https://support.example.com"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
	if public := server.get(t, "/realms/acme/api/branding"); !strings.Contains(
		public.Body.String(), "https://support.example.com") {
		t.Fatalf("対照のリンクが公開 branding に出ていない: %s", public.Body.String())
	}
}

// groupNames は呼び出し元から見えるグループ一覧の名前を返す。
// 「作成されていない」を、保存層の内部表現ではなく管理 API の読み取りモデルで読む。
func (s *refusalServer) groupNames(t *testing.T, realm string) []string {
	t.Helper()
	listed := s.get(t, "/realms/"+realm+"/api/admin/v1/groups")
	if listed.Code != http.StatusOK {
		t.Fatalf("group list status=%d body=%s", listed.Code, listed.Body.String())
	}
	var body struct {
		Groups []struct {
			Name string `json:"name"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatalf("group list body=%s: %v", listed.Body.String(), err)
	}
	names := make([]string, 0, len(body.Groups))
	for _, group := range body.Groups {
		names = append(names, group.Name)
	}
	return names
}

// groupUsage は現在の groups 使用量を値で返す。Repository の GetUsage は保存中の構造体を
// そのまま返すので、返り値を持ち越して前後を比べると必ず一致してしまう。
func (s *refusalServer) groupUsage(t *testing.T) int {
	t.Helper()
	usage, err := s.quotas.GetUsage(context.Background(), domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Groups
}

// EX-TENANCY-013-01: 上限に達したテナントの Group 作成は `QuotaExceededError` で拒否され、
// Group は作成されず、使用量も増えない。
//
// 使用量だけ増える実装は、応答からは正しい拒否と区別できないのに、以後の正当な作成まで
// 拒否し続ける。上限を上げても使用量が先に進んでいるので、被害は運用で消えない。
// だから拒否の効果は「作られていない」と「使用量が動いていない」の両方で読む。
func TestCreateGroupRefusesOverHardQuotaAndLeavesUsageUnchanged(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin(domain.DefaultTenantID))
	ctx := context.Background()
	limit := 1
	if err := server.quotas.SetQuota(ctx, domain.DefaultTenantID, &domain.TenantQuota{Groups: &limit}); err != nil {
		t.Fatal(err)
	}

	// 上限まで埋める。ここまでは同じ入口が通ることが、この後の拒否の対照にもなる。
	filled := server.send(t, http.MethodPost, "/realms/default/api/admin/v1/groups",
		map[string]any{"name": "engineering"})
	if filled.Code != http.StatusCreated {
		t.Fatalf("対照 status=%d body=%s, want 201", filled.Code, filled.Body.String())
	}
	// 値でとる。この Repository の GetUsage は生きたポインターを返すので、
	// 構造体を持ち越して前後を比べると、増えた使用量まで一緒に動いて必ず一致する。
	usageAtLimit := server.groupUsage(t)
	if usageAtLimit != 1 {
		t.Fatalf("前提が壊れている: usage.groups = %d, want 1", usageAtLimit)
	}

	refused := server.send(t, http.MethodPost, "/realms/default/api/admin/v1/groups",
		map[string]any{"name": "support"})
	if refused.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "quota_exceeded") {
		t.Fatalf("body=%s, want the quota refusal", refused.Body.String())
	}

	// 効果 1: Group が作られていない。読み取りモデルで名前まで確かめる。
	names := server.groupNames(t, "default")
	if len(names) != 1 || names[0] != "engineering" {
		t.Fatalf("拒否されたのにグループ一覧が %v", names)
	}
	// 効果 2: 使用量が動いていない。使用量だけ増える実装は応答からは正しい拒否に見えるのに、
	// 上限を上げても使用量が先に進んでいるので、以後の正当な作成まで拒否し続ける。
	if usageAfter := server.groupUsage(t); usageAfter != usageAtLimit {
		t.Fatalf("usage.groups = %d, want it to stay %d", usageAfter, usageAtLimit)
	}

	// 対照 2: 上限を上げれば同じ要求が通る。拒否が Quota に由来し、
	// 作成の経路そのものが塞がっていたのではないことを示す。
	raised := 2
	if err := server.quotas.SetQuota(ctx, domain.DefaultTenantID, &domain.TenantQuota{Groups: &raised}); err != nil {
		t.Fatal(err)
	}
	allowed := server.send(t, http.MethodPost, "/realms/default/api/admin/v1/groups",
		map[string]any{"name": "support"})
	if allowed.Code != http.StatusCreated {
		t.Fatalf("上限を上げた後の status=%d body=%s, want 201", allowed.Code, allowed.Body.String())
	}
}

const refusalTemplatePath = "/realms/acme/api/admin/v1/tenant/notification_templates"

// templateDetail は 1 テンプレートの現在値を管理 API から読み直す。
// 「上書きが保存されていない」は保存層ではなく、次に編集画面を開いた人が見る値で読む。
func (s *refusalServer) templateDetail(t *testing.T) struct {
	Customized      bool   `json:"customized"`
	Subject         string `json:"subject"`
	BodyText        string `json:"body_text"`
	BodyHTML        string `json:"body_html"`
	FromDisplayName string `json:"from_display_name"`
} {
	t.Helper()
	var detail struct {
		Customized      bool   `json:"customized"`
		Subject         string `json:"subject"`
		BodyText        string `json:"body_text"`
		BodyHTML        string `json:"body_html"`
		FromDisplayName string `json:"from_display_name"`
	}
	recorder := s.get(t, refusalTemplatePath+"/password_reset/ja")
	if recorder.Code != http.StatusOK {
		t.Fatalf("template detail status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	return detail
}

// EX-TENANCY-017-01: 許可集合外の差し込み変数を含む上書きは拒否され、保存されない。
// 以後も利用者には組込みデフォルトのリセットメールが届く。
//
// `{{password}}` のような変数を通せば、テンプレートは秘密の引き出し口になる。
// 逆に `{{reset_url}}` を落とした本文が保存されれば、リンクの無いリセットメールが
// 配られて利用者はパスワードを直せなくなる。だから効果は「保存されていない」だけでなく
// 「次に届くメールが組込みデフォルトのままである」ことまで読む。
func TestNotificationTemplateRefusesUnknownPlaceholderAndSendsTheBuiltinDefault(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	before := server.templateDetail(t)

	// 拒否の前に 1 通送っておく。「以後も組込みデフォルトが届く」は、保存された文面では
	// なく実際に描画されて届いたメールを前後で突き合わせないと確かめられない。
	baseline := server.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test", nil)
	if baseline.Code != http.StatusOK {
		t.Fatalf("baseline test send status=%d body=%s", baseline.Code, baseline.Body.String())
	}
	if len(server.sender.Sent) != 1 {
		t.Fatalf("前提が壊れている: 送信数 = %d, want 1", len(server.sender.Sent))
	}
	builtinMessage := server.sender.Sent[0]

	refused := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", map[string]any{
		"subject": "件名", "body_text": "{{password}}", "body_html": "<p>{{password}}</p>",
	})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "invalid_notification_template") {
		t.Fatalf("body=%s, want invalid_notification_template", refused.Body.String())
	}
	if len(*server.events) != 0 {
		t.Fatalf("拒否された更新が %d 件のイベントを発行した", len(*server.events))
	}

	after := server.templateDetail(t)
	if after.Customized || after != before {
		t.Fatalf("拒否されたのにテンプレートが変わった: %+v -> %+v", before, after)
	}

	// 効果の本体: 次に送られるメールが拒否の前と 1 文字も変わらないこと。
	// 保存の有無ではなく、利用者の受信箱に届くものを読む。
	sent := server.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test", nil)
	if sent.Code != http.StatusOK {
		t.Fatalf("test send status=%d body=%s", sent.Code, sent.Body.String())
	}
	if len(server.sender.Sent) != 2 {
		t.Fatalf("送信数 = %d, want 2", len(server.sender.Sent))
	}
	message := server.sender.Sent[1]
	if message != builtinMessage {
		t.Fatalf("拒否の後に届くメールが変わった: %+v -> %+v", builtinMessage, message)
	}
	// 差し込みは残らず解決されている。拒否した `{{password}}` はもちろん、
	// リンクを運ぶ変数が未解決のまま配られることも無い。
	if strings.Contains(message.Text, "{{") || strings.Contains(message.HTML, "{{") {
		t.Fatalf("差し込み変数が未解決のまま配られている: %q / %q", message.Text, message.HTML)
	}
	if strings.Contains(message.Text, "password}}") || strings.Contains(message.HTML, "password}}") {
		t.Fatalf("拒否された変数が本文に現れた: %q / %q", message.Text, message.HTML)
	}

	// 対照: 許可集合の変数だけなら同じ入口で保存され、以後のメールに反映される。
	accepted := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", map[string]any{
		"subject":   "【Acme】パスワード再設定",
		"body_text": "{{user_display_name}} さん {{reset_url}}",
		"body_html": "<p>{{user_display_name}} さん {{reset_url}}</p>",
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
	if detail := server.templateDetail(t); !detail.Customized ||
		detail.Subject != "【Acme】パスワード再設定" {
		t.Fatalf("対照の上書きが保存されていない: %+v", detail)
	}
}

// EX-TENANCY-017-02: テキスト本文だけ、あるいは HTML 本文だけの上書きは拒否され、
// 片方だけの上書きは作られない。
//
// 片方だけが保存されると、そのテンプレートは「HTML を読めない受信者には何も伝えない
// メール」か「HTML クライアントで空に見えるメール」のどちらかになる。
// 部分的に保存された状態は、次の編集者にも壊れているように見えない。
func TestNotificationTemplateRefusesPartialBodyAndCreatesNoOverride(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	before := server.templateDetail(t)

	for _, body := range []map[string]any{
		{"subject": "件名", "body_text": "{{reset_url}}", "body_html": ""},
		{"subject": "件名", "body_text": "", "body_html": "<p>{{reset_url}}</p>"},
	} {
		refused := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", body)
		if refused.Code != http.StatusBadRequest {
			t.Fatalf("%v: status=%d body=%s, want 400", body, refused.Code, refused.Body.String())
		}
		if after := server.templateDetail(t); after.Customized || after != before {
			t.Fatalf("%v: 片方だけの上書きが残った: %+v", body, after)
		}
	}

	// 対照: 両方揃えば同じ入口で保存される。
	accepted := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", map[string]any{
		"subject": "件名", "body_text": "{{reset_url}}", "body_html": "<p>{{reset_url}}</p>",
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
}

// EX-TENANCY-017-03: カタログに無い locale を指定した上書きは拒否され、
// その locale の上書きは作られない。
//
// カタログ外の locale に上書きが作られると、その locale の利用者には組込みデフォルトも
// 上書きも届かないテンプレートが生まれる。存在しない locale へ書けること自体が、
// 配信できないメールを静かに増やす。
func TestNotificationTemplateRefusesUnknownLocaleAndCreatesNoOverride(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	body := map[string]any{"subject": "件名", "body_text": "{{reset_url}}", "body_html": "<p>{{reset_url}}</p>"}

	for _, path := range []string{"password_reset/fr", "made_up/ja"} {
		refused := server.send(t, http.MethodPut, refusalTemplatePath+"/"+path, body)
		if refused.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s, want 400", path, refused.Code, refused.Body.String())
		}
		// 効果: 一覧にその行が生えていないこと。拒否した locale が
		// 「customized のまま読めない行」として残ると、編集画面が壊れる。
		listed := server.get(t, refusalTemplatePath)
		if listed.Code != http.StatusOK {
			t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
		}
		var catalog struct {
			Templates []struct {
				TemplateKey string `json:"template_key"`
				Locale      string `json:"locale"`
				Customized  bool   `json:"customized"`
			} `json:"templates"`
			SupportedLocales []string `json:"supported_locales"`
		}
		if err := json.Unmarshal(listed.Body.Bytes(), &catalog); err != nil {
			t.Fatal(err)
		}
		for _, row := range catalog.Templates {
			if row.Customized {
				t.Fatalf("%s: 拒否されたのに %s/%s が customized になった", path, row.TemplateKey, row.Locale)
			}
			if !slices.Contains(catalog.SupportedLocales, row.Locale) {
				t.Fatalf("%s: カタログ外の locale %q が一覧に現れた", path, row.Locale)
			}
		}
	}

	// 対照: カタログにある locale なら同じ入口で保存される。
	accepted := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", body)
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
}

// EX-TENANCY-017-04: 差出人メールアドレスを上書きする入力は受け付けない。
// 上書きできるのは表示名だけである。
//
// 差出人アドレスをテナントが決められると、この IdP は任意の送信元を名乗るメールの
// 中継になる。SPF と DKIM は送信ドメインに紐づくので、こちらの正当な署名がついたまま
// 別のドメインを名乗るメールが出ていく。表示名だけなら、この危険は生まれない。
func TestNotificationTemplateRefusesFromAddressOverrideAndKeepsDisplayNameOnly(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))
	before := server.templateDetail(t)

	for _, field := range []string{"from_email", "from_address", "from"} {
		refused := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", map[string]any{
			"subject": "件名", "body_text": "{{reset_url}}", "body_html": "<p>{{reset_url}}</p>",
			field: "billing@victim.example",
		})
		if refused.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s, want 400", field, refused.Code, refused.Body.String())
		}
		if after := server.templateDetail(t); after != before {
			t.Fatalf("%s: 拒否されたのにテンプレートが変わった: %+v", field, after)
		}
	}

	// 対照: 表示名だけの上書きは通り、実際に送られるメールへ反映される。
	// アドレスはサーバー設定のままなので、送信メッセージにはそもそもアドレスの欄が無い。
	accepted := server.send(t, http.MethodPut, refusalTemplatePath+"/password_reset/ja", map[string]any{
		"subject": "件名", "body_text": "{{reset_url}}", "body_html": "<p>{{reset_url}}</p>",
		"from_display_name": "Acme サポート",
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", accepted.Code, accepted.Body.String())
	}
	sent := server.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test", nil)
	if sent.Code != http.StatusOK {
		t.Fatalf("test send status=%d body=%s", sent.Code, sent.Body.String())
	}
	if len(server.sender.Sent) != 1 || server.sender.Sent[0].FromDisplayName != "Acme サポート" {
		t.Fatalf("送信メッセージ = %+v, want the display name only", server.sender.Sent)
	}
	if strings.Contains(server.sender.Sent[0].FromDisplayName, "victim.example") {
		t.Fatalf("拒否したアドレスが差出人に混ざった: %+v", server.sender.Sent[0])
	}
}

// EX-TENANCY-018-03: テスト送信には宛先の指定手段が無く、常に操作者本人へ送られる。
//
// この拒否が素通りすれば、テナント管理者の権限がそのまま任意宛先メールの送信手段になる。
// 送られるのはこちらのドメインから出る正当なパスワードリセットメールなので、
// 受け取った側には見分けがつかない。
func TestSendTestNotificationRefusesARequestedRecipient(t *testing.T) {
	server := newRefusalServer(t, refusalAdmin("acme"))

	// 宛先らしい名前を 1 つずつ単独で送る。まとめて送ると、本文の厳格な復号が
	// 「知らないフィールドがある」という別の理由で先に落ち、宛先を読む実装が
	// 混ざっていても気づけない。
	for _, field := range []string{"to", "recipient", "email", "to_address"} {
		sent := server.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test",
			map[string]any{field: "victim@example.test"})
		if sent.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s, want 200", field, sent.Code, sent.Body.String())
		}
		var result struct {
			Delivered bool   `json:"delivered"`
			To        string `json:"to"`
		}
		if err := json.Unmarshal(sent.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !result.Delivered || result.To != "operator@example.test" {
			t.Fatalf("%s: result=%+v, want delivery to the actor", field, result)
		}
	}

	// 効果: 指定した宛先へは 1 通も届いていない。応答の `to` だけを読むと、
	// 本人へ返事をしつつ裏で指定先にも送る実装を見逃す。
	if len(server.sender.Sent) != 4 {
		t.Fatalf("送信数 = %d, want one per attempt", len(server.sender.Sent))
	}
	for _, message := range server.sender.Sent {
		if message.To != "operator@example.test" {
			t.Fatalf("宛先 = %q, want the actor's own address", message.To)
		}
		if strings.Contains(message.To, "victim@example.test") {
			t.Fatalf("指定された宛先へ送信された: %+v", message)
		}
	}
}

// EX-TENANCY-018-04: 検証済みメールアドレスを持たない操作者のテスト送信は
// `InvalidRequestError` で拒否され、メールは送信されない。
//
// 未検証のアドレスへ送れば、そのアドレスの持ち主が操作者本人だという保証が無いまま
// メールが出る。アドレスの検証は「本人にしか届かない」の前提そのものなので、
// 検証されていない時点で本人宛だと言えない。
func TestSendTestNotificationRefusesUnverifiedActorAndSendsNothing(t *testing.T) {
	unverified := settingsActor("operator", "acme", []string{"admin"})
	address := "operator@example.test"
	unverified.Email = &address // EmailVerified は false のまま。
	server := newRefusalServer(t, unverified)

	refused := server.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test", nil)
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}
	if len(server.sender.Sent) != 0 {
		t.Fatalf("拒否されたのに %d 通送信された: %+v", len(server.sender.Sent), server.sender.Sent)
	}

	// 対照: 同じ操作者のアドレスが検証済みなら、同じ入口で本人へ届く。
	// 拒否が検証状態に由来し、送信の配線が無かったのではないことを示す。
	verified := newRefusalServer(t, refusalAdmin("acme"))
	sent := verified.send(t, http.MethodPost, refusalTemplatePath+"/password_reset/ja/test", nil)
	if sent.Code != http.StatusOK {
		t.Fatalf("対照 status=%d body=%s, want 200", sent.Code, sent.Body.String())
	}
	if len(verified.sender.Sent) != 1 || verified.sender.Sent[0].To != address {
		t.Fatalf("対照の送信 = %+v, want one message to %q", verified.sender.Sent, address)
	}
}
