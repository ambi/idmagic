package domain_test

import (
	"encoding/json"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
)

func TestClaimMappingPolicyJSONContract(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: "persistent", SourceAttribute: "user_id"},
		Rules: []claimdomain.ClaimMappingRule{{
			ClaimType: "email", Source: claimdomain.ClaimSourceUserAttribute,
			SourceKey: "email", Required: true,
		}},
	}
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name_id":{"format":"persistent","source_attribute":"user_id"},"rules":[{"claim_type":"email","source":"user_attribute","source_key":"email","required":true}]}`
	if string(data) != want {
		t.Fatalf("claim mapping wire changed:\n got: %s\nwant: %s", data, want)
	}
}
