namespace IdMagic.Contract {

@doc("By the time the worker sent a heartbeat, a completion report, or a failure report, its lease had already expired or been taken by another worker. The calling worker should abort the handler. A worker-facing domain condition, not an HTTP response body.")
model JobLeaseLostError {

}


@doc("One unit of enqueued asynchronous work. A worker takes a lease and moves it to Running, and the handler causes side\neffects only inside the boundary where `handler_context.tenant_id == job.tenant_id`.\nRecords are kept for 30 days (`delete_after`).")
model Job {
  @doc("The internally generated job id (UUIDv4).")
  id: string;
  @doc("The tenant boundary the Job belongs to. A worker causes side effects only inside it.")
  tenant_id: string;
  @doc("The job kind, which decides the handler that runs.")
  kind: JobKind;
  @doc("The execution lane, fixed at enqueue time by the registration of the kind.\nThe caller cannot choose it, and it is immutable once the Job exists. A claim only looks at Jobs on its own lane.")
  lane: ExecutionLane;
  status: JobStatus;
  @doc("The input handed to the handler at enqueue time. Callers keep raw PII out of it as far as they can.")
  params: Record<unknown>;
  @doc("The output the handler leaves behind when it succeeds.")
  result?: Record<unknown>;
  @doc("The most recent failure. On reaching Failed (dead-letter) it holds the final error.")
  error?: string;
  @doc("How many times execution has been attempted so far.")
  attempts: int64;
  @doc("Failing more times than this settles the Job as a dead letter (Failed).")
  max_attempts: int64;
  @doc("Unique on `(tenant_id, dedup_key)`. The idempotency key that keeps the same side effect from being enqueued twice; it is the main reason a handler's side effect is observed only once even under at-least-once delivery.")
  dedup_key?: string;
  @doc("Progress the handler may report while Running.")
  progress?: JobProgress;
  @doc("Identifies the worker currently holding Running. Unset in a terminal state and while Queued.")
  lease_owner?: string;
  @doc("When the lease expires. A heartbeat extends it; once it lapses another worker may claim the Job again.")
  lease_expires_at?: utcDateTime;
  @doc("The Job becomes claimable at this instant: now, right after enqueue, and pushed out by the backoff on a retry.")
  run_at: utcDateTime;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The states of JobLifecycle.")
enum JobStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
  canceled: "canceled",
}


@doc("The allow list of job kinds registered in the worker's handler registry. A consumer feature that adds a new job kind\nadds its kind to this enum, and declares its ExecutionLane at the same time, before writing the implementation\n(specification first). The core runtime contributes only `noop_echo`, for connectivity checks.\n`backchannel_logout_delivery` is declared ahead of the feature that will use it, and `data_key_reencryption` is added\nby DataKeys for the re-encryption job of envelope encryption.")
enum JobKind {
  noop_echo: "noop_echo",
  user_import_preview: "user_import_preview",
  user_import_apply: "user_import_apply",
  dynamic_group_reconcile: "dynamic_group_reconcile",
  lifecycle_workflow_run: "lifecycle_workflow_run",
  backchannel_logout_delivery: "backchannel_logout_delivery",
  provisioning_delivery: "provisioning_delivery",
  data_key_reencryption: "data_key_reencryption",
}


@doc("The partition that isolates execution slots for Jobs. The registration of a JobKind fixes its lane, and the caller\nthat enqueues cannot choose. `latency_sensitive` guarantees low-latency delivery its own slots, isolated from a bulk\nbacklog. `bulk` is for long-running, high-volume jobs, and `default` for everything that is neither.")
enum ExecutionLane {
  latency_sensitive: "latency_sensitive",
  default: "default",
  bulk: "bulk",
}


@doc("A progress snapshot the handler may report while Running.")
model JobProgress {
  @doc("Approximate progress, 0 to 100.")
  percent?: int64;
  @doc("A human-readable progress message.")
  message?: string;
  updated_at: utcDateTime;
}


@doc("The published language another bounded context holds when it has enqueued a Job or wants to refer to one. It carries\nthe smallest reference fit to publish, not the whole Job aggregate.")
model JobRef {
  id: string;
  tenant_id: string;
  kind: JobKind;
  status: JobStatus;
}


@doc("A Job was put on the queue.")
model JobEnqueued {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  kind: JobKind;
}


@doc("A worker claimed a Job and moved it to Running.")
model JobStarted {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  workerId: string;
  attempt: int64;
}


@doc("The handler finished normally.")
model JobSucceeded {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}


@doc("The handler failed. `terminal` being true means `max_attempts` was passed and the Job is settled as a dead letter.")
model JobFailed {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  terminal: boolean;
  error: string;
}


@doc("After a failure and its backoff, the Job returned to Queued for another attempt.")
model JobRetried {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  nextRunAt: utcDateTime;
}


@doc("The Job was cancelled before reaching a terminal state.")
model JobCanceled {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}

}
