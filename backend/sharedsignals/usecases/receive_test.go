package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentmodel "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// fakeVerifier scripts a Verify result/error for ReceiveSecurityEvent tests.
type fakeVerifier struct {
	result *ssports.VerifiedSecurityEvent
	err    error
}

func (f *fakeVerifier) Verify(context.Context, *ssdomain.SsfReceiverConfig, string) (*ssports.VerifiedSecurityEvent, error) {
	return f.result, f.err
}

func validAgentEvent(jti, tenantID, agentID string) *ssports.VerifiedSecurityEvent {
	return &ssports.VerifiedSecurityEvent{
		JTI: jti, Issuer: "https://transmitter.example", Audience: []string{"aud"}, IssuedAt: time.Now().UTC(),
		Events: map[string]any{
			"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{
				"subject": map[string]any{"subject_type": "Agent", "tenant_id": tenantID, "principal_id": agentID},
			},
		},
	}
}

type receiveTestDeps struct {
	deps         ssusecases.ReceiveDeps
	streamRepo   *ssmemory.SsfStreamRepository
	configRepo   *ssmemory.SsfReceiverConfigRepository
	receivedRepo *ssmemory.ReceivedSecurityEventRepository
	epochRepo    *ssmemory.AgentRevocationEpochRepository
	agentRepo    *agentmemory.AgentRepository
	events       *[]spec.DomainEvent
}

func newReceiveTestDeps(t *testing.T, verifier ssports.SecurityEventTokenVerifier) receiveTestDeps {
	t.Helper()
	events := &[]spec.DomainEvent{}
	d := receiveTestDeps{
		streamRepo: ssmemory.NewSsfStreamRepository(), configRepo: ssmemory.NewSsfReceiverConfigRepository(),
		receivedRepo: ssmemory.NewReceivedSecurityEventRepository(), epochRepo: ssmemory.NewAgentRevocationEpochRepository(),
		agentRepo: agentmemory.NewAgentRepository(), events: events,
	}
	d.deps = ssusecases.ReceiveDeps{
		StreamRepo: d.streamRepo, ReceiverConfigRepo: d.configRepo, ReceivedEventRepo: d.receivedRepo,
		EpochRepo: d.epochRepo, AgentRepo: d.agentRepo, Verifier: verifier,
		Emit: func(e spec.DomainEvent) error { *events = append(*events, e); return nil },
	}
	return d
}

const (
	receiveTestTenantID = "tenant-a"
	receiveTestStreamID = "stream-1"
)

func seedReceiveStream(t *testing.T, d receiveTestDeps, status ssdomain.SsfStreamStatus) {
	t.Helper()
	ctx := context.Background()
	if err := d.streamRepo.Save(ctx, &ssdomain.SsfStream{
		ID: receiveTestStreamID, TenantID: receiveTestTenantID, Direction: ssdomain.SsfStreamDirectionReceive,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, Status: status, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	jwksURI := "https://transmitter.example/jwks"
	if err := d.configRepo.Save(ctx, receiveTestTenantID, &ssdomain.SsfReceiverConfig{
		StreamID: receiveTestStreamID, TrustedIssuer: "https://transmitter.example", JWKSURI: &jwksURI, AcceptedAudiences: []string{"aud"},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func receiveTestCtx() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: receiveTestTenantID}, "", "")
}

// TestReceiveSecurityEvent_RejectsDisabledOrMissingStream — RED: stream が
// 存在しない/Receive でない/無効なら ErrSecurityEventRejected、検証は呼ばれない。
func TestReceiveSecurityEvent_RejectsDisabledOrMissingStream(t *testing.T) {
	verifier := &fakeVerifier{}
	d := newReceiveTestDeps(t, verifier)
	ctx := receiveTestCtx()
	now := time.Now().UTC()

	if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, "no-such-stream", "token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
		t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
	}

	seedReceiveStream(t, d, ssdomain.SsfStreamStatusDisabled)
	if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
		t.Fatalf("expected ErrSecurityEventRejected for disabled stream, got %v", err)
	}
}

// TestReceiveSecurityEvent_RejectsOnVerificationFailure — RED: 署名/iss/aud 検証
// 失敗は ErrSecurityEventRejected を返し、対応する VerificationResult で
// ReceivedSecurityEvent を記録し SecurityEventRejected を emit する。
func TestReceiveSecurityEvent_RejectsOnVerificationFailure(t *testing.T) {
	verifier := &fakeVerifier{err: ssports.ErrSecurityEventIssuerMismatch}
	d := newReceiveTestDeps(t, verifier)
	ctx := receiveTestCtx()
	now := time.Now().UTC()
	seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)

	if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "bad-token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
		t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
	}
	if len(*d.events) != 1 {
		t.Fatalf("expected exactly one emitted event, got %+v", *d.events)
	}
	rejected, ok := (*d.events)[0].(*ssdomain.SecurityEventRejected)
	if !ok || rejected.VerificationResult != ssdomain.SecurityEventVerificationRejectedUnknownIssuer {
		t.Fatalf("expected SecurityEventRejected(rejected_unknown_issuer), got %+v", (*d.events)[0])
	}
}

