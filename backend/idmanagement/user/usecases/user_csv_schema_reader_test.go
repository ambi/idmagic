package usecases

import (
	"bytes"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// Regression test for the bug where TenantUserCSVSchemaReader returned only
// the tenant's custom attribute definitions, causing ValidateAttributes to
// reject builtin attribute keys (e.g. "department") that were never absent
// from any allowlist, just missing from the merged defs.
func TestTenantUserCSVSchemaReaderIncludesBuiltinDefs(t *testing.T) {
	ctx := importPlannerContext()
	schemaRepo := usermemory.NewTenantUserAttributeSchemaRepository()
	reader := TenantUserCSVSchemaReader{Repository: schemaRepo}

	// No tenant custom schema saved at all.
	defs, err := reader.EffectiveUserAttributeDefs(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAttrKey(defs, "department") {
		t.Fatalf("expected builtin attribute %q in defs, got %+v", "department", defs)
	}

	// A tenant custom schema exists but only defines an unrelated attribute.
	if err := schemaRepo.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID:   "acme",
		Attributes: []userdomain.UserAttributeDef{{Key: "cost_code", Type: idmdomain.AttributeTypeString}},
	}); err != nil {
		t.Fatal(err)
	}
	defs, err = reader.EffectiveUserAttributeDefs(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if !containsAttrKey(defs, "department") || !containsAttrKey(defs, "middle_name") || !containsAttrKey(defs, "cost_code") {
		t.Fatalf("expected builtin + custom defs merged, got %+v", defs)
	}
}

func containsAttrKey(defs []userdomain.UserAttributeDef, key string) bool {
	for _, d := range defs {
		if d.Key == key {
			return true
		}
	}
	return false
}

// Regression test for ExportUserCSV via the real TenantUserCSVSchemaReader:
// a user with a value set on a builtin attribute (department) must not fail
// export even though the tenant has no matching custom attribute schema.
func TestExportUserCSVSucceedsWithBuiltinAttributeValue(t *testing.T) {
	ctx := importPlannerContext()
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice")) // has Attributes["department"] set

	schemaRepo := usermemory.NewTenantUserAttributeSchemaRepository()
	artifacts := &exportArtifactStore{}
	result, err := ExportUserCSV(ctx, UserCSVExportDeps{
		UserRepo:     repo,
		SchemaReader: TenantUserCSVSchemaReader{Repository: schemaRepo},
		Artifacts:    artifacts,
	}, []string{"id", "preferred_username"}, "", userdomain.DefaultUserCSVTransferPolicy())
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if result.TotalRows != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !bytes.Contains(artifacts.content, []byte("user-alice,alice")) {
		t.Fatalf("csv=%q", artifacts.content)
	}
}
