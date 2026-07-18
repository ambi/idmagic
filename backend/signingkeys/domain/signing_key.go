// Package domain owns tenant-scoped signing key lifecycle metadata.
package domain

import (
	"crypto"
	"time"
)

type SignatureAlgorithm string

const (
	SigAlgPS256 SignatureAlgorithm = "PS256"
	SigAlgES256 SignatureAlgorithm = "ES256"
)

func (s SignatureAlgorithm) Valid() bool { return s == SigAlgPS256 || s == SigAlgES256 }

type KeyProvider string

const (
	KeyProviderLocal        KeyProvider = "Local"
	KeyProviderPostgres     KeyProvider = "Postgres"
	KeyProviderVaultTransit KeyProvider = "VaultTransit"
)

func (p KeyProvider) Valid() bool {
	return p == KeyProviderLocal || p == KeyProviderPostgres || p == KeyProviderVaultTransit
}

type KeyUsage string

const KeyUsageSigning KeyUsage = "Signing"

func (u KeyUsage) Valid() bool { return u == KeyUsageSigning }

type SigningKey struct {
	TenantID   string
	Kid        string
	Alg        SignatureAlgorithm
	Provider   KeyProvider
	Usage      KeyUsage
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
	PublicJWK  map[string]any
	Active     bool
	CreatedAt  time.Time
	RetiredAt  *time.Time
	ExpiresAt  *time.Time
	ArchivedAt *time.Time
}

type TenantKeyHealth struct {
	TenantID     string
	Provider     KeyProvider
	Usage        KeyUsage
	ActiveKid    string
	JWKSKeyCount int
	Healthy      bool
}
