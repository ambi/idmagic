---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-07-16
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "既存の lifecycle workflow 管理面に対するテストの追加であり、契約も規範シナリオも変えない。" }
depends_on: [wi-219-lifecycle-workflow-admin-api, wi-220-lifecycle-workflow-admin-ui-and-operations, wi-222-lifecycle-workflow-dry-run-real-evaluation]
---

# lifecycle workflow 管理面に残る境界条件と E2E のテスト不足を解消する

## Motivation

本項目の作成後に `backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler_test.go` が追加され、管理 HTTP 経路の認可、テナント境界、revision、not-found などにはテストが入った。

domain と usecase にも lifecycle workflow のテストがあり、「HTTP レベルのテストが一切ない」という当初の前提は現在には当てはまらない。

一方、frontend の単体テストだけでは、画面、管理 API、永続化を結ぶ配線を検証できない。

`frontend/tests/e2e/` には lifecycle workflow の作成、enable、dry-run、実行履歴、retry を通す Playwright シナリオがまだなく、operator ごとの評価と filter/action 件数上限についても、規範上の境界を網羅しているか棚卸しが必要である。

本項目は既存テストを作り直さず、実際に残っている境界条件と E2E の不足だけを解消する。

## Scope

- lifecycle workflow の規範例を、handler、domain、usecase、frontend unit、E2E の既存テストへ対応付ける。
- `eq`、`not_eq`、`in`、`exists` の評価と、filter/action 件数上限の直前、上限、超過を確認し、欠けた domain または usecase テストだけを追加する。
- 管理 HTTP 経路について、規範例に対応しない認可、テナント境界、revision、not-found、dry-run、retry の不足だけを追加する。
- Playwright で、作成、enable、dry-run、実行履歴の確認、retry を製品と同じ UI と API から通す主要経路を追加する。
- テストで製品欠陥を検出した場合は、本項目で修正せず、該当する `REQ-*` を持つ bugfix work item に分ける。

## Out of Scope

- 既存テストの全面的な書き換え。
- lifecycle workflow の新機能または仕様変更。
- 汎用的な仕様適合フレームワークの新設。
- テストで見つかった製品欠陥の修正。

## Design

テスト件数ではなく、規範例が要求する観測点を基準に不足を判断する。

domain の表現検証、usecase の状態遷移、HTTP の契約、UI からの配線は異なる欠陥を検出するため、一つの層のテストで他の層を代用しない。

E2E は lifecycle workflow を名前に含むだけの別機能を証拠にせず、管理画面から作った定義が有効化され、dry-run の実評価と実行履歴を経て retry できることを観測する。

## Tasks

- [ ] T001 [Inventory] lifecycle workflow の `EX-*` と既存テストを対応付け、層ごとの不足を記録する。
- [ ] T002 [Unit] operator と filter/action 件数境界に不足する domain または usecase テストを追加する。
- [ ] T003 [HTTP] 認可、テナント境界、revision、not-found、dry-run、retry に残る handler test の不足を追加する。
- [ ] T004 [E2E RED] 管理画面から主要経路を通せない現状を Playwright で固定する。
- [ ] T005 [E2E] 作成、enable、dry-run、履歴確認、retry を通す Playwright シナリオを追加する。
- [ ] T006 [Triage] 検出した製品欠陥を個別の bugfix work item に分ける。
- [ ] T007 [Verify] 対象の検査と標準検証を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run verify`
- 主要 E2E の配線を一時的に外した場合に、追加した Playwright シナリオが失敗することを確認する。

## Risk Notes

リスクは medium であり、本項目ではテストだけを変更する。

広い E2E を一つ作るだけでは失敗原因が分かりにくくなるため、unit と HTTP で境界条件を固定し、E2E は層をまたぐ主要経路へ限定する。

未発見の製品欠陥をテストに合わせて同時修正すると変更の意味が混ざるため、個別の bugfix work item へ分ける。
