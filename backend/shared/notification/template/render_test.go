package template_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
)

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
// (extension at 2): HTML 側の差し込み値はエスケープされて描画され、タグとして
// 解釈されない。エスケープはレンダラの責務に閉じる。
func TestRenderEscapesVariablesInHTMLOnly(t *testing.T) {
	def := template.Definition{
		Subject:  "{{product_name}} からのお知らせ",
		BodyText: "{{user_display_name}} 様\n{{reset_url}}",
		BodyHTML: "<p>{{user_display_name}} 様</p><p><a href=\"{{reset_url}}\">再設定</a></p>",
	}
	vars := map[string]string{
		"product_name":      "IdMagic",
		"user_display_name": `<script>alert("x")</script>`,
		"reset_url":         "https://idp.test/reset_password?token=a&b=c",
	}

	rendered, err := template.Render(def, vars)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(rendered.Text, `<script>alert("x")</script>`) {
		t.Errorf("text body should keep the raw value, got %q", rendered.Text)
	}
	if strings.Contains(rendered.HTML, "<script>") {
		t.Errorf("html body must not contain an unescaped tag, got %q", rendered.HTML)
	}
	if !strings.Contains(rendered.HTML, "&lt;script&gt;") {
		t.Errorf("html body should contain the escaped value, got %q", rendered.HTML)
	}
	if !strings.Contains(rendered.HTML, "token=a&amp;b=c") {
		t.Errorf("html body should escape & in the URL, got %q", rendered.HTML)
	}
	if !strings.Contains(rendered.Text, "token=a&b=c") {
		t.Errorf("text body should keep the raw URL, got %q", rendered.Text)
	}
	if rendered.Subject != "IdMagic からのお知らせ" {
		t.Errorf("subject = %q", rendered.Subject)
	}
}

// レンダラは未定義の変数を空文字列へ潰さない。潰すと「リンクが欠けたメール」が
// 配られるため、描画側でも fail-closed にする。
func TestRenderRejectsMissingVariable(t *testing.T) {
	def := template.Definition{
		Subject:  "件名",
		BodyText: "{{reset_url}}",
		BodyHTML: "<p>{{reset_url}}</p>",
	}

	if _, err := template.Render(def, map[string]string{}); !errors.Is(err, template.ErrMissingVariable) {
		t.Fatalf("Render error = %v, want ErrMissingVariable", err)
	}
}

// 件名は単一行として扱う。改行を含む差し込み値でヘッダを分断できないようにする。
func TestRenderCollapsesSubjectToSingleLine(t *testing.T) {
	def := template.Definition{
		Subject:  "{{user_display_name}} 様へ",
		BodyText: "本文",
		BodyHTML: "<p>本文</p>",
	}

	rendered, err := template.Render(def, map[string]string{"user_display_name": "山田\r\nBcc: attacker@example.test"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.ContainsAny(rendered.Subject, "\r\n") {
		t.Fatalf("subject must be a single line, got %q", rendered.Subject)
	}
}

// scenario `Tenancy: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される`
func TestValidateDefinitionRejectsUnknownPlaceholder(t *testing.T) {
	cases := []struct {
		name string
		def  template.Definition
	}{
		{"unknown name in body text", template.Definition{
			Subject: "件名", BodyText: "{{password}}", BodyHTML: "<p>本文</p>",
		}},
		{"unknown name in html body", template.Definition{
			Subject: "件名", BodyText: "本文", BodyHTML: "<p>{{password}}</p>",
		}},
		{"unknown name in subject", template.Definition{
			Subject: "{{password}}", BodyText: "本文", BodyHTML: "<p>本文</p>",
		}},
		{"allowed name with wrong case", template.Definition{
			Subject: "件名", BodyText: "{{Reset_URL}}", BodyHTML: "<p>本文</p>",
		}},
		{"placeholder of another template key", template.Definition{
			Subject: "件名", BodyText: "{{new_email}}", BodyHTML: "<p>本文</p>",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := template.ValidateDefinition(notificationports.TemplateKeyPasswordReset, tc.def)
			if !errors.Is(err, template.ErrUnknownPlaceholder) {
				t.Fatalf("ValidateDefinition error = %v, want ErrUnknownPlaceholder", err)
			}
		})
	}
}

func TestValidateDefinitionAcceptsAllowedPlaceholders(t *testing.T) {
	def := template.Definition{
		Subject:  "{{product_name}} のパスワード再設定",
		BodyText: "{{user_display_name}} 様 {{reset_url}} {{expires_in_minutes}} {{tenant_display_name}}",
		BodyHTML: "<p>{{ user_display_name }}</p><a href=\"{{reset_url}}\">x</a>",
	}
	if err := template.ValidateDefinition(notificationports.TemplateKeyPasswordReset, def); err != nil {
		t.Fatalf("ValidateDefinition: %v", err)
	}
}

// scenario `Tenancy: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される`
// (extension at 1): HTML 本文を空にしてテキスト本文だけを保存しようとすると拒否される。
func TestValidateDefinitionRequiresSubjectTextAndHTMLTogether(t *testing.T) {
	cases := map[string]template.Definition{
		"missing html":    {Subject: "件名", BodyText: "本文", BodyHTML: ""},
		"missing text":    {Subject: "件名", BodyText: "", BodyHTML: "<p>本文</p>"},
		"missing subject": {Subject: "", BodyText: "本文", BodyHTML: "<p>本文</p>"},
		"blank html":      {Subject: "件名", BodyText: "本文", BodyHTML: "   \n "},
	}
	for name, def := range cases {
		t.Run(name, func(t *testing.T) {
			err := template.ValidateDefinition(notificationports.TemplateKeyPasswordReset, def)
			if !errors.Is(err, template.ErrIncompleteTemplate) {
				t.Fatalf("ValidateDefinition error = %v, want ErrIncompleteTemplate", err)
			}
		})
	}
}

func TestValidateDefinitionRejectsUnknownTemplateKey(t *testing.T) {
	def := template.Definition{Subject: "件名", BodyText: "本文", BodyHTML: "<p>本文</p>"}
	if err := template.ValidateDefinition(notificationports.TemplateKey("made_up"), def); !errors.Is(err, template.ErrUnknownTemplateKey) {
		t.Fatalf("ValidateDefinition error = %v, want ErrUnknownTemplateKey", err)
	}
}

// 許可集合は API から返して編集者に見せるため、key ごとに宣言されていること自体が契約。
func TestPlaceholdersAreDeclaredForEveryKey(t *testing.T) {
	shared := []string{"product_name", "tenant_display_name", "user_display_name"}
	for _, key := range notificationports.TemplateKeys() {
		placeholders := template.Placeholders(key)
		if len(placeholders) == 0 {
			t.Errorf("%s has no declared placeholders", key)
		}
		for _, want := range shared {
			if !slices.Contains(placeholders, want) {
				t.Errorf("%s does not allow the shared placeholder %q", key, want)
			}
		}
		// 決定 10: 資格情報・生 IP は許可集合に入れない。
		for _, forbidden := range []string{"password", "password_hash", "token", "totp_secret", "client_ip", "ip_address"} {
			if slices.Contains(placeholders, forbidden) {
				t.Errorf("%s must not allow the placeholder %q", key, forbidden)
			}
		}
	}
}
