---
status: suggested
authors: [tn]
created_at: 2026-07-19
---

# ADR-128: outbound provisioning (wi-45) を protocol 拡張可能な独立 `Provisioning` context として切り出し、protocol 非依存の配送コアを共有する

## コンテキスト

[[wi-45-outbound-scim-provisioning]] は idmagic を SCIM 2.0 **client** にし、下流 SaaS へ
user/group を push する outbound lifecycle management を実装する。実装前に確定すべき論点が
3 つある: (1) context 境界、(2) context 内部を将来の protocol 追加へどう耐えさせるか、
(3) 配送機構。

### (1) context 境界 — 方向で分ける

[[wi-31-scim2-provisioning]] (`Scim` context) は SCIM **server** として外部 IdP から受信する
inbound 方向を、本 WI は idmagic が SCIM **client** として下流へ送信する outbound 方向を扱う。
両者は「どちらも SCIM 2.0 を話す」点以外ほぼ重ならない: 権威方向 (外部が SoR / idmagic が SoR)、
主 aggregate (`ScimClient`/token vs `ProvisioningConnection`/`RemoteResourceLink`)、起動契機
(外部 HTTP request vs 内部 lifecycle event)、admin API が反転する。"User"/"Group" という名詞は
共通でも動詞と invariant が逆であり、1 context に同居させると ubiquitous language が曖昧になる。
しかもその「SCIM 2.0 を話す」共通性すら、受信して評価する server と、組み立てて送信する client では
実装が反転しており、コード共有の余地は小さい (decision 3)。→ 別 context にする。

### (2) 将来の protocol 追加と命名

outbound は近い将来 SCIM 以外の protocol (MS Graph / EntraID、Google Directory 等) を持つ
可能性が高い。ディレクトリ/アーキテクチャの後追い作り替えは高コストなので、仮に SCIM だけで
終わるとしても、context 内部は**最初から protocol 追加に耐える構造**で作る (単一 protocol を
前提にした flat 構成にはしない)。

命名について、`Inbound` / `Outbound` という対称 context にして `Scim` を `inbound/scim` へ移す
案も検討したが、本 WI では採らない:

- inbound 側の**正しい** taxonomy はまだ設計されていない。inbound には少なくとも (a) 外部が
  我々の API を叩く受動 server 型 (現 `Scim`)、(b) 管理者がファイルを上げる upload 型
  (現状 `backend/idmanagement/usecases/user_import.go` の CSV import。これは適所でないので
  別 WI で然るべき場所へリファクタすべき)、(c) 我々が外部 API を能動的に叩いて取り込む pull 型
  (予定は無いが将来可能性ゼロでない) が想定され、起動契機・権威・runtime が三者三様で、`Inbound`
  一語では判別できない。この設計は独立 WI に値する。
- 未設計の inbound スキームに outbound の命名を賭けるより、outbound を **capability 名
  `Provisioning`** (IAM 慣習で "provisioning" = outbound) で切り出す方が低リスク。inbound が
  どう決着しても常に正しく、将来 symmetric にしたくなれば `Provisioning` → `Outbound` は
  単純リネームで済み、**内部構造 (下記決定 2) は外側の名前に依存せず不変**。高コストなのは
  内部分解であって外側の名前ではない。
- 稼働中 `Scim` の physical rename も本 WI に含めない (別 refactor WI。[[wi-254-backend-feature-vertical-slice-convention]] /
  [[wi-255-authentication-feature-vertical-slices]] / [[wi-256-oauth2-feature-vertical-slices]]
  と同じく、物理再配置は専用 WI に切り出す)。

申し送り: 「client として外部 API を能動駆動する」machinery (connection 登録・credential 管理・
スケジューリング・retry・remote resource 相関) は、outbound push と将来の active-pull inbound で
大半が共通になり得る。本 WI で作るコアはその再利用元になりうるので、将来 inbound taxonomy を
設計する WI はこのコアの抽出可能性を出発点にする。

### (3) 配送機構 — 既存 outbox は流用できない

wi-45 の草案は「committed lifecycle event を既存 `outbox` (Relay drain) から観測する」ことを
前提にしていたが、コードベースの実態調査で成立しないことが分かった。

- `backend/shared/adapters/eventsink/relay.go` の `Relay` は `outbox` テーブルを drain して
  Kafka / PubSub / log という**外部 transport へ一方向配送するだけ**で、同一プロセス内の他
  context が購読できる in-process consumer 経路は存在しない。ADR-117 §2a も同じ制約を指摘し、
  `IdGovernance` でも「今 outbox 購読の consumer を作るのは本 WI に対して過大」と判断して
  published port 方式 (`UserWorkflowCapture`) に倒した前例がある。
