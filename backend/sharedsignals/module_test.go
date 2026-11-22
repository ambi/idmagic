package sharedsignals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedsignals "github.com/ambi/idmagic/backend/sharedsignals"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	dbpostgres "github.com/ambi/idmagic/backend/sharedsignals/internal/db_postgres"
)

type unavailableReceiver struct{}

func (unavailableReceiver) Push(context.Context, string, string, string) error {
	return errors.New("receiver unavailable")
}

//spec:covers REQ-SHAREDSIGNALS-006: worker と共通の組み立てで、乱数の指定がなくても保存する再試行時刻にばらつきを付ける。
func TestDeliveryProcessorDefaultsRandomizeRetryTime(t *testing.T) {
	ctx := context.Background()
	m := sharedsignals.NewMemoryModule()
	now := time.Now().UTC()
	const tenant = "tenant-a"
	const streamID = "stream-a"
	if err := m.StreamRepo.Save(ctx, &ssdomain.SsfStream{
		ID: streamID, TenantID: tenant, Direction: ssdomain.SsfStreamDirectionTransmit,
		Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked},
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.TransmitterConfigRepo.Save(ctx, tenant, &ssdomain.SsfTransmitterConfig{
		StreamID: streamID, DeliveryEndpoint: "https://receiver.example/events", MaxDeliveryAttempts: 8,
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.DeliveryRepo.Save(ctx, &ssdomain.SecurityEventDelivery{
		ID: "delivery-a", TenantID: tenant, StreamID: streamID, SetJTI: "jti-a",
		Status: ssdomain.SecurityEventDeliveryStatusPending, CreatedAt: now,
		Set: ssdomain.SecurityEventToken{Compact: "header.payload.sig"},
	}); err != nil {
		t.Fatal(err)
	}
	process := m.DeliveryProcessorWithPusher(unavailableReceiver{}, nil, "https://idp.example", nil)
	allAtLimit := true
	for attempt := range 4 {
		now = now.Add(time.Hour)
		count, err := process(ctx, now, 10)
		if err != nil || count != 1 {
			t.Fatalf("count=%d err=%v", count, err)
		}
		deliveries, err := m.DeliveryRepo.ListByStream(ctx, tenant, streamID)
		if err != nil || len(deliveries) != 1 {
			t.Fatalf("deliveries=%v err=%v", deliveries, err)
		}
		d := deliveries[0]
		if d.NextAttemptAt == nil {
			t.Fatal("retry time is missing")
		}
		delay := d.NextAttemptAt.Sub(now)
		limit := (30 * time.Second) << attempt
		if delay < 0 || delay > limit || d.Status != ssdomain.SecurityEventDeliveryStatusFailed || d.AttemptCount != attempt+1 {
			t.Fatalf("limit=%v delay=%v delivery=%+v", limit, delay, d)
		}
		allAtLimit = allAtLimit && delay == limit
	}
	// 4 回とも上限を選ぶ確率は 10^-43 未満。固定待機の配線を検出する。
	if allAtLimit {
		t.Fatal("default retry always uses the limit")
	}
}

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
