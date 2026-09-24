# wi-68415-rename-provisioning-delivery-to-provisioning-task

作業項目は `wi-68415-rename-provisioning-delivery-to-provisioning-task` である。
対象は [IdMagic.Provisioning.Operations.ListProvisioningTasks](../../../spec/contexts/provisioning/main.tsp)、[IdMagic.Provisioning.Operations.GetProvisioningTask](../../../spec/contexts/provisioning/main.tsp)、[IdMagic.Provisioning.Operations.RetryProvisioningTask](../../../spec/contexts/provisioning/main.tsp) と、[REQ-PROVISIONING-015](../../domain/provisioning/scenarios.feature.md) である。

## 管理 API

旧パスは取り除いた。
旧パスを呼ぶクライアントは次の表のとおり呼び先を改める。

| 取り除いた操作 | 置き換える操作 |
| --- | --- |
| `GET /api/admin/v1/applications/{id}/provisioning/deliveries`（`ListProvisioningDeliveries`） | `GET /api/admin/v1/applications/{id}/provisioning/tasks`（`ListProvisioningTasks`） |
| `GET /api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}`（`GetProvisioningDelivery`） | `GET /api/admin/v1/applications/{id}/provisioning/tasks/{task_id}`（`GetProvisioningTask`） |
| `POST /api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}/retry`（`RetryProvisioningDelivery`） | `POST /api/admin/v1/applications/{id}/provisioning/tasks/{task_id}/retry`（`RetryProvisioningTask`） |

一覧の応答は、配列を `deliveries` ではなく `tasks` に入れる。
要素の型は `ProvisioningDelivery` から `ProvisioningTask` へ改名したが、フィールドは変わらない。
存在しない id を指定したときの problem の `type` は変わらない。

## イベントとジョブ

ドメインイベント `ProvisioningDeliveryStarted` は `ProvisioningTaskStarted` になり、Provisioning のイベントのペイロードのフィールド `deliveryId` は `taskId` になる。
監査ログやイベントを名前で絞り込んでいる場合は、新しい名前へ改める。

ジョブの種類 `provisioning_delivery` は `provisioning_task` になる。
製品は未リリースなので、旧名で投入済みのジョブとテーブル `provisioning_deliveries` の行は移行しない。
開発環境のデータベースは、スキーマを適用し直してから使う。
