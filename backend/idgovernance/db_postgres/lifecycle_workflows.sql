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

-- name: CancelQueuedLifecycleWorkflowRuns :many
UPDATE lifecycle_workflow_runs SET status='canceled',updated_at=now() WHERE tenant_id=$1 AND workflow_id=$2 AND status='queued' RETURNING id,target_user_id;

-- name: InsertLifecycleWorkflowRun :one
INSERT INTO lifecycle_workflow_runs (id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,triggered_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (tenant_id,workflow_id,revision,source_occurrence_id,target_user_id) DO NOTHING RETURNING id;

-- name: InsertLifecycleWorkflowStep :exec
INSERT INTO lifecycle_workflow_steps (run_id,step_index,action,outcome) VALUES ($1,$2,$3,$4);

-- name: FindLifecycleWorkflowRun :one
SELECT id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,job_id,triggered_at FROM lifecycle_workflow_runs WHERE tenant_id=$1 AND id=$2;

-- name: ListUnenqueuedLifecycleWorkflowRuns :many
SELECT id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,job_id,triggered_at FROM lifecycle_workflow_runs WHERE status='queued' AND job_id IS NULL ORDER BY triggered_at LIMIT $1;

-- name: AttachLifecycleWorkflowRunJob :execrows
UPDATE lifecycle_workflow_runs SET job_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='queued' AND job_id IS NULL;

-- name: ListLifecycleWorkflowSteps :many
SELECT s.step_index,s.action,s.outcome,COALESCE(s.error_code,'') AS error_code,s.completed_at FROM lifecycle_workflow_steps s JOIN lifecycle_workflow_runs r ON r.id=s.run_id WHERE r.tenant_id=$1 AND s.run_id=$2 ORDER BY s.step_index;

-- name: StartLifecycleWorkflowRun :execrows
UPDATE lifecycle_workflow_runs candidate SET status='running',updated_at=now() WHERE candidate.tenant_id=$1 AND candidate.id=$2 AND candidate.status='queued' AND NOT EXISTS (SELECT 1 FROM lifecycle_workflow_runs prior WHERE prior.tenant_id=candidate.tenant_id AND prior.target_user_id=candidate.target_user_id AND prior.id<>candidate.id AND prior.status IN ('queued','running') AND prior.triggered_at<candidate.triggered_at);

-- name: CheckpointLifecycleWorkflowStep :exec
UPDATE lifecycle_workflow_steps SET outcome=$3,error_code=NULLIF($4::text,''),completed_at=$5 WHERE run_id=$1 AND step_index=$2 AND EXISTS (SELECT 1 FROM lifecycle_workflow_runs WHERE id=$1 AND tenant_id=$6);

-- name: CompleteLifecycleWorkflowRun :exec
UPDATE lifecycle_workflow_runs SET status=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2;
