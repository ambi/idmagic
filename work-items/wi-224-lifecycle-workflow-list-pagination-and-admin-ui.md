---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-219-lifecycle-workflow-admin-api, wi-220-lifecycle-workflow-admin-ui-and-operations]
---

# lifecycle workflow の一覧・実行履歴のページネーションと運用 UI を改善する

## Motivation
`ListLifecycleWorkflows` はテナント内の全 workflow を無制限に返し、`ListLifecycleWorkflowRuns` は
`limit=100` を決め打ちで offset/cursor を持たない
(`admin_lifecycle_workflow_handler.go` L280)。テナント規模や run 実行回数が増えると、古い run 履歴を
閲覧する手段がない。wi-153 は「list/history の pagination は既存 admin API の契約に合わせ、本 WI で
独自の pagination 方式を増やさない」と明記していたが、実際には他 admin 一覧とも揃っていない独自の
固定 limit になっている。

frontend (`AdminLifecycleWorkflowsPage.tsx`) は一覧・run 履歴に検索/フィルタ/ソートがなく、対象
ユーザー ID の指定に `window.prompt` (L97)、削除確認に `window.confirm` (L120) を使っている。この
2 つはリポジトリ全体で本機能にしか存在しないパターンであり、他の admin 機能が使っている確認導線と
一致しない。run 詳細も一覧行へのインライン展開のみで、専用の run detail 画面や step ごとの
timestamp 表示、queued run の cancel 操作がない。

## Scope
- `spec/contexts/identity-management.yaml` の `ListLifecycleWorkflows` / `ListLifecycleWorkflowRuns`
  interface に pagination 契約 (`page_size` / cursor、または既存 admin 一覧に揃えた形) を追加する。
  [[wi-159-admin-resource-cursor-pagination]] が完了していればその cursor 契約に合わせ、未完了なら
  暫定契約とし、wi-159 完了時に移行可能な設計にする (`depends_on` には加えない。完了前提ではなく
  整合性を取るべき参照)。
- backend: 上記 pagination を usecase / handler / repository に実装する。
- frontend: 一覧・run 履歴に検索/フィルタ/ページ送りを追加する。
- `window.prompt` / `window.confirm` を、確認ダイアログ・対象選択フォームに置き換える。リポジトリに
  流用できる確認ダイアログコンポーネントがなければ新設する (他機能への横展開は本 WI の範囲外)。
- run detail を専用画面または panel に切り出し、trigger snapshot、各 step の timestamp/outcome/
  error_code、job attempt 情報を表示する。
- queued 状態の run に対する cancel 操作を UI に追加する。

## Out of Scope
- workflow を図として可視化するダイアグラム UI ([[wi-226-lifecycle-workflow-templates-and-on-demand-run]]
  以降で検討)。
- [[wi-159-admin-resource-cursor-pagination]] 本体のスコープ (他 admin resource 全体の cursor 化)。

## Plan
- pagination は [[wi-159-admin-resource-cursor-pagination]] の `PageRequest`/`PageResult` 語彙を
  先取りして設計し、wi-159 が後から完了しても契約の衝突が起きないようにする。
- 確認ダイアログは新設する場合、他機能へ強制的に展開せず、本機能の置き換えに閉じる。

## Tasks
- [ ] T001 [SCL] 一覧系 interface に pagination 契約を追加する。
- [ ] T002 [Go] usecase/handler/repository を pagination 対応にする。
- [ ] T003 [UI] 一覧・run 履歴の検索/フィルタ/ページ送り、確認ダイアログ、run detail 画面、cancel
  操作を実装する。
- [ ] T004 [Verify] `just verify-go` / `just verify-ui` / `just test-ui-e2e` を通す。

## Verification
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動: 100 件超の workflow および 100 件超の run を持つテナントで一覧・履歴のページ送りが正しく
  動くことを確認する。
- 手動: `window.prompt`/`window.confirm` を使わずに対象ユーザー指定と削除確認ができることを確認する。

## Risk Notes
pagination 契約の変更は API の後方互換に影響するため、既存フロントエンドの呼び出し側を同時に
更新する。
