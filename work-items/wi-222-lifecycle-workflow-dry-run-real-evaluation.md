---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-219-lifecycle-workflow-admin-api]
---

# lifecycle workflow の dry-run を実態評価する simulation に修正する

## Motivation
wi-153 の設計は「dry-run は workflow revision と対象 user_id を受け、同じ validator / trigger
evaluator / action planner を使って action ごとに `would_change` / `no_op` / `blocked` と理由を
返す」ことを要求している。しかし実装 (`backend/identitymanagement/adapters/http/admin_lifecycle_workflow_handler.go`
L269-273) は `revision.Actions` を単に列挙し、各 action に `"would_change"` を決め打ちで返しており、
対象 User の group membership、application assignment、required action、status、email 検証状態と
いった現在の状態を一切評価していない。trigger/filter が対象 User に一致するかどうかの判定も行われて
いない。

さらに dry-run が参照する revision は常に `workflow.CurrentRevision` (handler L262) であり、spec
(L1809) が要求する `enabled_revision` (フォールバック `current_revision`) ではない。そのため、まだ
enable していない draft 編集が「本番相当の結果」として管理者に提示されてしまう。

dry-run は管理者が enable 前に安全性を確認するための唯一の手段であり、この欠陥は「dry-run で確認した
はずが実行結果と食い違う」という実運用上の誤操作リスクに直結する。

## Scope
- `backend/identitymanagement/` の action executor (`lifecycle_workflow_dispatcher.go` 等) から、
  副作用を起こさず現在状態を読んで `changed` / `no_op` / `blocked` 相当を判定する部分を
  純粋な評価関数として抽出する。
- dry-run usecase を、trigger/filter の一致判定と上記の状態評価関数を使って実装し直す。
  trigger/filter が対象 User に一致しない場合の結果表現を明確にする。
- dry-run が参照する revision を `enabled_revision` 優先 (未 enable なら `current_revision`
  フォールバック) に修正する。
- `admin_lifecycle_workflow_handler.go` の決め打ち `would_change` を削除し、usecase の結果を
  そのまま返すようにする。
- `spec/contexts/identity-management.yaml` の `LifecycleWorkflowDryRunStepResult` /
  `scenarios` を、実状態評価に基づく判定として記述を明確化する。

## Out of Scope
- 複数 User を対象にした一括 dry-run。対象は引き続き単一 `target_user_id` とする。
- dry-run 結果の UI 表示改善は [[wi-224-lifecycle-workflow-list-pagination-and-admin-ui]] で扱う。

## Plan
- action executor の「現在状態を読んで outcome を決める」ロジックを、本実行 (checkpoint を伴う) と
  dry-run (副作用なし) の両方から呼べる形に分離する。ADR-113 が定めた desired-state action の設計
  原則は変えない。
- trigger/filter の評価は、trigger capture で使っている評価器を対象 User の現在属性に対して再利用
  する。

## Tasks
- [ ] T001 [Domain] action の状態判定ロジックを副作用なしの評価関数として抽出する。
- [ ] T002 [App] dry-run usecase を `enabled_revision` 対象・実状態評価に置き換える。
- [ ] T003 [HTTP] handler の決め打ち `would_change` を削除し usecase の結果を返す。
- [ ] T004 [Verify] action 種別ごと、trigger 不一致、revision 分岐を table test で検証する。

## Verification
- `just test-go`
- `just verify-go`
- 自動: 既に group member である User への `add_group_member` dry-run が `no_op` を返す。
- 自動: 未 enable の draft 編集後の dry-run が `enabled_revision` の内容を反映し、draft の変更を
  反映しない。
- 自動: trigger/filter が対象 User に一致しない場合、dry-run が一致しない旨を返す。
- 手動: 管理画面で dry-run 結果と、実際に enable して得られる実行結果が一致することを確認する。

## Risk Notes
判定ロジックの共有化には本実行側 (executor) のリファクタリングが伴うため、既存の step 実行・retry・
冪等性テストが全て変わらず通ることを回帰確認する。dry-run が引き続き WorkflowRun / Job / membership /
assignment / required action / status / email を一切作成・変更しない制約は維持する。
