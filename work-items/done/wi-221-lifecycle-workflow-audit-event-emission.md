---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-153-identity-lifecycle-workflows, wi-218-lifecycle-workflow-action-execution-and-audit]
---

# lifecycle workflow の監査イベント発行を完成させる

## Motivation
`spec/contexts/identity-management.yaml` は `LifecycleWorkflowCreated` / `LifecycleWorkflowUpdated` /
`LifecycleWorkflowEnabled` / `LifecycleWorkflowDisabled` / `LifecycleWorkflowRunStarted` /
`LifecycleWorkflowRunSucceeded` / `LifecycleWorkflowRunPartiallyFailed` / `LifecycleWorkflowRunFailed` /
`LifecycleWorkflowRunCanceled` / `LifecycleWorkflowStepFailed` の計 10 個の event を `emits` として
宣言している (L1973-2108)。しかし `backend/shared/spec/events.go` にはこのうち
`LifecycleWorkflowDeleted` (L881) しか型が定義されておらず、
`backend/identitymanagement/usecases/lifecycle_workflows.go` でも `DeleteLifecycleWorkflow` (L292)
だけが `adminEmit` を呼んでいる。`CreateLifecycleWorkflow` (L75) / `UpdateLifecycleWorkflow` (L125) /
`EnableLifecycleWorkflow` (L162) / `DisableLifecycleWorkflow` (L183) と、run/step の状態遷移経路は
一切 event を発行しない。

ADR-113 と wi-153 は「workflow_id / run_id / step_id / outcome を監査検索可能にする」ことを設計の
柱としており、SCL の scenarios も `emitted.exists(...)` でこれを表明しているが、SCL scenario と Go
実装を結びつけるテストがないため、この欠落は wi-218/wi-219/wi-220 の完了報告を通過してしまった。
結果として、削除以外の workflow 操作 (作成・更新・enable・disable) と run の開始・成功・一部失敗・
失敗・中止・step 失敗には監査証跡が一切残っていない。

## Scope
- `backend/shared/spec/events.go` に不足している 9 個の event 型
  (`LifecycleWorkflowCreated`/`Updated`/`Enabled`/`Disabled`、
  `LifecycleWorkflowRunStarted`/`Succeeded`/`PartiallyFailed`/`Failed`/`Canceled`、
  `LifecycleWorkflowStepFailed`) を追加する。
- `backend/identitymanagement/usecases/lifecycle_workflows.go` の
  `CreateLifecycleWorkflow` / `UpdateLifecycleWorkflow` / `EnableLifecycleWorkflow` /
  `DisableLifecycleWorkflow` から対応する event を発行する。
- run/step の状態遷移を扱う usecase (dispatcher / step executor) から
  `LifecycleWorkflowRunStarted` 以降の run event と `LifecycleWorkflowStepFailed` を発行する。
- `spec/contexts/identity-management.yaml` の該当 scenarios (`emitted.exists(...)`) が Go の挙動と
  一致することを確認する統合テストを追加する。
- workflow_id / run_id / step_id で Audit event が実際に検索できることを確認する。

## Out of Scope
- Audit context の汎用 read model 改善や admin UI の検索条件追加。
- 汎用的な SCL scenario ↔ Go conformance framework の新設 (本 WI では lifecycle workflow に限定した
  最小限の対応漏れ検出に留める)。
- job admin 画面 ([[wi-157-job-admin-operations-surface]])。

## Plan
- 既に実装済みの `DeleteLifecycleWorkflow` の `adminEmit` パターンをそのまま残り 9 event へ機械的に
  適用する。ロジック変更は行わず event 発行の追加のみに限定し、regression risk を抑える。
- SCL scenario の `emitted.exists(...)` アサーションを Go の統合テストとして固定し、以後同種の
  発行漏れを CI で検出できるようにする。

## Tasks
- [x] T001 [SCL] emits 宣言と Go 実装の対応表を作り、必要なら scenario の記述を調整する。
- [x] T002 [Domain/Events] `backend/shared/spec/events.go` に不足イベント型を追加する。
- [x] T003 [App] Create/Update/Enable/Disable usecase と run/step 実行経路で `Emit` を呼ぶ。
- [x] T004 [Verify] 全イベント発行と Audit 検索を統合テストで検証する。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-go`
- 自動: create/update/enable/disable の各操作後、対応する Audit Event が workflow_id で検索できる。
- 自動: run の succeeded/partially_failed/failed/canceled と step failed 発生後、対応する Audit
  Event が run_id/step_id で検索できる。

## Risk Notes
イベント発行漏れは監査証跡の欠落であり、コンプライアンス上のリスクである。今回の修正は
desired-state action や run 実行ロジックを変更せず、`Emit` 呼び出しの追加に限定することで、
既存の実行・冪等性・retry の挙動への影響を避ける。

実装中に、`LifecycleWorkflowRunHandler` が「成功した step が一つもない」場合を判定しておらず、
`WorkflowRunFailed` (ADR-113 決定5) へ到達できない状態だったことが判明した (常に succeeded/
partially_failed のいずれかにしか終端しない)。`LifecycleWorkflowRunFailed` event を正しく発行
するには終端 status 判定そのものが正しい必要があるため、本 WI の範囲内で
「成功 0 件かつ失敗あり → failed、それ以外で失敗あり → partially_failed」という ADR-113 の
既定ロジックに修正した。action executor の個々の判定・冪等性・checkpoint の挙動自体は変更していない。

## Completion
- **Completed At**: 2026-07-16
- **Summary**:
  `LifecycleWorkflowCreated`/`Updated`/`Enabled`/`Disabled`/`RunStarted`/`RunSucceeded`/
  `RunPartiallyFailed`/`RunFailed`/`RunCanceled`/`StepFailed` の 9 event 型を追加し、対応する
  usecase (Create/Update/Enable/Disable と run/step 実行経路、queued run cancel 経路) から発行
  するようにした。`CancelQueuedByWorkflow` を canceled run を返す契約に変更し、Disable/Delete が
  canceled run ごとに `RunCanceled` を発行する。監査検索 registry に `workflow.id` /
  `workflow_run.id` / `workflow_step.id` を追加し、`ExtractSearchAttributes` が
  `workflowId`/`runId`/`stepIndex` payload から抽出するようにした。run 終端 status 判定を
  ADR-113 の「成功 0 件なら failed」ロジックに修正した。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` - passed
  - `just verify-go` - passed (lint 0 issues, race test 全パッケージ green)
- **affected_guarantees_state**: workflow の作成・更新・enable・disable と run の開始・成功・
  一部失敗・失敗・中止・step 失敗のすべてに audit event が伴い、workflow_id/workflow_run.id/
  workflow_step.id で検索できる。run は「成功 0 件かつ失敗あり」の場合に failed へ正しく終端する。
- **evidence**:
  - procedure: Go unit/integration tests (usecases, dispatcher, audit search extractor, bootstrap
    audit event record) と `just verify-go` (golangci-lint + `go test -race ./...`)
    result: passed
    artifacts: []
