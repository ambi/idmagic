---
depends_on: []
status: completed
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
    **本 WI では admin user の作成・更新・無効化/有効化と account security のパスワード変更を
    移行して機構を実証するところまでを完了の区切りとする。** 残りの mutation（OAuth2 の
    client/consent/token/PAR/device flow、application、SAML/WS-Fed、SCIM）と relay 実装は
    [[wi-190]] に引き継ぐ（下記 Out of Scope）。
- **architecture**:
  - `ARCHITECTURE.md` に event log / delivery state / relay / audit projection の依存方向と、
    外部 I/O を transaction 外に置く規則を同期する。

## Out of Scope

- Kafka の exactly-once 配送、Kafka transaction、分散 transaction / 2PC。
- 外部 SaaS への同期プロビジョニング。outbound provisioning は [[wi-45]] の job / retry 境界で扱う。
- CSV import の全件 atomic rollback。[[wi-96]] は import job とバッチ transaction で扱う。
- 長期アーカイブ、監査検索 read model の非同期化、検索 UI の変更（[[wi-185]]）。
- 高頻度 telemetry を全て event log に保存すること。
- **admin user 作成・更新・無効化/有効化とパスワード変更以外の mutation（OAuth2 の
  client/consent/token/PAR/device flow、application、SAML/WS-Fed、SCIM）の
  transaction runner への移行、relay の実装。** T004 で調査した結果、OAuth2 の
  `emit()` ヘルパーが単純 CRUD と複雑プロトコルフローの signature を共有していること、
  `refresh_tokens.go` が既に自前で transaction を持ち二重化のリスクがあること、
  SAML/WS-Fed がそもそも DomainEvent を emit していないことが分かり、横展開は
  単純な繰り返し作業ではなく個別の設計判断を要すると判断した。機構
  （schema・transaction runner・DomainEvent 分類の CI ガード）と最初の移行例が
  揃った時点でこの WI は完了とし、残りの展開は [[wi-190]] に引き継ぐ。

## Plan

1. SCL と ADR で、event log を正本、`event_deliveries` を可変な配送管理、監査検索を再構築可能な
   projection と定義する。全イベントを event log に流すのではなく、分類表を正本にする。
2. `wi-146` が audit の所有境界を確立した後、その port を介して event log / read model を接続する。
   `wi-146` 未完了中は既存の audit API 契約を変えない。
3. PostgreSQL transaction runner を command 単位で適用する。接続を HTTP request 全体で保持する
   middleware は採らず、SMTP / HTTP / Kafka を transaction 内に置かない。
4. 最初に admin user の単件 mutation と account security のパスワード変更を移行し、機構を実証する。
   各 command は業務更新、event log、必要な監査原本を一つの commit にする。group / agent / tenant /
   OAuth client / protocol / application mutation と relay（`event_id` を冪等キーに Kafka を
   at-least-once publish し、Kafka 配送障害は event log を消さず delivery state に残す）は、
   T004 の調査結果を踏まえ [[wi-190]] に引き継ぐ。

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
- [x] T004 [App] 未分類・未経路の DomainEvent を CI で検出する。
  **本 WI のスコープは「CI 検出」までとし、OAuth2/application/SAML/WS-Fed/SCIM の
  mutation 移行は [[wi-190]] へ引き継ぐ（Out of Scope 参照）。**
  - 完了: `backend/shared/spec` および各 context の `domain/` に分散する `spec.DomainEvent`
    実装を `backend/` 全体から正規表現スキャンし列挙したところ 107 種類あることが判明
    （`backend/shared/eventlog/classify_coverage_test.go`）。既存 outbox `eventTopics`
    map（`backend/oauth2/adapters/persistence/postgres/outbox.go`）に載っている 40 種を
    `public_integration`、残り 67 種（T001/T003 分含む）を `audit_only` として
    `spec/contexts/system.yaml` の分類方針どおり現状の実際の配送経路を保ったまま
    `classify.go` の `classification` map を網羅化した。
    `TestAllDomainEventTypesAreClassified`/`TestClassificationMapHasNoStaleEntries` が
    `just test-go`/`verify-go` の一部として毎回実行され、新規 DomainEvent 追加時に
    分類漏れがあれば CI で検出される（`ToRecord` は未分類 type を実行時エラーで拒否する
    設計と対になる静的側のガード）。
    分類は「今のKafka到達可否を保つ」だけで、`public_integration`への新規昇格や
    `telemetry`区分の切り出しは行っていない（将来の判断に委ねる）。
  - 未着手: OAuth2/application/SAML/WS-Fed/SCIM の実際の mutation を transaction runner へ
    配線すること。調査の結果、想定より結合度が高いことが分かった:
    `backend/oauth2/usecases/events.go` の共有 `emit()` ヘルパーが admin_clients.go /
    admin_consents.go のような単純 CRUD だけでなく exchange_code.go / exchange_token.go /
    refresh_tokens.go / revoke_token.go / device_flow.go / push_authorization_request.go /
    rotate_signing_key.go / register_client.go という複雑なプロトコルフローとも
    signature を共有しており、さらに `ConsentDeps`
    は identitymanagement/authentication の HTTP handler からも
    `d.ConsentDeps()` 経由で cross-context に再利用されている。
    `refresh_tokens.go` の `RefreshTokenStore.Rotate` は既に自前で `Pool.Begin`/`tx.Commit`
    を行っており、外側の `TxRunner.Run` にそのまま組み込むと二重 transaction になる
    危険がある — 個別の設計判断が必要。SAML/WS-Fed の admin SP/RP CRUD は
    現状そもそも DomainEvent を emit していない（HTTP handler が repository を直接呼ぶ）ため、
    「移行」ではなく「新規に emit を追加するか」という別の意思決定を要する。SCIM は
    `Emit` フィールドはあるが実際に呼んでいる箇所を要再確認。
    次段では、まず oauth2 の単純 admin CRUD（admin_clients.go/admin_consents.go）だけを
    T003 と同じパターンで切り出す（`emit()` 型変更の波及を認めた上で、複雑プロトコル
    フロー側は legacyEmit 相当のラッパーに留める）ところから始めるのが妥当。
