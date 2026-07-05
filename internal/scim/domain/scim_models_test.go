package domain_test

import (
	"testing"

	"github.com/ambi/idmagic/internal/scim/domain"
)

// NewScimError は RFC 7644 のエラースキーマを付与した ScimErrorResponse を組み立てる。
func TestNewScimError(t *testing.T) {
	got := domain.NewScimError("404", "not found", "invalidValue")

	if len(got.Schemas) != 1 ||
		got.Schemas[0] != "urn:ietf:params:scim:api:messages:2.0:Error" {
		t.Fatalf("unexpected schemas: %v", got.Schemas)
	}
	if got.Status != "404" {
		t.Errorf("Status = %q, want 404", got.Status)
	}
	if got.Detail != "not found" {
		t.Errorf("Detail = %q, want 'not found'", got.Detail)
	}
	if got.ScimType != "invalidValue" {
		t.Errorf("ScimType = %q, want invalidValue", got.ScimType)
	}
}

// detail / scimType が空でも schema と status は必ず設定される。
func TestNewScimErrorEmptyOptionalFields(t *testing.T) {
	got := domain.NewScimError("500", "", "")
	if got.Status != "500" {
		t.Errorf("Status = %q, want 500", got.Status)
	}
	if got.Detail != "" || got.ScimType != "" {
		t.Errorf("expected empty optional fields, got detail=%q scimType=%q", got.Detail, got.ScimType)
	}
	if len(got.Schemas) != 1 {
		t.Fatalf("schemas must always be set, got %v", got.Schemas)
	}
}
