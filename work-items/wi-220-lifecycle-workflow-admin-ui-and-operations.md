---
depends_on: [wi-218-lifecycle-workflow-action-execution-and-audit, wi-219-lifecycle-workflow-admin-api]
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-16
---

# lifecycle workflow の管理 UI と運用手順を提供する

## Motivation

型付き action の編集、dry-run の解釈、step failure の復旧を管理画面と運用文書で一貫して提供する必要がある。

## Scope

- `frontend/src/features/admin-lifecycle-workflows/` に一覧、editor、dry-run、run history/detail を追加する。
- ja/en 文言、destructive action warning、README の運用・障害対応手順を追加する。

## Out of Scope

- 汎用 Job 管理画面。

## Plan

- API contract の型を再利用し、自由記述 script を提供しない。
- destructive action と disable/archive の影響を保存前に表示する。

## Tasks

- [ ] T001 [UI] typed editor と validation summary を実装する。
- [ ] T002 [UI] dry-run/run history/failure detail を実装する。
- [ ] T003 [Docs] delivery semantics と recovery runbook を同期する。
- [ ] T004 [Verify] UI/E2E と全体 verify を通す。

## Verification

- `just verify-ui`
- `just test-ui-e2e`
- `just verify`

## Risk Notes

画面に attribute value、email 本文、secret を表示しない。API response と表示 model の両方で最小化する。
