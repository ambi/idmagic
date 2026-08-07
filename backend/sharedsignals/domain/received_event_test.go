package domain_test

import (
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validReceivedEvent() ssdomain.ReceivedSecurityEvent {
	return ssdomain.ReceivedSecurityEvent{
		ID:        "received_1",
		TenantID:  "tenant-a",
		StreamID:  "stream_1",
		SetJTI:    "jti_1",
		EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject: ssdomain.SsfSubject{
			SubjectType: ssdomain.SsfSubjectTypeAgent,
			TenantID:    "tenant-a",
			PrincipalID: "agent_1",
		},
		VerificationResult: ssdomain.SecurityEventVerificationAccepted,
		ReceivedAt:         time.Now().UTC(),
	}
}

// TestReceivedSecurityEventValidateHappyAndFailure — scenario
// `重複jtiのSETは一度だけ反映される` / `他テナントのstreamはissuer一致でも受理されない` の
// 前提となる構造的妥当性を検証する。
func TestReceivedSecurityEventValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.ReceivedSecurityEvent)
		wantErr bool
	}{
		{"ok accepted", func(*ssdomain.ReceivedSecurityEvent) {}, false},
		{"bad verification result", func(e *ssdomain.ReceivedSecurityEvent) {
			e.VerificationResult = ssdomain.SecurityEventVerificationResult("x")
		}, true},
		{"rejected without subject is ok", func(e *ssdomain.ReceivedSecurityEvent) {
			e.VerificationResult = ssdomain.SecurityEventVerificationRejectedSignature
			e.Subject = ssdomain.SsfSubject{}
		}, false},
		{"accepted without subject is rejected", func(e *ssdomain.ReceivedSecurityEvent) {
			e.Subject = ssdomain.SsfSubject{}
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validReceivedEvent()
			c.mutate(&e)
			err := e.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

func TestNewReceivedSecurityEventID(t *testing.T) {
	id, err := ssdomain.NewReceivedSecurityEventID()
	if err != nil {
		t.Fatalf("NewReceivedSecurityEventID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("NewReceivedSecurityEventID = %q, want UUID", id)
	}
}
