---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-219-lifecycle-workflow-admin-api, wi-220-lifecycle-workflow-admin-ui-and-operations, wi-222-lifecycle-workflow-dry-run-real-evaluation]
---

# lifecycle workflow admin API/UI のテスト網羅性を確立する

## Motivation
`backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` (327 行、
CRUD/enable/disable/archive/dry-run/run 一覧/詳細/retry の 11 以上のハンドラ) には HTTP レベルの
テストが一切なく、authorization (`requireWorkflowAdmin` 相当)、tenant 境界、revision precondition、
cross-tenant ID の not-found 正規化、エラーマッピングが未検証である。domain 層のテストも
`domain/lifecycle_workflows_test.go` (4件) / `usecases/lifecycle_workflows_test.go` (4件) /
`usecases/lifecycle_workflow_dispatcher_test.go` (2件) の計 10 件に留まり、`eq`/`not_eq`/`in`/`exists`
の operator 別テストや、filter/action の 20 件上限の境界テストがない。

frontend は `AdminLifecycleWorkflowPages.test.tsx` (3件) / `WorkflowDefinitionForm.test.tsx` (5件) の
React Testing Library テストのみで、Playwright e2e テストが存在しない。wi-220 の完了報告は
`just test-ui-e2e` を検証コマンドとして挙げているが、`frontend/tests/e2e/` に lifecycle workflow を
対象にした spec ファイルはなく (唯一 lifecycle を名に含む `ui-scenario-actions.spec.ts` は無関係な
"application lifecycle" を指す)、この検証は実質的に lifecycle workflow の挙動を確認していない。

この網羅性不足が、[[wi-221-lifecycle-workflow-audit-event-emission]] の監査イベント欠落や
[[wi-222-lifecycle-workflow-dry-run-real-evaluation]] の dry-run スタブ化のような重大な齟齬を
検出できないまま "completed" として出荷させた直接の原因である。

## Scope
- `backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler_test.go` を新設し、
  CRUD/enable/disable/archive/dry-run/run 一覧/詳細/retry の各ハンドラを authorization・tenant
  境界・revision precondition・not-found 正規化の観点でテストする。
- domain 層に filter operator (`eq`/`not_eq`/`in`/`exists`) ごとのテストと、filter/action 20 件
  上限の境界テストを追加する。
- `frontend/tests/e2e/` に、lifecycle workflow の作成 → enable → run 履歴確認 → retry の一連を
  検証する Playwright e2e テストを追加する。
- `spec/contexts/identity-management.yaml` の scenarios (`emitted.exists(...)` 等) と Go 実装の
  対応を確認する最小限の統合テストを追加する ([[wi-221-lifecycle-workflow-audit-event-emission]]
  の成果と連携する)。

## Out of Scope
- 新機能の追加。本 WI はテスト網羅性のみを対象とする。
- 汎用的な SCL-Go conformance framework の新設や他 context への横展開。

## Plan
- 既存の他 admin 機能 (admin-groups / admin-users 等) の handler test パターンに合わせて書く。
- e2e テストは dry-run の実態評価 ([[wi-222-lifecycle-workflow-dry-run-real-evaluation]]) が
  先に修正されている前提で書き、誤った挙動をテストとして固定しない。

## Tasks
- [ ] T001 [Go] `admin_lifecycle_workflow_handler_test.go` を追加する。
- [ ] T002 [Go] domain validator の operator/上限境界テストを追加する。
- [ ] T003 [UI] Playwright e2e シナリオを追加する。
- [ ] T004 [Verify] `just verify-go` / `just test-ui-e2e` / `just verify` を通す。

## Verification
- `just verify-go`
- `just test-ui-e2e`
- `just verify`

## Risk Notes
テスト追加のみで本番挙動は変えない想定だが、追加したテストが本 WI の対象外の未発見バグを検出した
場合は、その場で直さずスコープを切り分けて別 WI として起票する。
