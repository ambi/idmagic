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

`LifecycleWorkflow` は `draft` で作成する。完全な検証に成功したリビジョンを有効化すると、新しいトリガーの評価対象になる。有効化した後の編集は下書きとして `current_revision` に残り、有効化し直すまで評価に使わない。無効化すると新しいトリガーを止め、後から再び有効化できる。管理者は `draft`、`enabled`、`disabled` のどの状態からも削除できる。`enabled` から削除した場合は新しいトリガーを止め、`queued` の WorkflowRun をキャンセルする。削除済みの定義は参照整合性を保つため、内部では終端状態の `archived` として保持するが、管理画面と通常の API には公開しない。実行履歴と `LifecycleWorkflowDeleted` 監査イベントは保持する。

| State | Kind | Meaning |
|---|---|---|
| draft | initial | 作成直後。トリガーの評価対象にならない |
| enabled | — | 新しいトリガーの評価対象になる |
| disabled | — | 新しいトリガーを止めている。再び有効化できる |
| archived | terminal | 削除済み。参照整合性のために保持し、管理画面と通常の API には公開しない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| draft | LifecycleWorkflowEnabled | — | enabled |  |
| enabled | LifecycleWorkflowEnabled | — | enabled |  |
| enabled | LifecycleWorkflowDisabled | — | disabled |  |
| disabled | LifecycleWorkflowEnabled | — | enabled |  |
| draft | LifecycleWorkflowUpdated | — | draft |  |
| disabled | LifecycleWorkflowUpdated | — | disabled |  |
| draft | LifecycleWorkflowDeleted | — | archived |  |
| enabled | LifecycleWorkflowDeleted | — | archived |  |
| disabled | LifecycleWorkflowDeleted | — | archived |  |

| State | 編集 | 有効化 | 無効化 | 削除 |
|---|---|---|---|---|
| draft | → draft | → enabled（検証に成功）<br>拒否：400 invalid_request（検証に失敗） | 拒否：400 invalid_request | → archived |
| enabled | → enabled | → enabled（検証に成功）<br>拒否：400 invalid_request（検証に失敗） | → disabled | → archived |
| disabled | → disabled | → enabled（検証に成功）<br>拒否：400 invalid_request（検証に失敗） | 拒否：400 invalid_request | → archived |
| archived | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request |

## 操作

### 管理者によるワークフローの作成

#### REQ-IDGOVERNANCE-001 管理者はライフサイクルワークフローを作成できる

- 管理者がトリガーと順序付きのアクションを指定してワークフローを作成したとき、IdGovernance は、名前を前後の空白を除いて保存し、リビジョン 1 の `draft` のワークフローを作り、201 とワークフローを返し、`LifecycleWorkflowCreated` を発行する。
- 管理者が作成画面を開いたとき、IdGovernance は、トリガーとアクションの種類を日本語の説明とともに表示し、内部の識別子（`draft`、`revision`、`current_revision`）を表示しない。
- グループまたはアプリケーションを使うアクションに選べる参照先がない間、管理者が作成画面を開いたとき、IdGovernance は、先に参照先を作る必要があることを日本語で表示し、作成を受け付けない。
- トリガーまたはアクションの必須の設定が足りない間、管理者が作成画面で入力したとき、IdGovernance は、作成の操作を無効にし、足りない設定を日本語で表示する。
- 名前のないワークフローか、不正なトリガーかアクションを指定された場合、IdGovernance は、400 と `invalid_request` で拒否し、ワークフローを作らない。
- 同じテナントの削除していないほかのワークフローと、大文字と小文字を区別せずに同じ名前を指定された場合、IdGovernance は、409 と `workflow_name_conflict` で拒否する。
- **例**：EX-IDGOVERNANCE-001-01、EX-IDGOVERNANCE-001-02、EX-IDGOVERNANCE-001-03

#### REQ-IDGOVERNANCE-014 ライフサイクルワークフローの管理は管理者に限られる

- `admin` のロールを持たない利用者がワークフローの一覧、取得、作成、編集、有効化、無効化、削除、プレビュー、実行の一覧と取得と再試行を要求した場合、IdGovernance は、403 と `access_denied` で拒否し、ワークフローを作成も変更もせず、WorkflowRun と Job を作らない。
- **例**：EX-IDGOVERNANCE-014-01、EX-IDGOVERNANCE-014-02

### 管理者によるワークフローの編集

#### REQ-IDGOVERNANCE-002 管理者は既存ライフサイクルワークフローの定義を編集できる

