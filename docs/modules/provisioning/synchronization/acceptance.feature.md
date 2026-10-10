# Feature: 同期の例

## Rule: REQ-PROVISIONING-003 割り当て済みユーザーの作成は下流への作成としてプロビジョニングされる

### Example: EX-PROVISIONING-003-01 通常経路

- Given テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- And User "ユーザー-1" は Application "app-1" に割り当て済みである
- And IdManagement が User "ユーザー-1" の作成を commit した後で、イベント同期のポートを呼んでいる
- When Provisioning がイベント同期で受け取った変更を処理する
- Then `ProvisioningTask`（`operation=create`、`status=pending`）が作成されている
- Then dispatcher が Jobs.Job を関連付け ProvisioningTaskStarted が発行される
- Then `worker` プロセスが下流へ POST し、`UserProvisioned` が発行され、プロビジョニングタスクのステータスが `succeeded` になる

### Example: EX-PROVISIONING-003-02 User がこの Application に割り当て済みでない (scope=assigned_only)

- Given テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- And User "ユーザー-1" は Application "app-1" に割り当て済みである
- And IdManagement が User "ユーザー-1" の作成を commit した後で、イベント同期のポートを呼んでいる
- When Provisioning がイベント同期で受け取った変更を処理する
- Then User がこの Application に割り当て済みでない (scope=assigned_only)
- Then ProvisioningTask は作成されない

## Rule: REQ-PROVISIONING-004 ユーザーの無効化は下流への無効化としてプロビジョニングされる

### Example: EX-PROVISIONING-004-01 通常経路

- Given User "ユーザー-1" は下流に既に存在し RemoteResourceLink を持つ
- And IdManagement が User "ユーザー-1" の disable を commit し、イベント同期のポートを呼んでいる
- When Provisioning がイベント同期で受け取った変更を処理する
- Then ProvisioningTask (operation=deactivate) が作成される
- Then `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

## Rule: REQ-PROVISIONING-005 Application からの割り当て解除はデフォルトで下流の無効化としてプロビジョニングされる

### Example: EX-PROVISIONING-005-01 通常経路

- Given User "ユーザー-1" は Application "app-1" に割り当て済みで下流に存在する
- And DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- And Application が "ユーザー-1" の "app-1" への割り当て解除を commit し、イベント同期のポートを呼んでいる
- When Provisioning がイベント同期で受け取った変更を処理する
- Then ProvisioningTask (operation=deactivate) が作成される
- Then `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

## Rule: REQ-PROVISIONING-006 ユーザーの削除は猶予期間の経過後に下流への削除としてプロビジョニングされる

### Example: EX-PROVISIONING-006-01 通常経路

- Given DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- And User "ユーザー-1" は下流に存在する
- When User "ユーザー-1" が削除される
- Then 7日間は下流へ delete は送られない
- Then grace_period_days の経過後に purge の ProvisioningTask (operation=delete) が作成される
- Then `worker` プロセスが下流へ DELETE を送り、`UserDeprovisioned`（`action=delete`）が発行される

### Example: EX-PROVISIONING-006-02 猶予期間内に User "ユーザー-1" が Application "app-1" へ再び割り当てられる

- Given DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- And User "ユーザー-1" は下流に存在する
- When User "ユーザー-1" が削除される
- Then 7日間は下流へ delete は送られない
- Then 猶予期間内に User "ユーザー-1" が Application "app-1" へ再び割り当てられる
- Then 予約されていた purge の ProvisioningTask は取り消される

### Example: EX-PROVISIONING-006-03 猶予期間内に User "ユーザー-1" が復元される

- Given DeprovisionPolicy.on_delete=delete, grace_period_days=7 の接続 "app-1" と "app-2" がある
- And User "ユーザー-1" は下流に存在する
- When User "ユーザー-1" が削除される
- Then 猶予期間内に User "ユーザー-1" が復元され、再び有効になる
- Then "app-1" と "app-2" の削除の予約はどちらも取り消され、期限の経過後も下流へ DELETE は送られない

## Rule: REQ-PROVISIONING-011 誤削除ガードの閾値を超えると接続を隔離し、管理者が解除する

### Example: EX-PROVISIONING-011-01 通常経路

- Given accidental_deletion_count_threshold=5 が設定されている
- And 1 回のインクリメンタル同期で deactivate/delete 対象が 5 件を超える
- When インクリメンタル同期が閾値を超える deprovision アクションを計画する
- Then deprovision アクションを実行せず ConnectionQuarantined が発行される
- Then connection.health が quarantined になり notification_email へ通知される
- When 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
- Then ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る

