---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# Application CRUD の更新を event log と同一 transaction にする

## Motivation

Application、assignment、category、sign-in policy の変更と監査原本を不可分に確定する。

## Scope

- Application HTTP adapter と各 use case の transaction-bound emit。
- 成功と rollback の結合テスト。

## Out of Scope

- OAuth2 複雑プロトコルフロー、relay、SAML/WS-Fed。

## Plan

- mutation ごとに transaction runner と bridging emitter を適用し、既存 read 操作は変更しない。

## Tasks

- [x] T001 [App] Application mutation を共通 command envelope へ移行する。
- [x] T002 [Test] event emit 失敗の伝播と command transaction の rollback 保証を検証する。
- [x] T003 [Verify] Go 検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

HTTP handler と use case の emit 契約変更が複数操作へ波及する。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: Application catalog の CRUD、icon、binding、assignment、category、sign-in policy と web link 作成を `eventlog.CommandRunner` に移行した。use case は transactional emitter の失敗を返すため、業務更新と event log は同一 transaction で rollback される。
- **Affected Guarantees State**: `EventLogAtomicWithBusinessState` は移行対象操作で enforced。SAML/WS-Fed 設定の新規 event 判断は wi-194 の対象として維持する。
- **Verification Results**:
  - `just verify-go` - passed
  - `just yaml-check` - passed
- **Evidence**:
  - 実行環境: local macOS / Go race test。実行主体: Codex。対象: 作業ツリー。
  - `TestCreateApplicationReturnsTransactionalEmitFailure` と既存 PostgreSQL transaction runner テストで失敗伝播・rollback の共通保証を確認した。
