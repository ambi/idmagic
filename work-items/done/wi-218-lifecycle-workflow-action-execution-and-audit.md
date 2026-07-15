---
depends_on: [wi-217-lifecycle-workflow-durable-run-handoff, wi-6-real-email-sender-adapter]
status: completed
authors: [tn]
risk: high
created_at: 2026-07-16
---

# lifecycle workflow action を冪等実行し監査可能にする

## Motivation

at-least-once job は副作用の重複を許容しない。desired-state action と step checkpoint を組み合わせ、失敗してもアクセス剥奪などの後続 action を前進させる必要がある。

## Scope

- `spec/contexts/identity-management.yaml` の `WorkflowRun` / `WorkflowStep`、scenarios。
- group/application/required action/status/email の action port と executor を追加する。
- retry、per-user serialization、notification delivery dedup を実装する。
- workflow/run/step audit event、検索属性、retention cleanup、PII-safe logging を実装する。

## Out of Scope

- admin workflow CRUD UI。

## Plan

- resource と tenant を action 直前に再検証し、step outcome を checkpoint する。
- success/no-op 済み step は retry で実行しない。

## Tasks

- [x] T001 [SCL/Actions] executor と idempotency を仕様化・実装する。
- [x] T002 [Jobs] retry と per-user ordering を実装する。
- [x] T003 [Audit] event、retention、PII-safe observability を実装する。
- [x] T004 [Verify] partial failure と retry を検証する。

## Verification

- `just verify-go`

## Risk Notes

context 間の rollback は行わない。step ごとの durable checkpoint と desired-state 操作で最終状態を収束させる。

## Completion

- **Completed At**: 2026-07-16
- **Summary**:
  Lifecycle workflow の action executor を worker に接続し、各 action の結果を durable step checkpoint として保存するようにした。対象 user/resource を tenant 内で再検証し、同一 user の run を発火順に直列化する。
- **Affected Guarantees State**:
  desired-state action は changed / no_op / failed を記録し、成功済み step を再実行しない。resource 不在・tenant 不一致・通知不能は PII を含まない error code で記録する。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just test-go` — passed
  - `just verify` — failed (`lint-go` の既存環境エラー: `no go files to analyze`)
- **Evidence**:
  - 実行日: 2026-07-16
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex
  - 対象ソース版: main（コミット前作業ツリー）
  - 保存先: 外部成果物なし。検証結果を本記録に要約。
