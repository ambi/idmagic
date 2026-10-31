package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// TokenExchangeGrantTTL は交換の許可の有効期間で、再束縛したトークンの有効期間の上限でもある。
const TokenExchangeGrantTTL = 60 * time.Second

// TokenExchangeGrant は評価が発行した一回限りの再束縛の根拠である (REQ-OAUTH2-079)。
type TokenExchangeGrant struct {
	ClientID       string
	SubjectTokenID string
	Resource       string
	ExpiresAt      time.Time
}

// HashTokenExchangeGrant は保存と照合に使う値を返す。平文の許可を保存しない。
func HashTokenExchangeGrant(grant string) string {
	sum := sha256.Sum256([]byte(grant))
	return hex.EncodeToString(sum[:])
}
