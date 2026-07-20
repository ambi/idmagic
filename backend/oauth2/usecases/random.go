package usecases

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateOpaqueToken は base64url(no-padding) で32バイトのランダム値を返す。
// authorization_code 等の opaque token に使用する。
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateOpaqueToken is shared by feature use cases that issue opaque values.
func GenerateOpaqueToken() (string, error) {
	return generateOpaqueToken()
}
