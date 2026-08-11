# Provisioning Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-PROVISIONING-001: management API clientはProvisioning scope内のconnectionとdeliveryだけを操作できる
- Actor: ManagementApiClient
- Given: client は対象 tenant の有効な API access token を提示している
- Then: provisioning:read scope で application connection と tenant connection 一覧を参照できる
- Then: provisioning:write scope で connection の変更と delivery action を実行できる
- Alternative (provisioning:read だけで connection 変更または delivery action を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant と request tenant が一致しない): 操作は AccessDeniedError で拒否される

### REQ-PROVISIONING-002: 管理者はProvisioningConnectionを登録し接続テストでcapabilitiesを取得できる
- Actor: TenantAdministrator
- Given: Application "app-1" は存在し ProvisioningConnection を持たない
- Then: 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
- Then: ProvisioningConnectionRegistered が発行される
- Then: 管理者が TestProvisioningConnection を実行する
- Then: 下流 /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる
- Alternative (base_url が https でない、または内部/リンクローカル IP を指す): InvalidRequestError が返り ProvisioningConnection は作成されない
- Alternative (Application "app-1" に既に ProvisioningConnection が存在する): ProvisioningConnectionAlreadyExistsError が返る

### REQ-PROVISIONING-003: 割当済みユーザーの作成が下流へcreateとして配送される
- Actor: System
- Given: テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- Given: User "user-1" は Application "app-1" に割当済みである
- Then: IdManagement が User "user-1" を作成し commit する
- Then: 同一トランザクションで ProvisioningDelivery (operation=create, status=pending) が作成される
- Then: dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- Then: worker が下流へ POST し UserProvisioned が発行され配送が succeeded になる
- Alternative (User がこの Application に割当済みでない (scope=assigned_only)): ProvisioningDelivery は作成されない

### REQ-PROVISIONING-004: ユーザーのdisableがdeactivateとして下流へ配送される
- Actor: System
- Given: User "user-1" は下流に既に存在し RemoteResourceLink を持つ
- Then: IdManagement が User "user-1" を disable する
- Then: ProvisioningDelivery (operation=deactivate) が作成される
- Then: worker が下流へ active=false の PATCH を送り UserDeprovisioned が発行される

### REQ-PROVISIONING-005: Applicationからのunassignは既定でdeactivateとして配送される
- Actor: System
- Given: User "user-1" は Application "app-1" に割当済みで下流に存在する
- Given: DeprovisionPolicy.on_unassign は既定値 deactivate のままである
- Then: 管理者が User "user-1" の Application "app-1" への割当を解除する
- Then: ProvisioningDelivery (operation=deactivate) が作成される
- Then: worker が下流へ active=false の PATCH を送り UserDeprovisioned が発行される

### REQ-PROVISIONING-006: ユーザー削除はgrace_period_days経過後にdeleteとして配送される
- Actor: System
- Given: DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- Given: User "user-1" は下流に存在する
- Then: User "user-1" が削除される
- Then: 7日間は下流へ delete が配送されない
- Then: grace_period_days の経過後に purge の ProvisioningDelivery (operation=delete) が作成される
- Then: worker が下流へ DELETE を送り UserDeprovisioned (action=delete) が発行される
- Alternative (猶予期間内に User "user-1" が Application "app-1" へ再割当される): 予約されていた purge の ProvisioningDelivery は取り消される

