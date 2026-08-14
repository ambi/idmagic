---
context: provisioning
updated_at: 2026-08-11
---

# Provisioning Specification

## Overview

idmagic を SCIM 2.0 クライアントとして動かす外向きのプロビジョニング、すなわち下流 SaaS へのユーザーとグループの反映およびライフサイクル管理を所有する。情報の正は idmagic 側の User と Group であり、下流はその写しである。接続は Application 1 件につき最大 1 件とし、対象範囲には既存の ApplicationAssignment を再利用する。Scim は内向きのサーバー専用のままであり、プロトコル以外の関心事がほとんど重ならない別の Context とする。

`Sourcing` が外部から受け取るのに対し、`Provisioning` は外部へ送り出す。両者では処理の向き、情報の権威（idmagic が記録の正であるかどうか）、語彙が反転するため、`Tenancy`、`Application`、`IdManagement`、`Jobs` への公開された参照を除いてコードを共有しない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ProvisioningConnection | Application 1 件に対して最大 1 件だけ存在する outbound provisioning の設定。接続先 base_url・認証・機能トグル・スコープ属性マッピング・deprovision ポリシー・信頼性設定を束ねる。 | connection, 接続 |
| RemoteResourceLink | idmagic の `User` または `Group` と、下流の SCIM サービスプロバイダー上のリソース（リモート ID、`externalId`、`etag`）との対応を保持するエンティティ。HTTP 409（既存リソースとの衝突）では照合属性を使って既存リソースへ関連付け、HTTP 404（リソースの消失）では再作成して関連付けを更新する。 | remote link, 相関 |
| ProvisioningDelivery | 内部のライフサイクルイベント 1 件を下流へ反映するための配信単位。冪等キー（`tenant_id`、`connection_id`、`source_type`、`source_id`、`source_version`）により、重複する投入を no-op とする。実行は Jobs の `Job` に委譲する。 | delivery, 配信 |
| Deprovision | 割り当て解除、無効化、削除という内部のライフサイクルイベントを、下流に対する無効化、削除、無操作のいずれかへ変換すること。変換規則は `DeprovisionPolicy` が持つ。 |  |
| Grace Period | 削除操作を直ちに実行せず、`DeprovisionPolicy.grace_period_days` の経過後に完全削除するまでの猶予期間。期間内に対象が適用範囲へ戻れば取り消す。 | grace period, 猶予期間 |
| Accidental Deletion Guard | 1 回の sync で deactivate/delete 対象が accidental_deletion_count_threshold を超えたら実行せず connection を quarantine する誤削除防止策。 | 誤削除ガード |
| Quarantine | 連続失敗または誤削除ガード超過により配送を停止した ProvisioningConnection.health の状態。管理者が ResumeProvisioningConnection で解除するまで再開しない。 | quarantine, 隔離 |
| On-Demand Provision | 管理者が単一 subject を指定し、即時に試験配送する手動運用。 |  |
| Full Resync | 管理者が接続の適用範囲に含まれるすべての対象を再走査し、下流の状態を idmagic の状態へ収束させる手動運用。 |  |
| Mirror | ダウンストリーム SaaS 上の resource が idmagic を真実源とする射影であり、ダウンストリームでの手動変更は次回配送で上書きされること。 |  |
| Push Groups | ProvisioningFeatureFlags.push_groups が有効なとき、Group と membership をダウンストリームへ配送する機能。 |  |
| System | Provisioning の配送エンジンそのもの。人間の操作者を伴わない技術的主体を指す。 |  |

## State Transitions

### ProvisioningDeliveryLifecycle

`ProvisioningDelivery.status` の状態遷移を表す。`pending` で作成し、ディスパッチャーが Jobs の `Job` を関連付けると `in_flight` に遷移する。Jobs 側で再試行している間は `in_flight` のまま保持し、`succeeded` または `dead_letter` で終了する。`RetryProvisioningDelivery` は、管理者の操作によって `dead_letter` の配信を基に新しい `pending` の配信行を作る。このため、同じ配信行を戻す状態遷移ではなく、ユースケースの事後条件として扱う。

Initial: `pending` Terminal: `succeeded`, `dead_letter`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | ProvisioningDeliveryStarted | — | in_flight |  |
| in_flight | UserProvisioned | source_type == 'user' | succeeded |  |
| in_flight | UserDeprovisioned | source_type == 'user' | succeeded |  |
| in_flight | GroupPushed | source_type == 'group' | succeeded |  |
| in_flight | GroupMembershipPushed | source_type == 'group' | succeeded |  |
| in_flight | UserProvisioningFailed | — | dead_letter |  |

