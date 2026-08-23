---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
priority: p2
change_kind: feature
affected_spec:
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-001 }
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-002 }
depends_on: [wi-219-lifecycle-workflow-admin-api, wi-220-lifecycle-workflow-admin-ui-and-operations]
---

# lifecycle workflow の一覧・実行履歴のページネーションと運用 UI を改善する

## Motivation
`ListLifecycleWorkflows` はテナント内の全 workflow を無制限に返し、`ListLifecycleWorkflowRuns` は
`limit=100` を決め打ちで offset も cursor も持たない
(`backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` の `handleListLifecycleWorkflowRuns`)。
テナント規模や run 実行回数が増えると、古い run 履歴を閲覧する手段がない。
wi-153 は「list/history の pagination は既存 admin API の契約に合わせ、本 WI で独自の pagination 方式を
増やさない」と明記していたが、実際には他 admin 一覧とも揃っていない独自の固定 limit になっている。

**この乖離は起票時より大きくなっている。** [[wi-159-admin-resource-cursor-pagination]] と
[[wi-347-admin-pagination-totals-and-compact-cursors]] が完了し、`docs/api-rules.md` は管理用一覧 API に
「署名済みで版の付いたキーセット方式のカーソルを RFC 8288 の `Link` ヘッダーで返す」ことを規則として定めた。
lifecycle workflow の 2 本だけがこの規則の外に取り残されている。

frontend (`AdminLifecycleWorkflowsPage.tsx`) は一覧・run 履歴に検索/フィルタ/ソートがない。
run 詳細も一覧行へのインライン展開のみで、専用の run detail 画面や step ごとの timestamp 表示、
queued run の cancel 操作がない。
なお起票時に挙げていた `window.prompt` / `window.confirm` は既に取り除かれており、
テストが `prompt` を呼ばないことを固定している。この項目は解消済みとして扱う。

## Scope
- `spec/contexts/identity-governance/main.tsp` の `ListLifecycleWorkflows` / `ListLifecycleWorkflowRuns`
  interface に、`docs/api-rules.md` が定めるキーセットカーソルと `Link` ヘッダーの契約を適用する。
  独自の pagination 方式は増やさない。
- backend: 上記 pagination を usecase / handler / repository に実装する。
- frontend: 一覧・run 履歴に検索/フィルタ/ページ送りを追加する。
- run detail を専用画面または panel に切り出し、trigger snapshot、各 step の timestamp/outcome/
  error_code、job attempt 情報を表示する。
- queued 状態の run に対する cancel 操作を UI に追加する。

## Out of Scope
- workflow を図として可視化するダイアグラム UI ([[wi-226-lifecycle-workflow-templates-and-on-demand-run]]
  以降で検討)。
- 他 admin resource の cursor 化。[[wi-159-admin-resource-cursor-pagination]] で完了済みであり、
  本 work item は取り残された lifecycle workflow の 2 本を合流させるだけである。

## Plan
- pagination は既存のキーセットカーソル契約 (`docs/api-rules.md`、[[wi-347-admin-pagination-totals-and-compact-cursors]])
  をそのまま使い、新しい語彙を作らない。他の管理一覧と同じ形になることが成功条件である。
- 確認ダイアログは他機能で使っている導線に合わせ、本機能専用の作法を増やさない。

## Tasks
- [ ] T001 [Spec] 一覧系 interface に既存のカーソル pagination 契約を適用する。
- [ ] T002 [Go] usecase/handler/repository を pagination 対応にする。
- [ ] T003 [UI] 一覧・run 履歴の検索/フィルタ/ページ送り、run detail 画面、cancel 操作を実装する。
- [ ] T004 [Verify] `mise run verify-go` / `mise run verify-ui` / `mise run test-ui-e2e` を通す。

## Verification
- `mise run verify-go`
- `mise run verify-ui`
- `mise run test-ui-e2e`
- 手動: 100 件超の workflow および 100 件超の run を持つテナントで一覧・履歴のページ送りが正しく
  動くことを確認する。
- 手動: 他の管理一覧と同じページ送りの操作感で run 履歴をたどれることを確認する。

## Risk Notes
pagination 契約の変更は API の後方互換に影響するため、既存フロントエンドの呼び出し側を同時に
更新する。
