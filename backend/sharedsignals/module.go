// Package sharedsignals composes the SharedSignals module
// ([[wi-58-continuous-access-evaluation-agent-revocation]]).
package sharedsignals

import (
	"context"

	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// Module holds the dependencies SharedSignals's admin API, the token-exchange
// enforcement path (Introspect revocation check), and the transmitter/receiver
// pipeline need. Bootstrap assembles these per persistence backend (memory /
// postgres) and passes the Module through.
type Module struct {
	RevocationEpochRepo     ssports.AgentRevocationEpochRepository
	UserRevocationEpochRepo ssports.UserRevocationEpochRepository
	StreamRepo              ssports.SsfStreamRepository
	TransmitterConfigRepo   ssports.SsfTransmitterConfigRepository
	ReceiverConfigRepo      ssports.SsfReceiverConfigRepository
	DeliveryRepo            ssports.SecurityEventDeliveryRepository
	ReceivedEventRepo       ssports.ReceivedSecurityEventRepository
	// CheckDeliveryEndpoint は登録時に配送先の解決先を検査する。nil なら字句の検査だけを行う。
	// 解決を伴うので、組み立て地点が WithDeliveryEndpointCheck で有効にする。
	CheckDeliveryEndpoint func(ctx context.Context, endpoint string) error
}