- ~~T005 [Relay] relay を event log / delivery state に対応させる。~~ → [[wi-190]] T006 へ引き継ぎ。
- ~~T006 [Arch/Verify] 残りの回帰・障害注入・relay 再実行検証。~~ → [[wi-190]] T007 へ引き継ぎ。

## Verification

- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just test-go`
- `just verify-go`
- `just build-go`
- PostgreSQL 結合テスト: 業務更新、event log 追記のいずれかを失敗させた場合に全て rollback
  されることを確認する（`transaction_runner_test.go`）。
- relay 結合テストは relay 実装が [[wi-190]] に引き継がれたため本 WI の対象外。

## Risk Notes

high。既存の event emit は HTTP handler、use case、複数 context に分散し、transaction 境界を
誤ると mutation 成功後にイベントを失うか、外部 I/O 待ちで DB connection を長時間保持する。
command 単位の transaction に限定し、各移行で rollback / commit / duplicate delivery を
PostgreSQL 結合テストで固定する。既存 `outbox` の廃止は event log への二重書込み・照合期間を
設け、データ移行と relay 切替を一度に不可逆化しない。

T004 着手時点で、OAuth2/application/SAML/WS-Fed/SCIM への横展開が当初見積もりより
高リスクであることが判明した（`emit()` ヘルパーの cross-context 共有、
`refresh_tokens.go` の既存 transaction との二重化リスク、SAML/WS-Fed の emit 未実装）。
無理に一つの WI で背負わず機構実証の範囲でクローズし、残りを [[wi-190]] へ切り出す
判断とした。

## Completion

- **Completed At**: 2026-07-11
- **Summary**:
  業務状態と不変 event log を同一 PostgreSQL transaction で確定する機構を実装した。
  `deploy/schema/postgres.sql` に `event_logs`（不変・追記専用）と `event_deliveries`
  （Kafka 配送状態）を追加し、`backend/shared/adapters/persistence/postgres` に
  ctx 経由で `pgx.Tx` を伝播する `WithTx`/`TxFromContext`/`Runner` を実装した。
  この方式により port interface（`idmports.UserRepository` 等）は無変更のまま、
  対象 repository に `db(ctx)` ヘルパーを足すだけで transaction 参加を実現できた。
  `backend/shared/eventlog` に `DomainEvent` → `event_logs` record 変換（`ToRecord`）と
  `Recorder.Append`/`AppendDelivery` を呼ぶ `NewEmit` を実装し、
  `backend/` 全体に分散する 107 種類の `DomainEvent` 実装を正規表現スキャンで列挙して
  分類 map（`classify.go`）を網羅化した。`classify_coverage_test.go` が
  `just test-go`/`verify-go` の一部として毎回実行され、新規 DomainEvent 追加時に
  分類漏れがあれば CI で検出する。
  移行実証として、identitymanagement の admin user 作成・更新・無効化/有効化
  （`CreateUser`/`UpdateUser`/`SetUserDisabled`）と authentication のパスワード変更
  （`ChangePassword`）を、対応する HTTP handler で `TxRunner.Run` によりラップし、
  `deps.Emit` を transaction 参加する `NewEmit` へ差し替える形で移行した。PostgreSQL
  結合テスト（`backend/identitymanagement/adapters/persistence/postgres/transaction_runner_test.go`）
  で、業務更新 + event_logs 追記の commit、event_logs 追記失敗時の業務更新 rollback、
  後続処理が失敗した場合の rollback、`public_integration` event の `event_deliveries`
  同時挿入を検証した。
  OAuth2/application/SAML/WS-Fed/SCIM の残り mutation 移行と relay 実装は、
  横展開に個別の設計判断（`emit()` ヘルパーの cross-context 共有、
  `refresh_tokens.go` の既存 transaction との統合、SAML/WS-Fed への emit 新設要否）
  を要することが判明したため、[[wi-190]] に引き継いだ。
- **Affected Guarantees State**:
  admin user の作成・更新・無効化/有効化と account security のパスワード変更は、
  業務状態更新と event log 追記が同一 PostgreSQL transaction で確定するようになった
  （どちらかが失敗すれば両方 rollback）。それ以外の既存 mutation の挙動・API contract
  は変更していない（引き続き fire-and-forget の outbox/audit_events 経路）。
  `outbox`/`audit_events` テーブルは削除しておらず、既存コードが引き続き参照する。
- **Verification Results**:
  - `just yaml-check` — passed（SCL / work-items / ids / architecture）
  - `just build-go` — passed
  - `just test-go` — passed
  - `just verify-go` — passed（golangci-lint 0 issues、race-enabled Go tests）
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude (Sonnet 5)
  - 対象ソース版: main（コミット前）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
