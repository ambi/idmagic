// redirect_uri の照合規則。RFC 6749 3.1.2.3 が要求する登録済み URI との厳密一致と、
// RFC 8252 7.3 が認可要求に限って求めるループバックの IP のポートの例外を表す。
package domain

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// RedirectURIAllowed は presented が登録済み URI のいずれかとバイト単位で完全一致するかを返す。
//
// 接頭辞一致、大文字小文字の同一視、パーセントエンコーディングやパスの正規化は行わない。
// いずれを許しても、攻撃者が登録済み URI を接頭辞に持つ別の宛先へ認可コードを配送できるようになる。
// 認可要求の redirect_uri は AuthorizationRedirectURIAllowed で照合する。
func RedirectURIAllowed(registered []string, presented string) bool {
	if presented == "" {
		return false
	}
	return slices.Contains(registered, presented)
}

// AuthorizationRedirectURIAllowed は認可要求の redirect_uri を登録済み URI と照合する。
//
// 完全一致に加え、登録値と presented がどちらも http のループバックの IP（字句 127.0.0.1 か
// [::1]）で、ポートを除いた字句が一致するなら受け入れる（RFC 8252 7.3）。ネイティブアプリは
// OS から得た一時的なポートで待ち受けるため、登録時にポートを決められない。localhost は
// 名前解決に依存するため例外に含めない（RFC 8252 8.3）。
func AuthorizationRedirectURIAllowed(registered []string, presented string) bool {
	if RedirectURIAllowed(registered, presented) {
		return true
	}
	host, rest, ok := splitLoopbackRedirectURI(presented)
	if !ok {
		return false
	}
	return slices.ContainsFunc(registered, func(candidate string) bool {
		candidateHost, candidateRest, ok := splitLoopbackRedirectURI(candidate)
		return ok && candidateHost == host && candidateRest == rest
	})
}

// loopbackHosts はポートの例外を適用するホストの字句である。
var loopbackHosts = []string{"127.0.0.1", "[::1]"}

// splitLoopbackRedirectURI は http://<host>[:<port>]<rest> の形の値を host と rest に分ける。
//
// host はループバックの IP の字句に限り、port は 1〜65535 の十進数、rest は空か / か ? で
// 始まり、フラグメントを含まない。ホストの直後に @、\、. などが続く値は別のホストへ届き得る
// ため、字句の段階で拒否する。字句の分割に加えて url.Parse の結果とも照らし、両者が同じ
// ホストとポートを読む値に限って ok を返す。
func splitLoopbackRedirectURI(raw string) (host, rest string, ok bool) {
	const scheme = "http://"
	if !strings.HasPrefix(raw, scheme) || strings.Contains(raw, "#") {
		return "", "", false
	}
	for _, candidate := range loopbackHosts {
		after, found := strings.CutPrefix(raw[len(scheme):], candidate)
		if !found {
			continue
		}
		port, rest := "", after
		if afterColon, hasPort := strings.CutPrefix(after, ":"); hasPort {
			end := strings.IndexAny(afterColon, "/?")
			if end < 0 {
				end = len(afterColon)
			}
			port, rest = afterColon[:end], afterColon[end:]
			if !validPort(port) {
				return "", "", false
			}
		}
		if rest != "" && rest[0] != '/' && rest[0] != '?' {
			return "", "", false
		}
		if !parsedAsLoopback(raw, candidate, port) {
			return "", "", false
		}
		return candidate, rest, true
	}
	return "", "", false
}

// validPort は port が先頭のゼロを持たない 1〜65535 の十進数かを返す。
func validPort(port string) bool {
	if port == "" || len(port) > 5 || port[0] == '0' {
		return false
	}
	for _, r := range port {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

// parsedAsLoopback は url.Parse が raw を字句の分割と同じ host と port の http の URL として
// 読むかを返す。字句の分割だけに頼ると、解析器との解釈の差がそのまま配送先の取り違えになる。
func parsedAsLoopback(raw, host, port string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" && parsed.User == nil && parsed.Opaque == "" && parsed.Fragment == "" &&
		parsed.Hostname() == strings.Trim(host, "[]") && parsed.Port() == port
}

// ResolveRedirectURI は認可要求の応答を届ける配送先を決める。
//
// presented を示した要求は、AuthorizationRedirectURIAllowed が登録値と一致するとした値だけを
// 受け入れ、示された値をそのまま配送先とする。省いた要求は、登録が一件で
// openID でない場合に限り、その登録値へ届ける（RFC 6749 3.1.2.3、OAuth 2.1 2.3.2）。
// OpenID Connect Core 3.1.2.1 は認証要求の redirect_uri を必須とする。登録が複数なら、
// どれを選んでもクライアントの意図しない宛先になり得る。値のない redirect_uri は、
// RFC 6749 3.1 に従い省いたものとして扱う。
func ResolveRedirectURI(registered []string, presented string, openID bool) (string, error) {
	switch {
	case presented != "":
		if !AuthorizationRedirectURIAllowed(registered, presented) {
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
