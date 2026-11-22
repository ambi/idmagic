// Package signingkeys composes the SigningKeys module.
package signingkeys

import signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

type Module struct {
	KeyStore signingports.KeyStore
}
