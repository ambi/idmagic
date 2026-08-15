---
context: provisioning
updated_at: 2026-08-15
---

# Provisioning Specification

## Overview

下流の SaaS へユーザーとグループを反映する外向きのプロビジョニングを所有する。情報の正は idmagic 側の User と Group であり、下流はその写しである。接続は Application 1 件につき最大 1 件とし、配信対象の範囲には既存の ApplicationAssignment を再利用する。

`Sourcing` が外部から受け取るのに対し、この Context は外部へ送り出す。処理の向き、記録の正がどちらにあるか、語彙のすべてが反転するため、`Tenancy`、`Application`、`IdManagement`、`Jobs` への公開された参照を除いてコードを共有しない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ProvisioningConnection | Application 1 件に対して最大 1 件だけ存在する outbound provisioning の設定。接続先 base_url・認証・機能トグル・スコープ属性マッピング・deprovision ポリシー・信頼性設定を束ねる。 | connection, 接続 |
| RemoteResourceLink | idmagic の `User` または `Group` と、下流の SCIM サービスプロバイダー上のリソース（リモート ID、`externalId`、`etag`）との対応を保持するエンティティ。HTTP 409（既存リソースとの衝突）では照合属性を使って既存リソースへ関連付け、HTTP 404（リソースの消失）では再作成して関連付けを更新する。 | remote link, 相関 |
| ProvisioningDelivery | 内部のライフサイクルイベント 1 件を下流へ反映するための配信単位。冪等キー（`tenant_id`、`connection_id`、`source_type`、`source_id`、`source_version`）により、重複する投入を no-op とする。実行は Jobs の `Job` に委譲する。 | delivery, 配信 |
| Deprovision | 割り当て解除、無効化、削除という内部のライフサイクルイベントを、下流に対する無効化、削除、無操作のいずれかへ変換すること。変換規則は `DeprovisionPolicy` が持つ。 |  |
| Grace Period | 削除操作を直ちに実行せず、`DeprovisionPolicy.grace_period_days` の経過後に完全削除するまでの猶予期間。期間内に対象が適用範囲へ戻れば取り消す。 | grace period, 猶予期間 |
| Accidental Deletion Guard | 1 回の同期で無効化または削除の対象が `accidental_deletion_count_threshold` を超えた場合に、実行せず接続を Quarantine へ移す誤削除防止策。 | 誤削除ガード |
| Quarantine | 連続失敗または誤削除ガードの超過によって配信を停止した `ProvisioningConnection.health` の状態。管理者が `ResumeProvisioningConnection` で解除するまで再開しない。 | quarantine, 隔離 |
| On-Demand Provision | 管理者が対象を 1 件だけ指定し、即時に試験配信する手動運用。 |  |
| Full Resync | 管理者が接続の適用範囲に含まれるすべての対象を再走査し、下流の状態を idmagic の状態へ収束させる手動運用。 |  |
| Mirror | 下流の SaaS 上のリソースが idmagic を記録の正とする写しであり、下流での手動変更は次回の配信で上書きされること。 |  |
| Push Groups | `ProvisioningFeatureFlags.push_groups` が有効なとき、Group とメンバーシップを下流へ配信する機能。 |  |
| System | Provisioning の配信エンジンそのもの。人間の操作者を伴わない技術的主体を指す。 |  |

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

接続の登録、更新、削除、接続テスト、On-Demand Provision、Full Resync、Quarantine の解除、配信の一覧と再試行は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが所属テナントに対して行える。API アクセストークンでは `provisioning:read` が接続と配信の参照だけを、`provisioning:write` が接続の変更と配信操作を許可する。

接続と配信はテナントを越えない。別テナントの接続 ID や配信 ID を指定した参照は、存在しないものとして拒否する。

配信そのものは管理者の権限を借りない。`worker` が下流へ提示するのは接続に保存した資格情報だけであり、idmagic 側の管理者権限が下流へ伝わることはない。下流の URL は登録時に HTTPS であることと、内部アドレスやリンクローカルアドレスを指さないことを検証する。

## Design

### Protocol-agnostic core with protocol feature slices