// TestReceiveSecurityEvent_RejectsReplay — RED: 既に受理済みの jti は
// rejected_replay として拒否する。
func TestReceiveSecurityEvent_RejectsReplay(t *testing.T) {
	verified := validAgentEvent("jti-1", receiveTestTenantID, "agent_1")
	verifier := &fakeVerifier{result: verified}
	d := newReceiveTestDeps(t, verifier)
	ctx := receiveTestCtx()
	now := time.Now().UTC()
	seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
	if err := d.receivedRepo.Save(context.Background(), &ssdomain.ReceivedSecurityEvent{
		ID: "existing", TenantID: receiveTestTenantID, StreamID: receiveTestStreamID, SetJTI: "jti-1", EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject:            ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: receiveTestTenantID, PrincipalID: "agent_1"},
		VerificationResult: ssdomain.SecurityEventVerificationAccepted, ReceivedAt: now,
	}); err != nil {
		t.Fatalf("seed prior received event: %v", err)
	}

	if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
		t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
	}
	rejected, ok := (*d.events)[len(*d.events)-1].(*ssdomain.SecurityEventRejected)
	if !ok || rejected.VerificationResult != ssdomain.SecurityEventVerificationRejectedReplay {
		t.Fatalf("expected rejected_replay, got %+v", (*d.events)[len(*d.events)-1])
	}
}

// TestReceiveSecurityEvent_RejectsUnresolvedSubject — RED: subject_type が
// Agent 以外、cross-tenant、または Agent が存在しない場合は
// rejected_subject_unresolved。
func TestReceiveSecurityEvent_RejectsUnresolvedSubject(t *testing.T) {
	t.Run("unknown agent", func(t *testing.T) {
		verified := validAgentEvent("jti-unknown", receiveTestTenantID, "no-such-agent")
		verifier := &fakeVerifier{result: verified}
		d := newReceiveTestDeps(t, verifier)
		ctx := receiveTestCtx()
		now := time.Now().UTC()
		seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)

		if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
			t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
		}
		rejected := (*d.events)[len(*d.events)-1].(*ssdomain.SecurityEventRejected)
		if rejected.VerificationResult != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
			t.Fatalf("expected rejected_subject_unresolved, got %+v", rejected)
		}
	})

	t.Run("cross-tenant subject", func(t *testing.T) {
		verified := validAgentEvent("jti-cross", "tenant-b", "agent_1")
		verifier := &fakeVerifier{result: verified}
		d := newReceiveTestDeps(t, verifier)
		ctx := receiveTestCtx()
		now := time.Now().UTC()
		seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)

		if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "token", now); !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
			t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
		}
	})
}

// TestReceiveSecurityEvent_AcceptsAndReflectsLocalRevocation — RED: 検証を通過した
// イベントは ReceivedSecurityEvent(accepted) を記録し、AdvanceRevocationEpoch で
// LocalRevocation へ反映し、SecurityEventReceived を emit する。
func TestReceiveSecurityEvent_AcceptsAndReflectsLocalRevocation(t *testing.T) {
	verified := validAgentEvent("jti-ok", receiveTestTenantID, "agent_1")
	verifier := &fakeVerifier{result: verified}
	d := newReceiveTestDeps(t, verifier)
	ctx := receiveTestCtx()
	now := time.Now().UTC()
	seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
	if err := d.agentRepo.Save(context.Background(), &agentmodel.Agent{
		ID: "agent_1", TenantID: receiveTestTenantID, Name: "agent_1", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "user_1", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	if err := ssusecases.ReceiveSecurityEvent(ctx, d.deps, receiveTestStreamID, "token", now); err != nil {
		t.Fatalf("ReceiveSecurityEvent: %v", err)
	}

	replayed, err := d.receivedRepo.ExistsByJTI(context.Background(), receiveTestTenantID, receiveTestStreamID, "jti-ok")
	if err != nil || !replayed {
		t.Fatalf("expected the accepted event recorded for replay dedup, exists=%v err=%v", replayed, err)
	}
	epoch, err := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, "agent_1")
	if err != nil || epoch == nil || epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
		t.Fatalf("expected epoch advanced with reason=InboundSecurityEvent, got %+v err=%v", epoch, err)
	}
	last := (*d.events)[len(*d.events)-1]
	if last.EventType() != "SecurityEventReceived" {
		t.Fatalf("expected the last emitted event to be SecurityEventReceived, got %s", last.EventType())
	}
}
