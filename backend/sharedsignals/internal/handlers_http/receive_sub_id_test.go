package handlers_http_test

import (
	"net/http"
	"testing"
	"time"

	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers SSF10-RECEIVE-SUB-ID, REQ-SHAREDSIGNALS-010, EX-SHAREDSIGNALS-010-07: subject を省いた署名済み SET を HTTP で受理し、失効と受信記録とイベントを観測する。
//spec:covers EX-SHAREDSIGNALS-010-08: 二つの表現が Agent と束縛先クライアントを指していても同じ Agent として記録する。
//spec:covers RFC9493-SUBID-ISS-SUB, REQ-SHAREDSIGNALS-005: 発行者の不一致と別テナントの主体を拒否し、失効を反映しない。
func TestReceiveTopLevelSubjectHTTP(t *testing.T) {
	cases := []struct {
		name     string
		subID    any
		subject  map[string]any
		accepted bool
	}{
		{name: "opaque のみ", subID: map[string]any{"format": "opaque", "id": exampleAgentID}, accepted: true},
		{name: "iss_sub のみ", subID: issSub(exampleAgentID), accepted: true},
		{name: "同じ Agent の別名", subID: map[string]any{"format": "opaque", "id": exampleAgentID}, subject: issSub(agentClientID), accepted: true},
		{name: "別の発行者", subID: map[string]any{"format": "iss_sub", "iss": "https://other.example", "sub": exampleAgentID}},
		{name: "別テナントの主体", subID: map[string]any{"format": "opaque", "id": otherTenantAgentID}},
		{name: "null を subject で補わない", subID: nil, subject: issSub(exampleAgentID)},
		{name: "異なる主体", subID: map[string]any{"format": "opaque", "id": otherTenantAgentID}, subject: issSub(exampleAgentID)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newExampleStack(t)
			s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
			s.seedAgent(t, exampleOtherTenant, otherTenantAgentID)
			key := newTransmitterKey(t)
			streamID := s.registerReceiver(t, key)
			// SSF 1.0 §3.1 の独立した要求例。subject は必要な事例だけに付ける。
			event := map[string]any{}
			if tc.subject != nil {
				event["subject"] = tc.subject
			}
			token, err := tokensjose.SignPS256(key.private, key.kid, map[string]string{"typ": "secevent+jwt"}, map[string]any{
				"iss": trustedIssuer, "aud": receiverAudience, "jti": "sub-id-set", "iat": time.Now().Unix(),
				"sub_id": tc.subID, "events": map[string]any{"https://schemas.openid.net/secevent/caep/event-type/session-revoked": event},
			})
			if err != nil {
				t.Fatal(err)
			}
			rec := s.postSET(t, streamID, token)
			if s.epoch(t, exampleOtherTenant, otherTenantAgentID) != nil {
				t.Fatal("別テナントの失効エポックが進んだ")
			}
			if len(s.received.Saved) != 1 {
				t.Fatalf("受信記録=%+v", s.received.Saved)
			}
			record := s.received.Saved[0]
			if record.TenantID != tenancydomain.DefaultTenantID || record.StreamID != streamID || record.SetJTI != "sub-id-set" {
				t.Fatalf("受信記録=%+v", record)
			}
			if !tc.accepted {
				assertSETRejected(t, rec)
				if record.VerificationResult != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved || record.ReflectedAt != nil {
					t.Fatalf("拒否記録=%+v", record)
				}
				if s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID) != nil || s.events.count("SecurityEventReceived") != 0 {
					t.Fatal("拒否した SET が失効を反映した")
				}
				if rejections := s.events.rejections(); len(rejections) != 1 || rejections[0] != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
					t.Fatalf("拒否イベント=%v", rejections)
				}
				return
			}
			if rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if record.VerificationResult != ssdomain.SecurityEventVerificationAccepted || record.ReflectedAt == nil || record.Subject.PrincipalID != exampleAgentID || record.Subject.SubjectType != ssdomain.SsfSubjectTypeAgent {
				t.Fatalf("受理記録=%+v", record)
			}
			if epoch := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); epoch == nil || epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
				t.Fatalf("失効エポック=%+v", epoch)
			}
			if s.events.count("SecurityEventReceived") != 1 || s.events.count("SecurityEventRejected") != 0 {
				t.Fatalf("受信イベント=%+v", s.events.all())
			}
		})
	}
}
