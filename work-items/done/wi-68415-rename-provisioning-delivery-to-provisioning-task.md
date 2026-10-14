---
status: completed
archived_from: 8fde8e6d80edf2c191cea8b3913e305b829be78d
---

# ProvisioningDelivery を ProvisioningTask へ改名し、日本語では「プロビジョニングタスク」と呼ぶ

`mise run spec-diff` は、`REQ-PLATFORM-003` と `REQ-PROVISIONING-001`、`003` から `010`、`012` から `018` のシナリオ、`RFC7643-OUT-CORE-RESOURCES`、`RFC7643-OUT-GROUP-RESOURCES`、`RFC7644-OUT-BULK` の標準行、状態遷移 `ProvisioningDeliveryLifecycle` から `ProvisioningTaskLifecycle` への改名を変更として挙げた。
どのシナリオと標準行も、「配信」を「プロビジョニングタスク」または「プロビジョニングする」へ改めただけで、条件と要求する結果は変わらない。
TypeSpec では、`ListProvisioningDeliveries`、`GetProvisioningDelivery`、`RetryProvisioningDelivery`、`ProvisioningDelivery`、`ProvisioningDeliveryStatus`、`ProvisioningDeliveryStarted` と二つのエラーを取り除き、同じ内容の `ProvisioningTask` の名前の宣言を加えた。
管理 API のパスは `/provisioning/tasks` と `task_id` に、一覧の応答は `tasks` に、ジョブの種類は `provisioning_task` に、テーブルは `provisioning_tasks` に変わった。
管理画面の ja の「配送」と en の delivery も、プロビジョニングタスクへ改めた。

全文は `git show 8fde8e6d80edf2c191cea8b3913e305b829be78d:<このファイルのパス>` で読む。
