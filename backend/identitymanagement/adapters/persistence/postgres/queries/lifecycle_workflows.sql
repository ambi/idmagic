-- name: ListLifecycleWorkflowsByTenant :many
SELECT id,tenant_id,name,description,status,current_revision,enabled_revision,created_at,updated_at FROM lifecycle_workflows WHERE tenant_id=$1 ORDER BY name;

-- name: FindLifecycleWorkflow :one
SELECT id,tenant_id,name,description,status,current_revision,enabled_revision,created_at,updated_at FROM lifecycle_workflows WHERE tenant_id=$1 AND id=$2;

-- name: SaveLifecycleWorkflow :exec
INSERT INTO lifecycle_workflows (id,tenant_id,name,description,status,current_revision,enabled_revision,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,status=EXCLUDED.status,current_revision=EXCLUDED.current_revision,enabled_revision=EXCLUDED.enabled_revision,updated_at=EXCLUDED.updated_at WHERE lifecycle_workflows.tenant_id=EXCLUDED.tenant_id;

-- name: FindLifecycleWorkflowRevision :one
SELECT workflow_id,tenant_id,revision,trigger,actions,created_at FROM lifecycle_workflow_revisions WHERE tenant_id=$1 AND workflow_id=$2 AND revision=$3;

-- name: SaveLifecycleWorkflowRevision :exec
INSERT INTO lifecycle_workflow_revisions (workflow_id,tenant_id,revision,trigger,actions,created_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (workflow_id,revision) DO NOTHING;

-- name: ListLifecycleWorkflowRuns :many
SELECT id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,job_id,triggered_at FROM lifecycle_workflow_runs WHERE tenant_id=$1 AND workflow_id=$2 ORDER BY triggered_at DESC LIMIT $3;

-- name: RetryLifecycleWorkflowRun :execrows
UPDATE lifecycle_workflow_runs SET status='queued',job_id=NULL,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status IN ('failed','partially_failed');

-- name: ResetFailedLifecycleWorkflowSteps :exec
UPDATE lifecycle_workflow_steps SET outcome='pending',error_code=NULL,completed_at=NULL WHERE run_id=$1 AND outcome='failed';

-- name: CancelQueuedLifecycleWorkflowRuns :exec
UPDATE lifecycle_workflow_runs SET status='canceled',updated_at=now() WHERE tenant_id=$1 AND workflow_id=$2 AND status='queued';
