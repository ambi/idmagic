---
context: provisioning
updated_at: 2026-08-11
---

# Provisioning Specification

## Overview

idmagic を SCIM 2.0 client にした outbound provisioning (下流 SaaS への
user/group push lifecycle management) を所有する。真実源は idmagic 側の User/Group、
下流は mirror。connection は Application 1 件に対し最大 1 件、scope は既存
ApplicationAssignment を再利用する。Scim は inbound server 専用のままで、本 context とは
protocol を話す点以外ほぼ重ならない別 context。

The `Provisioning` context owns outbound delivery of identity changes to downstream SaaS targets —
the push counterpart to `Sourcing`'s pull/receive side. It is named for the capability
(`Provisioning`), not a direction word, so it stays correct regardless of how the inbound side is
later taxonomized; direction, authority (idmagic is source of record), and vocabulary invert
between the two contexts, so they do not share code beyond published references to `Tenancy` /
`Application` / `IdManagement` / `Jobs`.

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

## State Transitions

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

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Protocol-agnostic core with protocol feature slices

Most outbound behavior — the connection envelope, `DeprovisionPolicy`, `AttributeMappingRule`,
and the `ProvisioningDelivery`/`RemoteResourceLink` delivery engine (queueing, retry/backoff,
quarantine, ordering, resync) — does not depend on the wire protocol. Only the wire client and
some connection setup (auth method, capability discovery, default attribute schema) are
protocol-specific. The context root (`domain`, `ports`, `usecases`, `handlers_http`) therefore
holds the protocol-agnostic core, and each protocol gets its own feature slice implementing the
`ProvisioningTargetClient` port — currently `client_scim`, with `entraid`/`googledir` expected to
follow as siblings without touching the core. This is a deliberate variant of the repo's usual
"fat slice, thin shared root" convention: here the domain shape is mostly protocol-neutral with
protocol as the driven-adapter axis, so the core is fat and the slices are thin.

No shared SCIM wire kernel exists between this context's `client_scim` slice and `Sourcing`'s
`scim` slice: inbound's filter parser/evaluator and fixed response structs serve a receiver that
evaluates incoming SCIM against its own data, while outbound needs to *build* filter strings and
serialize a broader, mapping-driven attribute set (`externalId`, enterprise extensions). The
actual overlap (discovery structs, RFC schema URNs) is small enough that sharing now would
constrain both sides prematurely; extraction is deferred to when real duplication appears.

### Same-transaction delivery capture

Delivery does not observe the existing `outbox`/Relay drain — that path only forwards to external
transports (Kafka/PubSub/log) with no in-process consumer, and its topic registration is
incomplete and non-transactional for the events this context needs. Instead, `Provisioning`
implements a published capture port that `IdManagement`'s user-mutation path and `Application`'s
assignment path call inside their own Postgres transaction, inserting one `pending`
`ProvisioningDelivery` row per matching active connection. This mirrors the same-transaction
capture pattern established for lifecycle-workflow triggers: the User/assignment commit and the
delivery row become durable atomically, so `ProvisioningDelivery` itself is this context's outbox
equivalent, without depending on the shared outbox's atomicity. Delivery idempotency uses the key
`(tenant, connection, source_type, source_id, source_version)`.

### Design Decisions

- `Provisioning` is split out as its own bounded context — named for the capability rather than a
  direction word — with a protocol-agnostic core (connection, mapping, delivery engine) and thin
  per-protocol feature slices (`client_scim` today, `entraid`/`googledir` to follow) implementing
  `ProvisioningTargetClient`, rather than a shared SCIM wire kernel with the inbound side
  ([ADR-128](../../../decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md)).
- `Sourcing` (inbound identity intake) does not mirror `Provisioning`'s protocol-agnostic core,
  because unlike outbound delivery it has no shared engine already implemented to extract into one
  ([ADR-141](../../../decisions/ADR-141-inbound-identity-sourcing-taxonomy.md)).
- The transactional trigger-capture pattern — writing the triggering mutation and its queued
  follow-up row in the same transaction as the record context's commit — originated for
  lifecycle-workflow triggers in IdManagement and was preserved when that ownership moved to
  IdGovernance; Provisioning reuses the same pattern for its own delivery capture
  ([ADR-113](../../../decisions/ADR-113-identity-lifecycle-workflow-execution-model.md),
  [ADR-117](../../../decisions/ADR-117-extract-identity-governance-context.md)).

## Scenarios

