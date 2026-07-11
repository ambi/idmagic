---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-11
---

# 業務状態と不変 event log を同一 PostgreSQL transaction で確定する

## Motivation

現在の DomainEvent は HTTP 層の fire-and-forget `Emit` から、業務更新とは別の
PostgreSQL 操作で outbox / audit_events へ記録される。このため、業務状態だけが
commit されて event が失われる経路がある。また `outbox` はイベント本文と Kafka
配送状態を同じ可変行に持つため、長期の監査原本としての責務と配送キューとしての
責務が混在している。

IdMagic が確定したセキュリティ・管理上の事実を再生可能に保ちつつ、Kafka や外部
SaaS 呼出を API transaction に入れないため、業務状態と不変 event log だけを短い
PostgreSQL transaction で確定する。外部配送・検索投影は commit 後に別 worker が
処理する。

## Scope

- **decision**:
  - 新規 ADR で canonical event log、delivery state、監査 read model の責務を定義する。
    event log は監査原本兼配送源、Kafka は at-least-once、`event_id` は消費側の冪等キーとする。
  - DB transaction の対象を「同一 PostgreSQL 内の業務状態 + event log」に限定する。
    Kafka、SMTP、HTTP、CSV 全件、外部 SaaS の結果待ちは transaction に含めない。
  - DomainEvent を public integration event / audit-only event / telemetry の三分類に棚卸しし、
    public / audit event の routing と payload 必須属性を固定する。
- **scl**:
  - `spec/contexts/system.yaml` の glossary、models、invariants、scenarios、objectives に
    EventLog / EventDelivery、原子的記録、at-least-once 配送、重複冪等性、relay 障害時の
    再送を追加する。
  - `wi-146` 完了後の audit context に、event log を原本、検索用 audit record を projection とする
    所有関係を追加する。未完了の場合は System context の横断仕様だけを先行する。
  - 各 context の発行イベントについて、tenant、actor、subject、occurred_at、correlation ID、
    payload の PII 方針を SCL に反映する。
- **persistence / go**:
  - 不変の `event_logs` と、Kafka 配送の試行・状態を持つ `event_deliveries` を PostgreSQL に追加する。
    本文行は更新せず、delivery state だけを更新する。既存 outbox のデータ移行・後方互換方針を実装前に確定する。
  - application command ごとの transaction runner を導入し、transaction-bound repository と
    event recorder を使って業務更新と event log 追記を同一 commit にする。
  - HTTP 層の fire-and-forget emit を廃止し、失敗時に mutation を rollback する同期的な event
    記録へ移す。対象は通常の単件・軽量 mutation とし、CSV import はバッチ単位で適用する。
  - relay は commit 済み event log だけを取得して Kafka へ publish し、同じ `event_id` を header に
    付与して delivery state を更新する。Kafka 成功後の停止による重複は許容する。
- **architecture**:
  - `ARCHITECTURE.md` に event log / delivery state / relay / audit projection の依存方向と、
    外部 I/O を transaction 外に置く規則を同期する。

## Out of Scope

- Kafka の exactly-once 配送、Kafka transaction、分散 transaction / 2PC。
- 外部 SaaS への同期プロビジョニング。outbound provisioning は [[wi-45]] の job / retry 境界で扱う。
- CSV import の全件 atomic rollback。[[wi-96]] は import job とバッチ transaction で扱う。
- 長期アーカイブ、監査検索 read model の非同期化、検索 UI の変更（後続 WI）。
- 高頻度 telemetry を全て event log に保存すること。

## Plan

1. SCL と ADR で、event log を正本、`event_deliveries` を可変な配送管理、監査検索を再構築可能な
   projection と定義する。全イベントを event log に流すのではなく、分類表を正本にする。
2. `wi-146` が audit の所有境界を確立した後、その port を介して event log / read model を接続する。
   `wi-146` 未完了中は既存の audit API 契約を変えない。
3. PostgreSQL transaction runner を command 単位で適用する。接続を HTTP request 全体で保持する
   middleware は採らず、SMTP / HTTP / Kafka を transaction 内に置かない。
4. 最初に admin user / group / agent / tenant / OAuth client と account security の単件 mutation を
   移行し、次に protocol / application mutation を移行する。各 command は業務更新、event log、
   必要な監査原本を一つの commit にする。
5. relay は `event_id` を冪等キーに Kafka を at-least-once publish する。Kafka 配送障害は event log を
   消さず delivery state に残す。

## Tasks

- [x] T001 [Decision/SCL] ADR と SCL の event log / delivery / event classification / failure scenarios を
  追加し、`just yaml-check-scl` と `just scl-render` を通す。[[ADR-094]]。
  `spec/contexts/system.yaml`（glossary/models/invariants/objectives/scenarios）と
  `spec/contexts/audit.yaml`（AuditEventProjection、ownership 明記）を更新済み。
  T002 以降（schema/transaction runner/relay 実装）は未着手。
