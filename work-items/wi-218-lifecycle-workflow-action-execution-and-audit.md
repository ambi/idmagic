---
depends_on: [wi-217-lifecycle-workflow-durable-run-handoff, wi-6-real-email-sender-adapter]
status: pending
authors: [tn]
risk: high
created_at: 2026-07-16
---

# lifecycle workflow action を冪等実行し監査可能にする

## Motivation

at-least-once job は副作用の重複を許容しない。desired-state action と step checkpoint を組み合わせ、失敗してもアクセス剥奪などの後続 action を前進させる必要がある。

## Scope

- group/application/required action/status/email の action port と executor を追加する。
- retry、per-user serialization、notification delivery dedup を実装する。
- workflow/run/step audit event、検索属性、retention cleanup、PII-safe logging を実装する。

## Out of Scope

- admin workflow CRUD UI。

## Plan

- resource と tenant を action 直前に再検証し、step outcome を checkpoint する。
- success/no-op 済み step は retry で実行しない。

## Tasks

- [ ] T001 [Actions] action port/executor と idempotency を実装する。
- [ ] T002 [Jobs] retry と per-user ordering を実装する。
- [ ] T003 [Audit] event、retention、PII-safe observability を実装する。
- [ ] T004 [Verify] partial failure と retry を検証する。

## Verification

- `just verify-go`

## Risk Notes

context 間の rollback は行わない。step ごとの durable checkpoint と desired-state 操作で最終状態を収束させる。
