# Provisioning Scenarios

### REQ-PROVISIONING-001: 管理 API クライアントは Provisioning スコープの範囲でだけ接続と配信を操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- WHEN クライアントがアプリケーションの接続、テナントの接続、または配信操作を要求する
  - ALT `provisioning:read` だけで接続の変更または配信操作を要求する → 操作は `AccessDeniedError` で拒否される
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作は `AccessDeniedError` で拒否される
- THEN `provisioning:read` スコープはアプリケーションの接続とテナントの接続一覧の参照だけを許可する
- THEN `provisioning:write` スコープは接続の変更と配信操作の実行だけを許可する

### REQ-PROVISIONING-002: 管理者は接続を登録し、接続テストで下流の対応機能を取得できる
- ACTOR TenantAdministrator
- GIVEN Application "app-1" は存在し ProvisioningConnection を持たない
- WHEN 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
  - ALT base_url が https でない、または内部/リンクローカル IP を指す → InvalidRequestError が返り ProvisioningConnection は作成されない
  - ALT Application "app-1" に既に ProvisioningConnection が存在する → ProvisioningConnectionAlreadyExistsError が返る
- THEN ProvisioningConnectionRegistered が発行される
- WHEN 管理者が TestProvisioningConnection を実行する
- THEN 下流 /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる

### REQ-PROVISIONING-003: 割り当て済みユーザーの作成は下流への作成として配信される
- ACTOR System
- GIVEN テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- GIVEN User "ユーザー-1" は Application "app-1" に割り当て済みである
- GIVEN IdManagement が User "ユーザー-1" の作成を commit し、配信捕捉のポートを同一トランザクションで呼んでいる
- WHEN 捕捉した変更を Provisioning が処理する
- THEN `ProvisioningDelivery`（`operation=create`、`status=pending`）が発火元と同じトランザクションで作成されている
  - ALT User がこの Application に割り当て済みでない (scope=assigned_only) → ProvisioningDelivery は作成されない
- THEN dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- THEN `worker` プロセスが下流へ POST し、`UserProvisioned` が発行され、配信のステータスが `succeeded` になる

### REQ-PROVISIONING-004: ユーザーの無効化は下流への無効化として配信される
- ACTOR System
- GIVEN User "ユーザー-1" は下流に既に存在し RemoteResourceLink を持つ
- GIVEN IdManagement が User "ユーザー-1" の disable を commit し、配信捕捉のポートを呼んでいる
- WHEN 捕捉した変更を Provisioning が処理する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

### REQ-PROVISIONING-005: Application からの割り当て解除は既定で下流の無効化として配信される
- ACTOR System
- GIVEN User "ユーザー-1" は Application "app-1" に割り当て済みで下流に存在する
- GIVEN DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- GIVEN Application が "ユーザー-1" の "app-1" への割り当て解除を commit し、配信捕捉のポートを呼んでいる
- WHEN 捕捉した変更を Provisioning が処理する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

### REQ-PROVISIONING-006: ユーザーの削除は猶予期間の経過後に下流への削除として配信される
- ACTOR System
- GIVEN DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- GIVEN User "ユーザー-1" は下流に存在する
- WHEN User "ユーザー-1" が削除される
- THEN 7日間は下流へ delete が配信されない
- THEN grace_period_days の経過後に purge の ProvisioningDelivery (operation=delete) が作成される
  - ALT 猶予期間内に User "ユーザー-1" が Application "app-1" へ再び割り当てられる → 予約されていた purge の ProvisioningDelivery は取り消される
- THEN `worker` プロセスが下流へ DELETE を送り、`UserDeprovisioned`（`action=delete`）が発行される

### REQ-PROVISIONING-007: 下流の 409 衝突は既存リソースへの関連付けとして解決する
- ACTOR System
- GIVEN User "ユーザー-1" 向けの ProvisioningDelivery (operation=create) が in_flight である
- GIVEN 下流に同じ userName を持つ resource が既に存在し externalId が未設定である
- WHEN `worker` プロセスが下流へ POST し、HTTP 409 Conflict を受け取る
- THEN `worker` プロセスが `conflict_match_attribute`（`userName`）で既存リソースを検索し、`RemoteResourceLink` を作成する
- THEN 以後の配信はこの RemoteResourceLink を使い PATCH で更新する
- THEN UserProvisioned が発行され配信が succeeded になる

### REQ-PROVISIONING-008: 下流で消失したリソースは 404 の検出後に再作成する
- ACTOR System
- GIVEN User "ユーザー-1" の RemoteResourceLink が存在するが下流の resource は削除されている
- WHEN `worker` プロセスが下流へ PATCH し、HTTP 404 Not Found を受け取る
- THEN `worker` プロセスが下流へ新たに POST し、`RemoteResourceLink.remote_id` を更新する
- THEN UserProvisioned が発行され配信が succeeded になる

