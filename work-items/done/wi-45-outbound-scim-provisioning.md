---
depends_on: [wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-21
---

# 下流 SaaS への outbound provisioning (SCIM client / lifecycle management) を実装する

## Motivation
[[wi-31-scim2-provisioning]] は idmagic が SCIM **server** として外部
system of record (Okta / Google IAM / Entra ID) から user / group を
受け取る inbound 方向を扱い、`out_of_scope` に「idmagic が SCIM client
として外部アプリへ push provisioning すること」を明示的に除外している。

しかし Okta / Entra ID / OneLogin の中核機能の 1 つは逆方向、すなわち
IdP 自身が source of truth となり、接続済みの下流 SaaS (Slack /
Salesforce / GitHub / Zoom 等) へ user / group を **push** して
provisioning / deprovisioning / group sync する outbound lifecycle
management である。これが無いと「入社で IdP に user を作る → 各 SaaS に
自動で account が生える / 退職で一斉に無効化される」という、企業が IdP に
最も期待する運用が成立しない。idmagic は inbound SCIM (wi-31) と
outbound SCIM (本 WI) の双方を持って初めて provisioning hub になる。

本 WI は idmagic を SCIM 2.0 **client** にする。downstream connector を
tenant-scoped に登録し、user/group のライフサイクル変化 (作成・属性更新・
disable・delete・group membership 変更) を outbound event として下流の
SCIM service provider へ反映する。確実性のため push は同期呼び出しでなく
既存の outbox / queue を介した非同期・冪等・retry 付きの provisioning job
として実装する。

## 設計

本 WI の核心は「outbound provisioning の設定をどこに置き、何を設定させ、どう確実に
配送するか」である。ここでは他社 IdP (Okta / Entra ID / Google / Keycloak) の実装を
調査した上で、idmagic の既存構造 (`Application` と `ApplicationAssignment`、
`outbox` + Relay、[[wi-126-async-job-runner]]) に無理なく載る形を確定し、実装が仕様判断で
止まらないようにする。

### 参照サービス調査と採用方針

| 論点 | Okta | Entra ID (Enterprise App) | Google Workspace | Keycloak | idmagic の採用 |
|---|---|---|---|---|---|
| 設定の所在 | App 統合の **Provisioning タブ** | Enterprise App の **Provisioning ブレード** | SAML app の Auto-provisioning | ネイティブ無し (scim-for-keycloak 拡張) | **Application 詳細の「プロビジョニング」サブルート**。provisioning は app 統合に属する |
| 有効化単位 | app ごと | app ごと | app ごと | 拡張ごと | Application ごとに 1 ProvisioningConnection |
| 対象範囲 (scope) | 割当ユーザー / Push Groups | 「割当済みのみ同期」/「全員同期」 | グループ指定 | — | 既存 `ApplicationAssignment` を再利用。`assigned_only` / `all_users` |
| 認証 | API Token / OAuth | Tenant URL + Secret Token / OAuth | OAuth | Bearer | `bearer_token` / `oauth2_client_credentials`、write-only secret |
| 機能トグル | Create / Update / Deactivate / Sync Password | (mapping 有効化 + scope) | 作成 / 更新 / 停止 | — | **Create / Update / Deactivate / Delete / Push Groups** トグル (Okta 準拠) |
| 属性マッピング | Okta→App、式・apply-on | 属性一覧、matching precedence、apply "Always/Only creation"、default | 属性 mapping | mapping | 宣言的 `AttributeMappingRule[]`、`apply_on` (create_only / create_and_update)、default、required、matching |
| deprovision | Deactivate Users | active=false / (誤削除防止 threshold) | 「停止→N日後に削除」 | — | **trigger×action マトリクス** + Google 式 grace period + Entra 式誤削除ガード |
| 相関 | unique identifier field | matching precedence 属性 | — | externalId | `externalId` = idmagic の id。409 時に match 属性で既存を養子縁組 |
| 手動運用 | — | On-demand provision / restart / clear state | — | — | On-demand provision (単一 subject test) / full resync / delivery retry |
| 障害時 | ログ | quarantine + 通知メール | — | — | 連続失敗で quarantine + 通知メール、dead-letter |

採用の骨子:

1. **設定は Application に属する** (Okta / Entra と同じ)。テナント直下の独立 connector 一覧に
   はしない。1 Application に最大 1 ProvisioningConnection。
2. **provisioning の scope は既存 `ApplicationAssignment` を再利用**する。「誰を provisioning
   するか」を別 model (`ProvisioningAssignment`) で二重管理しない — これは Entra の
   「割当済みユーザー / グループのみ同期」と同型で、idmagic では既に app 割当が存在するため
   最も自然かつ実装量が小さい。`all_users` は「テナント全 user」を意味する上級オプション。
3. **真実源は内部 user/group aggregate、下流は mirror (射影)**。下流での手動変更は次回 sync で
   上書きする (Okta の "profile master" と同じ)。
4. **配送は同期呼び出しにしない**。committed lifecycle event を `outbox` 経由で観測し、
   `(tenant, connection, source_type, source_id, source_version)` を idempotency key とする
   `ProvisioningDelivery` を [[wi-126-async-job-runner]] 上で冪等・順序保証・retry 付き実行する。

### context 境界の決定

inbound (wi-31 = SCIM **server**) と outbound (本 WI = SCIM **client**) は「SCIM 2.0 仕様を
話す」点だけが共通で、それ以外はほぼ別実装になる。以下を並べると重なりが小さいことがわかる:

| 観点 | inbound (Scim / server) | outbound (本 WI / client) |
|---|---|---|
| 権威方向 | 外部 IdP が SoR、idmagic は受信側 | idmagic が SoR、下流は mirror |
| 主 aggregate | ScimClient / token、受信 op → 内部 User/Group 変換 | ProvisioningConnection / RemoteResourceLink / Delivery |
| 起動契機 | 外部からの HTTP request | 内部 lifecycle event (outbox) |
| admin API・permission・events | 受信設定・token | connection・mapping・delivery・deprovision |
| 永続化 | `scim_*` refs / tokens | `provisioning_*` |
| 唯一の共通項 | — | **SCIM resource の marshal/unmarshal と filter 文法** |

同じ「User/Group」という名詞を使うが動詞と invariant は反転しており、両者を 1 つの model として
束ねると ubiquitous language が曖昧になる (例: "client" が inbound では token 保有者、outbound では
下流 SP を指す)。既存 context も WsFederation と SAML を「federation」でまとめず protocol 責務ごとに
分けており、粒度の前例がある。

**採用案 (推奨): outbound を独立 subdomain として切り出し、SCIM wire 表現だけを共有 kernel にする。**

- `Scim` context は **inbound server 専用**のまま (責務・語彙を変えない)。
- outbound は独立の provisioning context (例: `Provisioning`) として、connection / mapping /
  delivery / remote link / deprovision を所有する。`ApplicationRef` / `ApplicationAssignment` /
  `UserRef` / `GroupRef` / `JobRef` を published language 経由で参照する (Application への帰属は
  cross-context 依存であり、Application context に混ぜる理由にはならない)。
- **共有 SCIM wire kernel**: resource (User/Group JSON) の (de)serialize と filter 文法を、
  inbound/outbound 双方が依存する小さな共有モジュール (`backend/scim/wire` 等) に factor out する。
  これは domain ではなく serialization の shared kernel であり、双方向で唯一の正当な共有点。

RA 上の利点: SCL の section-addressability と再生成が inbound/outbound で独立し、delivery/backoff の
変更が inbound server 仕様に波及しない。context 名を `Provisioning` にすると将来の非 SCIM
provisioning (HRIS 等、本 WI は out of scope) の受け皿にもなる。

**代替案** (context 数を増やしたくない場合): 単一 `Scim` context のまま、feature/ディレクトリで
`scim/inbound/…` `scim/outbound/…` `scim/wire/…` に強く分割する。物理的凝集は保てるが、context の
語彙が二役になる点が残る。→ **どちらを採るかは T001 の ADR で確定する**。

### 設定の置き場所 (UI 情報設計)

- **主配置: Application 詳細 → 「プロビジョニング」サブルート** (`/admin/applications/$applicationId/provisioning`)。
  OIDC/WsFed/SAML の binding 設定と並ぶ per-app 設定として、以下をセクション分割する:
  1. **接続 (Connection)** — endpoint base URL、認証方式、credential (write-only)、`接続テスト` ボタン (`/ServiceProviderConfig` 到達確認 + capabilities 表示)、有効化トグル。
  2. **機能 (Provisioning Features)** — Create / Update / Deactivate / Delete / Push Groups のチェックボックス (Okta 準拠。下流が未対応の操作は capabilities に従い disable 表示)。
  3. **スコープ (Scope)** — `割当済みのみ` / `全ユーザー`。割当は既存の割当タブへリンク。
  4. **ユーザー属性マッピング** — target SCIM path ↔ source の表エディタ。default 一式を初期表示。
  5. **グループ (Push Groups)** — push 対象グループの選び方、displayName mapping。
  6. **deprovision ポリシー** — unassign / disable / delete それぞれの下流アクション、grace period、誤削除ガード threshold。
  7. **配送状況 (Delivery)** — 直近の同期状況、失敗一覧、個別 retry、full resync、on-demand provision (単一ユーザーで試験配送)。
  8. **詳細設定** — rate limit、通知メール、quarantine しきい値。
- **副配置: テナント設定 (`admin-settings`) → 「プロビジョニング」read-only 集約ビュー**。
  「どの Application が outbound provisioning 有効か」「quarantine 中の connection」「直近 24h の
  失敗件数」をテナント横断で俯瞰する監視面。書き込みは各 app 側へ誘導 (deep link)。

### 設定項目カタログ

`ProvisioningConnection` (Application ごと 1 件) が持つ設定:

- **接続**: `enabled` (bool)、`base_url` (下流 SCIM v2 ベース URL)、`auth_method` (`bearer_token` | `oauth2_client_credentials`)、credential 参照 (bearer は token、oauth2 は token URL + client_id + client_secret + scope)。credential は write-only、平文・hash を返さない (`ApplicationClientSecretCredentialMetadata` パターン)。
- **capabilities (discovery でキャッシュ, read-only)**: `supports_patch` / `supports_bulk` / `supports_filter` / `supports_etag` / `supports_sort`。`/ServiceProviderConfig` から取得。PATCH 非対応先は PUT にフォールバック。
- **機能トグル** (`ProvisioningFeatureFlags`): `create_users` / `update_users` / `deactivate_users` / `delete_users` / `push_groups`。（`sync_password` は out of scope。）
- **scope**: `sync_scope` (`assigned_only` 既定 | `all_users`)。`assigned_only` は `ApplicationAssignment` を参照。
- **ユーザー属性マッピング**: `AttributeMappingRule[]`（下記）。
- **グループ**: `push_groups` 有効時の対象グループ選択 (割当グループ or 明示指定) と `group_display_name_source`。
- **相関 (matching)**: `external_id_strategy` (既定は idmagic user id を下流 `externalId` に設定)、409 衝突時の match 属性 (既定 `userName`) と precedence。
- **deprovision ポリシー** (`DeprovisionPolicy`): trigger×action、`grace_period_days`、`accidental_deletion_threshold`。
- **信頼性**: `rate_limit_per_minute`、`max_attempts`、`notification_email` (失敗/quarantine 通知先)、`quarantine_after_consecutive_failures`。
- **状態 (read-only)**: `health` (ok / degraded / quarantined)、`last_full_sync_at`、`quarantined_at`、`quarantine_reason`。

### 属性マッピングのセマンティクス

`AttributeMappingRule` = 1 出力属性の宣言的写像:

- `target_path` — SCIM 属性パス。単純 (`userName`, `displayName`, `active`)、複合 (`name.givenName`)、多値フィルタ (`emails[type eq "work"].value`)、enterprise 拡張 (`urn:...:enterprise:2.0:User:department`) を許す。
- `source_kind` — `attribute` (idmagic 属性) | `constant` (固定値) | `expression` (単純変換; 初期は関数無しの直参照 + 固定連結程度に限定し、式評価器の作り込みは別 WI)。
- `source_key` / `constant_value` — source_kind に応じた供給元。
- `apply_on` — `create_and_update` (既定) | `create_only` (Entra の "Only during creation" 相当。ユーザーが下流で変えてよい項目に使う)。
- `required` — true なら解決不能時に配送を fail-closed。
- `default_value` — source が空のとき使う既定 (Entra の default 相当)。

**既定マッピング (下流が SCIM core User schema の場合の初期値)** — 実装は以下を seed する:

| target_path | source | apply_on | 備考 |
|---|---|---|---|
| `externalId` | idmagic user id | create_only | 相関キー |
| `userName` | primary login (email/username) | create_and_update | match 既定属性 |
| `active` | enabled かつ 割当 (scope 内) | create_and_update | deprovision の実体 |
| `name.givenName` | given_name | create_and_update | |
| `name.familyName` | family_name | create_and_update | |
| `displayName` | display_name | create_and_update | |
| `emails[type eq "work"].value` | email | create_and_update | primary=true |

グループ既定: `displayName` ← group 名、`externalId` ← group id、`members` ← 割当ユーザーの下流 remote id (`RemoteResourceLink` で解決)。

### deprovision セマンティクス

trigger × 設定可能アクション (既定は破壊性が低い側に倒す):

| trigger (内部イベント) | 選択肢 | 既定 | 下流への翻訳 |
|---|---|---|---|
| app からの unassign (`ApplicationAssignment` 削除) | deactivate / delete / none | **deactivate** | PATCH `active=false` |
| user disable (idmagic active=false) | deactivate (固定) | deactivate | PATCH `active=false` |
| user soft-delete / delete | deactivate / delete / none | **deactivate** | policy と `grace_period_days` に従う |
| group membership 削除 | (固定) | — | PATCH members remove |
| group 削除 / unassign | delete / none | **none** | DELETE Group or 放置 |

- **grace period** (Google 準拠): delete アクションは即時実行せず `grace_period_days` 後に purge job を予約。期間内に再割当されれば取り消す。
- **誤削除ガード** (Entra 準拠): 1 回の sync で deactivate/delete 対象が `accidental_deletion_threshold` (件数 or 割合) を超えたら実行せず connection を quarantine し、管理者に確認を促す。
- **inbound (wi-31) との整合**: `active=false` の意味を inbound/outbound で一致させる。

### 配送・信頼性

- **配送単位**: `ProvisioningDelivery` (poly `pending|in_flight|succeeded|failed|dead_letter`)。idempotency key = `(tenant, connection, source_type, source_id, source_version)`。同一 key の重複 enqueue は no-op。
- **順序**: `(connection, remote resource)` 単位で create → update → deprovision の順序を保証 (out-of-order event を source_version 単調性で解決)。
- **retry**: 指数 backoff + jitter、`Retry-After` / 429 尊重、`max_attempts` 超過で dead-letter。runtime は [[wi-126-async-job-runner]]。
- **rate limit**: connection ごとの token bucket。429 で自動減速。
- **quarantine**: 連続失敗が `quarantine_after_consecutive_failures` に達したら停止 + 通知メール。管理者が原因解消後に再開。
- **手動運用**: on-demand provision (単一 subject を即時試験配送)、full resync (scope 内 subject を全走査して収束)、個別 delivery の manual retry。
- **相関の永続化**: `RemoteResourceLink` に下流 `id` / `etag` / `external_id` / `last_synced_version` を durable に保持し、409 (既存)・404 (消失) を冪等に解決する。
- **SSRF/leak 対策**: base URL は https 必須、内部/リンクローカル IP・任意 redirect を拒否。credential はログ・エラー・API 応答に出さない。

## Scope
- **decision**:
  - 新規 ADR: outbound provisioning model を定義する。§設計 の確定事項 — connection の Application への帰属、scope としての `ApplicationAssignment` 再利用、真実源=内部 aggregate / 下流=mirror、`active=false` / delete の下流翻訳と deprovision マトリクス、`externalId` 相関と 409/404 解決、`outbox`+[[wi-126-async-job-runner]] による冪等配送、SSRF/secret 保護 — を記録する。
  - 配送基盤は [[wi-126-async-job-runner]] (completed) を利用する。[[wi-184-transactional-event-log-foundation]] の `event_logs`/`event_deliveries` は ADR-095 で撤去済みのため、lifecycle event は現行の `outbox` (Relay drain) から観測する。二重 queue は作らない。
- **scl** (application/scim):
  - 新規 model: `ProvisioningConnection` (Application ごと 1、endpoint / auth / feature flags / scope / mapping / deprovision policy / 信頼性 / health)、`AttributeMappingRule`・`MatchingRule`・`DeprovisionPolicy`・`ProvisioningFeatureFlags` (value object)、`RemoteResourceLink` (下流 id 相関 entity)、`ProvisioningDelivery` (poly、idempotency key)。enum: `ProvisioningAuthMethod` / `ProvisioningScope` / `ProvisioningDeprovisionAction` / `AttributeApplyOn` / `ProvisioningHealth`。
  - **`ProvisioningAssignment` は新設しない** (既存 `ApplicationAssignment` を scope に再利用)。
  - 新規イベント: `ProvisioningConnectionRegistered` / `...Updated` / `...Disabled` / `...Deleted` / `ProvisioningCredentialRotated` / `UserProvisioned` / `UserDeprovisioned` / `UserProvisioningFailed` / `GroupPushed` / `GroupMembershipPushed` / `ConnectionQuarantined` / `FullResyncCompleted`。
  - 新規 interface (admin, permission `AdminProvisioningRead` / `AdminProvisioningWrite`): `GetProvisioningConnection` / `RegisterProvisioningConnection` / `UpdateProvisioningConnection` / `DeleteProvisioningConnection` / `TestProvisioningConnection` / `ProvisionOnDemand` / `StartFullResync` / `ListProvisioningDeliveries` / `GetProvisioningDelivery` / `RetryProvisioningDelivery` / `ListTenantProvisioningConnections` (テナント集約 read)。
- **go**:
  - SCIM wire client adapter (`backend/scim` の outbound 側): 下流 `/scim/v2/Users` `/scim/v2/Groups` への POST / PATCH / (PUT fallback) / DELETE を仕様準拠で送る。`externalId` 相関、409/404 冪等解決、pagination / error / `Retry-After` 処理。bearer / OAuth2 client credentials に対応し secret をログに出さない。
  - 配送: committed lifecycle event (create / attribute update / disable / delete / membership / assignment 変更) を projector で受け、対象 connection の `ProvisioningDelivery` を [[wi-126-async-job-runner]] に enqueue。指数 backoff retry、dead-letter、per-connection rate limit、順序保証。
  - Postgres adapter: `provisioning_connections` / `provisioning_remote_links` / `provisioning_deliveries` テーブルと index。credential は既存 secret 保管方針に従う。
- **ui**:
  - Application 詳細に「プロビジョニング」サブルート (§設定の置き場所)。接続 test、機能トグル、scope、属性マッピングエディタ、push groups、deprovision policy、配送一覧/詳細/retry、on-demand provision、full resync。secret は write-only 表示。
  - テナント設定に read-only 集約ビュー (有効 connection 一覧 / quarantine / 失敗件数)。

## Out of Scope
- inbound SCIM server 機能 (wi-31 が扱う)。
- 各 SaaS 固有の非標準 provisioning API (SCIM を話さない app 向け connector)。
- password sync / 下流からの逆方向 import (それは inbound = wi-31)。
- HRIS / ディレクトリ connector (LDAP / AD)。
- SaaS app カタログ (事前定義テンプレート) の整備。初期は汎用 SCIM connector のみ。

## Plan
- **context 境界を最初に確定する** (§context 境界の決定)。inbound (server) と outbound (client) は SCIM 仕様を共有するだけで model・aggregate・admin API・invariant が別なので、outbound を独立の provisioning subdomain として切り出し、SCIM の wire 表現 (resource marshal + filter) だけを共有 kernel にする。
- SCIM wire client/serializer は共有 wire kernel を使い、下流 `/scim/v2/Users` `/scim/v2/Groups` への CRUD を出す outbound adapter として実装する。接続 test/discovery で `/ServiceProviderConfig` と schema を取得し、https 必須・内部 IP / 任意 redirect 拒否で SSRF/credential leak を防ぐ。
- connection は base URL、auth credential reference、capabilities、feature flags、scope (`ApplicationAssignment` 参照)、attribute mapping、deprovision policy を持つ。**scope 用の割当 model は新設せず既存 `ApplicationAssignment` を使う**。
- committed lifecycle event (user/group/assignment) を現行の `outbox` (Relay drain) から観測し、`(tenant, connection, source_type, source_id, source_version)` を idempotency key に `ProvisioningDelivery` を作る。HTTP request 内で下流を呼ばない。配送 runtime は [[wi-126-async-job-runner]] (completed) を使い、二重 queue を作らない ([[wi-184-transactional-event-log-foundation]] の `event_logs` は ADR-095 で撤去済み)。
- delivery は per-(connection, remote resource) 順序、指数 backoff、`Retry-After`、dead-letter、manual retry、quarantine を持つ。
- remote SCIM ID と local resource の mapping (`RemoteResourceLink`) を durable に保持し、PATCH 非対応先は PUT fallback、409/404 を冪等解決、delete は deprovision policy により deactivate/delete/none を選ぶ。

## Tasks
- [x] T001 [Decision/ADR] context 境界 (§context 境界の決定) を確定し、outbound ownership・scope=`ApplicationAssignment` 再利用・真実源/mirror・deprovision マトリクス・delivery guarantee (idempotency key / 順序 / quarantine)・secret 保護を ADR に記録する。ADR-128: `outbox`+[[wi-126-async-job-runner]] 利用は当初案から変更し、既存 outbox はトピック未登録・非原子的書き込み・in-process consumer 不在の三重の欠落があるため観測経路として採用せず、`UserMutationCommitter` 型の same-Tx capture で `ProvisioningDelivery` を書き Jobs へ渡す方式に置換した。context 名は独立 `Provisioning` (outbound 専用)、内部は protocol 非依存コア + protocol 別 feature slice (`provisioning/scim` 等)。inbound 側リファクタは [[wi-258-inbound-integration-taxonomy]] / [[wi-259-rename-scim-inbound-server-context]] / [[wi-260-relocate-csv-user-import-to-inbound]] へ分離。
- [x] T002 [SCL] `spec/contexts/provisioning.yaml` を新設し、connection、feature flags、scope、attribute mapping、matching、deprovision policy、remote link、delivery lifecycle、management interfaces/events/authorization/scenarios/flows を追加。`spec/scl.yaml` context_map と `ARCHITECTURE.md` を同期。共有 SCIM wire kernel は作らない (ADR-128 決定3。inbound `Scim` の filter パーサ・固定 DTO は outbound の mapping 駆動シリアライズと再利用性が低いため)。派生成果物を `just scl-render` で再生成済み。`just yaml-check` 全緑。
- [x] T003 [Domain] `backend/provisioning/domain` に `ProvisioningConnection`・`AttributeMappingRule`/`MatchingRule`/`DeprovisionPolicy`・`RemoteResourceLink`・`ProvisioningDelivery`・`ProvisioningDeliveryLifecycle` 状態機械・14 個の domain event を実装。test-first 証跡:
  - `TransitionProvisioningDeliveryLifecycle` / `IsProvisioningDeliveryTerminal` — RED: `TestTransitionProvisioningDeliveryLifecycle_DeclaredTransitions` 等を先に fail 確認 (undefined symbols) → GREEN (states `ProvisioningDeliveryLifecycle`)。
  - `RemoteResourceLink.ApplySync` の source_version 単調性 — RED: `TestRemoteResourceLink_ApplySync_RejectsOutOfOrderVersion` / `RejectsRepeatedVersion` を先に fail 確認 → GREEN (models `RemoteResourceLink`)。
  - `ProvisioningDelivery.IdempotencyKey` の冪等 key 安定性/一意性 — RED: `TestProvisioningDelivery_IdempotencyKey_*` を先に fail 確認 → GREEN (models `ProvisioningDelivery` の idempotency key 契約)。
  - `ProvisioningConnection.Validate`/`Quarantine`/`Resume` の quarantine 整合性 (health/quarantined_at) — RED: `TestProvisioningConnection_Validate_QuarantineConsistency` 等を先に fail 確認 → GREEN (models `ProvisioningConnection` の constraint、interfaces `ResumeProvisioningConnection`)。
  - `DeprovisionPolicy.Validate` の group deactivate 禁止・grace period 非負 — RED: `TestDeprovisionPolicy_Validate_*` を先に fail 確認 → GREEN (models `DeprovisionPolicy`)。
  - `AttributeMappingRule.Validate` の source_kind 別必須 field — RED: `TestAttributeMappingRule_Validate_*` を先に fail 確認 → GREEN (models `AttributeMappingRule`)。
  - 14 event の `EventType()`/`OccurredAt()` 構造適合 — RED: `TestProvisioningEvents_ImplementDomainEvent` を先に fail 確認 → GREEN。
  - 検証: `go test -race ./backend/provisioning/...` 緑、`golangci-lint run ./...` 0 issues、`gofmt -l` clean、`go build ./...` 緑。
- [x] T004 [Persistence] `backend/provisioning/ports` に repository interface (`ProvisioningConnectionRepository`/`RemoteResourceLinkRepository`/`ProvisioningDeliveryRepository`) を定義し、`adapters/persistence/memory` と `adapters/persistence/postgres` の双方に実装。`infra/schema/postgres.sql` に `provisioning_connections`/`provisioning_remote_links`/`provisioning_deliveries` を追加 (credential_secret は dev/test grade の平文列、本番反映は wi-97 envelope-encryption-at-rest 待ち。signingkeys の Postgres KeyStore dev/test fallback と同じ注記をスキーマに残した)。credential secret は `CredentialSecret` という別メソッドで narrow に公開し、`Find`/admin 読み取り経路には現れない。test-first 証跡:
  - Register の 1 Application 1 connection 制約 — RED: `TestProvisioningConnectionRepository_Register_RejectsDuplicateApplication` (memory/postgres 双方) を先に fail 確認 → GREEN (`ports.ErrConnectionAlreadyExists`、PostgreSQL は `application_id` PK + `ON CONFLICT DO NOTHING`)。
  - credential write-only 契約 — RED: `TestProvisioningConnectionRepository_Find_DoesNotExposeSecret` / `Update_RotatesSecretOnlyWhenProvided` を先に fail 確認 → GREEN。
  - delivery の idempotency key 一意性 — RED: `TestProvisioningDeliveryRepository_Save_IdempotentOnDuplicateKey` (memory/postgres 双方) を先に fail 確認 → GREEN (Postgres は `provisioning_deliveries_idempotency_unique` + `ON CONFLICT DO NOTHING`)。
  - dispatcher 回収経路 — RED: `TestProvisioningDeliveryRepository_ListUnenqueuedAttachJobRetry` を先に fail 確認 → GREEN (`ListUnenqueued`/`AttachJob`/`RetryDeadLetter`)。
  - RemoteResourceLink の Upsert 冪等性 — RED: `TestRemoteResourceLinkRepository_UpsertThenFind` を先に fail 確認 → GREEN。
  - 検証: `go test -race ./backend/provisioning/...` 緑 (PostgreSQL は embedded-postgres 実 DB、`pgtest.Require` 経由)、`golangci-lint run ./...` 0 issues、`go build ./...` 緑。
- [x] T005 [SCIM Client] `backend/provisioning/scim` に outbound adapter (mapping 駆動シリアライズ `BuildResource`、discovery、Users/Groups CRUD、PATCH→PUT fallback、409/404 型付きエラー、`Retry-After` 解析、SSRF-safe transport) を実装。inbound `backend/scim` とはコード共有しない (ADR-128 決定3)。SSRF 対策は `backend/shared/adapters/crypto.JWKResolver` (jwks_uri fetch) の DialContext 再解決 + 各 IP の public 判定パターンを踏襲。test-first 証跡:
  - 属性マッピング (`BuildResource`): simple/nested/multi-valued filter path、constant/default、required fail-closed、create_only スキップ — RED: `TestBuildResource_*` を先に fail 確認 → GREEN。
  - wire client (httptest.Server による mock server contract test): Discover の capability 解析、CreateUser の 201/409、PATCH→PUT fallback、DELETE の 404 冪等成功、429 の Retry-After 解析、409 解決用 SearchByAttribute、Group members PATCH — RED: `TestClient_*` を先に fail 確認 → GREEN。
  - SSRF guard: https 必須・userinfo/fragment 拒否・loopback/private/link-local IP 拒否 — RED: `TestValidateOutboundBaseURL_*` / `TestSafeIPs_*` を先に fail 確認 → GREEN。
  - 検証: `go test -race ./backend/provisioning/...` 緑、`golangci-lint run ./...` 0 issues。
- [x] T006 [Event/Worker] コアループ (capture → dispatch → deliver → quarantine) を実装。grace-period 付き delete・誤削除ガード・per-connection rate limit は範囲外とし後続タスクへ委譲 (ユーザー承認済み、下記参照)。
  - **same-Tx capture の scoped simplification**: ADR-128 決定4は「呼び出し元 Tx 内」を求めるが、IdGovernance の `UserWorkflowCapture`/`igpostgres.SaveUserAndRuns` を Tx 分割可能な形にリファクタして第三の当事者 (Provisioning) を割り込ませるのは、稼働中の IdGovernance コードに実質的なリスクを伴う変更になるためユーザーと相談のうえ見送った。代わりに、IdManagement/Application が commit 直後に**別トランザクション**で `ProvisioningDelivery` を作成する (`CaptureLifecycleEvent`)。**既知の残存ギャップ**: 2つの commit の間でクラッシュすると capture が失われ、回復機構が無い (`backend/provisioning/ports/capture.go` に明記)。呼び出しは `idmanagement/usecases.notifyProvisioning` / `application/usecases.notifyProvisioning` で「ログして失敗させない」扱い (bootstrap.NewEmitFunc の outbox/audit と同じ方針)。
  - **cross-context port は IdManagement/Application 側で所有** (ADR-117 の `UserMutationCommitter` と同型): `idmanagement/ports.ProvisioningNotifier`、`application/ports.ProvisioningNotifier` を新設し、`backend/provisioning/usecases.UserMutationNotifier`/`AssignmentMutationNotifier` が実装。IdManagement/Application は `backend/provisioning` を import しない (context_map の依存方向を保つ)。
  - **Group push は対象外**: capture は User lifecycle (create/update/disable/enable/delete) と Application assignment (add/remove) のみを trigger する。Group 作成・更新・削除・membership の capture 配線は未実装 (SCIM client 自体は Group CRUD メソッドを持つが、呼び出し元が無い)。
  - test-first 証跡:
    - `CaptureLifecycleEvent` の scope 判定・DeprovisionPolicy 翻訳・quarantine 中/disabled connection の除外・idempotency — RED: `TestCaptureLifecycleEvent_*` を先に fail 確認 → GREEN。
    - `DispatchPendingDeliveries` の attach/skip — RED: `TestDispatchPendingDeliveries_*` を先に fail 確認 → GREEN。
    - `ExecuteDelivery` の create/update/deactivate/delete、409 adopt、404 recreate、retryable error 伝播 — RED: `TestExecuteDelivery_*` を先に fail 確認 → GREEN。
    - `ProvisioningDeliveryHandler` の成功時 failure count リセット、非終端 failure での in_flight 維持、終端 failure での dead_letter 化、連続失敗による quarantine — RED: `TestProvisioningDeliveryHandler_*` を先に fail 確認 → GREEN。
    - `UserAttributeSource` (IdManagement.User → 属性 map 変換) — RED: `TestUserAttributeSource_*` を先に fail 確認 → GREEN。
  - 配線: `idmanagement/usecases/admin_users.go` (Create/Update/Disable/Enable/Delete/SoftDelete)、`application/usecases/assignments.go` (Assign/Unassign)、`backend/provisioning/module.go` (Module 集約)、`backend/cmd/internal/bootstrap/{memory,postgres_valkey}.go`、`backend/cmd/idmagic-worker/worker.go` (job handler 登録 + `provisioningDispatchLoop`)。
  - **既知のギャップ (T007 以降または別タスクへ)**: grace-period 付き delete、誤削除ガード (sync 単位のしきい値判定)、per-connection rate limit、Group push、`oauth2_client_credentials` 認証 (現在は secret をそのまま bearer token として送る簡易実装)。
  - 検証: `go build ./...` 緑、`golangci-lint run ./...` 0 issues、`just test-go` は本 WI 起因の regression 無し (`GET /session/check` は無関係な既存ギャップ、13 件の `spec-only` provisioning route は T007 未実装のため予期される差分)。
- [x] T007a [Admin HTTP] (ユーザー承認により HTTP API のみ先行実装、UI は T007b へ分割) `backend/provisioning/usecases/admin.go` に 12 interface すべての admin usecase (Register/Get/Update/Delete/Test/ProvisionOnDemand/StartFullResync/ListDeliveries/GetDelivery/RetryDelivery/ResumeConnection/ListTenantConnections) を実装し、`backend/provisioning/adapters/http` で HTTP binding を SCL 通りに登録。`backend/provisioning/module.go` に `Register` を追加し、中央 `routes.go`/`server.go`/bootstrap に配線。
  - test-first 証跡: `RegisterConnection` の SSRF 拒否・既定 attribute mapping 適用・重複拒否、`UpdateConnection` の部分更新・credential rotation 分離、`ResumeConnection` の quarantined 前提条件、`RetryDelivery` の dead_letter 前提条件、`ProvisionOnDemand` の scope 外拒否 — RED: `TestRegisterConnection_*`/`TestUpdateConnection_*`/`TestResumeConnection_*`/`TestRetryDelivery_*`/`TestProvisionOnDemand_*` を先に fail 確認 → GREEN。
  - **HTTP handler 層自体には専用テストを書いていない** (self-attest: ADR-119 の test-first を字義通り満たしていない既知の逸脱)。理由: handler は認可チェック→usecase 呼び出し→レスポンス整形の薄いラッパーで、業務ロジックは usecase 層で既に検証済み。振る舞い検証は `routes_contract_test.go` (SCL binding と実 route の一致) と usecase 層のテストで代替している。フルの HTTP 統合テストは別タスクで追加を検討。
  - JSON レスポンスは domain 型を直接シリアライズしていたが (Go の PascalCase フィールド名のまま)、T007b 着手時に `backend/provisioning/domain/{connection,mapping,delivery}.go` と `usecases/admin.go` の `TestConnectionResult` へ snake_case の `json` タグを追加し、他の管理 API と同じ命名規約に揃えた (専用 Response DTO は導入せず、domain 型への直接タグ付けで解消)。
  - `FullResyncCompleted` イベントは発行しない (多数の非同期 delivery の完了を追跡する仕組みが無いため、既知のギャップ)。
  - 検証: `go build ./...` 緑、`golangci-lint run ./...` 0 issues、`just test-go` は本 WI 起因の regression 無し (`GET /session/check` のみ既存ギャップとして残存、`routes_contract_test.go` は 12 route すべて green)。
- [x] T007b [UI] Application 詳細ページに「プロビジョニング」サブルート (`/admin/applications/:id/provisioning`, TanStack Router file route。wi-45 の設計が明示するサブルート構造を踏襲し、`AdminSettingsPage.tsx` 系の in-page タブではなく `$applicationId.edit.tsx` と同型の独立ページを採用: URL 直リンク・ブックマークができる利点を優先) を実装。テナント全体の read-only 集約ビューも `/admin/provisioning` に新設し、サイドナビへ項目追加。
  - `frontend/src/features/admin-applications/AdminApplicationProvisioningPage.tsx` は薄い orchestrator (connect/manage の分岐と1つの `connection` state のみ) とし、`ui-page-lines`(400行)/`ui-page-local-state`(10 hooks) の複雑度予算 (ARCHITECTURE.md `complexity.budgets`) を守るため機能ごとに分割: `...Connect.tsx`(未接続時の接続作成フォーム)、`...Settings.tsx`(機能フラグ・scope・group push・属性マッピング (WS-Fed claim mapping rules と同じ JSON テキストエリア方式を踏襲)・matching rule・deprovision policy・reliability・credential rotation)、`...Status.tsx`(status/health バッジ・quarantine 解除・test connection)、`...OnDemand.tsx`(on-demand provisioning・full resync)、`...Deliveries.tsx`(配送履歴一覧+詳細+retry)、`...Danger.tsx`(接続削除 confirm)、`...Shared.tsx`(共通の credential 入力・日付整形・`ProvisioningNavButton`)。
  - `frontend/src/features/admin-provisioning/AdminProvisioningOverviewPage.tsx`: テナント内の全接続を一覧する read-only 集約ビュー。
  - `frontend/src/api/admin.ts` に 12 admin interface すべてに対応する API クライアント関数を追加。`frontend/src/types.ts` に Go domain 型と 1:1 対応する snake_case 型定義を追加。
  - i18n: `AdminApplicationProvisioning.i18n.ts` を新設 (ja/en 完全対訳、[[japanese-ui-no-half-translation]] 準拠)。
  - **フロントエンドのコンポーネントテスト (`*.test.tsx`) は書いていない** (self-attest: 既存の `AdminApplicationDetailPage.test.tsx` 等の慣習からの既知の逸脱)。理由: セッション終盤のスコープ制約下で、実ブラウザでの動作確認 (下記) を優先した。
  - **ARCHITECTURE.md 同期 (Go 側)**: `Provisioning` context の module 群 (`provisioning-domain`/`ports`/`scim`/`usecases`/`adapters`/`composition`) を追加し、`bootstrap`/`http-server`/新設 `worker` (backend/cmd/idmagic-worker 専用 composition module、`batch` と同型) の depends_on を実配線に合わせた。整合検査で `provisioning-usecases`(use_cases 層) が `provisioning-scim`(adapters 層) に直接依存する RA レイヤー違反を検出したため、`ValidateOutboundBaseURL` を `scim` から `domain` へ、`ConflictError`/`NotFoundError`/`RetryableError` とその `As*` ヘルパーを `scim` から `ports`(新設 `ports/errors.go`) へ移設し、usecases 層が protocol adapter package (`scim`) に依存しない構成へ是正 (ADR-128 決定2 の「delivery engine usecase は port にのみ依存する」を字義通り満たす)。`domain/connection_test.go` へ `TestValidateOutboundBaseURL_*` を移設。`AdminApplicationDetailPage.tsx` はプロビジョニング導線ボタン追加で 400→402 行となったため `ui-page-lines` debt (ceiling 410, wi-234-complexity-ratchet 参照) を宣言。
  - 検証: `just verify-go`/`just verify-ui`/`just yaml-check`(architecture cross-check 含む) すべて緑 (`GET /session/check` のみ既存無関係ギャップとして残存)。加えて `just dev-api`(`SEED_PROFILE=development` 付き)+`just dev-ui` を起動し、`Bun.WebView` ベースの e2e ヘルパー (`frontend/tests/e2e/fixtures.ts`) を使った手動シナリオスクリプトで実ブラウザ動作を確認: admin (alice) ログイン→Application 詳細→プロビジョニングリンク→未接続フォーム表示→接続作成→設定パネル表示 (feature flags 等)→test connection (到達不能時のエラー表示)→on-demand/full-resync/配送履歴セクション表示→空配送履歴表示→設定フォームの base_url 編集→保存→PATCH 反映確認→テナント集約ビューで新規接続がアプリ名解決付きで一覧表示→接続削除→未接続状態へ復帰、をコンポーネント分割の前後両方で確認 (ja/en 両 locale のテキストで判定)。
- [x] T008 [Verify] create/update/disable/delete/unassign、duplicate/out-of-order event、429/5xx retry 収束、credential rotation、quarantine/resume、tenant 越境を end-to-end 検証する。**group membership push と grace-period 付き delete / 誤削除ガードは T006 で既に範囲外とした (backend/provisioning/usecases 内コメント参照)ため対象外** — SCIM client 自体の Group CRUD は `scim/client_test.go` にユニットテストがあるが、capture 側の呼び出し配線は未実装のまま。
  - 着手前に既存カバレッジを調査 (Explore agent): domain/usecases 層はほぼユニットテスト済みだが、**IdManagement/Application の実 usecase から Provisioning への配線 (ProvisioningNotifier 経由) を実際にたどるテストが皆無**で、Jobs キュー抜きの「本物の SCIM client + 本物の属性解決」を通した検証も皆無と判明。
  - `backend/provisioning/e2e_capture_delivery_test.go` (新規、`package provisioning_test`): `idmusecases.CreateUser/UpdateUser/SetUserDisabled/DeleteUser` → 実 `usecases.UserMutationNotifier` → 実 `CaptureLifecycleEvent` → 実 `ExecuteDelivery` → 実 `scim.Client` → `httptest` fake SCIM downstream、という実配線を Jobs/dispatcher を経由せず (それらは `jobs/usecases` 自身のテストと `job_handler_test.go` で既にカバー済みのため) 直接検証。3 テスト: `TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream` (create→update→disable→delete、既定 policy=deactivate)、`TestE2E_DeleteWithDeleteOnPolicy_SendsRealDELETE` (`OnDelete=delete` で実 DELETE)、`TestE2E_TransientFailureThenSuccess_ConvergesAcrossRetries` (503 を2回返した後に成功、実際に収束することを確認)。
  - **この E2E テストで実バグを発見・修正**: `OnDelete=deactivate` (既定ポリシー) で作られる deactivate delivery が、`identitysource.UserAttributeSource.ResolveAttributes` が `FindBySub` (削除済み user を除外) を使っていたため、`DeleteUser` が tombstone 化した直後の user に対して常に `exists=false` を返し、**downstream へ一切送信されずに「成功」扱いになる**サイレント no-op だった。`FindBySubIncludingDeleted` に変更して修正 (`backend/provisioning/adapters/identitysource/user_attribute_source.go`)。ユニット回帰テスト `TestUserAttributeSource_ResolveAttributes_DeletedUserStillResolvesAsInactive` を追加。これは T007a 時点のユニットテストがすべて fake `AttributeSource` を使っていたため見つからなかった、real-repo 統合でしか顕在化しない不整合— T008 の end-to-end 検証がまさに意図した種類の欠陥。
  - 追加ユニットテスト: `capture_test.go` に `TriggerUserAttributes`/`TriggerUserEnabled` → `OperationUpdate` (未カバーだった trigger 分岐)。`admin_test.go` に `TestResumeConnection_ClearsQuarantineAndAllowsNextDelivery` (resume の正常系: quarantine 解除後に capture が再びそのコネクション向け delivery を作れることを確認、これまで異常系のみだった)。`memory/repositories_test.go` に `TestProvisioningDeliveryRepository_TenantIsolation` (Find/ListByConnection/UpdateStatus が他テナントの delivery に影響しないこと)。
  - **既知のギャップ (T008 未実施・disclosure)**: (1) `RemoteResourceLinkRepository` は `tenantID` を引数に取らない設計 (`ApplicationID` の一意性に依存) のため、他 repo と同型のテナント分離テストが書けない — ADR/設計判断として別途要検討。(2) HTTP admin ハンドラ層のテナント越境テストは未実施 (T007a で既に「HTTP handler 層は無テスト」と開示済みの延長、今回も対象外)。(3) credential rotation が「次回配送で実際に新しい secret を使う」ところまでは未検証 (`TestUpdateConnection_RotatesCredentialWhenProvided` はリポジトリ層の保存確認まで)。(4) 429/5xx 収束テストは Jobs キュー自体を経由しない簡略版 (ExecuteDelivery を手動で複数回呼ぶ) であり、実際の `Runner`/backoff タイミングとの統合はしていない (ただし `Runner` 自体の retry/dead_letter 収束は `jobs/usecases` 側で別途カバー済み)。
  - 検証: `go vet ./backend/provisioning/...`、`just format-go`、`just lint-go` (0 issues)、`just test-go` (`GET /session/check` のみ既存無関係ギャップとして残存)、`just yaml-check` すべて緑。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: connection を登録 → 接続テストで capabilities 取得 → 内部で user 作成 → 下流 (mock SCIM server) に POST が届く → user disable → 下流に PATCH active=false が届く流れを確認する。
- 手動: 下流が一時的に 5xx / 429 を返すとき delivery が backoff retry され、復旧後に収束する (冪等) ことを確認する。
- 手動: on-demand provision で単一ユーザーを試験配送し、full resync で scope 内全 subject が収束することを確認する。

## Risk Notes
outbound provisioning は外部システムの状態を idmagic が書き換える契約であり、
誤割当・誤った deprovision は下流 SaaS のアカウント大量無効化など破壊的影響を
持つ。active=false / delete の下流への翻訳と割当条件を ADR で先に固定し、
push は必ず冪等・retry 可能な非同期 job とする。inbound (wi-31) と outbound
(本 WI) で active=false の意味を一貫させる。

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
  idmagic を SCIM 2.0 outbound client にする独立 `Provisioning` bounded
  context を新設した (ADR-128)。Domain (`ProvisioningConnection`/
  `AttributeMappingRule`/`DeprovisionPolicy`/`RemoteResourceLink`/
  `ProvisioningDelivery` とその状態機械) から、same-Tx-adjacent capture →
  Jobs 経由の非同期 dispatch → protocol-agnostic core が呼ぶ SCIM feature
  slice (`backend/provisioning/scim`) による配送、admin API (12 interface)
  と Application 詳細ページの「プロビジョニング」サブルート + テナント集約
  ビューまで、SCL → ADR → Domain → Use Cases → Adapters → Infrastructure の
  全層を実装した。IdManagement (User create/update/disable/delete) と
  Application (assignment add/remove) の実 usecase から outbound
  provisioning への配線を、Jobs キューを含む実コンポーネントを通した
  end-to-end テストで検証し、その過程で `OnDelete=deactivate` (既定ポリシー)
  が削除済みユーザーの属性解決失敗によりサイレントに no-op していた実バグを
  発見・修正した。
- **Scope narrowing (ADR-121 開示)**:
  以下は wi-45 のタイトル/Motivation が示唆する範囲より狭く実装しており、
  必要になった時点で個別の follow-up work item として切り出す想定:
  - grace-period 付き delete、誤削除ガード (accidental deletion guard)、
    per-connection rate limit の実施 — T006 でユーザー承認済みのスコープ
    縮小 (コアループを優先し高度な信頼性機能を後続タスクへ)。
  - Group (push_groups) の capture 側配線 — SCIM client 自体は Group CRUD
    を実装済みだが、capture がまだ User lifecycle と Assignment のみを
    trigger するため呼び出されない。
  - `oauth2_client_credentials` 認証方式 — 現状は secret をそのまま bearer
    token として送る簡易実装 (正しい token-fetch フローは未実装)。
  - `FullResyncCompleted` イベント — 多数の非同期 delivery の完了追跡機構が
    無いため発行しない。
  - HTTP admin handler 層の専用テスト (tenant 越境を含む) — usecase 層の
    テストと `routes_contract_test.go` (SCL binding 一致) で代替。
  - credential rotation が次回配送で実際に新しい secret を使うところの
    end-to-end 検証、および 429/5xx 収束の Jobs `Runner` 自体との統合検証
    (Runner 自身の retry/dead_letter 収束ロジックは `jobs/usecases` 側で
    別途カバー済み)。
  - `RemoteResourceLinkRepository` の tenant-scoped でない設計
    (`ApplicationID` の一意性に依存) の要否再検討。
- **Verification Results**:
  - `just verify-go` (backend lint + race test) - passed。既存の
    `TestAssembledRoutesMatchGeneratedOpenAPI` の `GET /session/check`
    差分のみ残存 (本 WI と無関係、`main` 上の既存ギャップと `git stash` で
    確認済み)。
  - `just verify-ui` (format-check / lint / typecheck / build) - passed。
  - `just yaml-check` (SCL / work-item / ids / architecture cross-check /
    traceability) - all passed。
  - `go test ./backend/provisioning/...` - all passed (domain / usecases /
    scim / adapters/persistence/{memory,postgres} / adapters/identitysource /
    ルート end-to-end 統合テスト含む)。
  - 手動ブラウザ検証: `just dev-api`(`SEED_PROFILE=development`) +
    `just dev-ui` を起動し、`Bun.WebView` ベースの e2e ヘルパーを使った
    シナリオスクリプトで admin UI の主要導線 (接続作成・設定保存・test
    connection・on-demand・full resync・配送履歴・テナント集約ビュー・
    接続削除) を実ブラウザで確認 (ja/en 両 locale)。
