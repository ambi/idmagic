# ライフサイクルワークフローの定義

## 概要

この文書は、テナント管理者が `LifecycleWorkflow` を定義し、有効化し、試算する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | ワークフローの作成、編集、有効化と無効化、削除、一覧と取得、プレビュー（`dry_run`） |
| 行為者 | テナント管理者 |
| 扱わないもの | トリガーの評価とアクションの実行は[ワークフローの実行](../workflow-run/README.md)が扱う |

## モデル

編集は `current_revision` を進める。
有効化したリビジョンが `enabled_revision` になり、それだけが新しい WorkflowRun を作る。
有効化した後の編集は下書きとして `current_revision` に残り、有効化し直すまでトリガーの評価にもプレビューにも使わない。

## 状態遷移

### WorkflowDefinitionLifecycle

`LifecycleWorkflow` は `draft` で作成する。完全な検証に成功したリビジョンを有効化すると、新しいトリガーの評価対象になる。無効化すると新しいトリガーを止め、後から再び有効化できる。管理者は `draft`、`enabled`、`disabled` のどの状態からも削除できる。`enabled` から削除した場合は新しいトリガーを止め、`queued` の WorkflowRun をキャンセルする。削除済みの定義は参照整合性を保つため、内部では終端状態の `archived` として保持するが、管理画面と通常の API には公開しない。実行履歴と `LifecycleWorkflowDeleted` 監査イベントは保持する。

| State | Kind | Meaning |
|---|---|---|
| draft | initial | 作成直後。トリガーの評価対象にならない |
| enabled | — | 新しいトリガーの評価対象になる |
| disabled | — | 新しいトリガーを止めている。再び有効化できる |
| archived | terminal | 削除済み。参照整合性のために保持し、管理画面と通常の API には公開しない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| draft | LifecycleWorkflowEnabled | — | enabled |  |
| enabled | LifecycleWorkflowDisabled | — | disabled |  |
| disabled | LifecycleWorkflowEnabled | — | enabled |  |
| draft | LifecycleWorkflowDeleted | — | archived |  |
| enabled | LifecycleWorkflowDeleted | — | archived |  |
| disabled | LifecycleWorkflowDeleted | — | archived |  |

## 操作

### 管理者によるワークフローの作成

#### REQ-IDGOVERNANCE-001 管理者はライフサイクルワークフローを作成できる

#### REQ-IDGOVERNANCE-014 ライフサイクルワークフローの管理は管理者に限られる

### 管理者によるワークフローの編集

#### REQ-IDGOVERNANCE-002 管理者は既存ライフサイクルワークフローの定義を編集できる

#### REQ-IDGOVERNANCE-007 未知のフィールドや別テナントのリソースを参照するワークフローは有効化できない

### 管理者によるワークフローの無効化

#### REQ-IDGOVERNANCE-011 無効化すると未開始の WorkflowRun はキャンセルされ、実行中の WorkflowRun はステップ境界で止まる

### 管理者によるワークフローの取得

#### REQ-IDGOVERNANCE-012 別テナントのワークフローとリソースはテナント境界を越えない

### 管理者によるワークフローのプレビュー

#### REQ-IDGOVERNANCE-013 プレビューではアクションの結果を試算するが WorkflowRun や Job を作成しない

## セキュリティ上の考慮

`LifecycleWorkflow` の作成、編集、有効化、無効化、削除、プレビューは、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
プレビューは評価の結果を返すだけで保存しないので、参照の操作に置く。

定義の時点で別のテナントの識別子を指定すれば、保存で拒否する。
実行の時点の再取得は[ワークフローの実行](../workflow-run/README.md)が扱う。
