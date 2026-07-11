---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-11
depends_on: [wi-184-transactional-event-log-foundation]
---

# audit-only event log を best-effort 記録と reconciliation に切り替える

## Motivation

業務 mutation と event log を任意の HTTP handler で同一 transaction に囲う実装は、
transaction の範囲、依存、失敗処理を各 context に広げる。業務可用性を優先しつつ、
監査記録の失敗を観測・回復可能にする。

## Scope

- `spec/contexts/system.yaml` の `invariants.EventLogFailureIsObservable`、
  `objectives.EventLogBestEffortRecovery`、対応 scenarios。
- `decisions/ADR-094-transactional-event-log-and-audit-projection.md`。
- transaction-bound command envelope と emitter 注入の撤去。
- event sink / audit record 書込み失敗の構造化観測。
- 残る context の audit-only event を既存 best-effort 経路に統一する。

## Out of Scope

- reconciliation worker の実装、Kafka relay、exactly-once 配送。
- 外部連携に厳密な原子性を求める将来の repository-local outbox。

## Plan

- audit_only event は commit 後に best-effort で記録し、失敗を業務応答へ返さない。
- 原子性が必要な連携操作だけは、将来の専用 persistence port が短い transaction を所有する。

## Tasks

- [x] T001 [SCL/Decision] 仕様と ADR を best-effort + reconciliation に更新する。
- [x] T002 [App] transaction-bound command envelope を撤去し、失敗観測を実装する。
- [x] T003 [Verify] work item と検証結果を同期する。

## Verification

- `just yaml-check`
- `just verify-go`

## Risk Notes

audit_only event は一時的に欠落し得る。event type と tenant を含む失敗観測と将来の
reconciliation を必須にして、欠落を不可視にしない。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: HTTP adapter まで transaction を伝播する command envelope を撤去し、
  audit-only event は既存の best-effort 経路で記録する形に戻した。event sink、audit record
  変換、audit record append の各失敗を構造化ログに残すようにした。
- **Affected Guarantees State**: `EventLogFailureIsObservable` と
  `EventLogBestEffortRecovery` を enforced。業務操作と audit-only event の原子性は
  intentionally not guaranteed。
- **Verification Results**:
  - `just verify-go` - passed
  - `just yaml-check` - passed
- **Evidence**:
  - 実行環境: local macOS / Go race test。実行主体: Codex。対象: 作業ツリー。
  - HTTP 結合テストを含む Go race test と SCL / work-item / Architecture 整合検査を実行した。