- `backend/oauth2/adapters/persistence/postgres/outbox.go` の `eventTopics` トピックマップは
  `GroupCreated`/`GroupMemberAdded` など Group 系のみ登録済みで、`UserCreated` / `UserUpdated` /
  `UserDisabled` / `UserEnabled` / `UserDeleted` や `ApplicationAssigned` / `ApplicationUnassigned`
  は未登録。`OutboxEventSink.Emit` は未登録イベントに対し outbox 行を書かずエラーを返すのみで、
  本 WI が観測対象とする User/Assignment イベントは現状 outbox に一切現れない。
- そのエラーは呼び出し元 `bootstrap.Dependencies.NewEmitFunc`
  (`backend/cmd/internal/bootstrap/audit_event_record.go`) がログ出力するだけで、呼び出し元の
  DB トランザクションを失敗させない。しかも `NewEmitFunc` は
  `context.WithTimeout(context.Background(), 2*time.Second)` という**呼び出し元 context と
  無関係な新規 context** を使うため、`admin_users.go` 等が呼ぶ `deps.Emit(...)` は User/Assignment
  の Save と同一 Postgres トランザクションでは実行されない。つまり `Deps.Emit` 経由の outbox
  書き込みはベストエフォート・非トランザクショナルであり、ADR-113/117 の「同一 Tx capture」
  (`UserMutationCommitter` / `UserWorkflowCapture` という明示的な `Pool.Begin(ctx)` adapter) とは
  別物で、`Deps.Emit` はそこに関与しない。

三重 (トピック未登録・非原子的書き込み・in-process consumer 不在) の欠落があるため、
「既存 outbox をそのまま流用する」では wi-45 が要求する「確実性のため push は同期呼び出しでなく
outbox/queue を介した非同期・冪等・retry 付き」を満たせない。

## 決定

### 1. 独立 bounded context `Provisioning` を新設する (outbound 専用)

`Scim` context は inbound server 専用のまま責務・語彙を変えない。outbound (connection / mapping /
delivery / remote link / deprovision) は新規 `Provisioning` context
(`spec/contexts/provisioning.yaml`) に置く。`TenantRef` / `ApplicationRef` /
`ApplicationAssignmentRef` / `UserRef` / `GroupRef` / `JobRef` を published language 経由で
参照する。`spec/scl.yaml` の `context_map` に `Provisioning` を追加し、`depends_on` は
`Tenancy` / `Application` / `IdManagement` / `Jobs`、すべて `via: published_language`
(`IdGovernance` の `depends_on` 構造と同型)。命名は §コンテキスト (2) の通り capability 名を採る
(`Outbound` でも可。symmetric 化は inbound taxonomy WI で再検討)。

### 2. 内部構造 = protocol 非依存コア (context ルート) + protocol 別 feature slice

outbound provisioning の振る舞いの**大半は protocol 非依存**である: connection の envelope
(scope、feature flags、信頼性、通知)、`DeprovisionPolicy`、`AttributeMappingRule`、
`ProvisioningDelivery` / `RemoteResourceLink` の lifecycle、配送エンジン (決定 4 の same-Tx
capture、queue、retry/backoff、quarantine、順序保証、full resync、on-demand)。protocol 固有なのは
wire client (resource marshal / endpoint / verb / filter) と一部の接続設定 (auth 方式、capability
discovery の仕方、属性 schema 既定) のみ。

したがって:

- **protocol 非依存コア**を context ルート `backend/provisioning/{domain,ports,usecases,adapters}`
  に置く。ここに neutral aggregates (`ProvisioningConnection` envelope / `ProvisioningDelivery` /
  `RemoteResourceLink` / `DeprovisionPolicy` / `AttributeMappingRule`)、配送エンジン usecase、
  `ProvisioningTargetClient` port (protocol の seam)、`provisioning_*` postgres adapter、
  connection CRUD・delivery 一覧/retry の admin http を置く。
- **protocol ごとに feature slice** `backend/provisioning/<protocol>/{domain,ports,usecases,adapters}`
  を置き、`ProvisioningTargetClient` を実装する。本 WI は `provisioning/scim/` (SCIM wire client +
  SCIM 固有の接続設定/capability discovery)。将来 `provisioning/entraid/` (MS Graph)、
  `provisioning/googledir/` を並べる。新 protocol の追加はコアを触らず feature slice の追加で済む。
- `module.go` は context ルートに 1 つ据え置く。package 名は各層名のまま (`domain`/`ports`/
  `usecases`/`http`)、`provisioning/adapters/http/routes.go` のような集約点では named import
  (`scimtarget` 等) を用いる。

これは idmagic 自身の先例と同型である: `Application` (protocol 中立の上位概念) + OIDC/SAML/WsFed の
protocol binding (各 protocol context が wire 挙動を所有し opaque key で参照)。ただし outbound の
各 protocol target は OAuth2 ほど巨大でないため、別 context ではなく単一 `Provisioning` 内の
**feature slice 粒度**にする。

