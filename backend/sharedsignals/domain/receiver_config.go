package domain

import (
	"strings"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SsfReceiverConfig は direction=Receive の SsfStream に 1 対 1 で付随する、外部
// transmitter の信頼設定。WorkloadTrustBundle と同型の登録済み issuer
// 検証を再利用する。
type SsfReceiverConfig struct {
	StreamID          string
	TrustedIssuer     string
	JWKSURI           *string
	JWKS              map[string]any
	AcceptedAudiences []string
}

var ssfReceiverConfigSchema = z.Struct(z.Shape{
	"StreamID":          z.String().Min(1).Required(),
	"TrustedIssuer":     z.String().Min(1).Required(),
	"AcceptedAudiences": z.Slice(z.String().Min(1)).Min(1).Required(),
})

// Validate は構造的妥当性 (spec/contexts/sharedsignals.yaml SsfReceiverConfig
// constraints) を検証する: trusted_issuer は https、jwks_uri / jwks の少なくとも
// 一方が必須、accepted_audiences は非空。
func (c SsfReceiverConfig) Validate() error {
	if err := spec.Validate(ssfReceiverConfigSchema, &c); err != nil {
		return err
	}
	if !strings.HasPrefix(c.TrustedIssuer, "https://") {
		return errInvalidTrustedIssuer
	}
	if c.JWKSURI == nil && c.JWKS == nil {
		return errMissingJWKSSource
	}
	return nil
}