## Authorization Boundary

認可の意味づけはアプリケーションとそのテストが強制する。本仕様は API の認証を記録するが、ポリシーの DSL は意図的に定義しない。ポリシーの言語を採用する前に、別の work item で Cedar を評価する。

## Design

### Protocol-agnostic core with protocol feature slices

外向きの振る舞いの大半、すなわち接続のエンベロープ、`DeprovisionPolicy`、`AttributeMappingRule`、`ProvisioningDelivery` と `RemoteResourceLink` による配信処理（キュー、再試行と間隔、隔離、順序、再同期）はプロトコルに依存しない。プロトコル固有なのは通信クライアントと接続設定の一部（認証方式、能力の探索、デフォルト属性スキーマ）だけである。したがって Context のルート (`domain`、`ports`、`usecase`、`handlers_http`) がプロトコル非依存の中核を持ち、プロトコルごとに `ProvisioningTargetClient` ポートを実装する機能単位を設ける。現在は `client_scim` があり、将来は中核に触れず `entraid` と `googledir` を同階層へ追加する想定である。これは、このリポジトリで一般的な「厚い機能単位、薄い共通ルート」を意図的に反転した形である。ここではドメインの形がほぼプロトコルに依存せず、プロトコルが駆動される側のアダプターの軸になるため、中核が厚く機能単位が薄い。

この Context の `client_scim` 機能と `Sourcing` の `scim` 機能の間には、共有する SCIM 通信中核を置かない。内向き側のフィルター構文の解析と評価、および固定レスポンスの構造体は、受信した SCIM リクエストを自身のデータに対して評価する側のためのものである。一方、外向き側はフィルター文字列を組み立て、マッピングに基づくより広い属性集合（`externalId`、Enterprise 拡張）を直列化する必要がある。実際に重なる部分（Discovery の構造体、RFC が定めるスキーマ URN）は十分に小さく、現時点で共有すると双方を早すぎる段階で結合することになる。本当の重複が現れるまで切り出しを先送りする。

### Same-transaction delivery capture

配信処理は、共通の `outbox` テーブルを外部の Kafka、Pub/Sub、ログへ転送するイベントリレーを利用しない。そのリレーは `IdManagement` や `Application` の書き込みトランザクション内で `ProvisioningDelivery` を作成できないためである。代わりに `Provisioning` は配信捕捉用の公開ポートを提供し、`IdManagement` のユーザー変更処理と `Application` の割り当て変更処理が、それぞれ自身の PostgreSQL トランザクション内でこのポートを呼び出す。ポートは一致する有効な接続ごとに `pending` の `ProvisioningDelivery` 行を 1 件挿入する。発火元の変更と配信行は同時にコミットまたはロールバックされるため、`ProvisioningDelivery` がこの Context 専用の Transactional Outbox となる。配信の冪等性には `(tenant, connection, source_type, source_id, source_version)` をキーとして使う。

### Design Decisions

- `Provisioning` は独立した Bounded Context として切り出す。名前は向きを表す語ではなく能力から取る。プロトコルに依存しない中核 (接続、対応付け、配送の処理系) と、`ProvisioningTargetClient` を実装するプロトコルごとの薄い機能スライス (現在は `client_scim`、続いて `entraid` と `googledir`) を持ち、内向き側と SCIM の通信の核を共有することはしない。
- `Sourcing` (内向きのアイデンティティの取り込み) は `Provisioning` のプロトコルに依存しない中核を写さない。外向きの配送と違い、切り出す対象になる共通の処理系が既に実装されているわけではないからである。
- `User` やアプリケーション割り当ての変更を保存するユースケースは、Provisioning が公開する捕捉ポートを同じ PostgreSQL トランザクション内で呼び出し、対応する `ProvisioningDelivery` を `pending` で保存する。変更と配信行は共にコミットまたはロールバックされるため、変更だけが確定して配信予定が失われることはない。`ProvisioningDelivery` は Provisioning 固有の Transactional Outbox として機能する。

## Scenarios

