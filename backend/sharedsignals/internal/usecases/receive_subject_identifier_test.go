package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmodel "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

const caepSessionRevokedURI = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"

// subjectIdentifierEvent wraps an RFC 9493 Subject Identifier as the sole
// `subject` member of the sole CAEP event claim.
func subjectIdentifierEvent(jti string, subjectID map[string]any) *ssports.VerifiedSecurityEvent {
	return &ssports.VerifiedSecurityEvent{
		JTI: jti, Issuer: "https://transmitter.example", Audience: []string{"aud"}, IssuedAt: time.Now().UTC(),
		Events: map[string]any{caepSessionRevokedURI: map[string]any{"subject": subjectID}},
	}
}

// TestReceiveSecurityEvent_Rfc9493SubjectIdentifiers — RED: REQ-SHAREDSIGNALS-010
// (docs/modules/sharedsignals/features/receiver.md)。外部の transmitter が RFC 9493 の
// Subject Identifier (`iss_sub` / `opaque`) で送った SET でも主体を解決する。
// テナントは受信ストリームが属するテナントで決まり、識別子は Agent の識別子か、
// Agent に束縛済みの OAuth2Client の識別子として解決する。
//
// 解釈する `iss_sub` と `opaque` が主体を解決すること、解釈しない `format` は
// 解決できる識別子を同じ Subject Identifier の中に持っていても拒否されること、
// そして拒否が失効エポックを進めないことを併せて観測する。
// `trusted_issuer` と完全一致しない場合、SET 自体の検証を通っていても主体を解決
// せず、失効エポックを進めないことを固定する。
//
//spec:covers RFC9493-SUBID-FORMAT: 種別を決めるのが `format` の値だけであることを固定する。
//spec:covers RFC9493-SUBID-ISS-SUB: `format=iss_sub` の `iss` が受信ストリームの `trusted_issuer` と一致しなければ主体を解決せず、失効エポックを進めないことを固定する。
//spec:covers EX-SHAREDSIGNALS-010-01: `iss_sub` で Agent を名指した SET が、その Agent の失効エポックを InboundSecurityEvent で進め、SecurityEventReceived を発行する。
//spec:covers EX-SHAREDSIGNALS-010-02: `sub` が束縛先クライアントの識別子でも、束縛元の Agent の失効エポックが進む。
//spec:covers EX-SHAREDSIGNALS-010-03: 自身の発行者もテナントも持たない `opaque` の `id` が、受信ストリームのテナントの Agent として解決される。
//spec:covers EX-SHAREDSIGNALS-010-04: `iss` が `trusted_issuer` と一致しない SET を ErrSecurityEventRejected で拒否し、rejected_subject_unresolved を発行し、失効エポックを進めない。
//spec:covers EX-SHAREDSIGNALS-010-05: 解釈しない `format=email` を、解決できる識別子を同居させても拒否し、失効エポックを進めない。
//spec:covers EX-SHAREDSIGNALS-010-06: どの Agent にも束縛先クライアントにも一致しない識別子を拒否し、失効エポックを進めない。
func TestReceiveSecurityEvent_Rfc9493SubjectIdentifiers(t *testing.T) {
	const agentID = "agent_1"
	const boundClientID = "client_1"

	seed := func(t *testing.T, verified *ssports.VerifiedSecurityEvent) receiveTestDeps {
		t.Helper()
		d := newReceiveTestDeps(t, &fakeVerifier{result: verified})
		seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
		now := time.Now().UTC()
		seedReceiveAgent(t, d, agentID)
		if _, err := d.agentRepo.AddBinding(context.Background(), &agentmodel.AgentCredentialBinding{
			AgentID: agentID, ClientID: boundClientID, CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
		return d
	}

	assertAccepted := func(t *testing.T, d receiveTestDeps) {
		t.Helper()
		epoch, err := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, agentID)
		if err != nil {
			t.Fatal(err)
		}
		if epoch == nil {
			t.Fatal("expected the agent's revocation epoch to advance")
		}
		if epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
			t.Fatalf("reason = %q, want %q", epoch.Reason, ssdomain.RevocationReasonInboundSecurityEvent)
		}
		var received bool
		for _, event := range *d.events {
			if _, ok := event.(*ssdomain.SecurityEventReceived); ok {
				received = true
			}
		}
		if !received {
			t.Fatalf("expected SecurityEventReceived, got %+v", *d.events)
		}
	}

	accepted := []struct {
		name      string
		subjectID map[string]any
	}{
		{
			name:      "iss_sub naming the agent",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://transmitter.example", "sub": agentID},
		},
		{
			name:      "iss_sub naming the bound client",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://transmitter.example", "sub": boundClientID},
		},
		{
			name:      "opaque naming the agent",
			subjectID: map[string]any{"format": "opaque", "id": agentID},
		},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			d := seed(t, subjectIdentifierEvent("jti-"+tc.name, tc.subjectID))
			if err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now); err != nil {
				t.Fatalf("expected the SET to be accepted: %v", err)
			}
			assertAccepted(t, d)
		})
	}

	rejected := []struct {
		name      string
		subjectID map[string]any
	}{
		{
			name:      "iss_sub whose iss is not the stream's trusted issuer",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://elsewhere.example", "sub": agentID},
		},
		{
			name:      "a format idmagic does not interpret",
			subjectID: map[string]any{"format": "email", "email": "owner@example.com"},
		},
		{
			// 解決可能な `id` を同居させても、`format` が解釈対象でなければ拒否する。
			// メンバーの有無で種別を当てにいく実装なら、ここで主体が解決してしまう。
			name:      "an uninterpreted format carrying a resolvable identifier anyway",
			subjectID: map[string]any{"format": "email", "email": "owner@example.com", "id": agentID, "sub": agentID},
		},
		{
			name:      "an identifier matching no agent and no bound client",
			subjectID: map[string]any{"format": "opaque", "id": "unknown"},
		},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			d := seed(t, subjectIdentifierEvent("jti-"+tc.name, tc.subjectID))
			err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now)
			if !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
				t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
			}
			epoch, findErr := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, agentID)
			if findErr != nil {
				t.Fatal(findErr)
			}
			if epoch != nil {
				t.Fatal("expected no revocation epoch to advance for a rejected SET")
			}
			var rejectedEvent *ssdomain.SecurityEventRejected
			for _, event := range *d.events {
				if e, ok := event.(*ssdomain.SecurityEventRejected); ok {
					rejectedEvent = e
				}
			}
			if rejectedEvent == nil || rejectedEvent.VerificationResult != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
				t.Fatalf("expected SecurityEventRejected(rejected_subject_unresolved), got %+v", *d.events)
			}
		})
	}

	// idmagic 自身の transmitter が使う独自形式は引き続き受理する (共存)。
	t.Run("idmagic's own wire format still resolves", func(t *testing.T) {
		now := time.Now().UTC()
		d := seed(t, validAgentEvent("jti-native", receiveTestTenantID, agentID))
		if err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now); err != nil {
			t.Fatalf("expected the native format to keep working: %v", err)
		}
		assertAccepted(t, d)
	})
}

