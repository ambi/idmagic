// Package sharedsignals composes the SharedSignals bounded context
// (ADR-057, [[wi-58-continuous-access-evaluation-agent-revocation]]).
package sharedsignals

import (
	"github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// Module holds the dependencies SharedSignals's admin API, the token-exchange
// enforcement path (Introspect revocation check), and the transmitter/receiver
// pipeline need. Bootstrap assembles these per persistence backend (memory /
// postgres) and passes the Module through.
type Module struct {
	RevocationEpochRepo   ports.AgentRevocationEpochRepository
	StreamRepo            ports.SsfStreamRepository
	TransmitterConfigRepo ports.SsfTransmitterConfigRepository
	ReceiverConfigRepo    ports.SsfReceiverConfigRepository
	DeliveryRepo          ports.SecurityEventDeliveryRepository
	ReceivedEventRepo     ports.ReceivedSecurityEventRepository
}
