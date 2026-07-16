// Package signingkeys composes the SigningKeys bounded context.
package signingkeys

import "github.com/ambi/idmagic/backend/signingkeys/ports"

type Module struct {
	KeyStore ports.KeyStore
}
