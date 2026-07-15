package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestPostgresSchemaReferentialIntegrityConstraints(t *testing.T) {
	sql, err := os.ReadFile("../../../../../infra/schema/postgres.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(sql)
	required := []string{
		"CONSTRAINT users_tenant_id_unique UNIQUE (tenant_id, id)",
		"CONSTRAINT consents_user_fkey\n        FOREIGN KEY (user_id)\n        REFERENCES users(id) ON DELETE RESTRICT",
		"CONSTRAINT refresh_tokens_user_fkey\n        FOREIGN KEY (user_id)\n        REFERENCES users(id) ON DELETE RESTRICT",
		"CONSTRAINT signing_keys_tenant_id_fkey\n        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT",
		"CONSTRAINT agents_owner_fkey\n        FOREIGN KEY (owner_user_id)\n        REFERENCES users(id) ON DELETE RESTRICT",
		"CONSTRAINT agent_credential_bindings_client_fkey\n        FOREIGN KEY (client_id)\n        REFERENCES clients(client_id) ON DELETE RESTRICT",
		"CONSTRAINT applications_tenant_id_fkey\n        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT",
		"CONSTRAINT password_history_user_id_fkey\n        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
	}
	for _, want := range required {
		if !strings.Contains(schema, want) {
			t.Fatalf("postgres schema missing referential-integrity constraint:\n%s", want)
		}
	}
}

func TestPostgresSchemaTimestampColumnPolicy(t *testing.T) {
	sql, err := os.ReadFile("../../../../../infra/schema/postgres.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(sql)
	allTables := []string{
		"tenants",
		"tenant_brandings",
		"tenant_branding_assets",
		"clients",
		"users",
		"mfa_factors",
		"consents",
		"refresh_tokens",
		"signing_keys",
		"outbox",
		"password_history",
		"password_reset_tokens",
		"groups",
		"group_members",
		"tenant_user_attribute_schemas",
		"email_change_tokens",
		"audit_events",
		"authentication_event_buckets",
		"agents",
		"agent_credential_bindings",
		"authorization_detail_types",
		"applications",
		"application_icons",
		"application_sign_in_policies",
		"tenant_default_sign_in_policies",
		"application_assignments",
		"saml_service_providers",
		"wsfed_relying_parties",
		"application_orderings",
		"application_categories",
		"scim_tokens",
		"scim_user_refs",
		"scim_group_refs",
		"lifecycle_workflows",
		"lifecycle_workflow_revisions",
		"lifecycle_workflow_runs",
		"lifecycle_workflow_steps",
	}
	for _, table := range allTables {
		block := postgresTableBlock(t, schema, table)
		if !strings.Contains(block, "created_at TIMESTAMPTZ NOT NULL DEFAULT now()") {
			t.Fatalf("%s must have created_at with NOT NULL DEFAULT now()", table)
		}
	}

	updatedTables := []string{
		"tenants",
		"tenant_brandings",
		"tenant_branding_assets",
		"clients",
		"users",
		"mfa_factors",
		"consents",
		"refresh_tokens",
		"signing_keys",
		"outbox",
		"authentication_event_buckets",
		"groups",
		"tenant_user_attribute_schemas",
		"agents",
		"authorization_detail_types",
		"applications",
		"application_icons",
		"application_sign_in_policies",
		"tenant_default_sign_in_policies",
		"application_assignments",
		"saml_service_providers",
		"wsfed_relying_parties",
		"application_orderings",
		"application_categories",
		"scim_tokens",
		"scim_user_refs",
		"scim_group_refs",
		"lifecycle_workflows",
		"lifecycle_workflow_runs",
	}
	for _, table := range updatedTables {
		block := postgresTableBlock(t, schema, table)
		if !strings.Contains(block, "updated_at TIMESTAMPTZ NOT NULL DEFAULT now()") {
			t.Fatalf("%s can be updated and must have updated_at with NOT NULL DEFAULT now()", table)
		}
	}

	insertOrDeleteOnlyTables := []string{
		"password_history",
		"password_reset_tokens",
		"group_members",
		"email_change_tokens",
		"audit_events",
		"agent_credential_bindings",
	}
	for _, table := range insertOrDeleteOnlyTables {
		block := postgresTableBlock(t, schema, table)
		if strings.Contains(block, "updated_at") {
			t.Fatalf("%s is insert/delete-only and must not have updated_at", table)
		}
	}
}

func postgresTableBlock(t *testing.T, schema, table string) string {
	t.Helper()
	start := strings.Index(schema, "CREATE TABLE "+table+" (")
	if start < 0 {
		t.Fatalf("postgres schema missing table %s", table)
	}
	rest := schema[start:]
	block, _, ok := strings.Cut(rest, "\n);")
	if !ok {
		t.Fatalf("postgres schema table %s is not closed", table)
	}
	return block
}
