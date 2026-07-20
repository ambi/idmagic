package domain

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"
)

// ClientSecretCredential は client_secret の保存単位。平文は一切保持しない。
// ExpiresAt は rotation overlap の終了、RevokedAt は fail-close の即時失効を表す。
type ClientSecretCredential struct {
	CredentialID string     `json:"credential_id"`
	ClientID     string     `json:"client_id"`
	SecretHash   string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

func (c ClientSecretCredential) IsActiveAt(now time.Time) bool {
	return c.RevokedAt == nil && (c.ExpiresAt == nil || now.Before(*c.ExpiresAt))
}

func HashClientSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func VerifyClientSecret(secret, encodedHash string) bool {
	actual := HashClientSecret(secret)
	if len(actual) != len(encodedHash) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(encodedHash)) == 1
}
