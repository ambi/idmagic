-- Preflight checks for adding tenant-local referential integrity constraints.
-- Run before applying deploy/schema/postgres.sql with psqldef. Every query must
-- return zero rows before the declarative schema change is applied.

SELECT 'consents missing same-tenant user' AS check_name, c.tenant_id, c.sub AS ref_id
  FROM consents c
  LEFT JOIN users u ON u.tenant_id = c.tenant_id AND u.sub = c.sub
 WHERE u.sub IS NULL;

SELECT 'refresh_tokens missing same-tenant user' AS check_name, r.tenant_id, r.sub AS ref_id
  FROM refresh_tokens r
  LEFT JOIN users u ON u.tenant_id = r.tenant_id AND u.sub = r.sub
 WHERE u.sub IS NULL;

SELECT 'signing_keys missing tenant' AS check_name, s.tenant_id, s.kid AS ref_id
  FROM signing_keys s
  LEFT JOIN tenants t ON t.id = s.tenant_id
 WHERE t.id IS NULL;

SELECT 'agents missing same-tenant owner' AS check_name, a.tenant_id, a.owner_sub AS ref_id
  FROM agents a
  LEFT JOIN users u ON u.tenant_id = a.tenant_id AND u.sub = a.owner_sub
 WHERE u.sub IS NULL;

SELECT 'agent_credential_bindings missing same-tenant agent' AS check_name, b.tenant_id, b.agent_id AS ref_id
  FROM agent_credential_bindings b
  LEFT JOIN agents a ON a.tenant_id = b.tenant_id AND a.id = b.agent_id
 WHERE a.id IS NULL;

SELECT 'agent_credential_bindings missing same-tenant client' AS check_name, b.tenant_id, b.client_id AS ref_id
  FROM agent_credential_bindings b
  LEFT JOIN clients c ON c.tenant_id = b.tenant_id AND c.client_id = b.client_id
 WHERE c.client_id IS NULL;

SELECT 'applications missing tenant' AS check_name, a.tenant_id, a.application_id::text AS ref_id
  FROM applications a
  LEFT JOIN tenants t ON t.id = a.tenant_id
 WHERE t.id IS NULL;

SELECT 'application_categories missing tenant' AS check_name, c.tenant_id, c.category_id::text AS ref_id
  FROM application_categories c
  LEFT JOIN tenants t ON t.id = c.tenant_id
 WHERE t.id IS NULL;

SELECT 'application_orderings missing same-tenant user' AS check_name, o.tenant_id, o.user_sub AS ref_id
  FROM application_orderings o
  LEFT JOIN users u ON u.tenant_id = o.tenant_id AND u.sub = o.user_sub
 WHERE u.sub IS NULL;

SELECT 'application_assignments missing same-tenant user subject' AS check_name, a.tenant_id, a.subject_id AS ref_id
  FROM application_assignments a
  LEFT JOIN users u ON u.tenant_id = a.tenant_id AND u.sub = a.subject_id
 WHERE a.subject_type = 'user'
   AND u.sub IS NULL;

SELECT 'application_assignments missing same-tenant group subject' AS check_name, a.tenant_id, a.subject_id AS ref_id
  FROM application_assignments a
  LEFT JOIN groups g ON g.tenant_id = a.tenant_id AND g.id = a.subject_id
 WHERE a.subject_type = 'group'
   AND g.id IS NULL;
