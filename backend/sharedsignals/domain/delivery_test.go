package domain_test

import (
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validCaepEvent() ssdomain.CaepEvent {
	return ssdomain.CaepEvent{
		EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject: ssdomain.SsfSubject{
			SubjectType: ssdomain.SsfSubjectTypeAgent,
			TenantID:    "tenant-a",
			PrincipalID: "agent_1",
		},
		EventTimestamp:   time.Now().UTC(),
		InitiatingEntity: ssdomain.InitiatingEntityAdmin,
	}
}

func validSecurityEventToken() ssdomain.SecurityEventToken {
	return ssdomain.SecurityEventToken{
		JTI:      "jti_1",
		Issuer:   "https://idmagic.example",
		Audience: "https://receiver.example",
		IssuedAt: time.Now().UTC(),
		Event:    validCaepEvent(),
		Compact:  "eyJ...",
	}
}

func validDelivery() ssdomain.SecurityEventDelivery {
	return ssdomain.SecurityEventDelivery{
		ID:        "delivery_1",
		TenantID:  "tenant-a",
		StreamID:  "stream_1",
		SetJTI:    "jti_1",
		Set:       validSecurityEventToken(),
		Status:    ssdomain.SecurityEventDeliveryStatusPending,
		CreatedAt: time.Now().UTC(),
	}
}

// TestSecurityEventDeliveryValidateHappyAndFailure — scenario
// `配送失敗は再試行され上限超過でdead_letterへ遷移する` の前提となる構造的妥当性を検証する。
func TestSecurityEventDeliveryValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.SecurityEventDelivery)
		wantErr bool
	}{
		{"ok", func(*ssdomain.SecurityEventDelivery) {}, false},
		{"bad status", func(d *ssdomain.SecurityEventDelivery) { d.Status = ssdomain.SecurityEventDeliveryStatus("x") }, true},
		{"negative attempt count", func(d *ssdomain.SecurityEventDelivery) { d.AttemptCount = -1 }, true},
		{"invalid nested set", func(d *ssdomain.SecurityEventDelivery) { d.Set.Compact = "" }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := validDelivery()
			c.mutate(&d)
			err := d.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

func TestSecurityEventDeliveryIsTerminal(t *testing.T) {
	d := validDelivery()
	if d.IsTerminal() {
		t.Fatal("pending delivery must not be terminal")
	}
	d.Status = ssdomain.SecurityEventDeliveryStatusDelivered
	if !d.IsTerminal() {
		t.Fatal("delivered must be terminal")
	}
	d.Status = ssdomain.SecurityEventDeliveryStatusDeadLetter
	if !d.IsTerminal() {
		t.Fatal("dead_letter must be terminal")
	}
	d.Status = ssdomain.SecurityEventDeliveryStatusFailed
	if d.IsTerminal() {
		t.Fatal("failed must not be terminal (still retryable)")
	}
}
