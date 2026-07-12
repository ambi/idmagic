package kernel

import "unicode"

const genericEnglishErrorText = "The request could not be completed."

// EnglishErrorText preserves an English error text and replaces Japanese text
// with a stable English fallback before it crosses an API boundary.
func EnglishErrorText(text string) string {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			return genericEnglishErrorText
		}
	}
	return text
}
