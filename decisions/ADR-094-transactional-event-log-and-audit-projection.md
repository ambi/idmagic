---
status: accepted
authors: [tn]
created_at: 2026-07-11
---

# ADR-094: 業務状態と不変 event log の同一 transaction 確定、audit を projection とする責務分離

## コンテキスト

現在の `DomainEvent` は HTTP 層の fire-and-forget `Emit`（`backend/bootstrap/server.go`
の `emit` クロージャ）から、業務更新とは別の PostgreSQL 操作で `outbox` /
`audit_events` へ記録される。`emit` 内部の `EventSink.Emit` と `AuditEventRepo.Append`
はいずれもエラーを無視するため、業務状態だけが commit されて event が失われる経路が
ある。呼び出しは use case 層（`identitymanagement`/`authentication` の各 usecase）と
HTTP handler 層（`oauth2`/`tenancy` の admin handler）の双方に散在し、どの mutation が
どの event を確定させるべきかが呼び出し側の実装に委ねられている。

さらに `outbox`（`deploy/schema/postgres.sql`）はイベント本文（`event_type` /
`topic` / `payload`）と Kafka 配送状態（`published_at` / `attempts` / `last_error`）を
同じ可変行に持つ。行が配送処理で書き換わるため、長期の監査原本としての性質
（不変・追記専用）と配送キューとしての性質（可変・進行状態）が 1 テーブルに同居し、
どちらの責務も歪める。

[[wi-146]]（監査境界の抽出）により `backend/audit` が独立した bounded context として
成立し、監査イベントの検索用 read model（`audit_events` /
`audit_event_search_attributes`）を所有している。本 ADR はこの read model の「原本」を
どこに置くかを確定し、業務状態と event の記録を安全な transaction 境界に収める。

対象は [[wi-184]]。IdMagic は確定したセキュリティ・管理上の事実を再生可能に保つ
必要がある一方、Kafka や外部 SaaS 呼出を API transaction に含めると DB connection を
外部 I/O 待ちで保持し続けることになり、`CancellationConsistency` /
`DatabaseResilience`（`spec/contexts/system.yaml`）の運用目標と相反する。

## 決定

1. **`event_logs` を監査原本兼 Kafka 配送源とする。** 業務コマンドが確定させる
   `DomainEvent` のうち `public_integration` / `audit_only` に分類されるものは、
   `event_logs` に不変の 1 行として追記する。行は追記後に更新しない
   （SCL `EventLogRecord`、`spec/contexts/system.yaml`）。テーブル名は複数形
   （`audit_events` 等、既存 schema の大半に合わせる。単数の `outbox` は例外）。
2. **`event_deliveries` を可変な配送管理専用テーブルとする。** イベント本文は持たず
   `event_id` で `event_logs` を参照し、Kafka publish の試行回数・直近エラー・配送状態
   だけを保持する（SCL `EventDeliveryRecord`）。既存 `outbox` はこの二分割の前身であり、
   全 mutation の移行が完了し `outbox` を使うコードが無くなった時点で `DROP TABLE` する。
   IdMagic は未リリースのため保存データの互換維持・段階照合は不要で、`outbox` /
   `audit_events` はテーブルへの参照コードが残っている間だけ物理的に併存させれば足りる
   （データ移行 script は不要）。
3. **DB transaction の対象を同一 PostgreSQL 内の「業務状態 + event_logs」に限定する。**
   Kafka publish、SMTP 送信、HTTP 呼出、CSV 全件処理、外部 SaaS の結果待ちは
   transaction に含めない（SCL invariant `EventLogAtomicWithBusinessState`）。
   これにより mutation の transaction は短時間で完結し、外部 I/O 障害が DB connection を
   占有しない。
4. **`DomainEvent` を 3 分類に棚卸しする。** `public_integration`（他 context・外部への
   公開契約、`event_logs` 記録 + Kafka publish 対象）、`audit_only`（`event_logs` 記録のみ、
   Kafka publish 対象外）、`telemetry`（`event_logs` 対象外の運用計測）。分類は
   `DomainEventClassification`（SCL enum）で固定し、未分類・未経路の `DomainEvent` は
   CI で検出する（[[wi-184]] T004）。
5. **Kafka は at-least-once 配送とし、`event_id` を消費側の冪等キーとする。** relay は
   commit 済み `event_logs` だけを取得して publish し、同じ `event_id` を Kafka message
   header に付与して `event_deliveries` の状態を更新する。publish が ack された後の
   relay プロセス停止による重複配送は許容し、消費側が `event_id` で重複を排除する
   （SCL invariant `EventDeliveryEventuallyDelivered`、objective
   `RelayAtLeastOnceDelivery`）。Kafka の exactly-once 配送、Kafka transaction、
   分散 transaction / 2PC は対象外とする。
6. **監査検索 read model は `event_logs` からの projection とする。** `audit` context の
   `audit_events` / `audit_event_search_attributes`（`AuditEventProjection`、
   `spec/contexts/audit.yaml`）は正本ではなく、`event_logs` から再構築可能な派生データと
   位置付ける。原本（`event_logs`、System context）と projection（`audit` context）の
   所有は分離したまま、`audit` は `event_logs` を読む port を介して接続する
   （[[wi-146]] が確立した境界を変更しない）。read model 自体の非同期 projection worker・
   checkpoint・rebuild は [[wi-185]] の scope とし、schema・互換維持もそちらで自由に
   決める。
