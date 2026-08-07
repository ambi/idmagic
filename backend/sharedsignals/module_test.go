package sharedsignals_test

import (
	"testing"

	sharedsignals "github.com/ambi/idmagic/backend/sharedsignals"
	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	dbpostgres "github.com/ambi/idmagic/backend/sharedsignals/db_postgres"
)

// TestModuleMemoryWiringCompiles は db_memory の各 repository が対応する ports
// インターフェースを満たすことをコンパイル時に確認する。
func TestModuleMemoryWiringCompiles(t *testing.T) {
	_ = sharedsignals.Module{
		RevocationEpochRepo:   dbmemory.NewAgentRevocationEpochRepository(),
		StreamRepo:            dbmemory.NewSsfStreamRepository(),
		TransmitterConfigRepo: dbmemory.NewSsfTransmitterConfigRepository(),
		ReceiverConfigRepo:    dbmemory.NewSsfReceiverConfigRepository(),
		DeliveryRepo:          dbmemory.NewSecurityEventDeliveryRepository(),
		ReceivedEventRepo:     dbmemory.NewReceivedSecurityEventRepository(),
	}
}

// TestModulePostgresWiringCompiles は db_postgres の各 repository が対応する ports
// インターフェースを満たすことをコンパイル時に確認する (Pool は型検査のみで未使用)。
func TestModulePostgresWiringCompiles(t *testing.T) {
	_ = sharedsignals.Module{
		RevocationEpochRepo:   &dbpostgres.AgentRevocationEpochRepository{},
		StreamRepo:            &dbpostgres.SsfStreamRepository{},
		TransmitterConfigRepo: &dbpostgres.SsfTransmitterConfigRepository{},
		ReceiverConfigRepo:    &dbpostgres.SsfReceiverConfigRepository{},
		DeliveryRepo:          &dbpostgres.SecurityEventDeliveryRepository{},
		ReceivedEventRepo:     &dbpostgres.ReceivedSecurityEventRepository{},
	}
}