### Example: EX-PROVISIONING-011-02 connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ

- Given accidental_deletion_count_threshold=5 が設定されている
- And 1 回のインクリメンタル同期で deactivate/delete 対象が 5 件を超える
- When インクリメンタル同期が閾値を超える deprovision アクションを計画する
- Then deprovision アクションを実行せず ConnectionQuarantined が発行される
- Then connection.health が quarantined になり notification_email へ通知される
- When 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
- But connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ
- Then InvalidRequestError相当の拒否として動作せず対象が無いため何も変化しない (requires が resource.health=quarantined を要求し拒否する)

### Example: EX-PROVISIONING-011-03 割合の閾値を超える

- Given accidental_deletion_percent_threshold=50 が設定され、件数の閾値は設定されていない
- And 下流へ反映済みの User 10 人のうち 6 人が、イベント同期を通らずにスコープ外になっている
- When インクリメンタル同期が接続を処理する
- Then インクリメンタル同期はプロビジョニングタスクを 1 件も作らず、下流へは何も送られない
- Then ConnectionQuarantined が発行され connection.health が quarantined になる

### Example: EX-PROVISIONING-011-04 閾値ちょうどの deprovision は通常どおり実行する

- Given accidental_deletion_count_threshold=5 が設定されている
- And 下流へ反映済みの User のうち 5 人が、イベント同期を通らずにスコープ外になっている
- When インクリメンタル同期が接続を処理する
- Then インクリメンタル同期は 5 人の deactivate のプロビジョニングタスクを作る
- Then connection.health は ok のままで、ConnectionQuarantined は発行されない

## Rule: REQ-PROVISIONING-016 同じ冪等キーの重複したプロビジョニングタスクは既存のレコードに収束する

### Example: EX-PROVISIONING-016-01 通常経路

- Given (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningTask が既に存在する
- When 同じライフサイクルイベントが at-least-once の配送によって再びイベント同期へ渡される（ディスパッチャーの重複実行または再送を模す）
- Then 新規 ProvisioningTask は作成されず既存のレコードがそのまま使われる

## Rule: REQ-PROVISIONING-017 イベント同期直後のキュー投入に失敗しても、定期ディスパッチャーが未関連付けのプロビジョニングタスクを回収する

### Example: EX-PROVISIONING-017-01 通常経路

- Given イベント同期が `ProvisioningTask`（`job_id` 未設定、`status=pending`）を確定したが、API プロセスから Jobs への即時投入には失敗した
- When `worker` プロセスの定期ディスパッチャーが、`job_id` を関連付けていない `pending` のプロビジョニングタスクを再走査する
- Then dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- Then `ProvisioningTaskStarted` が発行され、`worker` プロセスがプロビジョニングタスクを実行する

## Rule: REQ-PROVISIONING-019 インクリメンタル同期は、あるべき状態と下流へ反映済みの状態の差分をプロビジョニングタスクにする

### Example: EX-PROVISIONING-019-01 通常経路

- Given テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- And User "ユーザー-1" は "app-1" に割り当て済みで有効だが、RemoteResourceLink がない
- When `worker` プロセスのインクリメンタル同期が接続 "app-1" を処理する
- Then `operation=create` の ProvisioningTask が作成される

### Example: EX-PROVISIONING-019-02 割り当てを解除された User のリンクが残っている

- Given User "ユーザー-1" は下流で有効な RemoteResourceLink を持つ
- And "ユーザー-1" の "app-1" への割り当ては解除されている
- And DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- When `worker` プロセスのインクリメンタル同期が接続 "app-1" を処理する
- Then `operation=deactivate` の ProvisioningTask が作成される

### Example: EX-PROVISIONING-019-03 未完了のプロビジョニングタスクがある

- Given User "ユーザー-1" は "app-1" に割り当て済みで有効だが、RemoteResourceLink がない
- And "ユーザー-1" には `pending` の ProvisioningTask が既にある
- When `worker` プロセスのインクリメンタル同期が接続 "app-1" を処理する
- Then 新しい ProvisioningTask は作成されない

### Example: EX-PROVISIONING-019-04 猶予期間つきの削除

- Given 接続 "app-1" の DeprovisionPolicy は on_delete=delete、grace_period_days=7 である
- And User "ユーザー-1" は削除済みで、下流で有効な RemoteResourceLink を持つ
- When `worker` プロセスのインクリメンタル同期が接続 "app-1" を処理する
- Then `operation=delete` の ProvisioningTask は作成されない
- Then 猶予期間の経過後に削除する予約が作成される
