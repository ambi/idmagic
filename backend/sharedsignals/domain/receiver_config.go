package domain

import (
	"strings"

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

// Validate は構造的妥当性 (docs/modules/sharedsignals/ SsfReceiverConfig
// constraints) を検証する: trusted_issuer は https、jwks_uri / jwks の少なくとも
// 一方が必須、accepted_audiences は非空。
func (c SsfReceiverConfig) Validate() error {
	if err := spec.All(
		spec.Required("StreamID", c.StreamID),
		spec.Required("TrustedIssuer", c.TrustedIssuer),
		spec.NonEmpty("AcceptedAudiences", c.AcceptedAudiences),
	); err != nil {
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
