package spec_test

import (
	"encoding/json"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

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
