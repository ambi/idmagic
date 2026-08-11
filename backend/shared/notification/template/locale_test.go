package template_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/notification/template"
)

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
// の主経路と 3 つの extension: 受信者 locale → テナント既定 → システム既定の順に、
// カタログが同梱翻訳を持つ最初の locale を採る。
func TestResolveLocaleFollowsRecipientThenTenantThenSystem(t *testing.T) {
	cases := []struct {
		name       string
		candidates []string
		want       string
	}{
		{"recipient locale wins", []string{"ja", "en", "en"}, "ja"},
		{"tenant default when recipient is unset", []string{"", "ja", "en"}, "ja"},
		{"system default when recipient and tenant are unset", []string{"", "", "en"}, "en"},
		{"unsupported recipient locale falls through", []string{"fr", "ja", "en"}, "ja"},
		{"unsupported everywhere falls back to en", []string{"fr", "de", ""}, "en"},
		{"bcp47 tag narrows to the primary language", []string{"ja-JP", "", "en"}, "ja"},
		{"case and whitespace are normalized", []string{" JA ", "", "en"}, "ja"},
		{"no candidates at all", nil, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := template.ResolveLocale(tc.candidates...); got != tc.want {
				t.Fatalf("ResolveLocale(%q) = %q, want %q", tc.candidates, got, tc.want)
			}
		})
	}
}

func TestSupportedLocalesCoversJaAndEn(t *testing.T) {
	locales := template.SupportedLocales()
	if len(locales) < 2 {
		t.Fatalf("SupportedLocales() = %q, want at least ja and en", locales)
	}
	for _, want := range []string{"ja", "en"} {
		found := false
		for _, locale := range locales {
			if locale == want {
				found = true
			}
		}
		if !found {
			t.Errorf("SupportedLocales() = %q, missing %q", locales, want)
		}
	}
	if !template.LocaleSupported("ja") || template.LocaleSupported("fr") {
		t.Errorf("LocaleSupported disagrees with SupportedLocales()")
	}
}

func TestNormalizeLocaleRejectsUnsupportedTags(t *testing.T) {
	cases := map[string]string{
		"ja":    "ja",
		"ja-JP": "ja",
		"EN":    "en",
		"en_US": "en",
		"fr":    "",
		"":      "",
		"jaja":  "",
	}
	for input, want := range cases {
		if got := template.NormalizeLocale(input); got != want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", input, got, want)
		}
	}
}
