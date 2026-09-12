# Feature: Provisioning のシナリオ

## Rule: REQ-PROVISIONING-001 管理 API クライアントは Provisioning スコープの範囲でだけ接続と配信を操作できる

Primary actor: `ManagementApiClient`

### Example: EX-PROVISIONING-001-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがアプリケーションの接続、テナントの接続、または配信操作を要求する
- Then `provisioning:read` スコープはアプリケーションの接続とテナントの接続一覧の参照だけを許可する
- Then `provisioning:write` スコープは接続の変更と配信操作の実行だけを許可する

### Example: EX-PROVISIONING-001-02 `provisioning:read` だけで接続の変更または配信操作を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがアプリケーションの接続、テナントの接続、または配信操作を要求する
- But `provisioning:read` だけで接続の変更または配信操作を要求する
- Then 操作は `AccessDeniedError` で拒否される

### Example: EX-PROVISIONING-001-03 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- When クライアントがアプリケーションの接続、テナントの接続、または配信操作を要求する
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作は `AccessDeniedError` で拒否される

## Rule: REQ-PROVISIONING-002 管理者は接続を登録し、接続テストで下流の対応機能を取得できる

Primary actor: `TenantAdministrator`

### Example: EX-PROVISIONING-002-01 通常経路

- Given Application "app-1" は存在し ProvisioningConnection を持たない
- When 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- Then ProvisioningConnectionRegistered が発行される
- When 管理者が TestProvisioningConnection を実行する
- Then 下流 /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる

### Example: EX-PROVISIONING-002-02 base_url が https でない、または内部/リンクローカル IP を指す

- Given Application "app-1" は存在し ProvisioningConnection を持たない
- When 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- But base_url が https でない、または内部/リンクローカル IP を指す
- Then InvalidRequestError が返り ProvisioningConnection は作成されない

### Example: EX-PROVISIONING-002-03 Application "app-1" に既に ProvisioningConnection が存在する

- Given Application "app-1" は存在し ProvisioningConnection を持たない
- When 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- But Application "app-1" に既に ProvisioningConnection が存在する
- Then ProvisioningConnectionAlreadyExistsError が返る

## Rule: REQ-PROVISIONING-003 割り当て済みユーザーの作成は下流への作成として配信される

Primary actor: `System`

### Example: EX-PROVISIONING-003-01 通常経路

- Given テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- And User "ユーザー-1" は Application "app-1" に割り当て済みである
- And IdManagement が User "ユーザー-1" の作成を commit し、配信捕捉のポートを同一トランザクションで呼んでいる
- When 捕捉した変更を Provisioning が処理する
- Then `ProvisioningDelivery`（`operation=create`、`status=pending`）が発火元と同じトランザクションで作成されている
- Then dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- Then `worker` プロセスが下流へ POST し、`UserProvisioned` が発行され、配信のステータスが `succeeded` になる

### Example: EX-PROVISIONING-003-02 User がこの Application に割り当て済みでない (scope=assigned_only)

- Given テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- And User "ユーザー-1" は Application "app-1" に割り当て済みである
- And IdManagement が User "ユーザー-1" の作成を commit し、配信捕捉のポートを同一トランザクションで呼んでいる
- When 捕捉した変更を Provisioning が処理する
- Then User がこの Application に割り当て済みでない (scope=assigned_only)
- Then ProvisioningDelivery は作成されない

## Rule: REQ-PROVISIONING-004 ユーザーの無効化は下流への無効化として配信される

Primary actor: `System`

### Example: EX-PROVISIONING-004-01 通常経路

- Given User "ユーザー-1" は下流に既に存在し RemoteResourceLink を持つ
- And IdManagement が User "ユーザー-1" の disable を commit し、配信捕捉のポートを呼んでいる
- When 捕捉した変更を Provisioning が処理する
- Then ProvisioningDelivery (operation=deactivate) が作成される
- Then `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

## Rule: REQ-PROVISIONING-005 Application からの割り当て解除は既定で下流の無効化として配信される

Primary actor: `System`

### Example: EX-PROVISIONING-005-01 通常経路

- Given User "ユーザー-1" は Application "app-1" に割り当て済みで下流に存在する
- And DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- And Application が "ユーザー-1" の "app-1" への割り当て解除を commit し、配信捕捉のポートを呼んでいる
- When 捕捉した変更を Provisioning が処理する
- Then ProvisioningDelivery (operation=deactivate) が作成される
- Then `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