### REQ-PROVISIONING-007: 下流の409衝突は既存resourceをexternalIdまたはmatch属性で養子縁組する
- Actor: System
- Given: User "user-1" 向けの ProvisioningDelivery (operation=create) が in_flight である
- Given: 下流に同じ userName を持つ resource が既に存在し externalId が未設定である
- Then: worker が下流へ POST し 409 Conflict を受け取る
- Then: worker が conflict_match_attribute (userName) で既存 resource を検索し RemoteResourceLink を作成する
- Then: 以後の配送はこの RemoteResourceLink を使い PATCH で更新する
- Then: UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-008: 下流で消失したresourceは404検出後に再作成される
- Actor: System
- Given: User "user-1" の RemoteResourceLink が存在するが下流の resource は削除されている
- Then: worker が下流へ PATCH し 404 Not Found を受け取る
- Then: worker が新規に下流へ POST し RemoteResourceLink の remote_id を更新する
- Then: UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-009: 下流の一時的な429と5xxはbackoffで再試行され復旧後に収束する
- Actor: System
- Given: ProvisioningDelivery が in_flight で下流が一時的に 429 (Retry-After あり) を返す
- Then: worker が Retry-After に従い backoff し Jobs レベルで再試行する
- Then: ProvisioningDelivery.status は in_flight のまま保持される
- Then: 下流が復旧すると次の attempt が成功し UserProvisioned が発行され succeeded になる

### REQ-PROVISIONING-010: max_attempts超過でdead_letterになり管理者は手動retryできる
- Actor: System
- Given: ProvisioningDelivery が in_flight で Jobs.Job の attempts が max_attempts に達している
- Then: 最終 attempt も失敗し UserProvisioningFailed が発行され配送が dead_letter になる
- Then: 管理者が RetryProvisioningDelivery を呼ぶ
- Then: 配送が pending に戻り job_id がクリアされる
- Alternative (配送が dead_letter でない (pending/in_flight/succeeded)): ProvisioningDeliveryNotRetryableError が返る

### REQ-PROVISIONING-011: 誤削除ガードの閾値超過でconnectionがquarantineされ管理者が解除する
- Actor: TenantAdministrator
- Given: accidental_deletion_count_threshold=5 が設定されている
- Given: 1 回の resync で deactivate/delete 対象が 5 件を超える
- Then: full resync がこれらの deprovision アクションを実行せず ConnectionQuarantined が発行される
- Then: connection.health が quarantined になり notification_email へ通知される
- Then: 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
- Then: ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る
- Alternative (connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ): InvalidRequestError相当の拒否として動作せず対象が無いため何も変化しない (requires が resource.health=quarantined を要求し拒否する)

### REQ-PROVISIONING-012: 管理者はon-demand provisionで単一ユーザーを試験配送できる
- Actor: TenantAdministrator
- Given: User "user-1" は connection の scope 内である
- Then: 管理者が ProvisionOnDemand を実行する
- Then: ProvisioningDelivery (status=pending) が即時に作成される
- Alternative (指定した subject が scope 外である (scope=assigned_only で未割当)): ProvisioningSubjectNotInScopeError が返る

### REQ-PROVISIONING-013: 管理者はfull resyncでscope内subject全件を収束できる
- Actor: TenantAdministrator
- Given: connection の scope 内に複数 subject が存在し一部は下流と乖離している
- Then: 管理者が StartFullResync を実行する
- Then: scope 内の全 subject に対して ProvisioningDelivery が作成される
- Then: すべての配送が収束すると FullResyncCompleted が発行される

### REQ-PROVISIONING-014: credentialのrotationは監査可能でありrotated_atが更新される
- Actor: TenantAdministrator
- Given: connection は bearer_token 認証で稼働している
- Then: 管理者が UpdateProvisioningConnection に新しい credential を渡す
- Then: ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- Then: 以後の配送は新しい credential で下流へ認証する

### REQ-PROVISIONING-015: 他テナントのProvisioningConnectionと配送は越境しない
- Actor: System
- Given: テナント "tenant-a" の ProvisioningConnection と ProvisioningDelivery が存在する
- Then: テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningDelivery を同じ id/delivery_id で呼ぶ
- Then: ProvisioningConnectionNotFoundError または ProvisioningDeliveryNotFoundError が返る

