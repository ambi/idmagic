package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestPostgresSchemaReferentialIntegrityConstraints(t *testing.T) {
	sql, err := os.ReadFile("../../../../../deploy/schema/postgres.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(sql)
	required := []string{
		"CONSTRAINT users_tenant_sub_unique UNIQUE (tenant_id, sub)",
		"CONSTRAINT consents_tenant_sub_fkey\n        FOREIGN KEY (tenant_id, sub)\n        REFERENCES users(tenant_id, sub) ON DELETE RESTRICT",
		"CONSTRAINT refresh_tokens_tenant_sub_fkey\n        FOREIGN KEY (tenant_id, sub)\n        REFERENCES users(tenant_id, sub) ON DELETE RESTRICT",
		"CONSTRAINT signing_keys_tenant_id_fkey\n        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT",
		"CONSTRAINT agents_tenant_owner_fkey\n        FOREIGN KEY (tenant_id, owner_sub)\n        REFERENCES users(tenant_id, sub) ON DELETE RESTRICT",
		"CONSTRAINT agent_credential_bindings_tenant_client_fkey\n        FOREIGN KEY (tenant_id, client_id)\n        REFERENCES clients(tenant_id, client_id) ON DELETE RESTRICT",
		"FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT",
		"FOREIGN KEY (tenant_id, user_sub)\n        REFERENCES users (tenant_id, sub) ON DELETE CASCADE",
		"CREATE TRIGGER application_assignments_subject_ref_trigger",
	}
	for _, want := range required {
		if !strings.Contains(schema, want) {
			t.Fatalf("postgres schema missing referential-integrity constraint:\n%s", want)
		}
	}
}