### REQ-PROVISIONING-001: management API clientはProvisioning scope内のconnectionとdeliveryだけを操作できる
- ACTOR ManagementApiClient
- GIVEN client は対象 tenant の有効な API access token を提示している
- WHEN client が application connection、tenant connection、または delivery action の操作を要求する
  - ALT provisioning:read だけで connection 変更または delivery action を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant と request tenant が一致しない → 操作は AccessDeniedError で拒否される
- THEN provisioning:read scope は application connection と tenant connection 一覧の参照だけを許可する
- THEN provisioning:write scope は connection の変更と delivery action の実行だけを許可する

### REQ-PROVISIONING-002: 管理者はProvisioningConnectionを登録し接続テストでcapabilitiesを取得できる
- ACTOR TenantAdministrator
- GIVEN Application "app-1" は存在し ProvisioningConnection を持たない
- WHEN 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
  - ALT base_url が https でない、または内部/リンクローカル IP を指す → InvalidRequestError が返り ProvisioningConnection は作成されない
  - ALT Application "app-1" に既に ProvisioningConnection が存在する → ProvisioningConnectionAlreadyExistsError が返る
- THEN ProvisioningConnectionRegistered が発行される
- WHEN 管理者が TestProvisioningConnection を実行する
- THEN 下流 /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる

### REQ-PROVISIONING-003: 割当済みユーザーの作成が下流へcreateとして配送される
- ACTOR System
- GIVEN テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- GIVEN User "user-1" は Application "app-1" に割当済みである
- WHEN IdManagement が User "user-1" を作成し commit する
- THEN 同一トランザクションで ProvisioningDelivery (operation=create, status=pending) が作成される
  - ALT User がこの Application に割当済みでない (scope=assigned_only) → ProvisioningDelivery は作成されない
- THEN dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- THEN worker が下流へ POST し UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-004: ユーザーのdisableがdeactivateとして下流へ配送される
- ACTOR System
- GIVEN User "user-1" は下流に既に存在し RemoteResourceLink を持つ
- WHEN IdManagement が User "user-1" を disable する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN worker が下流へ active=false の PATCH を送り UserDeprovisioned が発行される

### REQ-PROVISIONING-005: Applicationからのunassignは既定でdeactivateとして配送される
- ACTOR System
- GIVEN User "user-1" は Application "app-1" に割当済みで下流に存在する
- GIVEN DeprovisionPolicy.on_unassign は既定値 deactivate のままである
- WHEN 管理者が User "user-1" の Application "app-1" への割当を解除する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN worker が下流へ active=false の PATCH を送り UserDeprovisioned が発行される

### REQ-PROVISIONING-006: ユーザー削除はgrace_period_days経過後にdeleteとして配送される
- ACTOR System
- GIVEN DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- GIVEN User "user-1" は下流に存在する
- WHEN User "user-1" が削除される
- THEN 7日間は下流へ delete が配送されない
- THEN grace_period_days の経過後に purge の ProvisioningDelivery (operation=delete) が作成される
  - ALT 猶予期間内に User "user-1" が Application "app-1" へ再割当される → 予約されていた purge の ProvisioningDelivery は取り消される
- THEN worker が下流へ DELETE を送り UserDeprovisioned (action=delete) が発行される

### REQ-PROVISIONING-007: 下流の409衝突は既存resourceをexternalIdまたはmatch属性で養子縁組する
- ACTOR System
- GIVEN User "user-1" 向けの ProvisioningDelivery (operation=create) が in_flight である
- GIVEN 下流に同じ userName を持つ resource が既に存在し externalId が未設定である
- WHEN worker が下流へ POST し 409 Conflict を受け取る
- THEN worker が conflict_match_attribute (userName) で既存 resource を検索し RemoteResourceLink を作成する
- THEN 以後の配送はこの RemoteResourceLink を使い PATCH で更新する
- THEN UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-008: 下流で消失したresourceは404検出後に再作成される
- ACTOR System
- GIVEN User "user-1" の RemoteResourceLink が存在するが下流の resource は削除されている
- WHEN worker が下流へ PATCH し 404 Not Found を受け取る
- THEN worker が新規に下流へ POST し RemoteResourceLink の remote_id を更新する
- THEN UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-009: 下流の一時的な429と5xxはbackoffで再試行され復旧後に収束する
- ACTOR System
- GIVEN ProvisioningDelivery が in_flight である
- WHEN worker が配送を試み、下流が一時的に 429 (Retry-After あり) を返す
- THEN worker は Retry-After に従って backoff し、Jobs レベルの再試行を予定する
- THEN ProvisioningDelivery.status は in_flight のまま保持される
- WHEN 下流の復旧後に次の attempt を実行する
- THEN 配送が成功し、UserProvisioned が発行され status は succeeded になる

