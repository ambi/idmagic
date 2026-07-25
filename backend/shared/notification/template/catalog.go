package template

import (
	"embed"
	"fmt"
	"sync"

	"github.com/goccy/go-yaml"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// defaultsFS は組込み既定テンプレートを locale 別ファイルとして持つ。同梱翻訳は
// ja / en だが、ファイルを足せば locale を増やせる形にしてある (ADR-142 決定 7)。
//
//go:embed defaults/*.yaml
var defaultsFS embed.FS

type builtinDefinition struct {
	Subject  string `yaml:"subject"`
	BodyText string `yaml:"body_text"`
	BodyHTML string `yaml:"body_html"`
}

// builtins は locale -> template_key -> 文面。埋め込みファイルの読み込みは一度だけ行う。
var builtins = sync.OnceValues(loadBuiltins)

func loadBuiltins() (map[string]map[notificationports.TemplateKey]Definition, error) {
	loaded := map[string]map[notificationports.TemplateKey]Definition{}
	for _, locale := range supportedLocales {
		raw, err := defaultsFS.ReadFile("defaults/" + locale + ".yaml")
		if err != nil {
			return nil, fmt.Errorf("read builtin notification templates for %q: %w", locale, err)
		}
		var parsed map[string]builtinDefinition
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("parse builtin notification templates for %q: %w", locale, err)
		}
		byKey := map[notificationports.TemplateKey]Definition{}
		for name, def := range parsed {
			key := notificationports.TemplateKey(name)
			if !key.Valid() {
				return nil, fmt.Errorf("%w: %q in defaults/%s.yaml", ErrUnknownTemplateKey, name, locale)
			}
			byKey[key] = Definition{Subject: def.Subject, BodyText: def.BodyText, BodyHTML: def.BodyHTML}
		}
		loaded[locale] = byKey
	}
	return loaded, nil
}

// Builtin は組込み既定テンプレートを返す。テナントは編集できず、上書きの削除で常に
// ここへ戻る (ADR-142 決定 1)。カタログの完全性は package のテストで固定しているため、
// 対応 locale と既知 key の組で not found にはならない。
func Builtin(key notificationports.TemplateKey, locale string) (Definition, error) {
	if !key.Valid() {
		return Definition{}, fmt.Errorf("%w: %q", ErrUnknownTemplateKey, key)
	}
	if !LocaleSupported(locale) {
		return Definition{}, fmt.Errorf("%w: %q", ErrUnsupportedLocale, locale)
	}
	loaded, err := builtins()
	if err != nil {
		return Definition{}, err
	}
	def, ok := loaded[locale][key]
	if !ok {
		return Definition{}, fmt.Errorf("%w: %q has no builtin template for %q", ErrUnknownTemplateKey, key, locale)
	}
	return def, nil
}
