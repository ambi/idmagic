---
status: completed
archived_from: 914c983349aa6698f26e2b093e7c666b8c7264c8
---

# 業務状態と不変 event log を同一 PostgreSQL transaction で確定する

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

全文は `git show 914c983349aa6698f26e2b093e7c666b8c7264c8:<このファイルのパス>` で読む。