## Rule: REQ-PROVISIONING-006 ユーザーの削除は猶予期間の経過後に下流への削除として配信される

Primary actor: `System`

### Example: EX-PROVISIONING-006-01 通常経路

- Given DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- And User "ユーザー-1" は下流に存在する
- When User "ユーザー-1" が削除される
- Then 7日間は下流へ delete が配信されない
- Then grace_period_days の経過後に purge の ProvisioningDelivery (operation=delete) が作成される
- Then `worker` プロセスが下流へ DELETE を送り、`UserDeprovisioned`（`action=delete`）が発行される

### Example: EX-PROVISIONING-006-02 猶予期間内に User "ユーザー-1" が Application "app-1" へ再び割り当てられる

- Given DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- And User "ユーザー-1" は下流に存在する
- When User "ユーザー-1" が削除される
- Then 7日間は下流へ delete が配信されない
- Then 猶予期間内に User "ユーザー-1" が Application "app-1" へ再び割り当てられる
- Then 予約されていた purge の ProvisioningDelivery は取り消される

## Rule: REQ-PROVISIONING-007 下流の 409 衝突は既存リソースへの関連付けとして解決する

Primary actor: `System`

### Example: EX-PROVISIONING-007-01 通常経路

- Given User "ユーザー-1" 向けの ProvisioningDelivery (operation=create) が in_flight である
- And 下流に同じ userName を持つ resource が既に存在し externalId が未設定である
- When `worker` プロセスが下流へ POST し、HTTP 409 Conflict を受け取る
- Then `worker` プロセスが `conflict_match_attribute`（`userName`）で既存リソースを検索し、`RemoteResourceLink` を作成する
- Then 以後の配信はこの RemoteResourceLink を使い PATCH で更新する
- Then UserProvisioned が発行され配信が succeeded になる

## Rule: REQ-PROVISIONING-008 下流で消失したリソースは 404 の検出後に再作成する

Primary actor: `System`

### Example: EX-PROVISIONING-008-01 通常経路

- Given User "ユーザー-1" の RemoteResourceLink が存在するが下流の resource は削除されている
- When `worker` プロセスが下流へ PATCH し、HTTP 404 Not Found を受け取る
- Then `worker` プロセスが下流へ新たに POST し、`RemoteResourceLink.remote_id` を更新する
- Then UserProvisioned が発行され配信が succeeded になる

## Rule: REQ-PROVISIONING-009 下流の一時的な 429 と 5xx はバックオフして再試行し、復旧後に収束する

Primary actor: `System`

### Example: EX-PROVISIONING-009-01 通常経路

- Given ProvisioningDelivery が in_flight である
- When `worker` プロセスが配信を試み、下流が一時的に HTTP 429（`Retry-After` あり）を返す
- Then `worker` プロセスは `Retry-After` に従って待機時間を延ばし、Jobs での再試行を予定する
- Then `ProvisioningDelivery.status` は `in_flight` のまま保持される
- When 下流の復旧後に次の attempt を実行する
- Then 配信が成功し、`UserProvisioned` が発行され、ステータスは `succeeded` になる

## Rule: REQ-PROVISIONING-010 試行上限を超えた配信は dead_letter となり、管理者が手動で再試行できる

Primary actor: `System`

### Example: EX-PROVISIONING-010-01 通常経路

- Given ProvisioningDelivery が in_flight で Jobs.Job の attempts が max_attempts に達している
- When 最終 attempt も失敗する
- Then UserProvisioningFailed が発行され配信が dead_letter になる
- When 管理者が RetryProvisioningDelivery を呼ぶ
- Then 配信が pending に戻り job_id がクリアされる

### Example: EX-PROVISIONING-010-02 配信が dead_letter でない (pending/in_flight/succeeded)

- Given ProvisioningDelivery が in_flight で Jobs.Job の attempts が max_attempts に達している
- When 最終 attempt も失敗する
- Then UserProvisioningFailed が発行され配信が dead_letter になる
- When 管理者が RetryProvisioningDelivery を呼ぶ
- But 配信が dead_letter でない (pending/in_flight/succeeded)
- Then ProvisioningDeliveryNotRetryableError が返る

## Rule: REQ-PROVISIONING-011 誤削除ガードの閾値を超えると接続を隔離し、管理者が解除する

Primary actor: `TenantAdministrator`

### Example: EX-PROVISIONING-011-01 通常経路

