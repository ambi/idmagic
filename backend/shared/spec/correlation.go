package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
)

// 相関ハッシュ / IP 丸めの単一ヘルパ (wi-145 / ADR-046)。
//
// username / IP の相関検索属性 (audit search)、throttle の keyHash (ADR-029)、bucket の keyHash は
// すべてこのヘルパを共有する。hash / truncation を誤ると個人情報が監査ログに平文で流れるため、
// 抽出と検索変換は必ずこの単一実装を使う。

// SaltedHash は tenant salt 付き SHA-256 を hex 文字列で返す (hex(sha256(salt || value)))。
// salt を前置することで同一値でも tenant が異なれば hash が異なり、cross-tenant 相関を防ぐ。
func SaltedHash(salt []byte, value string) string {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

// NormalizeUsername は username を相関ハッシュ前に正規化する (ADR-046: lowercased)。
// emit 側の抽出と検索側の変換が同じ hash を得るよう、両者はこの単一ヘルパを共有する。
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// TruncateIP は IP を粗い相関単位へ丸める: IPv4 は /24、IPv6 は /48 (ADR-046)。
// 丸めた結果を CIDR 表記 (例 "203.0.113.0/24" / "2001:db8::/48") で返す。
// パースできない入力は "" を返す (呼び出し側で空を弾く)。
func TruncateIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		masked := v4.Mask(net.CIDRMask(24, 32))
		return (&net.IPNet{IP: masked, Mask: net.CIDRMask(24, 32)}).String()
	}
	masked := parsed.Mask(net.CIDRMask(48, 128))
	return (&net.IPNet{IP: masked, Mask: net.CIDRMask(48, 128)}).String()
}