//spec:covers REQ-SHAREDSIGNALS-010: 両方の表現を検査して解決後に照合し、不正な主体を補わず、拒否時はどの Agent も失効させない。
//spec:covers RFC9493-SUBID-FORMAT, RFC9493-SUBID-ISS-SUB: 取り出し口によらず形式と trusted_issuer を検査する。
func TestReceiveSecurityEventTopLevelSubject(t *testing.T) {
	const agentID, otherAgentID, clientID = "agent_1", "agent_2", "client_1"
	opaque := map[string]any{"format": "opaque", "id": agentID}
	native := map[string]any{"subject_type": "Agent", "tenant_id": receiveTestTenantID, "principal_id": agentID}
	cases := []struct {
		name       string
		subID      any
		subject    any
		hasSubject bool
		accepted   bool
	}{
		{name: "最上位のみ", subID: opaque, accepted: true},
		{name: "同一識別子", subID: opaque, subject: opaque, accepted: true},
		{name: "束縛先クライアントの別名", subID: opaque, subject: map[string]any{"format": "iss_sub", "iss": "https://transmitter.example", "sub": clientID}, accepted: true},
		{name: "最上位が束縛先クライアント", subID: map[string]any{"format": "opaque", "id": clientID}, subject: native, accepted: true},
		{name: "独自形式との共存", subID: opaque, subject: native, accepted: true},
		{name: "異なる Agent", subID: map[string]any{"format": "opaque", "id": otherAgentID}, subject: opaque},
		{name: "最上位の発行者不一致", subID: map[string]any{"format": "iss_sub", "iss": "https://other.example", "sub": agentID}, subject: opaque},
		{name: "従来形式の発行者不一致", subID: opaque, subject: map[string]any{"format": "iss_sub", "iss": "https://other.example", "sub": agentID}},
		{name: "最上位が null", subID: nil, subject: opaque},
		{name: "最上位が文字列", subID: agentID, subject: opaque},
		{name: "最上位が独自形式", subID: native, subject: opaque},
		{name: "最上位の format 欠落", subID: map[string]any{"id": agentID}, subject: opaque},
		{name: "最上位の format 型不正", subID: map[string]any{"format": 1, "id": agentID}, subject: opaque},
		{name: "最上位の未対応形式", subID: map[string]any{"format": "complex", "id": agentID}, subject: opaque},
		{name: "最上位の識別子が空", subID: map[string]any{"format": "opaque", "id": ""}, subject: opaque},
		{name: "最上位の識別子が不明", subID: map[string]any{"format": "opaque", "id": "unknown"}, subject: opaque},
		{name: "従来形式の型不正", subID: opaque, subject: agentID},
		{name: "従来形式が null", subID: opaque, hasSubject: true},
		{name: "従来形式の種類が欠落", subID: opaque, subject: map[string]any{"tenant_id": receiveTestTenantID, "principal_id": agentID}},
		{name: "従来形式の種類が不明", subID: opaque, subject: map[string]any{"subject_type": "unknown", "tenant_id": receiveTestTenantID, "principal_id": agentID}},
		{name: "従来形式の format 型不正", subID: opaque, subject: map[string]any{"format": 1, "subject_type": "Agent", "tenant_id": receiveTestTenantID, "principal_id": agentID}},
		{name: "従来形式が別テナント", subID: opaque, subject: map[string]any{"subject_type": "Agent", "tenant_id": "other-tenant", "principal_id": agentID}},
		{name: "従来形式が別の種類", subID: opaque, subject: map[string]any{"subject_type": "User", "tenant_id": receiveTestTenantID, "principal_id": agentID}},
		{name: "最上位が混在形式", subID: map[string]any{"format": "opaque", "id": agentID, "principal_id": agentID}, subject: opaque},
		{name: "従来形式が混在形式", subID: opaque, subject: map[string]any{"format": "opaque", "id": agentID, "subject_type": "Agent"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eventClaims := map[string]any{}
			if tc.subject != nil || tc.hasSubject {
				eventClaims["subject"] = tc.subject
			}
			verified := &ssports.VerifiedSecurityEvent{
				JTI: "sub-id-set", SubID: tc.subID, HasSubID: true,
				Events: map[string]any{caepSessionRevokedURI: eventClaims},
			}
			d := newReceiveTestDeps(t, &fakeVerifier{result: verified})
			seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
			seedReceiveAgent(t, d, agentID)
			seedReceiveAgent(t, d, otherAgentID)
			now := time.Now().UTC()
			if _, err := d.agentRepo.AddBinding(context.Background(), &agentmodel.AgentCredentialBinding{AgentID: agentID, ClientID: clientID, CreatedAt: now}); err != nil {
				t.Fatal(err)
			}
			err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now)
			if other, findErr := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, otherAgentID); findErr != nil || other != nil {
				t.Fatalf("対象外の失効エポック=%+v err=%v", other, findErr)
			}
			epoch, findErr := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, agentID)
			if findErr != nil {
				t.Fatal(findErr)
			}
			var received, rejected int
			for _, event := range *d.events {
				switch e := event.(type) {
				case *ssdomain.SecurityEventReceived:
					received++
				case *ssdomain.SecurityEventRejected:
					rejected++
					if e.VerificationResult != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
						t.Fatalf("拒否結果=%s", e.VerificationResult)
					}
				}
			}
			if tc.accepted {
				if err != nil || epoch == nil || !epoch.Epoch.Equal(now) || received != 1 || rejected != 0 {
					t.Fatalf("err=%v epoch=%+v received=%d rejected=%d", err, epoch, received, rejected)
				}
			} else if !errors.Is(err, ssusecases.ErrSecurityEventRejected) || epoch != nil || received != 0 || rejected != 1 {
				t.Fatalf("err=%v epoch=%+v received=%d rejected=%d", err, epoch, received, rejected)
			}
		})
	}
}
