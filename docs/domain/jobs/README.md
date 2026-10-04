# Jobs

## 責務と境界

テナントの境界を保つ汎用の非同期ジョブの基盤として、永続キューと `worker` の実行環境を扱う。
対象は、非同期の処理に共通する投入、永続化、取得、リース、ハートビート、再試行、配信不能への退避、キャンセルである。

| 扱わないもの | 担当 |
| --- | --- |
| `JobKind` のパラメーターを解釈して作用を起こす処理 | 利用側の Context のユースケース。`backend/cmd/idmagic-worker/worker.go` が起動時にハンドラーを一覧へ登録する |
| 全テナントを対象とする定期の処理 | `idmagic-batch` |

この分離により、ジョブの基盤は利用側の Context の業務の論理に依存しない。
API のプロセスはジョブを投入するが実行せず、`worker` のプロセスはジョブを実行するが HTTP を提供しない。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Job` | 種類（`JobKind`）、レーン、状態、`params`、`result`、`error`、`run_at`、`attempts` と `max_attempts`、リースの所有者と期限、`dedup_key` | `Tenant` を `tenant_id` で参照する |

`JobKind` は、ちょうど一つの `ExecutionLane`（`latency_sensitive`、`default`、`bulk`）を、`domain.RegisterKind(kind, lane)` による登録の時点で固定する。
投入する呼び出し元はレーンを指定できない。

- **判断**：`params` と `result` は保存時に暗号化しない。`JobKind` ごとに解釈する不透明な値であり、シークレットを含めてはならないからである。

## 公開する契約

管理 API の操作とモデルの形は TypeSpec の `Jobs` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `EnqueueJob` | ジョブを投入する各 Context | この Context が提供する | 同じプロセスの中の Go の呼び出しで、ジョブを投入する。`dedup_key` で重複を防げる |
| ハンドラーの一覧 | `worker` の起動処理が、利用側の Context のハンドラーを登録する | この Context が定める | `JobKind` ごとに、テナントに固定した実行の文脈でハンドラーを呼ぶ |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `JobEnqueued`、`JobStarted`、`JobSucceeded`、`JobFailed`、`JobRetried`、`JobCanceled` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [永続キュー](queue/README.md) | 投入、取得とリース、再試行、配信不能、レーン、テナントの境界 |
| [ジョブの管理](admin/README.md) | テナントとシステムの経路での一覧、参照、取り消し |
| [標準開発環境](local-development/README.md) | Docker なしの開発環境での `worker` の動作 |

| 文書 | 内容 |
| --- | --- |
| [Jobs の用語集](glossary.md) | この Context での語義 |
| [Jobs の設計](design/README.md) | 話題ごとの設計と重要な判断 |
