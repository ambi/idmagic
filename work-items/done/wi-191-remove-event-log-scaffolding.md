---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: [wi-184-transactional-event-log-foundation, wi-190-transactional-event-log-rollout]
---

# 未使用の event_logs/event_deliveries scaffolding を撤去し outbox + best-effort audit_events に戻す

## Motivation

[[wi-184-transactional-event-log-foundation]] で導入した `event_logs` / `event_deliveries`、
`backend/shared/eventlog` パッケージ、command 単位の transaction runner は、
[[wi-190-transactional-event-log-rollout]] / [[ADR-094]] の best-effort への pivot によって
実質的に producer を失い、休眠状態になっていた。`event_deliveries` に書くのは
identitymanagement / authentication の bridge 経由の `PasswordChanged` のみ、Kafka 配送の
実正本は今も `outbox`、監査検索の実正本は今も `audit_events` である。それにもかかわらず
SCL は「relay は `event_deliveries` を読む」「`event_logs` が監査原本」と規定し、ADR-094 の
「outbox 維持・event_logs は将来用」と矛盾していた。

使われないテーブル・transaction runner・分類機構を実態のない複雑性として抱え続ける理由は
なく、将来 reconciliation / 外部連携が具体的要件になった時点で設計し直す方が確実である。
[[ADR-095]] の決定に基づき、この scaffolding を撤去して ADR-094 以前の
outbox + best-effort audit_events 構成に戻す。

## Scope

- **decision**: [[ADR-095]]。ADR-094 の event_log 保持条項と wi-184 foundation を supersede。
- **scl**:
  - `spec/contexts/system.yaml`: `EventLog` / `EventDelivery` / `DomainEventClassification` の
    glossary・models、`EventLogFailureIsObservable` / `EventDeliveryRetainsFailedEventLog` /
    `EventDeliveryEventuallyDelivered` invariants、`RelayAtLeastOnceDelivery` /
    `EventLogRetention` / `EventLogBestEffortRecovery` objectives、relay / best-effort
    scenarios を撤去する (wi-184 以前は relay / best-effort の SCL 記述自体が無かったため
    retarget ではなく撤去でよい)。
  - `spec/contexts/audit.yaml`: `AuditEventProjection` glossary と
    `AuditProjectionIsRebuildableFromEventLog` invariant を撤去し、`audit_events` を
    監査原本とする定義に戻す。
  - 派生成果物 (html / json schema / openapi) を再生成する。
- **schema**: `deploy/schema/postgres.sql` から `event_logs` / `event_deliveries` テーブルと
  index を削除し、識別子ポリシーのヘッダコメントから両テーブルの記述を除く。
- **go (削除)**: `backend/shared/eventlog`、`backend/shared/adapters/persistence/postgres/eventlog`、
  `backend/shared/adapters/persistence/memory/eventlog`、`backend/shared/txrunner`、
  `backend/shared/adapters/persistence/memory/txrunner`、
  `backend/shared/adapters/persistence/postgres/runner.go` / `txcontext.go`、
  transaction runner の結合テスト。`sqlc.yaml` の eventlog 生成ブロック。
- **go (revert)**: `admin_user_handler.go` / `account_authflow_handler.go` の
  `TxRunner.Run` + `NewBridgingEmit` を素の fire-and-forget emit (`legacyEmit` / `adminUserDeps`)
  に戻す。各 context routes.go / `bootstrap` の `TxRunner` / `EventLogRecorder` 配線を撤去する。
  `users.go` / `password_history.go` の tx 参加ヘルパー (`TxFromContext` 分岐) を直接プール
  利用に戻す。

## Out of Scope

- `outbox` / `audit_events` の挙動変更。Kafka 配送は outbox → relay、監査検索は audit_events の
  まま維持する。
- 将来の event log / reconciliation / 外部連携原子性の再設計 (要件が具体化した時点で別 ADR / WI)。
- `audit_events` の read model 化・長期保持運用 ([[wi-185]] は前提喪失により cancelled)。

## Plan

1. [[ADR-095]] を記録し、ADR-094 に supersede 注記を付す。
2. SCL ソース (system.yaml / audit.yaml) から event log 要素を撤去し、派生を再生成する
   (SCL-first)。
3. スキーマ・Go パッケージ・配線・sqlc 設定を撤去し、handler / repository を revert する。
4. `just build-go` / `just test-go` / `just verify-go` / `just yaml-check` を green にする。

## Tasks

- [x] T001 [Decision] [[ADR-095]] を作成し、[[ADR-094]] に supersede 注記を追記する。
- [x] T002 [SCL] system.yaml / audit.yaml から event log 要素を撤去し、`just yaml-check-scl` /
  `just scl-render` を通す。
- [x] T003 [Schema/Go] スキーマ・Go パッケージ・transaction runner・sqlc 設定を撤去し、
  handler / repository を fire-and-forget + 直接プール利用へ revert する。
- [x] T004 [Verify] build / test / verify / yaml-check を green にし、event log 参照の残存が
  無いことを確認する。

## Verification

- `just build-go`
- `just test-go`
- `just verify-go`
- `just yaml-check`
- `just scl-render`
- `grep` で `event_logs` / `event_deliveries` / `shared/eventlog` / `txrunner` の実装参照が
  残っていないこと (done work-item の履歴記述と本 WI / ADR は除く)。

## Risk Notes

medium。撤去対象が HTTP handler・DI・複数 context・SCL・スキーマに跨る。撤去漏れや
revert 誤りがあると、監査イベント (audit_events) や Kafka 配送 (outbox) の既存経路を壊す
恐れがある。build / race test / SCL 整合検査で回帰を固定し、outbox / audit_events の
挙動は一切変更しないことを保証する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  [[ADR-095]] に基づき、休眠していた `event_logs` / `event_deliveries` scaffolding を全撤去し、
  監査は `audit_events`、Kafka 配送は `outbox` を正本とする ADR-094 以前の構成へ戻した。
  SCL (system.yaml / audit.yaml) から EventLog / EventDelivery / DomainEventClassification と
  relay / best-effort / projection の記述を撤去し派生を再生成、`deploy/schema/postgres.sql`
  から両テーブルを削除、`backend/shared/eventlog` とその postgres / memory adapter、
  command transaction runner (`backend/shared/txrunner` ほか) と `sqlc.yaml` の生成ブロックを
  削除した。identitymanagement / authentication の admin user 単件 mutation と
  パスワード変更を `TxRunner.Run` + `NewBridgingEmit` から素の fire-and-forget emit
  (`legacyEmit` / `adminUserDeps`) に戻し、`users.go` / `password_history.go` の tx 参加
  ヘルパーを直接プール利用へ revert した。
- **Affected Guarantees State**:
  admin user 作成・更新・無効化/有効化とパスワード変更は、他 context と同じく audit_only
  event を best-effort (fire-and-forget) で記録する構成に統一した (ADR-094 の
  best-effort 方針を全 context で一貫させた)。業務更新と監査記録の原子性は
  intentionally not guaranteed。`outbox` / `audit_events` の挙動・API contract は不変。
- **Verification Results**:
  - `just build-go` — passed
  - `just test-go` — passed
  - `just verify-go` — passed
  - `just yaml-check` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Claude (Sonnet 5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