### REQ-PROVISIONING-009: 下流の一時的な 429 と 5xx はバックオフして再試行し、復旧後に収束する
- ACTOR System
- GIVEN ProvisioningDelivery が in_flight である
- WHEN `worker` プロセスが配信を試み、下流が一時的に HTTP 429（`Retry-After` あり）を返す
- THEN `worker` プロセスは `Retry-After` に従って待機時間を延ばし、Jobs での再試行を予定する
- THEN `ProvisioningDelivery.status` は `in_flight` のまま保持される
- WHEN 下流の復旧後に次の attempt を実行する
- THEN 配信が成功し、`UserProvisioned` が発行され、ステータスは `succeeded` になる

### REQ-PROVISIONING-010: 試行上限を超えた配信は dead_letter となり、管理者が手動で再試行できる
- ACTOR System
- GIVEN ProvisioningDelivery が in_flight で Jobs.Job の attempts が max_attempts に達している
- WHEN 最終 attempt も失敗する
- THEN UserProvisioningFailed が発行され配信が dead_letter になる
- WHEN 管理者が RetryProvisioningDelivery を呼ぶ
  - ALT 配信が dead_letter でない (pending/in_flight/succeeded) → ProvisioningDeliveryNotRetryableError が返る
- THEN 配信が pending に戻り job_id がクリアされる

### REQ-PROVISIONING-011: 誤削除ガードの閾値を超えると接続を隔離し、管理者が解除する
- ACTOR TenantAdministrator
- GIVEN accidental_deletion_count_threshold=5 が設定されている
- GIVEN 1 回の resync で deactivate/delete 対象が 5 件を超える
- WHEN full resync が閾値を超える deprovision アクションを検出する
- THEN deprovision アクションを実行せず ConnectionQuarantined が発行される
- THEN connection.health が quarantined になり notification_email へ通知される
- WHEN 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
  - ALT connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ → InvalidRequestError相当の拒否として動作せず対象が無いため何も変化しない (requires が resource.health=quarantined を要求し拒否する)
- THEN ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る

### REQ-PROVISIONING-012: 管理者は On-Demand Provision で 1 人のユーザーを試験配信できる
- ACTOR TenantAdministrator
- GIVEN User "ユーザー-1" は connection の scope 内である
- WHEN 管理者が ProvisionOnDemand を実行する
  - ALT 指定した subject が scope 外である (scope=assigned_only で未割り当て) → ProvisioningSubjectNotInScopeError が返る
- THEN `ProvisioningDelivery`（`status=pending`）が直ちに作成される

### REQ-PROVISIONING-013: 管理者は Full Resync で適用範囲の全対象を収束できる
- ACTOR TenantAdministrator
- GIVEN connection の scope 内に複数 subject が存在し一部は下流と乖離している
- WHEN 管理者が StartFullResync を実行する
- THEN scope 内の全 subject に対して ProvisioningDelivery が作成される
- THEN すべての配信が収束すると FullResyncCompleted が発行される

### REQ-PROVISIONING-014: 資格情報のローテーションは監査でき、`rotated_at` が更新される
- ACTOR TenantAdministrator
- GIVEN connection は bearer_token 認証で稼働している
- WHEN 管理者が UpdateProvisioningConnection に新しい credential を渡す
- THEN ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- THEN 以後の配信は新しい credential で下流へ認証する

### REQ-PROVISIONING-015: 他テナントの接続と配信はテナント境界を越えない
- ACTOR System
- GIVEN テナント "tenant-a" の ProvisioningConnection と ProvisioningDelivery が存在する
- WHEN テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningDelivery を同じ id/delivery_id で呼ぶ
- THEN ProvisioningConnectionNotFoundError または ProvisioningDeliveryNotFoundError が返る

### REQ-PROVISIONING-016: 同じ冪等キーの重複した配信は既存行に収束する
- ACTOR System
- GIVEN (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningDelivery が既に存在する
- WHEN 同じライフサイクルイベントが at-least-once 配信によって再び捕捉される（ディスパッチャーの重複実行または再送を模す）
- THEN 新規 ProvisioningDelivery は作成されず既存行がそのまま使われる

### REQ-PROVISIONING-017: 捕捉直後のキュー投入に失敗しても、定期ディスパッチャーが未関連付けの配信を回収する
- ACTOR System
- GIVEN IdManagement のコミットと同じトランザクションで `ProvisioningDelivery`（`job_id` 未設定、`status=pending`）が確定したが、API プロセスから Jobs への即時投入には失敗した
- WHEN `worker` プロセスの定期ディスパッチャーが、`job_id` を関連付けていない `pending` の配信を再走査する
- THEN dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- THEN `ProvisioningDeliveryStarted` が発行され、`worker` プロセスが配信を実行する

### REQ-PROVISIONING-018: 必須の属性マッピングを解決できない配信はフェイルクローズで失敗する
- ACTOR System
- GIVEN AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- WHEN `worker` プロセスが配信前に必須属性を解決する
- THEN 解決に失敗し、下流へは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)
