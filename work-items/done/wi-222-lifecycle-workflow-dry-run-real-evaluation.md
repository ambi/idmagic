---
status: completed
archived_from: 73b40fc320004d4cca8fd4e9d938df96091113ae
---

# lifecycle workflow の dry-run を実態評価する simulation に修正する

`idmdomain.EvaluateWorkflowAction` を副作用なしの評価関数として抽出し、action
executor (`executeLifecycleAction`) と新しい `DryRunLifecycleWorkflow` usecase の両方から呼ぶ形に
リファクタリングした。dry-run は `enabled_revision` (未 enable なら `current_revision`) を対象に、
対象 User の group membership / application assignment / required action / status / email 検証
状態を実際に読んで `would_change` / `no_op` / `blocked` を判定する。trigger の filter が対象 User
に一致しない場合は全 action を `blocked` (reason: `trigger_not_matched`) として返す。副次的に、
executor 側の `unassign_application` が既存の割当有無を確認せず常に `changed` を返していた不整合も
同じ評価関数で解消した (dry-run と実行結果の食い違いを防ぐため)。

全文は `git show 73b40fc320004d4cca8fd4e9d938df96091113ae:<このファイルのパス>` で読む。