- Given accidental_deletion_count_threshold=5 が設定されている
- And 1 回の resync で deactivate/delete 対象が 5 件を超える
- When full resync が閾値を超える deprovision アクションを検出する
- Then deprovision アクションを実行せず ConnectionQuarantined が発行される
- Then connection.health が quarantined になり notification_email へ通知される
- When 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
- Then ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る

### Example: EX-PROVISIONING-011-02 connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ

- Given accidental_deletion_count_threshold=5 が設定されている
- And 1 回の resync で deactivate/delete 対象が 5 件を超える
- When full resync が閾値を超える deprovision アクションを検出する
- Then deprovision アクションを実行せず ConnectionQuarantined が発行される
- Then connection.health が quarantined になり notification_email へ通知される
- When 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
- But connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ
- Then InvalidRequestError相当の拒否として動作せず対象が無いため何も変化しない (requires が resource.health=quarantined を要求し拒否する)

## Rule: REQ-PROVISIONING-012 管理者は On-Demand Provision で 1 人のユーザーを試験配信できる

Primary actor: `TenantAdministrator`

### Example: EX-PROVISIONING-012-01 通常経路

- Given User "ユーザー-1" は connection の scope 内である
- When 管理者が ProvisionOnDemand を実行する
- Then `ProvisioningDelivery`（`status=pending`）が直ちに作成される

### Example: EX-PROVISIONING-012-02 指定した subject が scope 外である (scope=assigned_only で未割り当て)

- Given User "ユーザー-1" は connection の scope 内である
- When 管理者が ProvisionOnDemand を実行する
- But 指定した subject が scope 外である (scope=assigned_only で未割り当て)
- Then ProvisioningSubjectNotInScopeError が返る

## Rule: REQ-PROVISIONING-013 管理者は Full Resync で適用範囲の全対象を収束できる

Primary actor: `TenantAdministrator`

### Example: EX-PROVISIONING-013-01 通常経路

- Given connection の scope 内に複数 subject が存在し一部は下流と乖離している
- When 管理者が StartFullResync を実行する
- Then scope 内の全 subject に対して ProvisioningDelivery が作成される
- Then すべての配信が収束すると FullResyncCompleted が発行される

## Rule: REQ-PROVISIONING-014 資格情報のローテーションは監査でき、`rotated_at` が更新される

Primary actor: `TenantAdministrator`

### Example: EX-PROVISIONING-014-01 通常経路

- Given connection は bearer_token 認証で稼働している
- When 管理者が UpdateProvisioningConnection に新しい credential を渡す
- Then ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- Then 以後の配信は新しい credential で下流へ認証する

## Rule: REQ-PROVISIONING-015 他テナントの接続と配信はテナント境界を越えない

Primary actor: `System`

### Example: EX-PROVISIONING-015-01 通常経路

- Given テナント "tenant-a" の ProvisioningConnection と ProvisioningDelivery が存在する
- When テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningDelivery を同じ id/delivery_id で呼ぶ
- Then ProvisioningConnectionNotFoundError または ProvisioningDeliveryNotFoundError が返る

## Rule: REQ-PROVISIONING-016 同じ冪等キーの重複した配信は既存行に収束する

Primary actor: `System`

### Example: EX-PROVISIONING-016-01 通常経路

- Given (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningDelivery が既に存在する
- When 同じライフサイクルイベントが at-least-once 配信によって再び捕捉される（ディスパッチャーの重複実行または再送を模す）
- Then 新規 ProvisioningDelivery は作成されず既存行がそのまま使われる

## Rule: REQ-PROVISIONING-017 捕捉直後のキュー投入に失敗しても、定期ディスパッチャーが未関連付けの配信を回収する

Primary actor: `System`

### Example: EX-PROVISIONING-017-01 通常経路

- Given IdManagement のコミットと同じトランザクションで `ProvisioningDelivery`（`job_id` 未設定、`status=pending`）が確定したが、API プロセスから Jobs への即時投入には失敗した
- When `worker` プロセスの定期ディスパッチャーが、`job_id` を関連付けていない `pending` の配信を再走査する
- Then dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- Then `ProvisioningDeliveryStarted` が発行され、`worker` プロセスが配信を実行する

## Rule: REQ-PROVISIONING-018 必須の属性マッピングを解決できない配信はフェイルクローズで失敗する

Primary actor: `System`

### Example: EX-PROVISIONING-018-01 通常経路

- Given AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- When `worker` プロセスが配信前に必須属性を解決する
- Then 解決に失敗し、下流へは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)
