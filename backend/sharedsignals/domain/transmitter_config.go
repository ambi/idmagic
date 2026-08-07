package domain

import (
	"strings"

	z "github.com/Oudwins/zog"

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

var ssfTransmitterConfigSchema = z.Struct(z.Shape{
	"StreamID":            z.String().Min(1).Required(),
	"DeliveryEndpoint":    z.String().Min(1).Required(),
	"Audience":            z.String().Min(1).Required(),
	"MaxDeliveryAttempts": z.Int().GT(0).Required(),
})

// Validate は構造的妥当性を検証する: delivery_endpoint は https 必須。
func (c SsfTransmitterConfig) Validate() error {
	if err := spec.Validate(ssfTransmitterConfigSchema, &c); err != nil {
		return err
	}
	if !strings.HasPrefix(c.DeliveryEndpoint, "https://") {
		return errInvalidDeliveryEndpoint
	}
	return nil
}

const DefaultMaxDeliveryAttempts = 8
