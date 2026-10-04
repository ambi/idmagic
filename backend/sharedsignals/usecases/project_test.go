package usecases_test

import (
	"context"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/sharedsignals/sign_jose"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

func newProjectorDeps(t *testing.T) (ssusecases.ProjectorDeps, *ssmemory.SsfStreamRepository, *ssmemory.SsfTransmitterConfigRepository, *ssmemory.SecurityEventDeliveryRepository) {
	t.Helper()
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("NewInMemoryKeyStore: %v", err)
	}
	streamRepo := ssmemory.NewSsfStreamRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	deps := ssusecases.ProjectorDeps{
		StreamRepo: streamRepo, TransmitterConfigRepo: configRepo, DeliveryRepo: deliveryRepo,
		Signer: &sign_jose.Signer{KeyStore: keyStore}, Issuer: "https://idp.example/tenant-a",
	}
	return deps, streamRepo, configRepo, deliveryRepo
}

const projectTestTenantID = "tenant-a"

func seedTransmitStream(t *testing.T, streamRepo *ssmemory.SsfStreamRepository, configRepo *ssmemory.SsfTransmitterConfigRepository, id string, status ssdomain.SsfStreamStatus, eventTypes []ssdomain.CaepEventType) {
	t.Helper()
	ctx := context.Background()
	if err := streamRepo.Save(ctx, &ssdomain.SsfStream{
		ID: id, TenantID: projectTestTenantID, Direction: ssdomain.SsfStreamDirectionTransmit,
		EventTypes: eventTypes, Status: status, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed stream %s: %v", id, err)
	}
	if err := configRepo.Save(ctx, projectTestTenantID, &ssdomain.SsfTransmitterConfig{
		StreamID: id, DeliveryEndpoint: "https://receiver.example/" + id, Audience: "https://receiver.example/" + id,
		MaxDeliveryAttempts: ssdomain.DefaultMaxDeliveryAttempts,
	}); err != nil {
		t.Fatalf("seed transmitter config %s: %v", id, err)
	}
}

// TestProjectAgentAccessRevoked_EnqueuesForSubscribedEnabledStream — RED:
// session-revoked を購読する有効な Transmit stream に pending delivery を1件
// enqueue する (EcosystemPropagation)。
func TestProjectAgentAccessRevoked_EnqueuesForSubscribedEnabledStream(t *testing.T) {
	ctx := context.Background()
	deps, streamRepo, configRepo, deliveryRepo := newProjectorDeps(t)
	seedTransmitStream(t, streamRepo, configRepo, "stream_1", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})

	event := &ssdomain.AgentAccessRevoked{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1", Reason: ssdomain.RevocationReasonAgentKilled}
	if err := ssusecases.ProjectAgentAccessRevoked(ctx, deps, event); err != nil {
		t.Fatalf("ProjectAgentAccessRevoked: %v", err)
	}

	deliveries, err := deliveryRepo.ListByStream(ctx, "tenant-a", "stream_1")
	if err != nil {
		t.Fatalf("ListByStream: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d: %+v", len(deliveries), deliveries)
	}
	d := deliveries[0]
	if d.Status != ssdomain.SecurityEventDeliveryStatusPending {
		t.Fatalf("expected pending status, got %+v", d)
	}
	if d.Set.Compact == "" || d.SetJTI != d.Set.JTI {
		t.Fatalf("expected a signed SET attached to the delivery: %+v", d)
	}
	if d.Set.Event.Subject.PrincipalID != "agent_1" || *d.Set.Event.Reason != ssdomain.RevocationReasonAgentKilled {
		t.Fatalf("unexpected CAEP event content: %+v", d.Set.Event)
	}
}

// TestProjectAgentAccessRevoked_SkipsDisabledUnsubscribedAndReceiveStreams —
// RED: 無効化された stream・session-revoked を購読していない stream・
// direction=Receive の stream には delivery を作らない。
func TestProjectAgentAccessRevoked_SkipsDisabledUnsubscribedAndReceiveStreams(t *testing.T) {
	ctx := context.Background()
	deps, streamRepo, configRepo, deliveryRepo := newProjectorDeps(t)
	seedTransmitStream(t, streamRepo, configRepo, "disabled_stream", ssdomain.SsfStreamStatusDisabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})
	seedTransmitStream(t, streamRepo, configRepo, "unsubscribed_stream", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeCredentialChange})
	if err := streamRepo.Save(ctx, &ssdomain.SsfStream{
		ID: "receive_stream", TenantID: "tenant-a", Direction: ssdomain.SsfStreamDirectionReceive,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed receive stream: %v", err)
	}

	event := &ssdomain.AgentAccessRevoked{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1", Reason: ssdomain.RevocationReasonAgentKilled}
	if err := ssusecases.ProjectAgentAccessRevoked(ctx, deps, event); err != nil {
		t.Fatalf("ProjectAgentAccessRevoked: %v", err)
	}

	for _, streamID := range []string{"disabled_stream", "unsubscribed_stream", "receive_stream"} {
		deliveries, err := deliveryRepo.ListByStream(ctx, "tenant-a", streamID)
		if err != nil {
			t.Fatalf("ListByStream(%s): %v", streamID, err)
		}
		if len(deliveries) != 0 {
			t.Fatalf("%s: expected no delivery, got %+v", streamID, deliveries)
		}
	}
}

// TestProjectAgentAccessRevoked_SkipsStreamWithoutTransmitterConfig — RED:
// SsfTransmitterConfig を持たない stream (整合性が崩れたデータ) は no-op で
// スキップする (エラーにはしない)。
func TestProjectAgentAccessRevoked_SkipsStreamWithoutTransmitterConfig(t *testing.T) {
	ctx := context.Background()
	deps, streamRepo, _, deliveryRepo := newProjectorDeps(t)
	if err := streamRepo.Save(ctx, &ssdomain.SsfStream{
		ID: "orphan_stream", TenantID: "tenant-a", Direction: ssdomain.SsfStreamDirectionTransmit,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed orphan stream: %v", err)
	}

	event := &ssdomain.AgentAccessRevoked{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1", Reason: ssdomain.RevocationReasonAgentKilled}
	if err := ssusecases.ProjectAgentAccessRevoked(ctx, deps, event); err != nil {
		t.Fatalf("ProjectAgentAccessRevoked: %v", err)
	}
	deliveries, err := deliveryRepo.ListByStream(ctx, "tenant-a", "orphan_stream")
	if err != nil {
		t.Fatalf("ListByStream: %v", err)
	}
	if len(deliveries) != 0 {
		t.Fatalf("expected no delivery for a stream without transmitter config, got %+v", deliveries)
	}
}

// TestProjectAgentAccessRevoked_NilStreamRepoIsNoop — RED: StreamRepo が nil の
// lightweight wiring (SharedSignals.Module 未配線のテスト等) では panic せず
// no-op で返す。composition root は AgentRevocationReactor.Emit から常に
// ProjectAgentAccessRevoked を呼ぶため、この nil-skip を projector 自身が
// 保証する必要がある。
func TestProjectAgentAccessRevoked_NilStreamRepoIsNoop(t *testing.T) {
	ctx := context.Background()
	event := &ssdomain.AgentAccessRevoked{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1", Reason: ssdomain.RevocationReasonAgentKilled}
	if err := ssusecases.ProjectAgentAccessRevoked(ctx, ssusecases.ProjectorDeps{}, event); err != nil {
		t.Fatalf("ProjectAgentAccessRevoked (nil StreamRepo): %v", err)
	}
}

// TestProjectAgentAccessRevoked_FansOutToMultipleStreams — RED: 複数の適格
// stream があれば、それぞれに1件ずつ delivery を作る。
func TestProjectAgentAccessRevoked_FansOutToMultipleStreams(t *testing.T) {
	ctx := context.Background()
	deps, streamRepo, configRepo, deliveryRepo := newProjectorDeps(t)
	seedTransmitStream(t, streamRepo, configRepo, "stream_1", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})
	seedTransmitStream(t, streamRepo, configRepo, "stream_2", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked, ssdomain.CaepEventTypeCredentialChange})

	event := &ssdomain.AgentAccessRevoked{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1", Reason: ssdomain.RevocationReasonOwnerDeleted}
	if err := ssusecases.ProjectAgentAccessRevoked(ctx, deps, event); err != nil {
		t.Fatalf("ProjectAgentAccessRevoked: %v", err)
	}
	for _, streamID := range []string{"stream_1", "stream_2"} {
		deliveries, err := deliveryRepo.ListByStream(ctx, "tenant-a", streamID)
		if err != nil {
			t.Fatalf("ListByStream(%s): %v", streamID, err)
		}
		if len(deliveries) != 1 {
			t.Fatalf("%s: expected 1 delivery, got %d", streamID, len(deliveries))
		}
	}
}

// NewAgentRevocationReactor は、派生イベントを記録し、AgentAccessRevoked だけを外部への伝播へ投影する。
// 投影の失敗はローカルの失効を失敗させない。
//
//spec:covers REQ-SHAREDSIGNALS-007: 組み立てた反応器が AgentKilled で失効エポックを進め、RevocationEpochAdvanced と AgentAccessRevoked を記録し、購読する Transmit ストリームへ pending の配送を 1 件作ること、投影が失敗しても React が成功しエポックが進むことを固定する。
func TestNewAgentRevocationReactor_RecordsAndProjectsTheRevocation(t *testing.T) {
	event := &idmdomain.AgentKilled{At: time.Now().UTC(), TenantID: projectTestTenantID, AgentID: "agent_1"}

	t.Run("Projects", func(t *testing.T) {
		projector, streamRepo, configRepo, deliveryRepo := newProjectorDeps(t)
		seedTransmitStream(t, streamRepo, configRepo, "stream_1", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})
		var recorded []string
		reactor := ssusecases.NewAgentRevocationReactor(ssusecases.RevocationReactorDeps{
			EpochRepo: ssmemory.NewAgentRevocationEpochRepository(), Projector: projector,
			Emit: func(e spec.DomainEvent) { recorded = append(recorded, e.EventType()) },
		})
		ctx := context.Background()
		if err := reactor.React(ctx, event); err != nil {
			t.Fatalf("React: %v", err)
		}
		if len(recorded) != 2 || recorded[0] != "RevocationEpochAdvanced" || recorded[1] != "AgentAccessRevoked" {
			t.Fatalf("recorded = %v, want [RevocationEpochAdvanced AgentAccessRevoked]", recorded)
		}
		deliveries, err := deliveryRepo.ListByStream(ctx, projectTestTenantID, "stream_1")
		if err != nil || len(deliveries) != 1 {
			t.Fatalf("deliveries = %+v, %v; want exactly one for AgentAccessRevoked", deliveries, err)
		}
	})

	t.Run("ProjectionFailureDoesNotFailTheRevocation", func(t *testing.T) {
		projector, streamRepo, configRepo, _ := newProjectorDeps(t)
		seedTransmitStream(t, streamRepo, configRepo, "stream_1", ssdomain.SsfStreamStatusEnabled, []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked})
		projector.Issuer = ""
		epochs := ssmemory.NewAgentRevocationEpochRepository()
		reactor := ssusecases.NewAgentRevocationReactor(ssusecases.RevocationReactorDeps{EpochRepo: epochs, Projector: projector})
		ctx := context.Background()
		if err := reactor.React(ctx, event); err != nil {
			t.Fatalf("React: %v, want the local revocation to succeed despite the projection failure", err)
		}
		if epoch, err := epochs.FindByAgent(ctx, projectTestTenantID, "agent_1"); err != nil || epoch == nil {
			t.Fatalf("epoch = %+v, %v; want advanced", epoch, err)
		}
	})
}
