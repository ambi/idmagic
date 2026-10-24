package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// 監査の検索は payload の userId で主体を引くので、項目名が TypeSpec と違うと、利用者で絞り込んだ
// 監査の検索から WS-Fed と WS-Trust の発行が黙って抜ける。
//
//spec:covers REQ-WSFEDERATION-002, REQ-WSFEDERATION-004: WsFedSignInIssued と WsTrustTokenIssued の payload は、TypeSpec と同じ userId で発行先の利用者を指す。
func TestIssuanceEventsNameTheUserAsInTheContract(t *testing.T) {
	at := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		event    spec.DomainEvent
		audience string
		value    string
	}{
		{event: &WsFedSignInIssued{At: at, TenantID: "tenant", Wtrealm: "urn:rp", UserID: "user-1"}, audience: "wtrealm", value: "urn:rp"},
		{event: &WsTrustTokenIssued{At: at, TenantID: "tenant", AppliesTo: "urn:rp", UserID: "user-1"}, audience: "appliesTo", value: "urn:rp"},
	} {
		t.Run(tc.event.EventType(), func(t *testing.T) {
			data, err := spec.MarshalDomainEvent(tc.event)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			if wire["userId"] != "user-1" || wire["tenantId"] != "tenant" || wire[tc.audience] != tc.value {
				t.Fatalf("payload = %s, want the %s fields of the contract", data, tc.event.EventType())
			}
		})
	}
}
