package spec

import (
	"crypto/sha256"
	"encoding/hex"
)

// SaltedHash は tenant salt 付き SHA-256 を hex 文字列で返す (hex(sha256(salt || value)))。
// salt を前置することで同一値でも tenant が異なれば hash が異なり、cross-tenant 相関を防ぐ。
// throttle / bucket の keyHash (ADR-029) が使う。監査検索の username/IP 相関には使わない
// (ADR-104: ADR-046 の username/IP 条項を撤回し、平文のまま扱う)。
func SaltedHash(salt []byte, value string) string {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}
