package bootstrap

import "crypto/rand"

// LoadPaginationCursorSecret returns the HMAC secret used to sign keyset
// pagination cursors. Set PAGINATION_CURSOR_SECRET explicitly in
// any multi-replica deployment: without it, each replica generates its own
// random secret at startup, so a cursor issued by one replica is rejected
// (InvalidRequestError) by another.
func LoadPaginationCursorSecret() []byte {
	if v := EnvDefault("PAGINATION_CURSOR_SECRET", ""); v != "" {
		return []byte(v)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("bootstrap: failed to generate pagination cursor secret: " + err.Error())
	}
	return secret
}
