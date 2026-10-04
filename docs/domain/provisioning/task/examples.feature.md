# Feature: プロビジョニングタスクの実行の例

## Rule: REQ-PROVISIONING-007 下流の 409 衝突は既存リソースへの関連付けとして解決する

### Example: EX-PROVISIONING-007-01 通常経路

- Given User "ユーザー-1" 向けの ProvisioningTask (operation=create) が in_flight である
- And 下流に同じ userName を持つ resource が既に存在し externalId が未設定である
- When `worker` プロセスが下流へ POST し、HTTP 409 Conflict を受け取る
- Then `worker` プロセスが `conflict_match_attribute`（`userName`）で既存リソースを検索し、`RemoteResourceLink` を作成する
- Then 以後のプロビジョニングタスクはこの RemoteResourceLink を使い PATCH で更新する
- Then UserProvisioned が発行されプロビジョニングタスクが succeeded になる

## Rule: REQ-PROVISIONING-008 下流で消失したリソースは 404 の検出後に再作成する

### Example: EX-PROVISIONING-008-01 通常経路

- Given User "ユーザー-1" の RemoteResourceLink が存在するが下流の resource は削除されている
- When `worker` プロセスが下流へ PATCH し、HTTP 404 Not Found を受け取る
- Then `worker` プロセスが下流へ新たに POST し、`RemoteResourceLink.remote_id` を更新する
- Then UserProvisioned が発行されプロビジョニングタスクが succeeded になる

## Rule: REQ-PROVISIONING-009 下流の一時的な 429 と 5xx はバックオフして再試行し、復旧後に収束する

### Example: EX-PROVISIONING-009-01 通常経路

- Given ProvisioningTask が in_flight である
- When `worker` プロセスがプロビジョニングタスクを実行し、下流が一時的に HTTP 429（`Retry-After` あり）を返す
- Then `worker` プロセスは `Retry-After` に従って待機時間を延ばし、Jobs での再試行を予定する
- Then `ProvisioningTask.status` は `in_flight` のまま保持される
- When 下流の復旧後に次の attempt を実行する
- Then プロビジョニングタスクが成功し、`UserProvisioned` が発行され、ステータスは `succeeded` になる

## Rule: REQ-PROVISIONING-010 試行上限を超えたプロビジョニングタスクは dead_letter となり、管理者が手動で再試行できる

### Example: EX-PROVISIONING-010-01 通常経路

- Given ProvisioningTask が in_flight で Jobs.Job の attempts が max_attempts に達している
- When 最終 attempt も失敗する
- Then UserProvisioningFailed が発行されプロビジョニングタスクが dead_letter になる
- When 管理者が RetryProvisioningTask を呼ぶ
- Then プロビジョニングタスクが pending に戻り job_id がクリアされる

### Example: EX-PROVISIONING-010-02 プロビジョニングタスクが dead_letter でない (pending/in_flight/succeeded)

- Given ProvisioningTask が in_flight で Jobs.Job の attempts が max_attempts に達している
- When 最終 attempt も失敗する
- Then UserProvisioningFailed が発行されプロビジョニングタスクが dead_letter になる
- When 管理者が RetryProvisioningTask を呼ぶ
- But プロビジョニングタスクが dead_letter でない (pending/in_flight/succeeded)
- Then ProvisioningTaskNotRetryableError が返る

## Rule: REQ-PROVISIONING-018 必須の属性マッピングを解決できないプロビジョニングタスクはフェイルクローズで失敗する

### Example: EX-PROVISIONING-018-01 通常経路

- Given AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- When `worker` プロセスがプロビジョニングタスクの実行前に必須属性を解決する
- Then 解決に失敗し、下流へは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)