### REQ-PROVISIONING-016: 同一idempotencyキーの重複配送は既存行へのno-opになる
- Actor: System
- Given: (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningDelivery が既に存在する
- Then: 同じ lifecycle event が at-least-once 配信で再度 capture される (dispatcher の重複実行や再送を模す)
- Then: 新規 ProvisioningDelivery は作成されず既存行がそのまま使われる

### REQ-PROVISIONING-017: APIprocessのcapture直後のenqueueが失敗してもperiodicなdispatcherが未関連付けのdeliveryを回収する
- Actor: System
- Given: IdManagement の commit と同一トランザクションで ProvisioningDelivery (job_id 未設定、status=pending) が確定したが、API process の即時 enqueue 呼び出しには失敗した
- Then: worker process の periodic dispatcher が job_id 未関連付けの pending delivery を再走査する
- Then: dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- Then: ProvisioningDeliveryStarted が発行され worker が配送を実行する

### REQ-PROVISIONING-018: requiredな属性マッピングが解決不能な配送はfail-closedで失敗する
- Actor: System
- Given: AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- Then: worker が配送前に required 属性の解決を試み失敗する
- Then: 下流へは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)

### REQ-PROVISIONING-019: RegisterProvisioningConnection
管理者が Application に ProvisioningConnection を新規登録する。属性マッピングは SCIM core User の既定を自動 seed する。
- Precondition: is_safe_outbound_url(input.base_url)
- Precondition: credential_matches_auth_method(input.credential)
- Postcondition: output.connection.application_id == input.id
- Postcondition: output.connection.status == "active"
- Postcondition: emitted.exists(e, e is ProvisioningConnectionRegistered)

### REQ-PROVISIONING-020: GetProvisioningConnection
管理者が Application の ProvisioningConnection を取得する。別テナントは未存在として扱う。

### REQ-PROVISIONING-021: UpdateProvisioningConnection
管理者が ProvisioningConnection の設定を更新する。credential を含む更新は
credential rotation として扱い ProvisioningCredentialRotated も発行する。
- Precondition: input.base_url == null || is_safe_outbound_url(input.base_url)
- Precondition: input.credential == null || credential_matches_auth_method(input.credential)
- Postcondition: input.credential == null || emitted.exists(e, e is ProvisioningCredentialRotated)
- Postcondition: input.status != "disabled" || emitted.exists(e, e is ProvisioningConnectionDisabled)

### REQ-PROVISIONING-022: DeleteProvisioningConnection
管理者が ProvisioningConnection を削除する (hard delete)。既存の RemoteResourceLink・ProvisioningDelivery は履歴として残す。

### REQ-PROVISIONING-023: TestProvisioningConnection
管理者が下流 /ServiceProviderConfig への到達性を確認し capabilities を再取得してキャッシュを更新する。

### REQ-PROVISIONING-024: ProvisionOnDemand
管理者が単一 subject を指定して即時に試験配送する ProvisioningDelivery を作成する。
- Postcondition: output.delivery.status == "pending"
- Postcondition: output.delivery.source_id == input.subject_id

### REQ-PROVISIONING-025: StartFullResync
管理者が connection の scope 内 subject 全件を再走査し収束させる full resync を開始する。完了は非同期で FullResyncCompleted が発行される。

### REQ-PROVISIONING-026: ListProvisioningDeliveries
管理者が connection の配送状況を created_at 降順 (同値は id で tie-break) の双方向 keyset
pagination で一覧する。status/source_type で絞り込める。cursor は応答の Link response header
(rel="prev" / rel="next") から取得する。

### REQ-PROVISIONING-027: GetProvisioningDelivery
管理者が配送 1 件の詳細を取得する。

### REQ-PROVISIONING-028: RetryProvisioningDelivery
管理者が dead_letter の配送を手動で再試行する。pending へ戻し dispatcher の再走査対象にする。
- Precondition: resource.status == "dead_letter"
- Postcondition: output.delivery.status == "pending"
- Postcondition: output.delivery.job_id == null

