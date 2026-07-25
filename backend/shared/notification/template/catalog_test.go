package template_test

import (
	"errors"
	"strings"
	"testing"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
)

// カタログの完全性は仕様。ある key × locale の組が欠けると、その言語の利用者に空の
// メールが届く (= 復旧導線が消える)。組込み既定は編集不可なので、ここで固定しておけば
// 実行時にこの穴は発生しない。
func TestBuiltinCatalogIsCompleteAndValid(t *testing.T) {
	for _, key := range notificationports.TemplateKeys() {
		for _, locale := range template.SupportedLocales() {
			def, err := template.Builtin(key, locale)
			if err != nil {
				t.Fatalf("Builtin(%s, %s): %v", key, locale, err)
			}
			if strings.TrimSpace(def.Subject) == "" || strings.TrimSpace(def.BodyText) == "" || strings.TrimSpace(def.BodyHTML) == "" {
				t.Errorf("Builtin(%s, %s) has an empty part", key, locale)
			}
			if err := template.ValidateDefinition(key, def); err != nil {
				t.Errorf("Builtin(%s, %s) is not a valid definition: %v", key, locale, err)
			}
			// 組込み既定はサンプル値だけで描画できなければならない。描画できない
			// 変数が残っていれば実送信でも同じ失敗が起きる。
			if _, err := template.Render(def, template.SampleVars(key)); err != nil {
				t.Errorf("Builtin(%s, %s) does not render with sample vars: %v", key, locale, err)
			}
		}
	}
}

func TestBuiltinRejectsUnknownKeyAndLocale(t *testing.T) {
	if _, err := template.Builtin(notificationports.TemplateKey("made_up"), "ja"); !errors.Is(err, template.ErrUnknownTemplateKey) {
		t.Errorf("Builtin with unknown key error = %v, want ErrUnknownTemplateKey", err)
	}
	if _, err := template.Builtin(notificationports.TemplateKeyPasswordReset, "fr"); !errors.Is(err, template.ErrUnsupportedLocale) {
		t.Errorf("Builtin with unsupported locale error = %v, want ErrUnsupportedLocale", err)
	}
}

// 日本語 UI に英語メールが届く不整合を解消したことを固定する。件名は組込み既定に
// 依存するので、日本語文字が含まれることだけを固定し文言そのものは縛らない。
func TestBuiltinJapaneseTemplatesAreLocalized(t *testing.T) {
	for _, key := range notificationports.TemplateKeys() {
		ja, err := template.Builtin(key, "ja")
		if err != nil {
			t.Fatalf("Builtin(%s, ja): %v", key, err)
		}
		en, err := template.Builtin(key, "en")
		if err != nil {
			t.Fatalf("Builtin(%s, en): %v", key, err)
		}
		if ja.Subject == en.Subject {
			t.Errorf("%s: ja and en share the subject %q", key, ja.Subject)
		}
		if !containsJapanese(ja.Subject) {
			t.Errorf("%s: ja subject %q has no Japanese characters", key, ja.Subject)
		}
		if containsJapanese(en.Subject) {
			t.Errorf("%s: en subject %q contains Japanese characters", key, en.Subject)
		}
	}
}

// SampleVars はプレビュー用の固定値。実データを流さないため、許可集合の全変数を
// 埋めていなければプレビューが失敗する。
func TestSampleVarsCoverEveryPlaceholder(t *testing.T) {
	for _, key := range notificationports.TemplateKeys() {
		sample := template.SampleVars(key)
		for _, name := range template.Placeholders(key) {
			if strings.TrimSpace(sample[name]) == "" {
				t.Errorf("SampleVars(%s) has no value for %q", key, name)
			}
		}
	}
}

func containsJapanese(value string) bool {
	for _, r := range value {
		if (r >= 0x3040 && r <= 0x30ff) || (r >= 0x4e00 && r <= 0x9fff) {
			return true
		}
	}
	return false
}
