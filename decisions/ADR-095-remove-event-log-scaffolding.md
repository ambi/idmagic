---
status: accepted
authors: [tn]
created_at: 2026-07-12
supersedes: [ADR-094]
---

# ADR-095: event_logs/event_deliveries scaffolding を撤去し outbox + best-effort audit_events に戻す

## コンテキスト

[[wi-184]] は業務状態と不変 event log を同一 transaction で確定する基盤として
`event_logs` / `event_deliveries` テーブル、`backend/shared/eventlog` パッケージ、
command 単位の transaction runner (`WithTx` / `TxFromContext` / `Runner`) を導入した。
その後 [[ADR-094]] は、業務可用性を監査記録より優先するため audit_only DomainEvent を
best-effort 記録へ切り替え、汎用 command envelope と transaction-bound emitter 注入を
撤去し、`event_logs` / `event_deliveries` schema は「将来の reconciliation と連携用途」の
ために保持すると決めた。

結果として現在の実装は次の中途半端な状態にある。

- `event_deliveries` に行を書くのは identitymanagement / authentication の
  `NewBridgingEmit` 経由の `PasswordChanged` のみで、他の public_integration event
  (OAuth2 の client / token / consent など 40 種) は今も `outbox` にしか入らない。
  Kafka 配送の実正本は依然 `outbox` である。
- `event_logs` に行を書くのも上記 2 context の bridge だけで、他 context の
  audit_only event は `audit_events` にしか入らない。監査検索の実正本も依然
  `audit_events` である。
- それにもかかわらず SCL は「relay は `event_deliveries` を読む」「`event_logs` が
  監査原本」「`audit_events` はその projection」と規定し、ADR-094 本文の
  「outbox 維持・event_logs は将来用」と矛盾している。

`event_logs` / `event_deliveries` は現に誰も読まず、producer も 1 event 型に留まる
休眠テーブルであり、それを前提とした SCL 記述・transaction runner・分類機構が
実態のない複雑性として残っている。将来 reconciliation や外部連携が実際に必要に
なった時点の設計は、その要件が具体化してから決めるべきで、使われない足場を先取りで
抱える理由はない。

## 決定

`event_logs` / `event_deliveries` に関わる scaffolding を撤去し、監査は `audit_events`、
Kafka 配送は `outbox` を正本とする ADR-094 以前の構成へ戻す。best-effort 記録と失敗の
観測可能性という ADR-094 の中核方針は維持する。

具体的には次を撤去する。

- スキーマ: `event_logs` / `event_deliveries` テーブル。
- Go: `backend/shared/eventlog`、その postgres / memory adapter、
  command 単位の transaction runner (`backend/shared/txrunner`、
  `backend/shared/adapters/persistence/postgres` の `Runner` / `WithTx` / `TxFromContext`、
  memory 版 txrunner)。runner は event_logs との同一 transaction 追記のためだけに
  導入され、event_logs 撤去後は単件 mutation を包むだけで目的を失うため併せて撤去する。
- 配線: HTTP handler の `TxRunner.Run` ラップと `NewBridgingEmit` を、他 context と同じ
  fire-and-forget emit に戻す。repository の tx 参加ヘルパー (`db(ctx)` / `TxFromContext`
  分岐) を直接プール利用に戻す。

将来 event log / reconciliation / 外部連携の原子性が具体的要件として現れた場合は、
その時点で新しい ADR と work item により設計する。

本 ADR は [[ADR-094]] の「`event_logs` / `event_deliveries` schema を将来用途のため保持する」
条項と、[[wi-184]] の transactional event log foundation を supersede する。ADR-094 の
best-effort 記録・失敗観測・reconciliation という方針自体は audit_events を対象として
存続する。

## 却下した代替案

- **event_logs を正本にする (wi-184 の完成)**: 全 context の public_integration event を
  `event_logs` / `event_deliveries` に同一 transaction で書く producer を実装し、relay と
  監査 projection をそれに向ける。実需のない段階で多 context 横断の大規模改修と二重書込み
  移行期間を負う。将来要件が具体化してから設計する方が確実。
- **休眠のまま schema と scaffolding を残す**: 使われないテーブル・transaction runner・
  分類機構が実態と乖離したまま残り、SCL とコードの矛盾・保守負荷が続く。
- **relay だけ event_deliveries へ移行する**: producer が 1 event 型しかないため、
  public_integration event の大半が Kafka に届かなくなる静かな配送障害を招く。

## 影響

- **SCL (`spec/contexts/system.yaml`)**: `models.EventLogRecord` /
  `models.EventDeliveryRecord` / `models.EventDeliveryStatus` /
  `models.DomainEventClassification`、glossary の `EventLog` / `EventDelivery` /
  `DomainEventClassification`、`invariants.EventLogFailureIsObservable` /
  `invariants.EventDeliveryRetainsFailedEventLog` /
  `invariants.EventDeliveryEventuallyDelivered`、`objectives.RelayAtLeastOnceDelivery` /
  `objectives.EventLogRetention` / `objectives.EventLogBestEffortRecovery`、
  対応する relay / best-effort scenario を撤去する。現に生きている保証は対象を
  移して残す: 監査記録の best-effort と失敗観測は `audit_events` を対象に、
  relay の at-least-once 配送は `outbox` を対象に記述する。
- **SCL (`spec/contexts/audit.yaml`)**: `models.AuditEventProjection` と
  `invariants.AuditProjectionIsRebuildableFromEventLog` を撤去し、`audit_events` を
  監査原本とする定義へ戻す。
- **データ / スキーマ**: `event_logs` / `event_deliveries` を削除する。両テーブルは
  実運用データの正本になっておらず移行は不要。`outbox` / `audit_events` は不変。
- **運用**: Kafka 配送は `outbox` → relay、監査検索は `audit_events` のまま変わらない。
- **work item**: 撤去作業を [[wi-191]] で行う。event_logs を原本前提とする [[wi-185]] は
  前提が失われるため cancelled とする。
