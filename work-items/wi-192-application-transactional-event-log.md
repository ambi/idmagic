---
status: pending
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

- [ ] T001 [App] Application mutation を移行する。
- [ ] T002 [Test] rollback を検証する。
- [ ] T003 [Verify] Go 検証を通す。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

HTTP handler と use case の emit 契約変更が複数操作へ波及する。
