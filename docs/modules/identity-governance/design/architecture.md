# IdGovernance のアーキテクチャ

この文書は、IdGovernance の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは、`IdManagement` と `Application` への一方向であり、循環しない。
具体的なアダプターは起動処理で注入する。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 捕捉が User の変更を受け、実行が User と Group を読み書きする | 相手が定める `UserMutationCommitter` を実装する。相手の `user` と `group` の Repository と、`UserLifecycle` の実装を使う |
| `Application` | 実行が割り当てを読み、変える | 相手の `AssignmentRepository` と、`ApplicationAssignments` の実装を使う |
| `Jobs` | ディスパッチャーが Job を投入し、`worker` がハンドラーを実行する | 相手の `JobRepository` と、Job の種類 `lifecycle_workflow_run` を使う |
| `Tenancy` | ディスパッチャーが Job の数を上限で確かめる | 相手の `QuotaRepository` を使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 記録の正を持たず、ポリシーとオーケストレーションだけを持つ | ワークフローが記録の正を持つモジュールの内部へ広がらない。詳細は[判断](decisions.md#ガバナンスのポリシーとオーケストレーションを-idmanagement-から切り出す) |
| WorkflowRun を User の変更と同じトランザクションで捕捉する | User だけが更新されて実行が作られない事態を防ぐ |
| アクションを冪等にし、ステップごとに結果を記録する | 再試行で、成功したステップを繰り返さない |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。
機能仕様に対応するコードは、要件の ID を `//spec:covers` で名指すテストからたどる。

| 層 | 責務 |
| --- | --- |
| `domain` | `LifecycleWorkflow`、トリガーの評価、WorkflowRun の計画、ドメインイベント |
| `ports` | 定義と実行の Repository、User と実行を一緒に保存する捕捉、`UserLifecycle`、`ApplicationAssignments` |
| `usecases` | 定義の管理、プレビュー、捕捉、ディスパッチ、実行のハンドラー |
| `handlers_http` | テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 実行の捕捉 | IdManagement の User の作成と変更 | User を変更したプロセスが、同じトランザクションで WorkflowRun を保存する | [ワークフローの実行の設計](../workflow-run/design.md) |
| Job の関連付け | 一定の間隔 | `worker` のディスパッチャー | [ワークフローの実行の設計](../workflow-run/design.md) |
| ステップの実行 | `lifecycle_workflow_run` の Job | `worker` が Job のハンドラーとして実行する | [ワークフローの実行](../workflow-run/README.md) |
| 管理 API | 解決済みのテナントの管理 API への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [ライフサイクルワークフローの定義](../lifecycle-workflow/README.md) |
