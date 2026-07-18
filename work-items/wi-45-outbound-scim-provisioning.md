---
depends_on: [wi-126-async-job-runner]
status: pending
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
- [ ] T001 [Decision/ADR] context 境界 (§context 境界の決定) を確定し、outbound ownership・scope=`ApplicationAssignment` 再利用・真実源/mirror・deprovision マトリクス・delivery guarantee (idempotency key / 順序 / quarantine)・secret 保護・`outbox`+[[wi-126-async-job-runner]] 利用を ADR に記録する。
- [ ] T002 [SCL] 確定 context に connection、feature flags、scope、attribute mapping、matching、deprovision policy、remote link、delivery lifecycle、management interfaces/events/constraints/contracts/scenarios を追加し、shared SCIM wire kernel の位置づけを記す。派生成果物を再生成する。
- [ ] T003 [Domain] `ProvisioningConnection` (Application ごと uniqueness)、`AttributeMappingRule`/`MatchingRule`/`DeprovisionPolicy`、`RemoteResourceLink`、`ProvisioningDelivery` を実装し、source version 単調性・冪等 key・tenant/connection 一意性をテストする。
- [ ] T004 [Persistence] `provisioning_connections`/`provisioning_remote_links`/`provisioning_deliveries` repository、credential reference (write-only)、lease/retry/next_attempt fields を memory/PostgreSQL に追加する。
- [ ] T005 [SCIM Client] wire kernel + outbound adapter で discovery、Users/Groups CRUD、PATCH→PUT fallback、pagination/error/`Retry-After`、409/404 冪等、redirect/TLS 制限を実装し mock server contract test を作る。
- [ ] T006 [Event/Worker] `outbox` からの user/group/assignment projector と、[[wi-126-async-job-runner]] 上の冪等 job handler、順序保証、retry/dead-letter、rate limit、quarantine、grace-period delete を実装する。
- [ ] T007 [Admin HTTP/UI] Application 詳細の「プロビジョニング」サブルート (接続 test / 機能トグル / scope / mapping editor / push groups / deprovision policy / delivery list・detail・retry / on-demand provision / full resync)、secret write-only 表示、テナント設定の read-only 集約ビューを追加する。
- [ ] T008 [Verify] create/update/disable/delete/unassign、group membership、duplicate/out-of-order event、429/5xx retry 収束、grace-period / 誤削除ガード、credential rotation、tenant 越境を end-to-end 検証する。

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
