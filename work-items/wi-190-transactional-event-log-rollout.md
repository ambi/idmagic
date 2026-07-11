---
status: in_progress
authors: ["tn"]
risk: high
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# transaction-bound event log を OAuth2 / application / SAML / WS-Fed / SCIM / relay へ広げる

## Motivation

[[wi-184]] は業務状態と不変 event log を同一 PostgreSQL transaction で確定する基盤
（`event_logs` / `event_deliveries` テーブル、ctx 経由で `pgx.Tx` を伝播する
transaction runner、107 種類の DomainEvent を網羅した分類 map と CI ガード）を作り、
identitymanagement の admin user 作成・更新・無効化と authentication の
パスワード変更をこの方式へ移行した。

wi-184 の完了時点で、それ以外の mutation（OAuth2 の client/consent/token/PAR/device
flow、application の CRUD、SAML/WS-Fed の admin SP/RP CRUD、SCIM）は引き続き従来の
fire-and-forget 経路（`legacyEmit()` 相当のラッパー）に残っている。wi-184 の調査で、
これらへの展開は単純な横展開ではなく個別の設計判断を要することが分かったため、
wi-184 のスコープを機構整備 + 最初の移行例に絞ってクローズし、残りの展開を本 WI に
引き継ぐ。

## Scope

- **persistence / go**（段階的に、小さい WI 相当の単位に分割して進める）:
  子 WI（wi-191 から wi-197）で段階的に実施する:
  1. OAuth2 の単純 admin CRUD（`admin_clients.go` の
     `CreateAdminOAuth2Client`/`UpdateAdminOAuth2Client`/`DeleteAdminOAuth2Client`、
     `admin_consents.go` の consent revoke）を wi-184 T003 と同じ方式（ctx 伝播 +
     `TxRunner.Run` + `sharedeventlog.NewEmit`）へ移行する。
  2. application context の CRUD（`applications.go`/`assignments.go`/
     `categories.go`/`sign_in_policy.go`）を同様に移行する。
  3. OAuth2 の複雑プロトコルフロー（`exchange_code.go`/`exchange_token.go`/
     `refresh_tokens.go`/`device_flow.go`/`push_authorization_request.go`/
     `revoke_token.go`/`rotate_signing_key.go`/`register_client.go`）を対象に、
     `emit()` ヘルパー共有による型変更の波及と、`refresh_tokens.go` の
     `RefreshTokenStore.Rotate` が既に自前で行っている `Pool.Begin`/`tx.Commit`
     との二重 transaction 衝突を個別に設計してから移行する。
  4. SAML/WS-Fed の admin SP/RP CRUD（`saml/adapters/http/admin_service_provider_handler.go`、
     `wsfederation/adapters/http/admin_relying_party_handler.go`）は現状 DomainEvent を
     一切 emit していないため、まず「新規に emit を追加すべきか」を decision として
     確定してから、追加する場合は同方式で実装する。
  5. SCIM（`backend/scim/usecases/usecases.go` の `Emit` フィールド）の実際の呼び出し
     箇所を確認し、inbound provisioning という性質を踏まえて migration 方式を決める。
  6. relay を `event_deliveries` に対応させ、`event_id` を冪等キーに Kafka へ
     at-least-once で publish する（wi-184 が Plan していた旧 T005 相当）。
- **decision**: 上記 4（SAML/WS-Fed への emit 新設）と 3（refresh token rotation の
  transaction 設計）は非自明な設計判断のため ADR を残す。

## Out of Scope

- wi-184 が既に完了させた範囲（`event_logs`/`event_deliveries` schema、transaction
  runner 基盤、DomainEvent 分類 CI ガード、admin user/password change の移行）の再変更。
- Kafka の exactly-once 配送、Kafka transaction、分散 transaction / 2PC（[[wi-184]] と同様）。
- CSV import の全件 atomic rollback（[[wi-96]]）。
- 監査検索 read model の非同期化（[[wi-185]]）。

## Plan

1. まず OAuth2 の単純 admin CRUD だけを切り出し、`emit()` ヘルパーの型変更の波及を
   受け止めつつ、複雑プロトコルフロー側は legacyEmit 相当のラッパーに留める
   （wi-184 T003/T004 の調査で確認済みの方針）。
2. application context の CRUD を同様に移行する。ここまでで「単純 CRUD の横展開」が完了する。
3. `refresh_tokens.go` の transaction 設計（外側 `TxRunner` と内側の自前 `Begin`/`Commit`
   をどう統合するか）を ADR として決定してから、複雑プロトコルフローへ着手する。
4. SAML/WS-Fed への emit 新設要否を decision として確定する。
5. SCIM の調査結果に基づき migration 方式を決める。
6. 上記が一定進んだ段階で relay を `event_deliveries` に対応させる。

## Tasks

- [x] T001 [wi-191] OAuth2 の単純 admin CRUD（client/consent）を共通 command envelope へ移行する。
- [x] T002 [wi-192] application context の CRUD を共通 command envelope へ移行する。
- [ ] T003 [wi-193] [Decision/App] `refresh_tokens.go` の transaction 設計を ADR で決定し、
  OAuth2 の複雑プロトコルフローを移行する。
- [ ] T004 [wi-194] [Decision/App] SAML/WS-Fed の admin SP/RP CRUD への emit 新設要否を決定し、
  必要なら実装する。
- [ ] T005 [wi-195] SCIM の実際の emit 呼び出し箇所を確認し、migration 方式を実装する。
- [ ] T006 [wi-196] [Relay] relay を `event_deliveries` に対応させ、at-least-once、stable event ID、
  failure recording をテストする。
- [ ] T007 [wi-197] [Arch/Verify] `ARCHITECTURE.md`、README の runtime 説明を同期し、回帰・障害注入・
  relay 再実行検証を完了する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- `just build-go`
- PostgreSQL 結合テスト: 各移行対象について、業務更新・event log 追記のいずれかを
  失敗させた場合に全て rollback されることを確認する（[[wi-184]] の
  `transaction_runner_test.go` と同じ形式）。
- relay 結合テスト: Kafka 障害時は delivery が未完了のまま残り、再実行で配送されること、
  publish 後の停止では同じ `event_id` の重複が起こり得ることを確認する。

## Risk Notes

high。[[wi-184]] の調査で判明した通り、OAuth2 の `emit()` ヘルパーは単純 CRUD と
複雑プロトコルフローの両方の `Deps.Emit` シグネチャを束ねており、`ConsentDeps` は
identitymanagement/authentication からも cross-context に再利用されている。
`refresh_tokens.go` の `RefreshTokenStore.Rotate` は既に自前で `Begin`/`Commit` を
行っており、外側の `TxRunner.Run` に無検討で組み込むと二重 transaction になる。
SAML/WS-Fed は現状 emit 自体が存在しないため、「移行」ではなく「新規機能追加」の
判断が要る。段階を大きく飛ばさず、T001 から順に小さい単位で commit し、各段階で
`just verify-go` を green に保つ。
