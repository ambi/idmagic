package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	audithttp "github.com/ambi/idmagic/backend/audit/handlers_http"
)

func TestCharacterizeAdminAuditSearchOptionsSupportedChoices(t *testing.T) {
	e := newAuditAdminServer(t, auditUser("user_admin", "acme", []string{"admin"}), nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events/search-options")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body audithttp.AuditEventSearchOptionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(body.Outcomes, []string{"success", "failure"}) {
		t.Fatalf("outcomes=%v, want [success failure]", body.Outcomes)
	}
	if !slices.IsSorted(body.EventTypes) {
		t.Fatalf("event_types must be sorted: %v", body.EventTypes)
	}
	for _, eventType := range []string{"FederatedAuthenticated", "UserCreated", "EmailChanged"} {
		if !slices.Contains(body.EventTypes, eventType) {
			t.Fatalf("event_types must contain %s: %v", eventType, body.EventTypes)
		}
	}
}
