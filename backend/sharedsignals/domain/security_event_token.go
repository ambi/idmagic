package domain

import (
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SecurityEventToken は RFC 8417 の Security Event Token。署名済み compact JWT と、
// そのデコード済み標準クレーム。
type SecurityEventToken struct {
	JTI      string
	Issuer   string
	Audience string
	IssuedAt time.Time
	Event    CaepEvent
	Compact  string
}

var securityEventTokenSchema = z.Struct(z.Shape{
	"JTI":      z.String().Min(1).Required(),
	"Issuer":   z.String().Min(1).Required(),
	"Audience": z.String().Min(1).Required(),
	"IssuedAt": z.Time().Required(),
	"Compact":  z.String().Min(1).Required(),
})

func (t SecurityEventToken) Validate() error {
	if err := spec.Validate(securityEventTokenSchema, &t); err != nil {
		return err
	}
	return t.Event.Validate()
}