### REQ-PROVISIONING-001: management API クライアントはProvisioning スコープのconnectionとdeliveryだけを操作できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API access トークンを提示している
- WHEN クライアントがアプリケーションの接続、テナントの接続、または配信操作を要求する
  - ALT `provisioning:read` だけで接続の変更または配信操作を要求する → 操作は `AccessDeniedError` で拒否される
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作は `AccessDeniedError` で拒否される
- THEN `provisioning:read` スコープはアプリケーションの接続とテナントの接続一覧の参照だけを許可する
- THEN `provisioning:write` スコープは接続の変更と配信操作の実行だけを許可する

### REQ-PROVISIONING-002: 管理者はProvisioningConnectionを登録し接続テストでcapabilitiesを取得できる
- ACTOR TenantAdministrator
- GIVEN Application "app-1" は存在し ProvisioningConnection を持たない
- WHEN 管理者が RegisterProvisioningConnection を https の base_url と bearer_token で実行する
  - ALT base_url が https でない、または内部/リンクローカル IP を指す → InvalidRequestError が返り ProvisioningConnection は作成されない
  - ALT Application "app-1" に既に ProvisioningConnection が存在する → ProvisioningConnectionAlreadyExistsError が返る
- THEN ProvisioningConnectionRegistered が発行される
- WHEN 管理者が TestProvisioningConnection を実行する
- THEN ダウンストリーム /ServiceProviderConfig への到達性が確認され capabilities がキャッシュされる

### REQ-PROVISIONING-003: 割当済みユーザーの作成がダウンストリームへcreateとして配送される
- ACTOR System
- GIVEN テナント "tenant-a" の Application "app-1" に有効な ProvisioningConnection (scope=assigned_only, create_users=true) が存在する
- GIVEN User "ユーザー-1" は Application "app-1" に割当済みである
- WHEN IdManagement が User "ユーザー-1" を作成し commit する
- THEN 同一トランザクションで `ProvisioningDelivery`（`operation=create`、`status=pending`）が作成される
  - ALT User がこの Application に割当済みでない (scope=assigned_only) → ProvisioningDelivery は作成されない
- THEN dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- THEN `worker` プロセスが下流へ POST し、`UserProvisioned` が発行され、配信のステータスが `succeeded` になる

### REQ-PROVISIONING-004: ユーザーのdisableがdeactivateとしてダウンストリームへ配送される
- ACTOR System
- GIVEN User "ユーザー-1" はダウンストリームに既に存在し RemoteResourceLink を持つ
- WHEN IdManagement が User "ユーザー-1" を disable する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

### REQ-PROVISIONING-005: Applicationからのunassignはデフォルトでdeactivateとして配送される
- ACTOR System
- GIVEN User "ユーザー-1" は Application "app-1" に割当済みでダウンストリームに存在する
- GIVEN DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- WHEN 管理者が User "ユーザー-1" の Application "app-1" への割当を解除する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

### REQ-PROVISIONING-006: ユーザー削除はgrace_period_days経過後にdeleteとして配送される
- ACTOR System
- GIVEN DeprovisionPolicy.on_delete=delete, grace_period_days=7 が設定されている
- GIVEN User "ユーザー-1" はダウンストリームに存在する
- WHEN User "ユーザー-1" が削除される
- THEN 7日間はダウンストリームへ delete が配送されない
- THEN grace_period_days の経過後に purge の ProvisioningDelivery (operation=delete) が作成される
  - ALT 猶予期間内に User "ユーザー-1" が Application "app-1" へ再割当される → 予約されていた purge の ProvisioningDelivery は取り消される
- THEN `worker` プロセスが下流へ DELETE を送り、`UserDeprovisioned`（`action=delete`）が発行される

### REQ-PROVISIONING-007: ダウンストリームの409衝突は既存リソースexternalIdまたはmatch属性で養子縁組する
- ACTOR System
- GIVEN User "ユーザー-1" 向けの ProvisioningDelivery (operation=create) が in_flight である
- GIVEN ダウンストリームに同じ userName を持つ resource が既に存在し externalId が未設定である
- WHEN `worker` プロセスが下流へ POST し、HTTP 409 Conflict を受け取る
- THEN `worker` プロセスが `conflict_match_attribute`（`userName`）で既存リソースを検索し、`RemoteResourceLink` を作成する
- THEN 以後の配送はこの RemoteResourceLink を使い PATCH で更新する
- THEN UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-008: ダウンストリームで消失したリソース404検出後に再作成される
- ACTOR System
- GIVEN User "ユーザー-1" の RemoteResourceLink が存在するがダウンストリームの resource は削除されている
- WHEN `worker` プロセスが下流へ PATCH し、HTTP 404 Not Found を受け取る
- THEN `worker` プロセスが下流へ新たに POST し、`RemoteResourceLink.remote_id` を更新する
- THEN UserProvisioned が発行され配送が succeeded になる

