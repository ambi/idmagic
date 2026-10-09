// Package signingkeys composes the SigningKeys module.
package signingkeys

import "github.com/ambi/idmagic/backend/signingkeys/ports"

type Module struct {
	KeyStore ports.KeyStore
}
