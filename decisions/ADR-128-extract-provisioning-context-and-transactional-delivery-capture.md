---
status: suggested
authors: [tn]
created_at: 2026-07-19
superseded_by: [ADR-141]
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

なお、この節が置いた仮の 3 分類 (受動 server / upload / 能動 pull) と「CSV import は適所でない」という
申し送りは [[ADR-141-inbound-identity-sourcing-taxonomy]] が上書きした (分類軸は権威と source binding、
管理者 CSV import は IdManagement に残す)。以下の決定 1〜5 は有効のまま。

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

outbound (connection / mapping / delivery / remote link / deprovision) を独立 bounded context
`Provisioning` として新設する (決定 1)。命名は §コンテキスト (2) の通り capability 名を採り、方向語
(`Outbound`) には依存しない。内部構造は protocol 非依存コア (context ルート) + protocol 別
feature slice とし (決定 2)、共有 SCIM wire kernel は作らず outbound は自前のシリアライズを持つ
(決定 3、rule of three)。配送は既存 `outbox` の非原子性・未登録トピック・in-process consumer 不在
という三重の欠落ゆえ流用できず、ADR-113/117 の same-Tx capture パターンを Provisioning コアにも
適用し、`IdManagement`/`Application` の commit と同一トランザクションで `ProvisioningDelivery`
(`pending`) 行を挿入する (決定 4)。deprovision / 相関 / 信頼性の骨子は wi-45 §設計のまま採用する
(決定 5)。この決定 1〜5 は [[ADR-141-inbound-identity-sourcing-taxonomy]] によっても有効のまま
維持されている (同 ADR が上書きしたのは inbound taxonomy に関する申し送り部分のみ)。メカニズムの
詳細は [backend/provisioning/ARCHITECTURE.md](../backend/provisioning/ARCHITECTURE.md) に移した。

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
