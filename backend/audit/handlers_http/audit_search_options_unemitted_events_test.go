package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	audithttp "github.com/ambi/idmagic/backend/audit/handlers_http"
)

func TestAdminAuditSearchOptionsOmitsUnemittedFederationEvents(t *testing.T) {
	e := newAuditAdminServer(t, auditUser("user_admin", "acme", []string{"admin"}), nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events/search-options")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body audithttp.AuditEventSearchOptionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"FederationLinked", "FederationUnlinked"} {
		if slices.Contains(body.EventTypes, eventType) {
			t.Errorf("unemitted event %s must not be a search option", eventType)
		}
	}
}