7. **HTTP 層の fire-and-forget `emit` を廃止し、command 単位の transaction runner へ
   置き換える。** transaction-bound repository と event recorder を使って業務更新と
   `event_logs` 追記を同一 commit にする。対象は通常の単件・軽量 mutation とし、CSV
   import はバッチ単位で適用する（[[wi-96]] の scope）。connection を HTTP request 全体で
   保持する middleware 方式は採らない。
8. **`event_logs` / `event_deliveries` の Go 実装は `backend/shared` に置く。**
   `spec/scl.yaml` の `context_map` で `Audit` の `depends_on` は `Tenancy` のみであり、
   Audit 自身の description も「イベントの発火元は各 context のままであり、Audit は
   それらを横断して読む窓口を所有する」と定める、読み取り専用の read model context
   である（[[wi-146]]）。一方 `event_logs` への書込みは identitymanagement /
   authentication / oauth2 / tenancy / application / saml / wsfederation / scim の
   あらゆる単件 mutation の transaction から呼ばれる、全 context 共通の書込み経路である。
   これを `backend/audit`（改名しても同様）に同居させると、Audit の「読み取り専用 read
   model」という境界を逆流させ、`context_map` に無い新しい依存の向き（各 context →
   Audit）を大量に生む。逆に、専用の新規 context package（`backend/system` 等）を
   起こすほどの context 固有ドメインロジックは無く、`event_logs` への記録は
   「DB pool・row scanner・transaction helper のような技術的共通部品」
   （ADR-090 が `backend/shared/adapters/persistence/` に許した範囲）そのものである。
   現行の outbox 経路もこのパターンを踏襲しており（`backend/shared/adapters/eventsink/`
   に `console.go` / `console_sink.go` / `kafka_relay.go` が同居）、`backend/shared/`
   直下にドメイン非依存の小 package を置く前例（`logging` / `resilience` / `version`）
   もある。したがって新しい Context Map 行や新規 bounded context を起こさず、
   既存の `backend/shared` 構造を拡張する。SCL 側も `EventLogRecord` /
   `EventDeliveryRecord` を `System` context（`spec/contexts/system.yaml`）に残し、
   `DatabaseResilience` / `StructuredApplicationLog` などの既存の「特定 context に
   属さない横断的技術方針」と同格に扱う。

## 却下した代替案

- **outbox を維持したまま配送状態カラムだけ分離せず運用する**: 監査原本の不変性を
  schema レベルで保証できず、配送処理のバグで本文が書き換わるリスクを排除できない。
- **Kafka を含めて 2PC / 分散 transaction にする**: 実装・運用コストが高く、Kafka
  broker 障害時に業務 mutation 自体をブロックする。IdP の可用性目標
  （`DatabaseResilience` / `ValkeyResilience`）と相反するため不採用。
- **HTTP request 全体を 1 transaction にする（long-lived connection middleware）**:
  外部 I/O 待ち（SMTP・Kafka・SaaS）の間 DB connection を保持し続け、接続枯渇を招く。
  command 単位の短い transaction に限定する方針を優先する。
- **全 `DomainEvent` を無条件で `event_logs` に記録する**: 高頻度 telemetry まで
  永続化すると `event_logs` が肥大化し監査原本としての可監査性を損なう。3 分類による
  棚卸しで対象を固定する。
- **`event_logs` / `event_deliveries` を `backend/audit`（または改名した
  `backend/event`）に同居させる**: Audit の「読み取り専用 read model」という境界と
  「全 context が書き込む transaction 中核」という責務が同居し、[[wi-146]] で分離した
  境界を逆流させる。書込み側の依存方向も `context_map` の現状（Audit は Tenancy にのみ
  依存）と整合しない。
- **新規 bounded context `backend/system` を起こす**: `event_logs` の記録は
  context 固有のドメインロジックではなく、DB pool や transaction helper と同種の
  技術的共通部品であり、専用 context package を正当化するだけの語彙・状態機械・
  認可ルールを持たない。

## 影響

- 新規 SCL 要素（`spec/contexts/system.yaml`）: glossary `EventLog` / `EventDelivery` /
  `DomainEventClassification`、models `EventLogRecord` / `EventDeliveryRecord` /
  `DomainEventClassification` / `EventDeliveryStatus`、invariants
  `EventLogAtomicWithBusinessState` / `EventDeliveryRetainsFailedEventLog` /
  `EventDeliveryEventuallyDelivered`、objectives `RelayAtLeastOnceDelivery` /
  `EventLogRetention`、scenarios（正常系・rollback・relay 再試行・重複許容の 4 件）。
- 新規 SCL 要素（`spec/contexts/audit.yaml`）: glossary `AuditEventProjection`、
  invariant `AuditProjectionIsRebuildableFromEventLog`。
- 今後の実装（[[wi-184]] T002 以降）: PostgreSQL に `event_logs` / `event_deliveries`
  テーブルを追加する。`outbox` はコード移行が終わるまで併存させ、移行後に
  `DROP TABLE`（データ移行 script 不要）。command transaction runner を新設し、
  `backend/identitymanagement`/`backend/authentication` の admin・account security
  mutation から順に移行する。`backend/shared`（`eventlog` package、
  `adapters/persistence/{postgres,memory}/eventlog`）に Go 実装を置き、
  `backend/relay` を `event_logs`/`event_deliveries` に対応させる。
- `ARCHITECTURE.md` の Context Map・Persistence 節は変更しない。`backend/shared` の
  技術的共通部品を 1 つ増やすだけで、新しい bounded context・Module は追加しないため
  （ADR-090 の既存範囲内）。
