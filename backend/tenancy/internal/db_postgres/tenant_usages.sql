-- name: EnsureTenantUsage :exec
INSERT INTO tenant_usages (tenant_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: IncrementTenantUsageUsers :one
UPDATE tenant_usages u
SET users = u.users + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.users FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.users + sqlc.arg(delta)::int
RETURNING u.users;

-- name: DecrementTenantUsageUsers :exec
UPDATE tenant_usages u SET users = GREATEST(0, u.users - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageGroups :one
UPDATE tenant_usages u
SET groups = u.groups + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.groups FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.groups + sqlc.arg(delta)::int
RETURNING u.groups;

-- name: DecrementTenantUsageGroups :exec
UPDATE tenant_usages u SET groups = GREATEST(0, u.groups - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageAgents :one
UPDATE tenant_usages u
SET agents = u.agents + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.agents FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.agents + sqlc.arg(delta)::int
RETURNING u.agents;

-- name: DecrementTenantUsageAgents :exec
UPDATE tenant_usages u SET agents = GREATEST(0, u.agents - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageApplications :one
UPDATE tenant_usages u
SET applications = u.applications + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.applications FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.applications + sqlc.arg(delta)::int
RETURNING u.applications;

-- name: DecrementTenantUsageApplications :exec
UPDATE tenant_usages u SET applications = GREATEST(0, u.applications - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageOAuth2Clients :one
UPDATE tenant_usages u
SET oauth2_clients = u.oauth2_clients + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.oauth2_clients FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.oauth2_clients + sqlc.arg(delta)::int
RETURNING u.oauth2_clients;

-- name: DecrementTenantUsageOAuth2Clients :exec
UPDATE tenant_usages u SET oauth2_clients = GREATEST(0, u.oauth2_clients - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageActiveSessions :one
UPDATE tenant_usages u
SET active_sessions = u.active_sessions + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.active_sessions FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.active_sessions + sqlc.arg(delta)::int
RETURNING u.active_sessions;

-- name: DecrementTenantUsageActiveSessions :exec
UPDATE tenant_usages u SET active_sessions = GREATEST(0, u.active_sessions - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageConsents :one
UPDATE tenant_usages u
SET consents = u.consents + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.consents FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.consents + sqlc.arg(delta)::int
RETURNING u.consents;

-- name: DecrementTenantUsageConsents :exec
UPDATE tenant_usages u SET consents = GREATEST(0, u.consents - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageActiveJobs :one
UPDATE tenant_usages u
SET active_jobs = u.active_jobs + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.active_jobs FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.active_jobs + sqlc.arg(delta)::int
RETURNING u.active_jobs;

-- name: DecrementTenantUsageActiveJobs :exec
UPDATE tenant_usages u SET active_jobs = GREATEST(0, u.active_jobs - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageSsfStreams :one
UPDATE tenant_usages u
SET ssf_streams = u.ssf_streams + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.ssf_streams FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.ssf_streams + sqlc.arg(delta)::int
RETURNING u.ssf_streams;

-- name: DecrementTenantUsageSsfStreams :exec
UPDATE tenant_usages u SET ssf_streams = GREATEST(0, u.ssf_streams - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageAuditEventsRetained :one
UPDATE tenant_usages u
SET audit_events_retained = u.audit_events_retained + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.audit_events_retained FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.audit_events_retained + sqlc.arg(delta)::int
RETURNING u.audit_events_retained;

-- name: DecrementTenantUsageAuditEventsRetained :exec
UPDATE tenant_usages u SET audit_events_retained = GREATEST(0, u.audit_events_retained - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);

-- name: IncrementTenantUsageExportArtifactsBytes :one
UPDATE tenant_usages u
SET export_artifacts_bytes = u.export_artifacts_bytes + sqlc.arg(delta)::int
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND COALESCE((SELECT q.export_artifacts_bytes FROM tenant_quotas q WHERE q.tenant_id = u.tenant_id), sqlc.arg(default_limit)::int) >= u.export_artifacts_bytes + sqlc.arg(delta)::int
RETURNING u.export_artifacts_bytes;

-- name: DecrementTenantUsageExportArtifactsBytes :exec
UPDATE tenant_usages u SET export_artifacts_bytes = GREATEST(0, u.export_artifacts_bytes - sqlc.arg(delta)::int) WHERE u.tenant_id = sqlc.arg(tenant_id);
