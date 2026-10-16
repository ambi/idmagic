// redirect_uri の照合規則。RFC 6749 3.1.2.3 が要求する登録済み URI との厳密一致を表す。
package domain

import "slices"

// RedirectURIAllowed は presented が登録済み URI のいずれかとバイト単位で完全一致するかを返す。
//
// 接頭辞一致、大文字小文字の同一視、パーセントエンコーディングやパスの正規化は行わない。
// いずれを許しても、攻撃者が登録済み URI を接頭辞に持つ別の宛先へ認可コードを配送できるようになる。
func RedirectURIAllowed(registered []string, presented string) bool {
	if presented == "" {
		return false
	}
	return slices.Contains(registered, presented)
}
