# wi-68415-rename-provisioning-delivery-to-provisioning-task

Provisioning の `ProvisioningDelivery` を `ProvisioningTask` へ改名し、日本語では「プロビジョニングタスク」と呼ぶ。
管理 API のパスは `/api/admin/v1/applications/{id}/provisioning/deliveries` から `/api/admin/v1/applications/{id}/provisioning/tasks` へ、パスパラメーターは `delivery_id` から `task_id` へ変わる。
一覧の応答のフィールドは `deliveries` から `tasks` へ変わる。
対象は [IdMagic.Provisioning.Operations.ListProvisioningTasks](../../../spec/contexts/provisioning/main.tsp)、[IdMagic.Provisioning.Operations.GetProvisioningTask](../../../spec/contexts/provisioning/main.tsp)、[IdMagic.Provisioning.Operations.RetryProvisioningTask](../../../spec/contexts/provisioning/main.tsp) と、[REQ-PROVISIONING-015](../../domain/provisioning/scenarios.feature.md) である。
ドメインイベントは `ProvisioningDeliveryStarted` から `ProvisioningTaskStarted` へ、ペイロードのフィールドは `deliveryId` から `taskId` へ、ジョブの種類は `provisioning_delivery` から `provisioning_task` へ変わる。
