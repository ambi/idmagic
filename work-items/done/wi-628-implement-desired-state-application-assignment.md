---
status: completed
archived_from: f890c5b40406abaf3a7f4d3e81a2c40e6939112b
---

# あるべき状態を指定する Application 割り当て操作を実装する

`mise run spec-diff` は main に対する規範的な仕様変更なしと報告した。
`REQ-APPLICATION-014` の宣言済みの具体例 2 件を、仕様を変えずに実装で満たした。
Application Context に内部インターフェース `DesiredStateAssignments` を加え、`AssignApplicationDesiredState` と `UnassignApplicationDesiredState` が User への直接割り当てだけをあるべき状態へそろえ、実際に変えたかを `bool` で返すようにした。差分がなければ保存、イベント発行、Provisioning への通知のいずれも行わない。
IdGovernance に port `ApplicationAssignments` を宣言し、LifecycleWorkflow の実行ハンドラーは assign_application と unassign_application をこの port へ渡すようにした。保存先への直接の書き込みはなくなり、ワークフローによる割り当ての変更が `ApplicationAssigned` と `ApplicationUnassigned` として監査へ届き、Provisioning へ通知される。
事前の評価は、assign_application について visibility まで一致する直接割り当てだけを `no_op` とするようにした。
worker は Application の Module から操作を組み立て、actor を `lifecycle-workflow` とする。
`tools/check/example-coverage-debt.json` から `EX-APPLICATION-014-01` と `EX-APPLICATION-014-02` を削除し、リリースノートを追加した。

全文は `git show f890c5b40406abaf3a7f4d3e81a2c40e6939112b:<このファイルのパス>` で読む。