- [x] T002 [Schema] `event_logs` と `event_deliveries` の schema、制約、既存 outbox からの移行・切替を
  実装し、PostgreSQL adapter の契約テストを追加する。`deploy/schema/postgres.sql` に両テーブルを
  追加（`outbox` / `audit_events` は既存コードが参照するため変更せず、削除は T005 で
  `DROP TABLE` のみ・データ移行不要。未リリースのため互換維持は不要）。Go 実装は
  `backend/shared/eventlog`（domain 型・`Recorder` port）、
  `backend/shared/adapters/persistence/postgres/eventlog`（sqlc adapter + 契約テスト）、
  `backend/shared/adapters/persistence/memory/eventlog`（memory adapter）に配置
  （`backend/audit` にも新規 `backend/system` にも置かない。理由は [[ADR-094]] 決定 8 項）。
  `just yaml-check` / `just sqlc-generate` / `just test-go` / `just verify-go` / `just build-go`
  すべて green。DI 配線（`bootstrap/deps.go`）・`emit` クロージャの置換・transaction runner
  自体は T003 のまま未着手。
- [x] T003 [App] command transaction runner と transaction-bound event recorder を実装し、
  admin / account security の単件 mutation を同一 commit へ移行する。
  `backend/shared/adapters/persistence/postgres`（`WithTx`/`TxFromContext`/`Runner`）+
  `backend/shared/txrunner`（Runner interface）+
  `backend/shared/adapters/persistence/memory/txrunner`（memory 用 passthrough）で
  transaction runner を実装。ctx 経由で tx を伝播させる方式のため port interface
  （`idmports.UserRepository` 等）は無変更（`UserRepository`/`PasswordHistoryRepository`
  postgres adapter に `db(ctx)` ヘルパーを追加しただけ）。
  `backend/shared/eventlog`（`ToRecord`/`NewEmit`）で `spec.DomainEvent` →
  `event_logs` record 変換と `Recorder.Append`/`AppendDelivery` 呼び出しを実装。
  分類は `UserCreated`/`UserUpdated`/`UserDisabled`/`UserEnabled`/
  `UserRequiredActionCleared` を `audit_only`、既存 outbox `eventTopics` に
  含まれる `PasswordChanged` のみ `public_integration` として棚卸し
  （残りは T004 で拡張）。
  移行対象: `backend/identitymanagement/usecases/admin_users.go` の `CreateUser`/
  `UpdateUser`/`SetUserDisabled` と、対応する HTTP handler
  （`handleCreateAdminUser`/`handleUpdateAdminUser`/`handleSetAdminUserDisabled`）。
  account security は `backend/authentication/usecases/change_password.go` の
  `ChangePassword` と `handleChangePasswordAPI`。
  `AdminUserDeps.Emit`/`AdminGroupDeps.Emit`/`AdminAgentDeps.Emit`/
  `AccountProfileDeps.Emit`/`ChangePasswordDeps.Emit` はいずれも
  `func(spec.DomainEvent) error` に変更（`adminEmit` ヘルパーを共有する group/agent/
  account-profile も型を揃えたが、これらの mutation 自体は today も transaction
  runner でラップしていない — legacy fire-and-forget を `error` 型に適合させる
  `legacyEmit()` ラッパー経由のまま。DeleteUser/SoftDeleteUser/RestoreUser/
  RequiredAction/groups/agents/tenant/OAuth2 client mutation は未移行）。
  PostgreSQL 結合テスト
  （`backend/identitymanagement/adapters/persistence/postgres/transaction_runner_test.go`）
  で、業務更新 + event_logs 追記の commit、event_logs 追記失敗時の業務更新
  rollback、fn が最後にエラーを返した場合の rollback、
  public_integration event の `event_deliveries` 同時挿入を検証。
  `just yaml-check` / `just build-go` / `just test-go` / `just verify-go`
  すべて green。audit_events を同一 transaction に含めることは行っていない
  （引き続き legacy fire-and-forget 経路）。
- [ ] T004 [App] OAuth2、application、SAML、WS-Fed、SCIM の通常 mutation を同一方式へ移行し、
  未分類・未経路の DomainEvent を CI で検出する。
- [ ] T005 [Relay] relay を event log / delivery state に対応させ、at-least-once、stable event ID、
  failure recording をテストする。
- [ ] T006 [Arch/Verify] `ARCHITECTURE.md`、README の runtime 説明を同期し、回帰・障害注入・
  relay 再実行検証を完了する。

## Verification

- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just test-go`
- `just verify-go`
- `just build-go`
- PostgreSQL 結合テスト: 業務更新、event log 追記、監査原本のいずれかを失敗させた場合に全て rollback
  されることを確認する。
- relay 結合テスト: Kafka 障害時は delivery が未完了のまま残り、再実行で配送されること、publish 後の
  停止では同じ `event_id` の重複が起こり得ることを確認する。
- 手動: 単件管理更新、CSV import の一バッチ、外部 I/O を伴う操作について、transaction が短時間で
  完了し外部 I/O を待たないことを確認する。

## Risk Notes

high。既存の event emit は HTTP handler、use case、複数 context に分散し、transaction 境界を
誤ると mutation 成功後にイベントを失うか、外部 I/O 待ちで DB connection を長時間保持する。
command 単位の transaction に限定し、各移行で rollback / commit / duplicate delivery を
PostgreSQL 結合テストで固定する。既存 `outbox` の廃止は event log への二重書込み・照合期間を
設け、データ移行と relay 切替を一度に不可逆化しない。