### REQ-PROVISIONING-010: max_attempts超過でdead_letterになり管理者は手動retryできる
- ACTOR System
- GIVEN ProvisioningDelivery が in_flight で Jobs.Job の attempts が max_attempts に達している
- WHEN 最終 attempt も失敗する
- THEN UserProvisioningFailed が発行され配送が dead_letter になる
- WHEN 管理者が RetryProvisioningDelivery を呼ぶ
  - ALT 配送が dead_letter でない (pending/in_flight/succeeded) → ProvisioningDeliveryNotRetryableError が返る
- THEN 配送が pending に戻り job_id がクリアされる

### REQ-PROVISIONING-011: 誤削除ガードの閾値超過でconnectionがquarantineされ管理者が解除する
- ACTOR TenantAdministrator
- GIVEN accidental_deletion_count_threshold=5 が設定されている
- GIVEN 1 回の resync で deactivate/delete 対象が 5 件を超える
- WHEN full resync が閾値を超える deprovision アクションを検出する
- THEN deprovision アクションを実行せず ConnectionQuarantined が発行される
- THEN connection.health が quarantined になり notification_email へ通知される
- WHEN 管理者が原因を確認したうえで ResumeProvisioningConnection を呼ぶ
  - ALT connection.health が quarantined でない状態で ResumeProvisioningConnection を呼ぶ → InvalidRequestError相当の拒否として動作せず対象が無いため何も変化しない (requires が resource.health=quarantined を要求し拒否する)
- THEN ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る

### REQ-PROVISIONING-012: 管理者はon-demand provisionで単一ユーザーを試験配送できる
- ACTOR TenantAdministrator
- GIVEN User "user-1" は connection の scope 内である
- WHEN 管理者が ProvisionOnDemand を実行する
  - ALT 指定した subject が scope 外である (scope=assigned_only で未割当) → ProvisioningSubjectNotInScopeError が返る
- THEN ProvisioningDelivery (status=pending) が即時に作成される

### REQ-PROVISIONING-013: 管理者はfull resyncでscope内subject全件を収束できる
- ACTOR TenantAdministrator
- GIVEN connection の scope 内に複数 subject が存在し一部は下流と乖離している
- WHEN 管理者が StartFullResync を実行する
- THEN scope 内の全 subject に対して ProvisioningDelivery が作成される
- THEN すべての配送が収束すると FullResyncCompleted が発行される

### REQ-PROVISIONING-014: credentialのrotationは監査可能でありrotated_atが更新される
- ACTOR TenantAdministrator
- GIVEN connection は bearer_token 認証で稼働している
- WHEN 管理者が UpdateProvisioningConnection に新しい credential を渡す
- THEN ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- THEN 以後の配送は新しい credential で下流へ認証する

### REQ-PROVISIONING-015: 他テナントのProvisioningConnectionと配送は越境しない
- ACTOR System
- GIVEN テナント "tenant-a" の ProvisioningConnection と ProvisioningDelivery が存在する
- WHEN テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningDelivery を同じ id/delivery_id で呼ぶ
- THEN ProvisioningConnectionNotFoundError または ProvisioningDeliveryNotFoundError が返る

### REQ-PROVISIONING-016: 同一idempotencyキーの重複配送は既存行へのno-opになる
- ACTOR System
- GIVEN (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningDelivery が既に存在する
- WHEN 同じ lifecycle event が at-least-once 配信で再度 capture される (dispatcher の重複実行や再送を模す)
- THEN 新規 ProvisioningDelivery は作成されず既存行がそのまま使われる

### REQ-PROVISIONING-017: APIprocessのcapture直後のenqueueが失敗してもperiodicなdispatcherが未関連付けのdeliveryを回収する
- ACTOR System
- GIVEN IdManagement の commit と同一トランザクションで ProvisioningDelivery (job_id 未設定、status=pending) が確定したが、API process の即時 enqueue 呼び出しには失敗した
- WHEN worker process の periodic dispatcher が job_id 未関連付けの pending delivery を再走査する
- THEN dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- THEN ProvisioningDeliveryStarted が発行され worker が配送を実行する

### REQ-PROVISIONING-018: requiredな属性マッピングが解決不能な配送はfail-closedで失敗する
- ACTOR System
- GIVEN AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- WHEN worker が配送前に required 属性を解決する
- THEN 解決に失敗し、下流へは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)
