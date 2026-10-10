namespace IdMagic.Contract {

@doc("ワーカーがハートビート、完了の報告、または失敗の報告を送った時点で、リースはすでに期限切れか、別のワーカーに引き取られていた。呼び出したワーカーはハンドラーを中止するべきである。これはワーカー向けのドメイン条件であり、HTTP のレスポンスボディではない。")
model JobLeaseLostError {

}


@doc("キューに入れた非同期処理の一単位。ワーカーがリースを取って Running へ移し、ハンドラーは `handler_context.tenant_id == job.tenant_id` の境界の中でだけ副作用を起こす。記録は 30 日間保持する（`delete_after`）。")
model Job {
  @doc("内部で生成したジョブの id（UUIDv4）。")
  id: string;
  @doc("Job が属するテナントの境界。ワーカーはその中でだけ副作用を起こす。")
  tenant_id: string;
  @doc("ジョブの種類。実行するハンドラーを決める。")
  kind: JobKind;
  @doc("実行レーン。キューへ入れるときに、種類の登録が決める。呼び出し元は選べず、Job ができた後は変わらない。引き受けは自身のレーンの Job だけを見る。")
  lane: ExecutionLane;
  status: JobStatus;
  @doc("キューへ入れるときにハンドラーへ渡す入力。呼び出し元は、生の個人情報をできるだけ含めない。")
  params: Record<unknown>;
  @doc("ハンドラーが成功したときに残す出力。")
  result?: Record<unknown>;
  @doc("直近の失敗。Failed（デッドレター）に達したときは最終的なエラーを保持する。")
  error?: string;
  @doc("これまでに実行を試みた回数。")
  attempts: int64;
  @doc("これを超えて失敗すると、Job をデッドレター（Failed）として確定する。")
  max_attempts: int64;
  @doc("`(tenant_id, dedup_key)` で一意である。同じ副作用を二度キューへ入れないための冪等キーであり、少なくとも一回の配送でもハンドラーの副作用が一度だけ観測される主な理由である。")
  dedup_key?: string;
  @doc("Running の間にハンドラーが報告できる進捗。")
  progress?: JobProgress;
  @doc("現在 Running を保持しているワーカーの識別子。終端の状態と Queued の間は設定しない。")
  lease_owner?: string;
  @doc("リースが期限切れになる時刻。ハートビートが延長し、期限が切れると別のワーカーが Job を再び引き受けられる。")
  lease_expires_at?: utcDateTime;
  @doc("Job を引き受けられるようになる時刻。キューへ入れた直後は現在時刻で、再試行ではバックオフの分だけ後ろへずれる。")
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


@doc("ワーカーのハンドラーのレジストリに登録したジョブの種類の許可リスト。新しいジョブの種類を加える利用側の機能は、実装を書く前に、その種類をこの列挙へ加え、同時にその ExecutionLane を宣言する（仕様先行）。中核のランタイムが加えるのは、疎通確認のための `noop_echo` だけである。`backchannel_logout_delivery` は、それを使う機能より先に宣言している。`data_key_reencryption` は、エンベロープ暗号化の再暗号化ジョブのために DataKeys が加える。")
enum JobKind {
  noop_echo: "noop_echo",
  user_import_preview: "user_import_preview",
  user_import_apply: "user_import_apply",
  group_import_preview: "group_import_preview",
  group_import_apply: "group_import_apply",
  dynamic_group_reconcile: "dynamic_group_reconcile",
  lifecycle_workflow_run: "lifecycle_workflow_run",
  backchannel_logout_delivery: "backchannel_logout_delivery",
  provisioning_task: "provisioning_task",
  data_key_reencryption: "data_key_reencryption",
}


@doc("Job の実行枠を隔離する区分。JobKind の登録がそのレーンを決め、キューへ入れる呼び出し元は選べない。`latency_sensitive` は、低レイテンシーの配送に、一括処理の滞留から隔離した専用の枠を保証する。`bulk` は長時間かかる大量のジョブ、`default` はそのどちらでもないすべてのジョブのためのレーンである。")
enum ExecutionLane {
  latency_sensitive: "latency_sensitive",
  default: "default",
  bulk: "bulk",
}


@doc("Running の間にハンドラーが報告できる進捗のスナップショット。")
model JobProgress {
  @doc("おおよその進捗。0 から 100。")
  percent?: int64;
  @doc("人が読む進捗のメッセージ。")
  message?: string;
  updated_at: utcDateTime;
}


@doc("指定した `job_id` の Job が存在しないか、他のテナントに属している。他のテナントの Job を存在しないものとして扱うのは、呼び出し元が見られないテナントの Job の存在を id だけで確かめられないようにするためである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:job_not_found` である。")
model JobNotFoundError is ProblemDetails;


@doc("すでに JobLifecycle の終端の状態（Succeeded、Failed、Canceled）に達した Job に対して CancelJob を呼んだ。終端の状態は変わらないので、何もせず成功を返すのではなくリクエストを拒否する。処理を止めるよう求めた運用者は、その処理がすでに終わっていたかを知る必要があるためである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:job_not_cancelable` である。")
model JobNotCancelableError is ProblemDetails;


@doc("管理 API が公開する Job。`params`、`result`、`dedup_key` は意図して省く。これらはキューへ入れたモジュールが選んだ値で、このモジュールは不透明なまま保存するだけなので、内容を保証できないためである（REQ-JOBS-014）。残るのは、運用者が Job を判断するのに必要なもの、すなわち何の Job か、どこにあるか、何回試行したか、どのワーカーが保持しているか、直近になぜ失敗したかである。")
model AdminJobResponse {
  id: string;
  @doc("Job が属するテナント。複数の値が現れるのはテナントを横断する一覧だけである。")
  tenant_id: string;
  kind: JobKind;
  lane: ExecutionLane;
  status: JobStatus;
  attempts: int64;
  max_attempts: int64;
  @doc("ハンドラーが報告した直近の失敗のメッセージ。一度でも失敗した Job に含まれ、その後再試行したものも含む。")
  error?: string;
  @doc("ハンドラーが最後に報告した進捗。ハンドラーが進捗を報告しなければ省く。")
  progress?: JobProgress;
  @doc("Running のリースを保持しているワーカー。Queued の間と終端の状態では省く。")
  lease_owner?: string;
  lease_expires_at?: utcDateTime;
  @doc("Job を次に引き受けられるようになる時刻。再試行のバックオフを待っている Job では未来の時刻になる。")
  run_at: utcDateTime;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("ListJobs と ListSystemJobs の絞り込み条件。テナントの範囲はパラメーターではなく、経路と認証コンテキストから決まる。ListJobs はリクエストのテナントに、ListSystemJobs はすべてのテナントに固定されるので、どの入力でも範囲は広がりも狭まりもしない。")
model AdminJobQuery {
  @doc("これらの JobStatus の値に絞り込む。省略するとすべての状態を返す。")
  status?: JobStatus[];
  @doc("これらの JobKind の値に絞り込む。省略するとすべての種類を返す。")
  kind?: JobKind[];
  @doc("この ExecutionLane に絞り込む。")
  lane?: ExecutionLane;
  @doc("ページの大きさの上限（デフォルトは 50、最大は 200）。")
  limit?: int64;
  @doc("前のレスポンスの `next_cursor` をそのまま渡す。`(created_at, id)` に対する不透明なキーセットの基準点である。別のテナントまたは別の絞り込み条件で発行したカーソルは、黙って基準点を置き直さずに拒否する。")
  cursor?: string;
}


@doc("Job の一覧の一ページ。作成の新しい順に並べる。")
model AdminJobListResponse {
  jobs: AdminJobResponse[];
  @doc("一覧を続けるための基準点。最後のページでは省く。")
  next_cursor?: string;
}


@doc("別の境界づけられたコンテキストが、Job をキューへ入れたとき、または Job を参照したいときに保持する公開契約。Job の集約全体ではなく、公開に足る最小の参照を含む。")
model JobRef {
  id: string;
  tenant_id: string;
  kind: JobKind;
  status: JobStatus;
}


@doc("Job をキューへ入れた。")
model JobEnqueued {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  kind: JobKind;
}


@doc("ワーカーが Job を引き受け、Running へ移した。")
model JobStarted {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  workerId: string;
  attempt: int64;
}


@doc("ハンドラーが正常に終わった。")
model JobSucceeded {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}


@doc("ハンドラーが失敗した。`terminal` が true のときは `max_attempts` を超えたので、Job をデッドレターとして確定した。")
model JobFailed {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  terminal: boolean;
  error: string;
}


@doc("失敗とそのバックオフの後、Job が次の試行のために Queued へ戻った。")
model JobRetried {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
  attempt: int64;
  nextRunAt: utcDateTime;
}


@doc("Job が終端の状態に達する前に取り消された。")
model JobCanceled {
  occurredAt: utcDateTime;
  jobId: string;
  tenantId: string;
}


@doc("エクスポートとインポートを支えるジョブのサービスが設定されていないか、到達できないので、リクエストをキューへ入れることも読むこともできなかった。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:jobs_unavailable` である。この条件はデプロイメントに起因し、サービスに到達できるようになれば同じリクエストが成功するので、500 ではなく 503 とする。")
model JobsUnavailableError is ProblemDetails;

}
