# Provisioning State Transitions

## ProvisioningDeliveryLifecycle

`ProvisioningDelivery.status` の状態遷移を表す。`pending` で作成し、ディスパッチャーが Jobs の `Job` を関連付けると `in_flight` に遷移する。Jobs 側で再試行している間は `in_flight` のまま保持し、`succeeded` または `dead_letter` で終了する。`RetryProvisioningDelivery` は、管理者の操作によって `dead_letter` の配信を基に新しい `pending` の配信行を作る。このため、同じ配信行を戻す状態遷移ではなく、ユースケースの事後条件として扱う。

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 作成直後。ディスパッチャーが Job を関連付けるのを待つ |
| in_flight | — | Job を関連付け、Jobs 側の再試行を含めて配信中である |
| succeeded | terminal | 下流への反映が完了した |
| dead_letter | terminal | 配信できずに終了した。管理者の再試行は新しい `pending` の行を作る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | ProvisioningDeliveryStarted | — | in_flight |  |
| in_flight | UserProvisioned | source_type == 'user' | succeeded |  |
| in_flight | UserDeprovisioned | source_type == 'user' | succeeded |  |
| in_flight | GroupPushed | source_type == 'group' | succeeded |  |
| in_flight | GroupMembershipPushed | source_type == 'group' | succeeded |  |
| in_flight | UserProvisioningFailed | — | dead_letter |  |
