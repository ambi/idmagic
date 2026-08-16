package template

import (
	"fmt"
	"html"
	"maps"
	"regexp"
	"slices"
	"strings"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// placeholderPattern は「差し込み変数のように見えるもの」を広く拾う。`{{Password}}` の
// ような綴り違いも拾って許可集合の検査対象にするため、名前の形は絞らない。絞ると
// 綴り違いが検査をすり抜けて本文にそのまま残り、編集者が気づけない。
var placeholderPattern = regexp.MustCompile(`\{\{[^{}]*\}\}`)

// 全 key 共通の差し込み変数。ブランディングと宛名は全通知で必要になるため key ごとに
// 差を付けない (「テンプレートキーごとの placeholder 許可集合」)。
var sharedPlaceholders = []string{"product_name", "tenant_display_name", "user_display_name"}

// keyPlaceholders は key 固有の差し込み変数。資格情報・単発トークン単体・生 IP は
// 意図的に含めない。
var keyPlaceholders = map[notificationports.TemplateKey][]string{
	notificationports.TemplateKeyPasswordReset:                 {"reset_url", "expires_in_minutes"},
	notificationports.TemplateKeyEmailVerification:             {"verification_url", "expires_in_minutes"},
	notificationports.TemplateKeyEmailChangeConfirmation:       {"confirmation_url", "expires_in_minutes", "new_email"},
	notificationports.TemplateKeyAccountSecurityAlert:          {"event_description", "occurred_at", "device_summary", "security_review_url"},
	notificationports.TemplateKeyAgentActionApprovalRequest:    {"approval_url", "client_name", "agent_name", "binding_message", "expires_in_minutes"},
	notificationports.TemplateKeyLifecycleWorkflowNotification: {"notification_key"},
}

// sampleValues はプレビュー用の固定値。実在の利用者名やトークンをプレビュー経路に
// 流さない。
var sampleValues = map[string]string{
	"product_name":        DefaultProductName,
	"tenant_display_name": "Example Inc.",
	"user_display_name":   "Taro Yamada",
	"reset_url":           "https://idp.example.test/reset_password?token=SAMPLE-TOKEN",
	"verification_url":    "https://idp.example.test/account/email/verify?token=SAMPLE-TOKEN",
	"confirmation_url":    "https://idp.example.test/account/email/verify?token=SAMPLE-TOKEN",
	"expires_in_minutes":  "30",
	"new_email":           "new-address@example.test",
	"event_description":   "Sign-in from a new device",
	"occurred_at":         "2026-01-01 12:00 UTC",
	"device_summary":      "Chrome / macOS (JP)",
	"security_review_url": "https://idp.example.test/account/security",
	"approval_url":        "https://idp.example.test/account/approvals",
	"client_name":         "Expense Agent",
	"agent_name":          "Travel Assistant",
	"binding_message":     "Trip W-123",
	"notification_key":    "welcome",
}

// Placeholders は template_key ごとの差し込み変数の許可集合を返す。管理 API がこの
// 集合を返すため、編集者は使える変数を推測しなくてよい。
func Placeholders(key notificationports.TemplateKey) []string {
	specific, ok := keyPlaceholders[key]
	if !ok {
		return nil
	}
	return append(slices.Clone(sharedPlaceholders), specific...)
}

// SampleVars はプレビューに使う固定のサンプル値。
func SampleVars(key notificationports.TemplateKey) map[string]string {
	vars := map[string]string{}
	for _, name := range Placeholders(key) {
		vars[name] = sampleValues[name]
	}
	return vars
}

// ValidateDefinition は保存前の fail-closed 検証。件名 / テキスト本文 / HTML 本文が
// 揃っていること、および使われている差し込み変数が許可集合に収まっていることを確かめる。
func ValidateDefinition(key notificationports.TemplateKey, def Definition) error {
	allowed := Placeholders(key)
	if allowed == nil {
		return fmt.Errorf("%w: %q", ErrUnknownTemplateKey, key)
	}
	if strings.TrimSpace(def.Subject) == "" || strings.TrimSpace(def.BodyText) == "" || strings.TrimSpace(def.BodyHTML) == "" {
		return ErrIncompleteTemplate
	}
	// 上限は notification_templates の CHECK と同じ数。ここで止めないと、
	// 制約違反がデータベースから返って 500 になる。
	for _, field := range []struct {
		name  string
		value string
		limit int
	}{
		{"subject", def.Subject, spec.LengthDisplayName},
		{"body_text", def.BodyText, spec.LengthPlainBody},
		{"body_html", def.BodyHTML, spec.LengthRichBody},
		{"from_display_name", def.FromDisplayName, spec.LengthChromeLabel},
	} {
		if err := spec.CheckMaxChars(field.name, field.value, field.limit); err != nil {
			return err
		}
	}
	for _, part := range []string{def.Subject, def.BodyText, def.BodyHTML} {
		for _, name := range placeholderNames(part) {
			if !slices.Contains(allowed, name) {
				return fmt.Errorf("%w: %q is not allowed for %s", ErrUnknownPlaceholder, name, key)
			}
		}
	}
	return nil
}

// Render は差し込み変数を展開する。HTML 側は必ずエスケープし、テキスト側は素で展開する。
// 値の無い変数は空文字列へ潰さずエラーにする。
func Render(def Definition, vars map[string]string) (Rendered, error) {
	subject, err := expand(def.Subject, vars, false)
	if err != nil {
		return Rendered{}, err
	}
	text, err := expand(def.BodyText, vars, false)
	if err != nil {
		return Rendered{}, err
	}
	fragment, err := expand(def.BodyHTML, vars, true)
	if err != nil {
		return Rendered{}, err
	}
	subject = collapseToSingleLine(subject)
	return Rendered{
		Subject:         subject,
		Text:            text,
		HTML:            wrapHTMLDocument(subject, fragment),
		FromDisplayName: collapseToSingleLine(def.FromDisplayName),
	}, nil
}

// expand は `{{name}}` を値へ置き換える。escapeValues が true なら値だけを HTML
// エスケープする (テンプレート本体の markup はそのまま通す)。
func expand(body string, vars map[string]string, escapeValues bool) (string, error) {
	var missing []string
	expanded := placeholderPattern.ReplaceAllStringFunc(body, func(token string) string {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(token, "{{"), "}}"))
		value, ok := vars[name]
		if !ok {
			missing = append(missing, name)
			return token
		}
		if escapeValues {
			return html.EscapeString(value)
		}
		return value
	})
	if len(missing) > 0 {
		slices.Sort(missing)
		return "", fmt.Errorf("%w: %s", ErrMissingVariable, strings.Join(slices.Compact(missing), ", "))
	}
	return expanded, nil
}

