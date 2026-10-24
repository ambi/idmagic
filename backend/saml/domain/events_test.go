package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 監査の検索は payload の userId で主体を引くので、項目名が TypeSpec と違うと、利用者で絞り込んだ
// 監査の検索から SAML のサインインが黙って抜ける。
//
//spec:covers REQ-SAML-006: SamlSignInIssued の payload は、TypeSpec の SamlSignInIssued と同じ userId でサインインした利用者を指す。
func TestSamlSignInIssuedNamesTheUserAsInTheContract(t *testing.T) {
	data, err := spec.MarshalDomainEvent(&domain.SamlSignInIssued{
		At: time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), TenantID: "tenant", EntityID: "https://sp.example", UserID: "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["userId"] != "user-1" || wire["tenantId"] != "tenant" || wire["entityId"] != "https://sp.example" {
		t.Fatalf("payload = %s, want the SamlSignInIssued fields of the contract", data)
	}
}
