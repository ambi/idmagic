namespace IdMagic.Contract {

@doc("worker が heartbeat / 完了報告 / 失敗報告を行った時点で、既にリースが失効しているか別 worker に奪われている。呼び出し元 worker はハンドラ実行を中断すべきである。")
model JobLeaseLostError {

}


@doc("enqueue された非同期処理の実行単位。worker がリースを確保して Running にし、\nハンドラが handler_context.tenant_id == job.tenant_id の境界内だけで副作用を起こす。\nレコードの保持期間は delete_after 30日とする。")
model Job {
  @doc("Job の内部生成 ID (UUIDv4)。")
  id: string;
  @doc("Job が帰属するテナント境界。worker はこの境界内でのみ副作用を起こす。")
  tenant_id: string;
  @doc("実行するハンドラを決めるジョブ種別。")
  kind: JobKind;
  @doc("kind の登録情報から enqueue 時に一意に決まる実行レーン。\n呼び出し元は指定できず、Job 作成後は不変。claim は対象 lane の Job だけを対象にする。")
  lane: ExecutionLane;
  status: JobStatus;
  @doc("enqueue 時にハンドラへ渡す入力。呼び出し側は生 PII を極力含めない。")
  params: unknown;
  @doc("ハンドラが Succeeded 時に残す出力。")
  result?: unknown;
  @doc("直近の失敗理由。Failed (dead-letter) 到達時は最終エラーを保持する。")
  error?: string;
  @doc("これまでの実行試行回数。")
  attempts: int64;
  @doc("この回数を超えて失敗すると dead-letter (Failed) に確定する。")
  max_attempts: int64;
  @doc("(tenant_id, dedup_key) で一意。同じ副作用を重複 enqueue しないための冪等キー。at-least-once 配信下でもハンドラの副作用は1回分しか観測されないことを保証する主な仕組み。")
  dedup_key?: string;
  @doc("ハンドラが Running 中に任意で報告する進捗。")
  progress?: JobProgress;
  @doc("現在 Running を保持する worker 識別子。terminal 状態や Queued では未設定。")
  lease_owner?: string;
  @doc("リースの失効時刻。heartbeat で延長する。期限切れになると別 worker が再 claim できる。")
  lease_expires_at?: utcDateTime;
  @doc("この時刻以降に claim 可能。enqueue 直後は now、リトライ時は backoff 分だけ先送りされる。")
  run_at: utcDateTime;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("JobLifecycle の状態。")
enum JobStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
  canceled: "canceled",
}


@doc("worker の handler registry に登録されたジョブ種別の allowlist。新しいジョブ種別を\n追加する consumer 機能は、実装前にこの enum へ自分の種別を足し、ExecutionLane を\n同時に宣言する (SCL-first)。core runtime は疎通確認用の noop_echo のみを\n追加する。backchannel_logout_delivery は未実装機能向けの先行宣言。data_key_reencryption は\nDataKeys がエンベロープ暗号化の再暗号化 job 用に追加する。")
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


@doc("Job の実行枠を隔離する区分。JobKind の登録情報が lane を一意に決め、\nenqueue 呼び出し元は指定できない。latency_sensitive は低遅延配送を専用実行枠で保証し、\nbulk backlog の滞留から隔離される。bulk は長時間・大量ジョブ、default はそのいずれでもない\nジョブに使う。")
enum ExecutionLane {
  latency_sensitive: "latency_sensitive",
  default: "default",
  bulk: "bulk",
}


@doc("ハンドラが Running 中に任意で報告する進捗スナップショット。")
model JobProgress {
  @doc("0-100 の概算進捗。")
  percent?: int64;
  @doc("人間可読な進捗メッセージ。")
  message?: string;
  updated_at: utcDateTime;
}


@doc("他 bounded context が「enqueue した/参照したい Job」を保持するための published\nlanguage。Job aggregate 全体ではなく、外部公開してよい最小の参照だけを持つ。")
model JobRef {
  id: string;
  tenant_id: string;
  kind: JobKind;
  status: JobStatus;
}


@doc("Job が queue に投入された。")
model JobEnqueued {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  kind: JobKind;
}


@doc("worker が Job を claim し Running に遷移した。")
model JobStarted {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  workerId: string;
  attempt: int64;
}


@doc("ハンドラが正常終了した。")
model JobSucceeded {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}


@doc("ハンドラが失敗した。terminal が true なら max_attempts 超過による dead-letter 確定。")
model JobFailed {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  terminal: boolean;
  error: string;
}


@doc("失敗後、backoff を経て再試行のため Queued に戻った。")
model JobRetried {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  nextRunAt: utcDateTime;
}


@doc("終端状態に達する前に取消された。")
model JobCanceled {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}

}
