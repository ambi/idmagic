---
depends_on: [wi-153-identity-lifecycle-workflows, wi-217-lifecycle-workflow-durable-run-handoff]
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-16
---

# lifecycle workflow の管理 API と dry-run を提供する

## Motivation

管理者が definition を安全に編集し、実行前に影響を確認し、run の失敗を診断・再実行できなければ workflow は運用できない。

## Scope

- CRUD、enable/disable/archive、dry-run、run list/detail/retry の admin API を追加する。
- CSRF、admin authentication、tenant-scoped authorization、revision precondition を強制する。

## Out of Scope

- React UI。

## Plan

- domain/usecase を HTTP request/response から分離する。
- cross-tenant ID は not-found に正規化する。

## Tasks

- [x] T001 [Use Case] dry-run/history/retry query を実装する。
- [x] T002 [HTTP] admin routes、validation、error mapping を実装する。
- [x] T003 [Verify] authorization、lost update、tenant isolation を検証する。

## Verification

- `just verify-go`

## Risk Notes

dry-run が run や action の副作用を作らないことを HTTP integration test で固定する。

## Completion

- **Completed At**: 2026-07-16
- **Summary**: Lifecycle workflow の管理 API、run 履歴・retry、および sqlc 永続化を実装した。
- **Affected Guarantees State**: tenant isolation、revision precondition、queued run cancel、failed step のみの retry を維持する。
- **Verification Results**:
  - `just sqlc-generate` — passed
  - `just verify` — passed
- **Evidence**: ローカル開発環境で Codex が SCL 同期・sqlc 生成・全体検証を実行した。