### REQ-PROVISIONING-009: ダウンストリームの一時的な429と5xxはbackoffで再試行され復旧後に収束する
- ACTOR System
- GIVEN ProvisioningDelivery が in_flight である
- WHEN `worker` プロセスが配信を試み、下流が一時的に HTTP 429（`Retry-After` あり）を返す
- THEN `worker` プロセスは `Retry-After` に従って待機時間を延ばし、Jobs での再試行を予定する
- THEN `ProvisioningDelivery.status` は `in_flight` のまま保持される
- WHEN ダウンストリームの復旧後に次の attempt を実行する
- THEN 配信が成功し、`UserProvisioned` が発行され、ステータスは `succeeded` になる

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
- GIVEN User "ユーザー-1" は connection の scope 内である
- WHEN 管理者が ProvisionOnDemand を実行する
  - ALT 指定した subject が scope 外である (scope=assigned_only で未割当) → ProvisioningSubjectNotInScopeError が返る
- THEN `ProvisioningDelivery`（`status=pending`）が直ちに作成される

### REQ-PROVISIONING-013: 管理者はfull resyncでスコープsubject全件を収束できる
- ACTOR TenantAdministrator
- GIVEN connection の scope 内に複数 subject が存在し一部はダウンストリームと乖離している
- WHEN 管理者が StartFullResync を実行する
- THEN scope 内の全 subject に対して ProvisioningDelivery が作成される
- THEN すべての配送が収束すると FullResyncCompleted が発行される

### REQ-PROVISIONING-014: credentialのrotationは監査可能でありrotated_atが更新される
- ACTOR TenantAdministrator
- GIVEN connection は bearer_token 認証で稼働している
- WHEN 管理者が UpdateProvisioningConnection に新しい credential を渡す
- THEN ProvisioningCredentialRotated が発行され credential.rotated_at が更新される
- THEN 以後の配送は新しい credential でダウンストリームへ認証する

### REQ-PROVISIONING-015: 他テナントのProvisioningConnectionと配送は越境しない
- ACTOR System
- GIVEN テナント "tenant-a" の ProvisioningConnection と ProvisioningDelivery が存在する
- WHEN テナント "tenant-b" の管理者が GetProvisioningConnection または GetProvisioningDelivery を同じ id/delivery_id で呼ぶ
- THEN ProvisioningConnectionNotFoundError または ProvisioningDeliveryNotFoundError が返る

### REQ-PROVISIONING-016: 同一idempotencyキーの重複配送は既存行へのno-opになる
- ACTOR System
- GIVEN (tenant_id, connection_id, source_type, source_id, source_version) が一致する ProvisioningDelivery が既に存在する
- WHEN 同じライフサイクルイベントが at-least-once 配信によって再び捕捉される（ディスパッチャーの重複実行または再送を模す）
- THEN 新規 ProvisioningDelivery は作成されず既存行がそのまま使われる

### REQ-PROVISIONING-017: APIprocessのcapture直後のenqueueが失敗してもperiodicなdispatcherが未関連付けのdeliveryを回収する
- ACTOR System
- GIVEN IdManagement のコミットと同じトランザクションで `ProvisioningDelivery`（`job_id` 未設定、`status=pending`）が確定したが、API プロセスから Jobs への即時投入には失敗した
- WHEN `worker` プロセスの定期ディスパッチャーが、`job_id` を関連付けていない `pending` の配信を再走査する
- THEN dispatcher が idempotency key を dedup_key として EnqueueJob を呼び job_id を関連付ける
- THEN `ProvisioningDeliveryStarted` が発行され、`worker` プロセスが配信を実行する

### REQ-PROVISIONING-018: requiredな属性マッピングが解決不能な配送はfail-closedで失敗する
- ACTOR System
- GIVEN AttributeMappingRule (target_path="emails[type eq \"work\"].value", required=true) を持つが対象 User に email が未設定である
- WHEN `worker` プロセスが配信前に必須属性を解決する
- THEN 解決に失敗し、ダウンストリームへは送信されず UserProvisioningFailed が発行される (max_attempts を待たず fail-closed)
