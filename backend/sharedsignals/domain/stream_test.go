package domain_test

import (
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validStream() ssdomain.SsfStream {
	return ssdomain.SsfStream{
		ID:         "stream_1",
		TenantID:   "tenant-a",
		Direction:  ssdomain.SsfStreamDirectionTransmit,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked},
		Status:     ssdomain.SsfStreamStatusEnabled,
		CreatedAt:  time.Now().UTC(),
	}
}

// TestSsfStreamValidateHappyAndFailure — scenario
// `無効化したstreamは配送も受理も行わない` の前提となる SsfStream の構造的妥当性を検証する。
func TestSsfStreamValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.SsfStream)
		wantErr bool
	}{
		{"ok", func(*ssdomain.SsfStream) {}, false},
		{"missing id", func(s *ssdomain.SsfStream) { s.ID = "" }, true},
		{"bad direction", func(s *ssdomain.SsfStream) { s.Direction = ssdomain.SsfStreamDirection("x") }, true},
		{"bad status", func(s *ssdomain.SsfStream) { s.Status = ssdomain.SsfStreamStatus("x") }, true},
		{"empty event_types", func(s *ssdomain.SsfStream) { s.EventTypes = nil }, true},
		{"invalid event type", func(s *ssdomain.SsfStream) {
			s.EventTypes = []ssdomain.CaepEventType{"not-a-real-type"}
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := validStream()
			c.mutate(&s)
			err := s.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

func TestSsfStreamIsEnabledAndSubscribes(t *testing.T) {
	s := validStream()
	if !s.IsEnabled() {
		t.Fatal("enabled stream must report IsEnabled() == true")
	}
	if !s.Subscribes(ssdomain.CaepEventTypeSessionRevoked) {
		t.Fatal("stream must subscribe to its own configured event type")
	}
	if s.Subscribes(ssdomain.CaepEventTypeCredentialChange) {
		t.Fatal("stream must not subscribe to an unconfigured event type")
	}

	s.Status = ssdomain.SsfStreamStatusDisabled
	if s.IsEnabled() {
		t.Fatal("disabled stream must report IsEnabled() == false")
	}
}

func TestNewSsfStreamID(t *testing.T) {
	id, err := ssdomain.NewSsfStreamID()
	if err != nil {
		t.Fatalf("NewSsfStreamID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("NewSsfStreamID = %q, want UUID", id)
	}
}