func placeholderNames(body string) []string {
	var names []string
	for _, token := range placeholderPattern.FindAllString(body, -1) {
		names = append(names, strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(token, "{{"), "}}")))
	}
	return names
}

// collapseToSingleLine は件名と差出人表示名を単一行に潰す。SMTP アダプタもヘッダを
// sanitize するが、描画結果そのものを単一行に確定させておくことでプレビューと実送信の
// 件名が一致する。
func collapseToSingleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// wrapHTMLDocument はテナントが上書きできない文書外枠を供給する。
// doctype / charset / viewport / 本文コンテナのスタイルはシステムが持ち、上書きできるのは
// `<body>` 内の fragment だけ。
func wrapHTMLDocument(subject, fragment string) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html>\n<head>\n")
	b.WriteString(`<meta charset="utf-8">` + "\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	b.WriteString("<title>" + html.EscapeString(subject) + "</title>\n")
	b.WriteString("</head>\n")
	b.WriteString(`<body style="margin:0;padding:24px;background-color:#f8fafc;` +
		`font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Hiragino Sans','Noto Sans JP',sans-serif;` +
		`color:#0f172a;line-height:1.7;">` + "\n")
	b.WriteString(`<div style="max-width:600px;margin:0 auto;padding:32px;background-color:#ffffff;` +
		`border-radius:12px;border:1px solid #e2e8f0;">` + "\n")
	b.WriteString(strings.TrimSpace(fragment))
	b.WriteString("\n</div>\n</body>\n</html>\n")
	return b.String()
}

// mergeVars は呼び出し元の変数に Notifier が補う変数を重ねる。呼び出し元の値を優先し、
// テナント由来の既定値は不足分だけを埋める。
func mergeVars(base, overlay map[string]string) map[string]string {
	merged := map[string]string{}
	maps.Copy(merged, base)
	for name, value := range overlay {
		if value != "" {
			merged[name] = value
		}
	}
	return merged
}
