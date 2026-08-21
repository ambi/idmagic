# Jobs Internals

## EnqueueJob
呼び出し元の Bounded Context が非同期処理をキューへ投入するための内部インターフェース。HTTP には公開せず、同一プロセス内の Go 呼び出しとして各 Context のユースケースから使う。`dedup_key` を指定した場合、`(tenant_id, dedup_key)` が既存の未終端 Job と一致すれば新しく作成せず、既存の JobRef を返す。作成する Job のレーンは JobKind の登録情報から一意に決まり、呼び出し元は指定できない。

## ClaimJobs
`worker` が指定したレーン内で取得可能な Job、すなわち `status=queued` かつ `run_at <= now` の Job、またはリースが失効した `running` の Job を一括取得する内部インターフェース。同一トランザクション内で `Running` に遷移させ、自身を `lease_owner` として `lease_expires_at` を設定する。対象レーン以外の Job は取得しない。
- Result invariant: all_jobs_leased_by(output.jobs, input.worker_id)
- Result invariant: single_worker_holds_active_lease(output.jobs)
- Result invariant: all_jobs_in_lane(output.jobs, input.lane)

## HeartbeatJob
`worker` が実行中の Job のリースを延長する。

## CompleteJob
`worker` がハンドラーの正常終了を報告し、Job を Succeeded に確定する。

## FailJob
`worker` がハンドラーの失敗を報告する。`attempts < max_attempts` ならバックオフ後の `run_at` を設定して Queued（Retry）へ、`attempts >= max_attempts` なら Failed（配信不能）へ確定する。
- Result invariant: output.status == 'failed' || output.status == 'queued'

## CancelJob
終端状態に達していない Job を取消す。既に終端状態の Job には作用しない (不可逆)。

## Execution lanes

`JobKind` はちょうど 1 つの `ExecutionLane` (`latency_sensitive`、`default`、`bulk`) を持ち、`domain.RegisterKind(kind, lane)` による登録の時点で固定される。`Job.Lane` は種別の登録から導かれ、取得は自分のレーンの中のジョブだけを対象にする。投入する呼び出し元がレーンを選べないのは、レーンが呼び出しごとの優先度ではなく容量を隔離する単位だからである。

各レーンは独自の並行数上限を持つ独立した `Runner` を備え、空いている実行枠の数だけジョブを一括取得して、ハンドラーを並行実行する（デフォルトは 4 枠）。1 つのプロセスで複数レーンの `Runner` を同時に起動することもできる。これは `JOB_WORKER_LANES` が未設定の場合の互換動作であり、開発環境と Docker Compose のデフォルトでもある。1 つのレーンの `Runner` だけを持つ専用の Deployment を動かすこともでき、こちらを本番のデフォルトとする。`infra/k8s/base/worker.yaml` が `idmagic-worker-{latency-sensitive,default,bulk}` の 3 つの Deployment を定義し、レーンごとの並行数は `JOB_WORKER_CONCURRENCY_<LANE>` で与える。

レーンが与えるのは順序ではなく容量の隔離である。`bulk` の滞留がどれだけ積み上がっても、`latency_sensitive` の実行枠を奪うことはない。同じ理由でレーンの中に数値の優先度は存在しない。取得の候補はおおむね `run_at` の古い順だが、並行実行、複数プロセス、同時刻のジョブがあるため、開始の順序も完了の順序も保証しない。

## Claim and lease

`ClaimBatch` は自分のレーンに限り、実行時刻に達した `queued` のジョブと、リースが切れた `running` のジョブを取得する。PostgreSQL では `WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED` を同じ文の `running` への更新と組み合わせることで、2 つの `worker` が同じジョブに対して有効なリースを同時に持つことを防ぐ。永続キューは専用の仲介サービスではなく PostgreSQL に置き、ミドルウェアを増やす代わりに、安全な並行取得のためにコードベースの他の箇所ですでに使っている `FOR UPDATE SKIP LOCKED` と同じ方式を再利用する。

実行の保証は **少なくとも 1 回** である。取得のたびに `attempts` を増やし、`lease_owner` と `lease_expires_at` を設定する。実行中のハンドラーはリース期間の 3 分の 1 ごとにハートビートを送り、完了させられるのはリースの所有者だけである。プロセスが異常終了したり停止させられたりしてハートビートが止まると、リースの期限が切れた後に別の `worker` がそのジョブを再取得できる。したがって **ハンドラーは冪等でなければならない**。重複排除キーと、利用側自身の一貫性境界を使うこと。

## Retry and dead-letter

失敗したジョブは指数的に後ろ倒しした `run_at` とともに `queued` へ戻り、`max_attempts` に達すると `failed` として確定する。取得の間隔、プロセス内の並行度、リース、再試行の間隔はプロセス全体の設定である。`JOB_POLL_INTERVAL`、`JOB_WORKER_CONCURRENCY`、`JOB_LEASE_DURATION`、`JOB_BACKOFF_BASE`、`JOB_BACKOFF_CAP` がそれにあたる。現時点では JobKind ごとの品質の制御も、利用側ごとの順序や流量の制限も提供しない。

SIGTERM や SIGINT を受けると `worker` は取得をやめ、停止猶予期間まで実行中のハンドラーを待つ。猶予期間の後は終了し、回復は明示的な再投入ではなくリースの自然な期限切れによって起こる。明示的な再投入を避けるのは、停止処理の打ち切りとハンドラーの完了が競合したときに二重実行の余地が生じるからである。

## Metrics

`idmagic-worker` は `/metrics` を管理専用の別の HTTP リスナーで公開する。これは `idmagic-api` の `/metrics` とは別のプロセスかつ別の実体である。レーンごとに `jobs_claim_latency_seconds`、`jobs_outcome_total`、`jobs_retry_total`、`jobs_queue_depth` を持つ。`tenant_id` と `job_id` は取りうる値を有限に保つためラベルから除外する。

## Boundary with scheduled batch

全テナントを対象とする定期的な保持期間の処理と署名鍵のライフサイクルの処理は、永続ジョブに混ぜない。代わりに外部のスケジューラーが `idmagic-batch` を 1 回限りで起動する。横断的な一掃にはテナントごとの作業単位が存在しないため、テナントのキューへ流し込むとキューの深さもレーンの隔離も意味を失う。

`worker` 内のライフサイクルワークフローの割り当て処理は、業務処理を直接行う定期バッチではなく、回復経路である。同じトランザクションで確定したものの Job と関連付けられなかった WorkflowRun を再走査し、永続キューへ安全に引き渡す。

## Schema notes

`Job` は自然キーの親を持たないが、`jobs.tenant_id` は必須である。リポジトリ全体の仕様にある [`tenant_id` retention classes](../../persistence.md#tenant_id-retention-classes) のうち、テナントが所有する Aggregate の分類に従う。`status` と `kind` は本書が規範として定める閉じた語彙であり、`CHECK` 制約はそれと並ぶ多層防御であって正ではない。`params` と `result` は `JobKind` ごとに解釈する不透明な JSONB であり、保存時に暗号化しない。終端行は専用の Job ではなく、`worker` の保持期間スイープが有効期間の経過後に削除する。`lane` を省略した行には `DEFAULT 'default'` を適用するが、通常の投入処理は `JobKind` の登録情報からレーンを明示する。`dedup_key` は `JobHandlerIdempotency` を支える。部分一意インデックスにより、`(tenant_id, dedup_key)` ごとに未終端の Job は最大 1 つに限られる。
