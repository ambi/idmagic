# IdGovernance のリスク

この文書は、IdGovernance の既知の欠陥、技術的負債、受容したリスクを扱う。
脅威は[脅威モデル](../../../design/security/threat-model.md)に置く。

| リスク | 現状 | 影響 |
| --- | --- | --- |
| グループのメンバーシップと必須操作の手順が、記録の正を持つ Context の操作を通らない | `add_group_member`、`remove_group_member` は Group の Repository を、`set_required_action`、`clear_required_action` は User の Repository を直接呼ぶ | 管理 API で同じ変更をしたときのドメインイベント、監査の記録、下流への通知が伴わない |
| 実行の記録の保持期間を適用する処理がない | WorkflowRun、WorkflowStep、通知の配送の記録を、Job と同じ 30 日で消す方針がある一方、それらを消す処理はない | 実行の記録が消えずに増え続ける |
| ワークフローの手順の User の無効化では、配下の Agent の失効エポックが進まない | `disable_user` は `worker` で実行され、失効エポックを進める反応は `api` のプロセスにだけある（[SharedSignals のリスク](../../sharedsignals/design/risks.md)） | 退職者のワークフローで User を止めても、所有する Agent の失効エポックは進まず、外部の受信側へも失効が伝わらない |