### REQ-PROVISIONING-029: ResumeProvisioningConnection
管理者が quarantine された ProvisioningConnection を解除し配送を再開する。運用手順に quarantine 解除の経路が明記されていなかったため、本 interface を追加した。
- Precondition: resource.health == "quarantined"
- Postcondition: output.connection.health != "quarantined"
- Postcondition: emitted.exists(e, e is ProvisioningConnectionQuarantineCleared)

### REQ-PROVISIONING-030: ListTenantProvisioningConnections
管理者がテナント内で有効な ProvisioningConnection を横断的に一覧する (テナント設定の read-only 集約ビュー)。書き込みは各 Application 側の interface へ誘導する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ProvisioningConnection | Application 1 件に対して最大 1 件だけ存在する outbound provisioning の設定。接続先 base_url・認証・機能トグル・scope・属性マッピング・deprovision policy・信頼性設定を束ねる。 | connection, 接続 |
| RemoteResourceLink | idmagic の User/Group と、下流 SCIM service provider 上の resource (remote id・externalId・etag) の相関を保持する entity。409 (既存衝突) は match 属性で養子縁組し、404 (消失) は再作成して更新する。 | remote link, 相関 |
| ProvisioningDelivery | 内部の lifecycle event 1 件を下流へ反映するための配送単位。idempotency key (tenant_id, connection_id, source_type, source_id, source_version) で重複 enqueue を no-op にする。実行は Jobs 上の Job に委譲する。 | delivery, 配送 |
| Deprovision | unassign・disable・delete という内部の lifecycle event を、下流での deactivate・delete・none のいずれかへ翻訳すること。翻訳規則は DeprovisionPolicy が持つ。 |  |
| Grace Period | delete アクションを即時実行せず、DeprovisionPolicy.grace_period_days の経過後に purge する猶予期間。期間内に対象が scope へ再度入れば取り消す。 | grace period, 猶予期間 |
| Accidental Deletion Guard | 1 回の sync で deactivate/delete 対象が accidental_deletion_count_threshold を超えたら実行せず connection を quarantine する誤削除防止策。 | 誤削除ガード |
| Quarantine | 連続失敗または誤削除ガード超過により配送を停止した ProvisioningConnection.health の状態。管理者が ResumeProvisioningConnection で解除するまで再開しない。 | quarantine, 隔離 |
| On-Demand Provision | 管理者が単一 subject を指定し、即時に試験配送する手動運用。 |  |
| Full Resync | 管理者が connection の scope 内 subject 全件を再走査し、下流の状態を idmagic の状態へ収束させる手動運用。 |  |
| Mirror | 下流 SaaS 上の resource が idmagic を真実源とする射影であり、下流での手動変更は次回配送で上書きされること。 |  |
| Push Groups | ProvisioningFeatureFlags.push_groups が有効なとき、Group と membership を下流へ配送する機能。 |  |
| System | Provisioning の配送エンジンそのもの。人間の操作者を伴わない技術的主体を指す。 |  |

## State machines

### ProvisioningDeliveryLifecycle

ProvisioningDelivery.status の状態機械。pending で作成され、dispatcher が Jobs.Job を
関連付けて in_flight に遷移する。Jobs レベルの attempt retry 中は in_flight のまま保持し
(WorkflowRunLifecycle と同じ扱い)、succeeded または dead_letter で終端する。dead_letter からは
RetryProvisioningDelivery が pending へ戻す (管理者操作であり本状態機械の transition としては
表さない — 新しい delivery 行の再作成に相当する運用のため RetryProvisioningDelivery の ensures で扱う)。

Initial: `pending`  
Terminal: `succeeded`, `dead_letter`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | ProvisioningDeliveryStarted | "" | in_flight |  |
| in_flight | UserProvisioned | source_type == 'user' | succeeded |  |
| in_flight | UserDeprovisioned | source_type == 'user' | succeeded |  |
| in_flight | GroupPushed | source_type == 'group' | succeeded |  |
| in_flight | GroupMembershipPushed | source_type == 'group' | succeeded |  |
| in_flight | UserProvisioningFailed | "" | dead_letter |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
