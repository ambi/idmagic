package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-IDMANAGEMENT-042: 初回パスワードの拒否が所有者の違反コードを返し、User とイベントを作らない。
//verify:operation CreateAdminUser: パスワード拒否の422の Problem Details、違反コードと保存とイベントの無変更
func TestAdminUserAPICreateReturnsOwnerPasswordViolationCodes(t *testing.T) {
	var events []spec.DomainEvent
	e, users := newAdminUserHandler(t, func(d *httpadapter.Deps) {
		d.Emit = func(event spec.DomainEvent) { events = append(events, event) }
	})
	csrf, cookie := adminCSRF(t, e)
	response := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/users", csrf, cookie, map[string]any{
		"preferred_username": "weak-password-subject", "password": "short",
	})
	var body struct {
		Type       string   `json:"type"`
		Status     int      `json:"status"`
		Violations []string `json:"violations"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusUnprocessableEntity || response.Header().Get("Content-Type") != "application/problem+json" ||
		body.Type != "urn:idmagic:error:password_policy" || body.Status != http.StatusUnprocessableEntity ||
		!reflect.DeepEqual(body.Violations, []string{"too_short"}) {
		t.Fatalf("status=%d content-type=%q body=%s, want 422 problem password_policy with too_short",
			response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	stored, err := users.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, "weak-password-subject")
	if err != nil || stored != nil || len(events) != 0 {
		t.Fatalf("user=%+v error=%v events=%v, want no state change", stored, err, events)
	}
}