外向きの振る舞いの大半、すなわち接続のエンベロープ、`DeprovisionPolicy`、`AttributeMappingRule`、`ProvisioningDelivery` と `RemoteResourceLink` による配信処理（キュー、再試行と間隔、隔離、順序、再同期）はプロトコルに依存しない。プロトコル固有なのは通信クライアントと接続設定の一部（認証方式、能力の探索、デフォルト属性スキーマ）だけである。したがって Context のルート (`domain`、`ports`、`usecase`、`handlers_http`) がプロトコル非依存の中核を持ち、プロトコルごとに `ProvisioningTargetClient` ポートを実装する機能単位を設ける。現在は `client_scim` があり、将来は中核に触れず `entraid` と `googledir` を同階層へ追加する想定である。これは、このリポジトリで一般的な「厚い機能単位、薄い共通ルート」を意図的に反転した形である。ここではドメインの形がほぼプロトコルに依存せず、プロトコルが駆動される側のアダプターの軸になるため、中核が厚く機能単位が薄い。

この Context の `client_scim` 機能と `Sourcing` の `scim` 機能の間には、共有する SCIM 通信中核を置かない。内向き側のフィルター構文の解析と評価、および固定レスポンスの構造体は、受信した SCIM リクエストを自身のデータに対して評価する側のためのものである。一方、外向き側はフィルター文字列を組み立て、マッピングに基づくより広い属性集合（`externalId`、Enterprise 拡張）を直列化する必要がある。実際に重なる部分（Discovery の構造体、RFC が定めるスキーマ URN）は十分に小さく、現時点で共有すると双方を早すぎる段階で結合することになる。本当の重複が現れるまで切り出しを先送りする。

### Same-transaction delivery capture

配信処理は、共通の `outbox` テーブルを外部の Kafka、Pub/Sub、ログへ転送するイベントリレーを利用しない。そのリレーは `IdManagement` や `Application` の書き込みトランザクション内で `ProvisioningDelivery` を作成できないためである。代わりに `Provisioning` は配信捕捉用の公開ポートを提供し、`IdManagement` のユーザー変更処理と `Application` の割り当て変更処理が、それぞれ自身の PostgreSQL トランザクション内でこのポートを呼び出す。ポートは一致する有効な接続ごとに `pending` の `ProvisioningDelivery` 行を 1 件挿入する。発火元の変更と配信行は同時にコミットまたはロールバックされるため、`ProvisioningDelivery` がこの Context 専用の Transactional Outbox となる。配信の冪等性には `(tenant, connection, source_type, source_id, source_version)` をキーとして使う。

### Design Decisions

- Context の名前は向きを表す語ではなく能力から取り、プロトコルに依存しない中核と、`ProvisioningTargetClient` を実装するプロトコルごとの薄い機能単位に分ける。ドメインの形がほぼプロトコルに依存しないため、リポジトリで一般的な「厚い機能単位、薄い共通ルート」を意図的に反転させている。
- 内向きの SCIM (`Sourcing`) と通信の中核を共有しない。実際に重なるのは Discovery の構造体と RFC が定めるスキーマ URN だけで、いま共有すると双方を早すぎる段階で結合することになるからである。
- 配信行は共通の outbox とイベントリレーではなく、この Context 専用の Transactional Outbox とする。汎用のリレーでは、`IdManagement` や `Application` の書き込みトランザクションの中で配信行を作れず、変更だけが確定して配信予定が失われる隙間が残るからである。

## Scenarios

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
- WHEN IdManagement が User "ユーザー-1" を作成し commit する
- THEN 同一トランザクションで `ProvisioningDelivery`（`operation=create`、`status=pending`）が作成される
  - ALT User がこの Application に割り当て済みでない (scope=assigned_only) → ProvisioningDelivery は作成されない
- THEN dispatcher が Jobs.Job を関連付け ProvisioningDeliveryStarted が発行される
- THEN `worker` プロセスが下流へ POST し、`UserProvisioned` が発行され、配信のステータスが `succeeded` になる

### REQ-PROVISIONING-004: ユーザーの無効化は下流への無効化として配信される
- ACTOR System
- GIVEN User "ユーザー-1" は下流に既に存在し RemoteResourceLink を持つ
- WHEN IdManagement が User "ユーザー-1" を disable する
- THEN ProvisioningDelivery (operation=deactivate) が作成される
- THEN `worker` プロセスが下流へ `active=false` の PATCH を送り、`UserDeprovisioned` が発行される

### REQ-PROVISIONING-005: Application からの割り当て解除は既定で下流の無効化として配信される
- ACTOR System
- GIVEN User "ユーザー-1" は Application "app-1" に割り当て済みで下流に存在する
- GIVEN DeprovisionPolicy.on_unassign はデフォルト値 deactivate のままである
- WHEN 管理者が User "ユーザー-1" の Application "app-1" への割り当てを解除する
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
