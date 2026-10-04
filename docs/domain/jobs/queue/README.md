# 永続キュー

## 概要

この文書は、ジョブの投入、取得、リース、再試行、配信不能、レーンの隔離、テナントの境界の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 投入と重複の排除、レーンごとの取得とリース、ハートビート、再試行と配信不能、レーンの登録の検証、実行の文脈のテナントへの固定 |
| 行為者 | System（ジョブを投入する Context のユースケースと、`worker` の実行環境） |
| 扱わないもの | ジョブの参照と取り消しは[ジョブの管理](../admin/README.md)が扱う |

## モデル

配送の保証は少なくとも 1 回である。
取得のたびに `attempts` を増やし、リースの所有者と期限を設定する。
完了を報告できるのは、リースの所有者だけである。
ハンドラーは冪等でなければならず、重複排除のキーと、利用側自身の一貫性の境界を使う。

`dedup_key` を伴う投入は、`(tenant_id, dedup_key)` に一致する未終端のジョブがあれば新しく作らず、既存の参照を返す。

レーンが与えるのは順序ではなく、キャパシティの隔離である。
`bulk` の滞留がどれだけ積み上がっても、`latency_sensitive` の実行枠を奪わない。
レーンの中に数値の優先度はない。
取得の候補はおおむね `run_at` の古い順だが、並行の実行、複数のプロセス、同じ時刻のジョブがあるので、開始の順序も完了の順序も保証しない。

- **判断**：レーンを種類の登録で決める理由は、[実行レーンを呼び出し元ではなく JobKind の登録で決める](../design/decisions.md#実行レーンを呼び出し元ではなく-jobkind-の登録で決める)。
- **判断**：少なくとも 1 回とする理由は、[配送を少なくとも 1 回とし冪等性をハンドラーに任せる](../design/decisions.md#配送を少なくとも-1-回とし冪等性をハンドラーに任せる)。

## 状態遷移

### JobLifecycle

`worker` が取得すると `queued` から `running` へ遷移する。実行に失敗した場合、`attempts` が `max_attempts` 未満ならバックオフを伴って `queued` に戻り、上限に達していれば配信不能を表す `failed` へ遷移する。`succeeded`、`failed`、`canceled` は不可逆の終端状態である。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 取得待ち。`run_at` に達すると自分のレーンの `worker` が取得できる |
| running | — | `worker` がリースを持ってハンドラーを実行している |
| succeeded | terminal | ハンドラーが正常終了した |
| failed | terminal | 試行上限に達し、配信不能として確定した |
| canceled | terminal | 終端に達する前に取り消した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | JobStarted | — | running |  |
| running | JobSucceeded | — | succeeded |  |
| running | JobFailed | attempts >= max_attempts | failed |  |
| running | JobRetried | attempts < max_attempts | queued |  |
| queued | JobCanceled | — | canceled |  |
| running | JobCanceled | — | canceled |  |

## 操作

### 利用側の Context によるジョブの投入

#### REQ-JOBS-002 ジョブを投入すると `worker` が実行して成功する

#### REQ-JOBS-007 同じ dedup_key の lifecycle_workflow_run は重複して投入されない

#### REQ-JOBS-008 API プロセスでの投入に失敗しても定期ディスパッチャーが未関連付けの実行を回収する

### worker によるジョブの実行

#### REQ-JOBS-003 同じ Job を再試行してもハンドラーの副作用は 1 回分だけ観測される

#### REQ-JOBS-004 `worker` が異常終了してもリース失効後に別の `worker` が再取得する

#### REQ-JOBS-009 bulk レーンに未処理ジョブが滞留しても latency_sensitive ジョブは専用実行枠で取得される

#### REQ-JOBS-011 レーンのカラムを省略したレコードは `default` レーンで補完され取得対象になる

#### REQ-JOBS-005 ハンドラーが失敗し続けると `max_attempts` 到達時に配信不能となる

#### REQ-JOBS-006 他テナントの Job は `worker` のテナント境界を越えない

### worker の起動

#### REQ-JOBS-010 レーン未登録の JobKind は `worker` 起動時に拒否される

## セキュリティ上の考慮

投入と取得を行う HTTP のエンドポイントは公開しない。
`EnqueueJob`、`ClaimJobs`、`HeartbeatJob`、`CompleteJob`、`FailJob` はいずれも同じプロセスの中の Go の呼び出しであり、テナント管理者や API アクセストークンから直接呼べる経路はない。
あるテナントが別のテナントのジョブを投入したり、実行枠を奪ったりすることはできない。

ハンドラーの実行の文脈は、その Job の `tenant_id` に固定する。
ハンドラーが誤って別のテナントの Aggregate の識別子を渡されても、実行の文脈のテナントと一致しないので、操作は拒否される。
`worker` のプロセスはすべてのテナントのジョブを実行するが、1 件のジョブを処理する間に見えるのは一つのテナントの範囲だけである。
