package domain

import (
	"strings"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SsfTransmitterConfig は direction=Transmit の SsfStream に 1 対 1 で付随する配送先設定。
type SsfTransmitterConfig struct {
	StreamID              string
	DeliveryEndpoint      string
	Audience              string
	DeliveryAuthorization *string
	MaxDeliveryAttempts   int
}

// Validate は構造的妥当性を検証する: delivery_endpoint は https 必須。
func (c SsfTransmitterConfig) Validate() error {
	if err := spec.All(
		spec.Required("StreamID", c.StreamID),
		spec.Required("DeliveryEndpoint", c.DeliveryEndpoint),
		spec.Required("Audience", c.Audience),
		spec.Required("MaxDeliveryAttempts", c.MaxDeliveryAttempts, spec.GreaterThan(0)),
	); err != nil {
		return err
	}
	if !strings.HasPrefix(c.DeliveryEndpoint, "https://") {
		return errInvalidDeliveryEndpoint
	}
	return nil
}

const DefaultMaxDeliveryAttempts = 8
