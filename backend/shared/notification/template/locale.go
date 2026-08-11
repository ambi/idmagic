package template

import (
	"slices"
	"strings"
)

// FallbackLocale は最終段の既定 locale。UI の FallbackLocale (spec/contexts/system.yaml の
// UX-LOCALE) と同じ値を使い、画面とメールで既定言語が食い違わないようにする。
const FallbackLocale = "en"

// supportedLocales はカタログが同梱翻訳を持つ locale。翻訳ファイル (defaults/<locale>.yaml)
// を足したらここに足す (決定 7: locale は enum ではなく言語タグとして一般化する)。
var supportedLocales = []string{"ja", "en"}

func SupportedLocales() []string { return slices.Clone(supportedLocales) }

func LocaleSupported(locale string) bool { return slices.Contains(supportedLocales, locale) }

// NormalizeLocale は BCP47 言語タグから primary language を取り出し、同梱翻訳を持つ
// locale なら小文字で返す。持たない locale と空文字列は空文字列を返す。
func NormalizeLocale(tag string) string {
	primary := strings.ToLower(strings.TrimSpace(tag))
	if cut := strings.IndexAny(primary, "-_"); cut >= 0 {
		primary = primary[:cut]
	}
	if !LocaleSupported(primary) {
		return ""
	}
	return primary
}

// ResolveLocale は候補を順に見て、最初にカタログが対応する locale を返す。どれも
// 対応していなければ FallbackLocale。呼び出し順が
// NotificationLocaleResolution (受信者 → テナント既定 → システム既定) を表す。
func ResolveLocale(candidates ...string) string {
	for _, candidate := range candidates {
		if locale := NormalizeLocale(candidate); locale != "" {
			return locale
		}
	}
	return FallbackLocale
}