なお本構造は [[wi-254-backend-feature-vertical-slice-convention]] の feature-slice 規約に対する
**明示的な variant** である: 通常は「fat な feature slice + thin な共有 root」だが、本 context は
domain の形 (大半が protocol 非依存、protocol は driven-adapter 軸) ゆえに「fat な protocol 非依存
コア (root) + protocol adapter の feature slice」を採る。この variant を `ARCHITECTURE.md` に
記す。

**却下**: (a) 単一 protocol のため flat で始め将来分割 — ディレクトリ/アーキテクチャの作り替えが
高コストで、protocol 追加は近い将来 likely なので最初から protocol slice 構造で作る。(b)
`connection`/`delivery` の関心軸 feature 分割 — 直感的でなく、両者は protocol 非依存コアの
neutral aggregate であって feature ではない。feature 軸は protocol にする。

### 3. 共有 SCIM wire kernel は作らない — outbound は自前の SCIM シリアライズを持つ

wi-45 草案は「唯一の共通項は SCIM の marshal/unmarshal と filter 文法」とし、本 ADR の初稿も
`scim_models.go` + `filter.go` を中立 kernel へ切り出す方針だったが、inbound コードを精査した結果、
実際の再利用は小さく共有はむしろ有害と判断し、方針を反転する。

- `backend/scim/domain/filter.go` (612 行) は inbound server 専用の **parse + 評価**エンジン
  (`ParseFilter(...)` → `FilterExpr.Matches(attrs map[string]any) bool`)。外部が投げる
  `?filter=userName eq "x"` を我々のデータに対して評価する、受信側だからこそ要る機能。outbound は
  409 相関で `userName eq "x"` 程度の filter 文字列を**組み立てる**だけで、パーサ・評価器は要らない。
- `backend/scim/domain/scim_models.go` の `UserResource` / `GroupResource` は inbound が**応答生成に
  使う固定最小サブセット**で、`externalId` も enterprise 拡張も持たず (inbound コードに両者は登場
  しない)、受信 body は `map[string]any` → `mutation.go` (`ParseUserWrite` 等) で別処理される
  (これらの struct は marshal 専用)。outbound は wi-45 の要求 (externalId 相関、enterprise 拡張、
  `AttributeMappingRule` 駆動の任意 `target_path`) を満たす**より柔軟な mapping 駆動シリアライズ**が
  要り、inbound の固定 struct を共有すると outbound を縛る。
- 実質的な重なりは discovery 系 struct (`ServiceProviderConfig` 等、inbound が生成 / outbound が
  接続テストで parse) と RFC 固定の schema URN 定数程度で、小さい。今は各自定義でよい。

よって outbound は `backend/provisioning/scim/` 配下に自前の SCIM シリアライズを持ち、inbound の
`backend/scim` には手を入れない。共有 package の抽出は、実際に重複が現れた時点で on-demand で行う
(rule of three)。この判断は decision 2 の「flat にしない」と矛盾しない: **構造 (context /
protocol slice) は作り替えが高コストゆえ前もって決め、共有コードは未成熟な二表現の早期結合が有害で
後からの抽出が安価ゆえ on-demand で切り出す**、という非対称に基づく。

### 4. 配送は「outbox を読む」でなく「同一 Tx で `ProvisioningDelivery` を書く」

既存 `outbox` (Relay drain) は観測経路として使わない。代わりに ADR-113/117 で確立済みの
**same-Tx capture** パターン (`UserMutationCommitter` / `UserWorkflowCapture` 型の明示的
トランザクション adapter) を Provisioning コア (決定 2) にも適用する。

- `IdManagement` の User 変更経路 (`captureUserMutation` が呼ぶ commit 処理) と `Application` の
  assignment 変更経路 (`assignments.go`) に、Provisioning が実装する published port
  (`ProvisioningCapture` 相当) を追加する。この port は**呼び出し元の Postgres トランザクション
  内**で、scope (`assigned_only` / `all_users`) に一致する有効な `ProvisioningConnection` ごとに
  `ProvisioningDelivery` (`pending`) 行を挿入する。
- これにより「User/Assignment の commit と provisioning delivery の生成が同一トランザクションで
  確定する」という wi-45 の確実性要件を、既存 outbox の非原子性に依存せず満たす。二重 queue には
  ならない — `ProvisioningDelivery` テーブル自体が Provisioning にとっての outbox 相当であり、
  [[wi-126-async-job-runner]] がそこから配送する。idempotency key
  `(tenant, connection, source_type, source_id, source_version)` は維持する。
- `Deps.Emit` (`NewEmitFunc`) 経由の既存 outbox 書き込み / audit mirroring は監査目的で現状維持し、
  本決定では変更しない。Provisioning はこれとは別経路 (same-Tx capture port) で確実性を得る。

