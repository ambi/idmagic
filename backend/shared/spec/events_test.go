package spec_test

import (
	"encoding/json"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type unmarshalableEvent struct {
	Fn func()
}

func (unmarshalableEvent) EventType() string     { return "Unmarshalable" }
func (unmarshalableEvent) OccurredAt() time.Time { return time.Now() }

func TestMarshalDomainEventPropagatesMarshalError(t *testing.T) {
	if _, err := spec.MarshalDomainEvent(unmarshalableEvent{Fn: func() {}}); err == nil {
		t.Fatal("expected error marshaling an event with an unsupported field type")
	}
}

type sliceEvent []int

func (sliceEvent) EventType() string     { return "Slice" }
func (sliceEvent) OccurredAt() time.Time { return time.Now() }

func TestMarshalDomainEventPropagatesReencodeError(t *testing.T) {
	// A DomainEvent that marshals to a non-object JSON value (here, a bare
	// array) succeeds at the first json.Marshal but fails the intermediate
	// decode into map[string]any that MarshalDomainEvent uses to splice in
	// type/occurredAt.
	if _, err := spec.MarshalDomainEvent(sliceEvent{1, 2, 3}); err == nil {
		t.Fatal("expected error re-decoding a non-object event into a wire map")
	}
}

func TestMarshalDomainEventUsesContractFieldNames(t *testing.T) {
	data, err := spec.MarshalDomainEvent(&oauthdomain.RefreshTokenIssued{
		At:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		TokenID: "token", FamilyID: "family", ClientID: "client", UserID: "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["type"] != "RefreshTokenIssued" || wire["familyId"] != "family" {
		t.Fatalf("unexpected event wire: %s", data)
	}
	if _, exists := wire["FamilyID"]; exists {
		t.Fatalf("Go field name leaked: %s", data)
	}
}
