// Package domain owns tenant-scoped signing key lifecycle metadata.
package domain

import (
	"crypto"
	"errors"
	"time"
)

var ErrActiveSigningKeyCannotBeDisabled = errors.New("active signing key cannot be disabled")

type SignatureAlgorithm string

const (
	SigAlgPS256 SignatureAlgorithm = "PS256"
	SigAlgES256 SignatureAlgorithm = "ES256"
	// SigAlgRS256 は ID-JAG の専用鍵だけが使う (REQ-SIGNINGKEYS-013)。クライアントが選べる
	// ID Token の署名方式ではないので、Valid には含めない。
	SigAlgRS256 SignatureAlgorithm = "RS256"
)

func (s SignatureAlgorithm) Valid() bool { return s == SigAlgPS256 || s == SigAlgES256 }

type KeyProvider string

const (
	KeyProviderLocal        KeyProvider = "Local"
	KeyProviderDatabase     KeyProvider = "Database"
	KeyProviderVaultTransit KeyProvider = "VaultTransit"
)

func (p KeyProvider) Valid() bool {
	return p == KeyProviderLocal || p == KeyProviderDatabase || p == KeyProviderVaultTransit
}

type KeyUsage string

const (
	KeyUsageSigning              KeyUsage = "Signing"
	KeyUsageXMLFederationSigning KeyUsage = "XmlFederationSigning"
	// KeyUsageIdJagSigning は ID-JAG の署名に使う。
	KeyUsageIdJagSigning KeyUsage = "IdJagSigning"
)

func (u KeyUsage) Valid() bool {
	return u == KeyUsageSigning || u == KeyUsageXMLFederationSigning || u == KeyUsageIdJagSigning
}

// AlgorithmFor は用途の鍵が宣言する署名方式を返す。
func AlgorithmFor(usage KeyUsage) SignatureAlgorithm {
	if usage == KeyUsageIdJagSigning {
		return SigAlgRS256
	}
	return SigAlgPS256
}

type SigningKey struct {
	TenantID   string
	Kid        string
	Alg        SignatureAlgorithm
	Provider   KeyProvider
	Usage      KeyUsage
	ScopeID    string
	PrivateKey crypto.PrivateKey `json:"-"`
	PublicKey  crypto.PublicKey  `json:"-"`
	PublicJWK  map[string]any
	// CertificateDER is required for XmlFederationSigning and public metadata only.
	CertificateDER []byte
	Active         bool
	Prepared       bool
	CreatedAt      time.Time
	RetiredAt      *time.Time
	ExpiresAt      *time.Time
	ArchivedAt     *time.Time
}

type RotationResult struct {
	Next, Previous *SigningKey
	Changed        bool
}

var (
	ErrPrepublicationRequired = errors.New("publish the XML key before activating it")
	ErrKeyNotPrepared         = errors.New("the XML key is not prepared")
)

func RequiresPrepublication(usage KeyUsage, scope string) bool {
	return usage == KeyUsageXMLFederationSigning && scope == "default"
}

func ActiveKey(keys []*SigningKey) *SigningKey {
	for _, key := range keys {
		if key.Active && key.ArchivedAt == nil {
			return key
		}
	}
	return nil
}

func PreparedKey(keys []*SigningKey) *SigningKey {
	for _, key := range keys {
		if key.Prepared && key.ArchivedAt == nil {
			return key
		}
	}
	return nil
}

// PreparationDue は、XML 鍵の集合に定期実行で次の鍵を準備すべきかを返す。有効な鍵が作成から
// cadence に達し、まだ準備済みの鍵がない集合だけが対象になる。
func PreparationDue(keys []*SigningKey, now time.Time, cadence time.Duration) bool {
	active := ActiveKey(keys)
	return active != nil && PreparedKey(keys) == nil && !now.Before(active.CreatedAt.Add(cadence))
}

// ActivatePrepared は鍵集合の所有者の直列化の内側で呼ぶ。変更する要素は複製し、
// 切り替え前に読み出した鍵の値を後から変更しない。
func ActivatePrepared(keys []*SigningKey, kid string, now time.Time, grace time.Duration) (RotationResult, error) {
	active := ActiveKey(keys)
	if active != nil && active.Kid == kid {
		return RotationResult{Next: active}, nil
	}
	prepared := PreparedKey(keys)
	if prepared == nil || prepared.Kid != kid || grace < 0 {
		return RotationResult{}, ErrKeyNotPrepared
	}
	next := *prepared
	next.Prepared, next.Active = false, true
	var previous *SigningKey
	for i, key := range keys {
		if key.Active {
			retired := *key
			retired.Active = false
			retired.RetiredAt = &now
			expires := now.Add(grace)
			retired.ExpiresAt = &expires
			keys[i], previous = &retired, &retired
		}
		if key.Kid == kid {
			keys[i] = &next
		}
	}
	return RotationResult{Next: &next, Previous: previous, Changed: true}, nil
}

type TenantKeyHealth struct {
	TenantID     string
	Provider     KeyProvider
	Usage        KeyUsage
	ActiveKid    string
	JWKSKeyCount int
	Healthy      bool
}
