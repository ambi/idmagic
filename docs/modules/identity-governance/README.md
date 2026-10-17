# IdGovernance

## 責務と境界

アイデンティティガバナンス（IGA）のポリシーとオーケストレーションを扱う。
JML を自動化する `LifecycleWorkflow` の定義、トリガーの評価、`WorkflowRun` の実行がここに属する。

| 扱わないもの | 担当 |
| --- | --- |
| User と Group の記録の正 | `IdManagement` |
| Application の割り当ての記録の正 | `Application` |
| WorkflowRun を実行する Job のキューと再試行 | `Jobs` |
| メールの送信の部品 | 共有の通知の部品 |

このモジュールは、User の変更を契機にワークフローを評価し、記録の正を持つモジュールの状態を冪等に変える。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 実行の捕捉 | IdManagement の User の作成と変更 | User を変更したプロセスが、同じトランザクションで WorkflowRun を保存する | [ワークフローの実行の設計](features/workflow-run.md#実現方式) |
| Job の関連付け | 一定の間隔 | `worker` のディスパッチャー | [ワークフローの実行の設計](features/workflow-run.md#実現方式) |
| ステップの実行 | `lifecycle_workflow_run` の Job | `worker` が Job のハンドラーとして実行する | [ワークフローの実行](features/workflow-run.md) |
| 管理 API | 解決済みのテナントの管理 API への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [ライフサイクルワークフローの定義](features/lifecycle-workflow.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 記録の正を持たず、ポリシーとオーケストレーションだけを持つ | ワークフローが記録の正を持つモジュールの内部へ広がらない。詳細は[判断](decisions.md#ガバナンスのポリシーとオーケストレーションを-idmanagement-から切り出す) |
| WorkflowRun を User の変更と同じトランザクションで捕捉する | User だけが更新されて実行が作られない事態を防ぐ |
| アクションを冪等にし、ステップごとに結果を記録する | 再試行で、成功したステップを繰り返さない |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [ライフサイクルワークフローの定義](features/lifecycle-workflow.md) | 作成、編集、有効化と無効化、削除、取得、プレビュー |
| [ワークフローの実行](features/workflow-run.md) | User の変更による実行の捕捉、Job の関連付け、ステップの実行と再試行 |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| データ設計 | [IdGovernance のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| 信頼性設計 | [ワークフローの実行の設計](features/workflow-run.md#信頼性) | Job の投入の失敗と再試行の扱いはワークフローの実行の設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [IdGovernance の用語集](glossary.md) | このモジュールでの語義 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
