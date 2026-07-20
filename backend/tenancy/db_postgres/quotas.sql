-- name: GetTenantQuota :one
SELECT * FROM tenant_quotas WHERE tenant_id = $1;

-- name: UpsertTenantQuota :exec
INSERT INTO tenant_quotas (
    tenant_id, users, groups, agents, applications, oauth2_clients,
    active_sessions, consents, active_jobs, audit_events_retained, export_artifacts_bytes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) ON CONFLICT (tenant_id) DO UPDATE SET
    users = EXCLUDED.users,
    groups = EXCLUDED.groups,
    agents = EXCLUDED.agents,
    applications = EXCLUDED.applications,
    oauth2_clients = EXCLUDED.oauth2_clients,
    active_sessions = EXCLUDED.active_sessions,
    consents = EXCLUDED.consents,
    active_jobs = EXCLUDED.active_jobs,
    audit_events_retained = EXCLUDED.audit_events_retained,
    export_artifacts_bytes = EXCLUDED.export_artifacts_bytes;

-- name: GetTenantUsage :one
SELECT * FROM tenant_usages WHERE tenant_id = $1;

-- name: UpsertTenantUsage :exec
INSERT INTO tenant_usages (
    tenant_id, users, groups, agents, applications, oauth2_clients,
    active_sessions, consents, active_jobs, audit_events_retained, export_artifacts_bytes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) ON CONFLICT (tenant_id) DO UPDATE SET
    users = EXCLUDED.users,
    groups = EXCLUDED.groups,
    agents = EXCLUDED.agents,
    applications = EXCLUDED.applications,
    oauth2_clients = EXCLUDED.oauth2_clients,
    active_sessions = EXCLUDED.active_sessions,
    consents = EXCLUDED.consents,
    active_jobs = EXCLUDED.active_jobs,
    audit_events_retained = EXCLUDED.audit_events_retained,
    export_artifacts_bytes = EXCLUDED.export_artifacts_bytes;
