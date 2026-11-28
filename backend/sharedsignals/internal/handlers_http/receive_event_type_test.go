package handlers_http_test

import (
	"net/http"
	"testing"
	"time"

	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SHAREDSIGNALS-019: 購読する種類と IdMagic が知らない種類の署名済み SET を HTTP で受け、202 と記録と失効エポックを観測する。
//spec:covers EX-SHAREDSIGNALS-019-02: 購読する `change_type=revoke` の credential-change は、reflected_at のない受理の記録だけを残し、Agent を失効させない。
//spec:covers EX-SHAREDSIGNALS-019-04: CaepEventType にない risk-level-change は、記録も拒否のイベントも残さずに 202 を返す。
func TestReceiveSecurityEventByEventTypeHTTP(t *testing.T) {
	credentialRevoked := map[string]any{"https://schemas.openid.net/secevent/caep/event-type/credential-change": map[string]any{
		"credential_type": "fido2-roaming", "change_type": "revoke", "event_timestamp": time.Now().Unix(),
	}}
	riskLevelChanged := map[string]any{"https://schemas.openid.net/secevent/caep/event-type/risk-level-change": map[string]any{
		"risk_reason": "PASSWORD_FOUND_IN_DATA_BREACH", "principal": "USER", "current_level": "LOW", "event_timestamp": time.Now().Unix(),
	}}

	t.Run("購読する credential-change の取り消しは記録だけを残す", func(t *testing.T) {
		s, streamID, key := newEventTypeReceiver(t)
		rec := s.postSET(t, streamID, signEventTypeSET(t, key, "credential-revoked", credentialRevoked))
		if rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
			t.Fatalf("status=%d body=%s, want 202 with an empty body", rec.Code, rec.Body.String())
		}
		if len(s.received.Saved) != 1 {
			t.Fatalf("受信記録=%+v, want one accepted record", s.received.Saved)
		}
		record := s.received.Saved[0]
		if record.VerificationResult != ssdomain.SecurityEventVerificationAccepted || record.ReflectedAt != nil ||
			record.EventType != ssdomain.CaepEventTypeCredentialChange || record.Subject.PrincipalID != exampleAgentID {
			t.Fatalf("受理記録=%+v, want accepted credential-change for %s without reflected_at", record, exampleAgentID)
		}
		if epoch := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); epoch != nil {
			t.Fatalf("失効エポック=%+v, want unchanged", epoch)
		}
		if s.events.count("SecurityEventReceived") != 0 || s.events.count("SecurityEventRejected") != 0 {
			t.Fatalf("イベント=%+v, want neither SecurityEventReceived nor SecurityEventRejected", s.events.all())
		}
	})

	t.Run("知らない種類は何も残さずに 202 を返す", func(t *testing.T) {
		s, streamID, key := newEventTypeReceiver(t)
		rec := s.postSET(t, streamID, signEventTypeSET(t, key, "risk-level", riskLevelChanged))
		if rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
			t.Fatalf("status=%d body=%s, want 202 with an empty body", rec.Code, rec.Body.String())
		}
		if len(s.received.Saved) != 0 {
			t.Fatalf("受信記録=%+v, want none", s.received.Saved)
		}
		if epoch := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); epoch != nil {
			t.Fatalf("失効エポック=%+v, want unchanged", epoch)
		}
		if s.events.count("SecurityEventReceived") != 0 || s.events.count("SecurityEventRejected") != 0 {
			t.Fatalf("イベント=%+v, want neither SecurityEventReceived nor SecurityEventRejected", s.events.all())
		}
	})
}

// newEventTypeReceiver は session-revoked と credential-change を購読する受信側のストリームを管理 API で登録する。
func newEventTypeReceiver(t *testing.T) (*exampleStack, string, *transmitterKey) {
	t.Helper()
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	key := newTransmitterKey(t)
	body := receiverRequest(key)
	body["event_types"] = []string{string(ssdomain.CaepEventTypeSessionRevoked), string(ssdomain.CaepEventTypeCredentialChange)}
	rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/receiver", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register receiver status=%d body=%s", rec.Code, rec.Body.String())
	}
	return s, decodeID(t, rec), key
}

func signEventTypeSET(t *testing.T, key *transmitterKey, jti string, events map[string]any) string {
	t.Helper()
	token, err := tokensjose.SignPS256(key.private, key.kid, map[string]string{"typ": "secevent+jwt"}, map[string]any{
		"iss": trustedIssuer, "aud": receiverAudience, "jti": jti, "iat": time.Now().Unix(),
		"sub_id": issSub(exampleAgentID), "events": events,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}
