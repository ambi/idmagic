// redirect_uri の照合規則。RFC 6749 3.1.2.3 が要求する登録済み URI との厳密一致を表す。
package domain

import (
	"errors"
	"slices"
)

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

// ResolveRedirectURI は認可要求の応答を届ける配送先を決める。
//
// presented を示した要求は、登録値との完全一致だけを受け入れる。省いた要求は、登録が一件で
// openID でない場合に限り、その登録値へ届ける（RFC 6749 3.1.2.3、OAuth 2.1 2.3.2）。
// OpenID Connect Core 3.1.2.1 は認証要求の redirect_uri を必須とする。登録が複数なら、
// どれを選んでもクライアントの意図しない宛先になり得る。値のない redirect_uri は、
// RFC 6749 3.1 に従い省いたものとして扱う。
func ResolveRedirectURI(registered []string, presented string, openID bool) (string, error) {
	switch {
	case presented != "":
		if !RedirectURIAllowed(registered, presented) {
			return "", errors.New("redirect_uri is not a registered URI")
		}
		return presented, nil
	case openID:
		return "", errors.New("redirect_uri is required for OpenID Connect requests")
	case len(registered) != 1:
		return "", errors.New("redirect_uri is required unless exactly one redirect URI is registered")
	default:
		return registered[0], nil
	}
}
