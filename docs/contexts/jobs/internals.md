# Jobs Internals

## Execution lanes

`JobKind` はちょうど 1 つの `ExecutionLane` (`latency_sensitive`、`default`、`bulk`) を持ち、`domain.RegisterKind(kind, lane)` による登録の時点で固定される。投入する呼び出し元はレーンを指定できず、`Job.Lane` は種別の登録から導かれる。

レーンが与えるのは順序ではなく容量の隔離である。`bulk` の滞留がどれだけ積み上がっても、`latency_sensitive` の実行枠を奪うことはない。同じ理由でレーンの中に数値の優先度は存在しない。取得の候補はおおむね `run_at` の古い順だが、並行実行、複数プロセス、同時刻のジョブがあるため、開始の順序も完了の順序も保証しない。

各レーンは独自の並行数上限を持つ独立した `Runner` を備え、空いている実行枠の数だけジョブを一括取得して、ハンドラーを並行実行する（デフォルトは 4 枠）。1 つのプロセスで複数レーンの `Runner` を同時に起動することもでき、これは `JOB_WORKER_LANES` が未設定の場合の互換動作である。本番のデフォルトは、1 つのレーンの `Runner` だけを持つ専用の Deployment を並べる構成であり、`infra/k8s/base/worker.yaml` が `idmagic-worker-{latency-sensitive,default,bulk}` の 3 つを定義する。

## Claim and lease

取得は自分のレーンに限り、実行時刻に達した `queued` のジョブと、リースが切れた `running` のジョブを対象にする。PostgreSQL では `WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED` を同じ文の `running` への更新と組み合わせるため、2 つの `worker` が同じジョブに対して有効なリースを同時に持つことはない。

実行の保証は **少なくとも 1 回** である。取得のたびに `attempts` を増やし、`lease_owner` と `lease_expires_at` を設定する。実行中のハンドラーはリース期間の 3 分の 1 ごとにハートビートを送り、完了を報告できるのはリースの所有者だけである。プロセスが異常終了したり停止させられたりしてハートビートが止まると、リースの期限が切れた後に別の `worker` がそのジョブを再取得できる。したがって **ハンドラーは冪等でなければならない**。重複排除キーと、利用側自身の一貫性境界を使うこと。

同じ作業が二重に積まれることは、投入の側でも防げる。`dedup_key` を伴う投入は、`(tenant_id, dedup_key)` に一致する未終端のジョブがあれば新しく作らず既存の参照を返す。この一意性はアプリケーションの検査ではなく部分一意インデックスが保証するので、同時に投入した 2 つの要求のうち片方だけが行を作る。

## Retry and dead-letter

失敗したジョブは指数的に後ろ倒しした `run_at` とともに `queued` へ戻り、`max_attempts` に達すると `failed` として確定する。取得の間隔、プロセス内の並行度、リース、再試行の間隔はプロセス全体の設定である（`JOB_POLL_INTERVAL`、`JOB_WORKER_CONCURRENCY`、`JOB_LEASE_DURATION`、`JOB_BACKOFF_BASE`、`JOB_BACKOFF_CAP`）。JobKind ごとの品質の制御も、利用側ごとの順序や流量の制限も提供しない。

SIGTERM や SIGINT を受けると `worker` は取得をやめ、停止猶予期間まで実行中のハンドラーを待つ。猶予期間の後は終了し、回復は明示的な再投入ではなくリースの自然な期限切れによって起こる。

## Boundary with scheduled batch

全テナントを対象とする定期的な保持期間の処理と署名鍵のライフサイクルの処理は、永続ジョブに混ぜず、外部のスケジューラーが `idmagic-batch` を 1 回限りで起動する。

一方、`worker` 内のライフサイクルワークフローの割り当て処理は定期バッチではなく回復経路である。同じトランザクションで確定したものの Job と関連付けられなかった `WorkflowRun` を再走査し、永続キューへ安全に引き渡す。