- 管理者が `expected_revision` を現在のリビジョンにしてワークフローを編集したとき、IdGovernance は、新しいリビジョンを保存して `current_revision` を一つ進め、200 とワークフローを返し、`LifecycleWorkflowUpdated` を発行する。
- ワークフローが `enabled` の間、管理者がワークフローを編集したとき、IdGovernance は、`enabled_revision` を変えず、有効化し直すまで編集をトリガーの評価とプレビューに使わない。
- 管理者が編集画面を開いたとき、IdGovernance は、現在のトリガーと順序付きのアクションを日本語の表示名と説明とともにフォームに復元する。
- `expected_revision` が現在のリビジョンと異なる編集、有効化、無効化、削除を要求された場合、IdGovernance は、409 と `workflow_revision_conflict` で拒否し、ワークフローを変えない。
- **例**：EX-IDGOVERNANCE-002-01

#### REQ-IDGOVERNANCE-007 未知のフィールドや別テナントのリソースを参照するワークフローは有効化できない

- 管理者が `expected_revision` を現在のリビジョンにしてワークフローを有効化したとき、IdGovernance は、そのリビジョンを完全に検証し、`enabled_revision` にし、200 とワークフローを返し、`LifecycleWorkflowEnabled` を発行する。
- テナントの属性スキーマにないフィールドのフィルター、別のテナントか存在しない Group か Application の参照、動的グループを対象にした `add_group_member` を含むワークフローの保存または有効化を要求された場合、IdGovernance は、400 と `invalid_request` で拒否し、ワークフローを変えない。
- **例**：EX-IDGOVERNANCE-007-01

### 管理者によるワークフローの無効化

#### REQ-IDGOVERNANCE-011 無効化すると未開始の WorkflowRun はキャンセルされ、実行中の WorkflowRun はステップ境界で止まる

- 管理者が `enabled` のワークフローを無効化したとき、IdGovernance は、ワークフローを `disabled` にし、`enabled_revision` を外し、`queued` の WorkflowRun をすべて `canceled` にして実行ごとに `LifecycleWorkflowRunCanceled` を発行し、200 とワークフローを返し、`LifecycleWorkflowDisabled` を発行する。
- ワークフローが `disabled` の間、`running` の WorkflowRun が次のステップを始めるとき、IdGovernance は、現在のステップのチェックポイントの後で実行を `canceled` にする。
- ワークフローが `enabled` でない間、User の変更があったとき、IdGovernance は、そのワークフローの WorkflowRun を作らない。
- `enabled` でないワークフローの無効化を要求された場合、IdGovernance は、400 と `invalid_request` で拒否する。
- 管理者がワークフローを削除したとき、IdGovernance は、ワークフローを `archived` にし、`queued` の WorkflowRun をキャンセルし、実行の履歴を残し、204 を返し、`LifecycleWorkflowDeleted` を発行する。
- **例**：EX-IDGOVERNANCE-011-01、EX-IDGOVERNANCE-011-02

### 管理者によるワークフローの取得

#### REQ-IDGOVERNANCE-012 別テナントのワークフローとリソースはテナント境界を越えない

- 管理者がワークフローを一覧または取得したとき、IdGovernance は、呼び出し元のテナントの削除していないワークフローだけを返す。
- 別のテナントか削除済みか存在しないワークフローを指定された場合、IdGovernance は、存在しないワークフローとして 400 と `invalid_request` で拒否する。
- 別のテナントの Group か Application を参照するアクションを受けた場合、IdGovernance は、保存の時点で拒否し、別のテナントの同じ名前のリソースへ読み替えない。
- **例**：EX-IDGOVERNANCE-012-01

### 管理者によるワークフローのプレビュー

#### REQ-IDGOVERNANCE-013 プレビューではアクションの結果を試算するが WorkflowRun や Job を作成しない

- 管理者が対象の User を指定してワークフローをプレビューしたとき、IdGovernance は、`enabled_revision`（有効化していないワークフローでは `current_revision`）のトリガーとアクションを、対象の User の現在の Group の所属、Application の割り当て、必須操作、状態、メールアドレスの確認の状態に対して評価する。
- 管理者がワークフローをプレビューしたとき、IdGovernance は、すでに目的の状態のアクションを `no_op`、状態が変わるアクションを `would_change`、実行できないアクションを `blocked` と理由とともに返す。
- 管理者がワークフローをプレビューしたとき、IdGovernance は、WorkflowRun、Job、所属、割り当て、必須操作、状態、メールアドレスを作成も変更もしない。
- トリガーのフィルターが対象の User の現在の属性に一致しない場合、IdGovernance は、すべてのアクションを理由 `trigger_not_matched` の `blocked` として返す。
- 対象の User を指定しないか、存在しない User を指定したプレビューを受けた場合、IdGovernance は、400 と `invalid_request` で拒否する。
- **例**：EX-IDGOVERNANCE-013-01、EX-IDGOVERNANCE-013-02、EX-IDGOVERNANCE-013-03

## セキュリティ上の考慮

`LifecycleWorkflow` の作成、編集、有効化、無効化、削除、プレビューは、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
プレビューは評価の結果を返すだけで保存しないので、参照の操作に置く。

定義の時点で別のテナントの識別子を指定すれば、保存で拒否する。
実行の時点の再取得は[ワークフローの実行](../workflow-run/README.md)が扱う。
