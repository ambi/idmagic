package handlers_http_test

// SCL scenario "テナントの通知テンプレート上書きは組込み既定より優先される" /
// "許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される" /
// "プレビューは実送信せずテスト送信は操作者本人にしか届かない" を
// /api/admin/tenant/notification_templates 経由で検証する。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const notificationTemplatePath = "/realms/acme/api/admin/tenant/notification_templates"

func newNotificationTemplateServer(t *testing.T, actor *userdomain.User) (*echo.Echo, *email_memory.NoopEmailSender, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	tenantRepo := memory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), activeTenant("acme", "Acme")); err != nil {
		t.Fatal(err)
	}
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	events := make([]spec.DomainEvent, 0)
	sender := &email_memory.NoopEmailSender{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test", SCL: spec.MustLoadSCL(),
			TenantRepo: tenantRepo,
			Emit:       func(event spec.DomainEvent) { events = append(events, event) },
		},
		UserRepo:      userRepo,
		EmailSender:   sender,
		AuthnResolver: resolver,
		Tenancy: tenancy.Module{
			TenantRepo:            tenantRepo,
			BrandingRepo:          memory.NewTenantBrandingRepository(),
			NotificationTemplates: memory.NewNotificationTemplateRepository(),
		},
	})
	return e, sender, &events
}

func notificationAdmin() *userdomain.User {
	actor := settingsActor("admin", "acme", []string{"admin"})
	email := "operator@example.test"
	actor.Email = &email
	actor.EmailVerified = true
	return actor
}

func notificationRequest(t *testing.T, e *echo.Echo, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	csrf, cookie := passwordResetContextCSRF(t, e, "/realms/acme/api/auth/password_reset_context")
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestNotificationTemplatesRejectNonAdmin(t *testing.T) {
	e, _, _ := newNotificationTemplateServer(t, settingsActor("alice", "acme", nil))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, notificationTemplatePath, http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNotificationTemplatesListReturnsWholeCatalog(t *testing.T) {
	e, _, _ := newNotificationTemplateServer(t, notificationAdmin())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, notificationTemplatePath, http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Templates []struct {
			TemplateKey string `json:"template_key"`
			Locale      string `json:"locale"`
			Customized  bool   `json:"customized"`
			Subject     string `json:"subject"`
		} `json:"templates"`
		SupportedLocales []string `json:"supported_locales"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Templates) == 0 || len(body.SupportedLocales) < 2 {
		t.Fatalf("body=%+v", body)
	}
	for _, tmpl := range body.Templates {
		if tmpl.Subject == "" || tmpl.Customized {
			t.Fatalf("unexpected row: %+v", tmpl)
		}
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
func TestNotificationTemplateUpdateAndReset(t *testing.T) {
	e, _, events := newNotificationTemplateServer(t, notificationAdmin())
	path := notificationTemplatePath + "/password_reset/ja"

	rec := notificationRequest(t, e, http.MethodPut, path, map[string]any{
		"subject":           "【Acme】パスワード再設定",
		"body_text":         "{{user_display_name}} さん {{reset_url}}",
		"body_html":         "<p>{{user_display_name}} さん {{reset_url}}</p>",
		"from_display_name": "Acme サポート",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Customized     bool     `json:"customized"`
		Subject        string   `json:"subject"`
		DefaultSubject string   `json:"default_subject"`
		Placeholders   []string `json:"placeholders"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if !updated.Customized || updated.Subject != "【Acme】パスワード再設定" {
		t.Fatalf("body=%+v", updated)
	}
	if updated.DefaultSubject == "" || len(updated.Placeholders) == 0 {
		t.Fatalf("the builtin default and the allowed placeholders must be returned: %+v", updated)
	}
	if len(*events) != 1 {
		t.Fatalf("events=%d, want 1 NotificationTemplateUpdated", len(*events))
	}
	if _, ok := (*events)[0].(*domain.NotificationTemplateUpdated); !ok {
		t.Fatalf("event type=%T", (*events)[0])
	}

	rec = notificationRequest(t, e, http.MethodDelete, path, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var reset struct {
		Customized bool `json:"customized"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reset); err != nil {
		t.Fatal(err)
	}
	if reset.Customized {
		t.Fatal("reset did not return to the builtin default")
	}
	if len(*events) != 2 {
		t.Fatalf("events=%d, want a NotificationTemplateReset too", len(*events))
	}
	if _, ok := (*events)[1].(*domain.NotificationTemplateReset); !ok {
		t.Fatalf("event type=%T", (*events)[1])
	}
}

// scenario `Tenancy: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される`
func TestNotificationTemplateUpdateRejectsUnknownPlaceholder(t *testing.T) {
	e, _, events := newNotificationTemplateServer(t, notificationAdmin())

	rec := notificationRequest(t, e, http.MethodPut, notificationTemplatePath+"/password_reset/ja", map[string]any{
		"subject":   "件名",
		"body_text": "{{password}}",
		"body_html": "<p>{{password}}</p>",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(*events) != 0 {
		t.Fatalf("a rejected update emitted %d events", len(*events))
	}
}

func TestNotificationTemplateUpdateRejectsUnknownKeyAndLocale(t *testing.T) {
	e, _, _ := newNotificationTemplateServer(t, notificationAdmin())
	body := map[string]any{"subject": "件名", "body_text": "本文", "body_html": "<p>本文</p>"}

	for _, path := range []string{
		notificationTemplatePath + "/made_up/ja",
		notificationTemplatePath + "/password_reset/fr",
	} {
		rec := notificationRequest(t, e, http.MethodPut, path, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
func TestNotificationTemplatePreviewDoesNotSend(t *testing.T) {
	e, sender, _ := newNotificationTemplateServer(t, notificationAdmin())

	rec := notificationRequest(t, e, http.MethodPost, notificationTemplatePath+"/password_reset/ja/preview", map[string]any{
		"subject":   "編集中の件名",
		"body_text": "{{user_display_name}} さん {{reset_url}}",
		"body_html": "<p>{{user_display_name}}</p>",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var preview struct {
		Subject  string `json:"subject"`
		BodyText string `json:"body_text"`
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Subject != "編集中の件名" || preview.BodyText == "" || preview.BodyHTML == "" {
		t.Fatalf("preview=%+v", preview)
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("preview sent %d emails", len(sender.Sent))
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
// 宛先は操作者本人に固定され、リクエストで指定できない (ADR-142 決定 8)。
func TestNotificationTemplateTestSendGoesToTheActorOnly(t *testing.T) {
	e, sender, _ := newNotificationTemplateServer(t, notificationAdmin())

	rec := notificationRequest(t, e, http.MethodPost, notificationTemplatePath+"/password_reset/ja/test",
		map[string]any{"to": "victim@example.test"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result struct {
		Delivered bool   `json:"delivered"`
		To        string `json:"to"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Delivered || result.To != "operator@example.test" {
		t.Fatalf("result=%+v", result)
	}
	if len(sender.Sent) != 1 || sender.Sent[0].To != "operator@example.test" {
		t.Fatalf("sent=%+v", sender.Sent)
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
// (extension at 4): 操作者に検証済みアドレスが無ければ拒否する。
func TestNotificationTemplateTestSendRequiresVerifiedActorAddress(t *testing.T) {
	actor := settingsActor("admin", "acme", []string{"admin"})
	unverified := "operator@example.test"
	actor.Email = &unverified
	e, sender, _ := newNotificationTemplateServer(t, actor)

	rec := notificationRequest(t, e, http.MethodPost, notificationTemplatePath+"/password_reset/ja/test", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("sent %d emails", len(sender.Sent))
	}
}
