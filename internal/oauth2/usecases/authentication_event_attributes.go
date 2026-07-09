package usecases

// Authentication event の PII-safe 属性を生成する helper (wi-46)。
// emit 経路と検索変換が同じ spec helper を共有し、hash / IP 丸めの実装を分散させない。

import (
	"github.com/ambi/idmagic/internal/shared/spec"
)

// AuthenticationEventAttributes は UserAuthenticated / AuthenticationFailed payload に載せる
// optional な PII-safe 属性。
type AuthenticationEventAttributes struct {
	UsernameHash string
	IPTruncated  string
	IPHash       string
	UAHash       string
}

// BuildAuthenticationEventAttributes は username / client IP / user-agent を payload 用に変換する。
// username は lowercased + trimmed 後に tenant salt 付き hash、IP は /24・/48 丸めと salted hash の
// 両方、user-agent は salted hash のみを返す。空または不正 IP は空値として落とす。
func BuildAuthenticationEventAttributes(
	salt []byte,
	username string,
	clientIP string,
	userAgent string,
) AuthenticationEventAttributes {
	out := AuthenticationEventAttributes{}
	if normalized := spec.NormalizeUsername(username); normalized != "" {
		out.UsernameHash = spec.SaltedHash(salt, normalized)
	}
	if clientIP != "" {
		out.IPTruncated = spec.TruncateIP(clientIP)
		if out.IPTruncated != "" {
			out.IPHash = spec.SaltedHash(salt, clientIP)
		}
	}
	if userAgent != "" {
		out.UAHash = spec.SaltedHash(salt, userAgent)
	}
	return out
}
