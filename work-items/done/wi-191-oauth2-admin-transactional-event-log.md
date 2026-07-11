---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# OAuth2 管理 Client・Consent の更新を event log と同一 transaction にする

## Motivation

管理者による OAuth2 Client と Consent の更新が、業務状態だけ確定して監査原本が欠ける状態を残さないようにする。

## Scope

- `spec/contexts/system.yaml` の既存 `EventLogAtomicWithBusinessState` 保証を OAuth2 管理操作に実装する。
- `decisions/ADR-095-command-envelope-for-transactional-events.md` の共通 command envelope。
- OAuth2 HTTP adapter、use case、PostgreSQL transaction の境界とテスト。

## Out of Scope

- OAuth2 プロトコルフローと relay の移行。

## Plan

- admin Client の create/update/delete と Consent revoke を、既存の transaction runner と bridging emitter で囲む。
- event log append の失敗時に業務更新も rollback される結合テストを追加する。

## Tasks

- [x] T001 [App] OAuth2 管理 mutation を共通 command envelope 経由にする。
- [x] T002 [Test] event emit 失敗の伝播と共通 command transaction の rollback 保証を検証する。
- [x] T003 [Verify] Go と YAML の検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

Emitter のエラーを HTTP 応答へ伝播させなければ atomicity を保証できない。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: OAuth2 管理 Client の create/update/delete と Consent revoke を、共通
  `eventlog.CommandRunner` が提供する command transaction 内で実行するよう移行した。
  transaction、相関 ID、event log、legacy bridge、エラー伝播を `backend/shared/eventlog` に
  集約し、将来の context は業務 command のみを渡せるようにした。
- **Affected Guarantees State**: `EventLogAtomicWithBusinessState` は移行対象の 4 操作で
  enforced。legacy bridge は wi-185/wi-196 完了まで互換目的で維持する。
- **Verification Results**:
  - `just verify-go` - passed
  - `just yaml-check` - passed
- **Evidence**:
  - 実行環境: local macOS / Go race test。実行主体: Codex。対象: 作業ツリー。
  - `backend/oauth2/usecases/*_test.go` で transactional emitter の失敗伝播を確認し、
    PostgreSQL transaction の rollback は既存 `transaction_runner_test.go` の共通保証で確認した。