### 5. deprovision / 相関 / 信頼性の骨子は wi-45 §設計 のまま採用する

connection の Application 帰属、scope=`ApplicationAssignment` 再利用、真実源=内部 aggregate /
下流=mirror、`active=false`/delete の deprovision マトリクス (trigger×action、grace period、
誤削除ガード)、`externalId` 相関と 409/404 冪等解決、SSRF/secret 保護は wi-45 本文の調査・設計を
正としてそのまま SCL (T002) へ変換する。

## 却下した代替案

- **単一 `Scim` context のまま `scim/inbound` / `scim/outbound` / `scim/wire` に分割**: 物理的
  凝集は保てるが、"client"/"User" 等の語彙が inbound/outbound で意味反転したまま同一 context に
  同居し、ubiquitous language が曖昧なまま残る。
- **`Inbound` / `Outbound` の対称 context を今作り `Scim` を `inbound/scim` へ移す**: inbound の
  正しい taxonomy (受動 server / upload / 能動 pull) が未設計で、単一 `Inbound` に束ねられるか
  未確定。稼働中 `Scim` の rename も高リスクで別 WI 相当。未設計の inbound スキームに outbound の
  命名を賭けず、outbound だけを capability 名で切り出す (決定 1)。
- **単一 protocol のため flat で始め将来 protocol 分割**: ディレクトリ/アーキテクチャの後追い
  作り替えが高コスト。protocol 追加は近い将来 likely なので最初から protocol slice 構造で作る
  (決定 2)。
- **`connection`/`delivery` の関心軸 feature 分割**: 直感的でなく、両者は protocol 非依存コアの
  neutral aggregate。feature 軸は protocol にする (決定 2)。
- **既存 `outbox` をそのまま観測する (wi-45 当初案)**: §コンテキスト (3) の通りトピック未登録・
  非原子的書き込み・in-process consumer 不在の三重の欠落があり、確実性要件を満たさない。
- **Relay の Publisher (Kafka/PubSub) を Provisioning 用 consumer で subscribe する**: 既定の
  `EVENT_SINK=console` 環境では動作せず、新規 consumer worker の構築も要る。ADR-117 §2a が同種の
  判断を「本 WI に対して過大」として既に退けている。
- **同期呼び出しで下流 SCIM へ即時 push する**: wi-45 Motivation が明示的に禁止する (下流障害が
  User 更新 API の可用性を巻き込む)。

## 影響

- `spec/scl.yaml` `context_map`: `Provisioning` エントリを新設 (`depends_on` =
  Tenancy/Application/IdManagement/Jobs、すべて `via: published_language`)。
- `spec/contexts/provisioning.yaml` (新規): `ProvisioningConnection` ほか wi-45 §Scope の
  models/interfaces/events/state machine/scenarios を追加 (T002)。protocol 非依存コアと
  `ProvisioningTargetClient` seam、SCIM binding の位置づけを記す。
- `spec/contexts/identity-management.yaml` / `spec/contexts/application.yaml`: User Save /
  Assignment 変更経路に Provisioning capture 用 published port 呼び出しを追加する契約を記載
  (詳細は T002/T003 で SCL 化)。
- `backend/scim`: 変更しない (共有 kernel を作らない。decision 3)。
- `backend/provisioning/` (新規): `module.go` を context ルートに 1 つ。ルート
  `{domain,ports,usecases,adapters}` に protocol 非依存コア (neutral aggregates + 配送エンジン +
  `ProvisioningTargetClient` port + `provisioning_*` postgres + connection/delivery admin http)、
  `provisioning/scim/{domain,ports,usecases,adapters}` に SCIM target feature (自前の SCIM
  シリアライズ + wire client + SCIM 固有設定)。
- `backend/idmanagement/usecases/admin_users.go` (`captureUserMutation` 経路) と
  `backend/application/usecases/assignments.go`: Provisioning capture port 呼び出しを追加する
  配線変更 (T003 以降)。
- `ARCHITECTURE.md`: `Provisioning` context + module 台帳 + depends_on エッジを同期し、
  決定 2 の feature-slice variant (protocol 非依存コア root + protocol adapter feature) を注記する。
- 既存 `outbox` (`eventTopics` マップ、`Deps.Emit`/`NewEmitFunc`) は変更しない (監査・外部
  transport 用途は現状維持)。
- **申し送り (別 WI、作成済み)**: (i) inbound taxonomy の設計 (受動 server / upload / 能動 pull の
  整理) = [[wi-258-inbound-integration-taxonomy]]、(ii) 稼働中 `Scim` context の inbound-honest な
  rename = [[wi-259-rename-scim-inbound-server-context]]、(iii) CSV import
  (`idmanagement/usecases/user_import.go`) の適所への移設 =
  [[wi-260-relocate-csv-user-import-to-inbound]]。いずれも本 WI では触れない。
