package domain

import (
	"time"

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

func (t SecurityEventToken) Validate() error {
	if err := spec.All(
		spec.Required("JTI", t.JTI),
		spec.Required("Issuer", t.Issuer),
		spec.Required("Audience", t.Audience),
		spec.Required("IssuedAt", t.IssuedAt),
		spec.Required("Compact", t.Compact),
	); err != nil {
		return err
	}
	return t.Event.Validate()
}
