---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
priority: p2
change_kind: feature
affected_spec:
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-003 }
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-007 }
depends_on: [wi-153-identity-lifecycle-workflows]
---

# lifecycle workflow の trigger/filter 表現力を拡張する

## Motivation
現在の `LifecycleWorkflow` は 1 revision につき trigger kind が 1 つ
(`spec/contexts/identity-governance/models.tsp` の `WorkflowTriggerDef` は配列ではなく単数)、filter は
field/operator/value の AND 結合のみで (`filters` は最大 20 項の AND、`WorkflowFilterOperator` は
`eq`/`not_eq`/`in`/`exists` に限定)。実運用の joiner/mover/leaver では「部署が Sales に変わった、または役職が Manager に
変わった」のような OR 条件、「入社日から 30 日以上経過」のような比較演算子が必要になる場面が多い。

現状はこれらを表現できず、管理者は同じ action セットを持つ複数の workflow 定義を条件ごとに複製する
運用を強いられ、revision 管理・監査の一貫性が損なわれる。

## Scope
- `WorkflowTriggerDef` を単一 kind の配列 (OR 結合、各要素は既存の kind + filters) に拡張するか、
  filter grouping (AND/OR の入れ子) を導入するかを検討し、いずれかを `docs/contexts/identity-governance/decisions.md` で確定する。
- `WorkflowFilterOperator` に、数値・日付型属性向けの比較演算子 (`gt`/`gte`/`lt`/`lte`) を追加する
  検証を行う。
- 現行 20 件の filter/action 上限を、拡張後のモデルでも妥当な値に再設定し、保存時 validation を
  更新する。

## Out of Scope
- 任意の expression 言語 (CEL 等) への全面移行。`docs/contexts/identity-management/decisions.md` の `DynamicGroupRule` とは異なる制約付き
  モデルを維持する方針を継続する (wi-153 の Plan が明示する「任意 expression engine は採らない」を
  踏襲する)。
- action 側の条件分岐 (per-action condition) や DAG/loop。これは別途検討する。
- 新しい trigger kind そのものの追加。日付起点の kind は
  [[wi-225-lifecycle-workflow-date-based-triggers]] が扱う。
  本 WI は `WorkflowTriggerDef` の**構造**を広げる側であり、wi-225 が足す kind もその構造に乗る。
  wi-225 を先に完了させてから着手すると、追加された kind を含めた形で OR と比較演算子を
  一度に設計できる。

## Plan
- まず `docs/contexts/identity-governance/decisions.md` で「複数 trigger kind (OR) を許すか」「filter に OR/NOT の grouping を許すか」「比較
  演算子をどこまで増やすか」を決定してから実装する。CEL 化は明示的に却下し、型付き vocabulary の
  拡張という既存方針を維持する。
- 既存 workflow の互換性 (既存の単一 trigger/filter 定義) は新モデルの特殊ケースとして扱えるように
  し、migration を不要にする。

## Tasks
- [ ] T001 [Decision] trigger/filter 拡張の範囲 (複数 trigger、filter grouping、比較演算子) を
  `docs/contexts/identity-governance/decisions.md` に記録する。
- [ ] T002 [Spec] 決定した拡張を models/scenarios に反映する。
- [ ] T003 [App] validator/trigger evaluator を拡張し、既存 workflow との互換性を保つ。
- [ ] T004 [UI] editor に複数 trigger/filter grouping の編集 UI を追加する。
- [ ] T005 [Verify] 既存 workflow の互換動作と新条件の評価を検証する。

## Verification
- `mise run check`
- `mise run test-go`
- `mise run verify-ui`
- 自動: 既存の単一 trigger/filter workflow が変更なく同じ挙動を維持する (後方互換)。
- 自動: OR 条件・比較演算子を使った filter が意図通りに評価される。

## Risk Notes
trigger/filter モデルの拡張は revision 管理・dry-run・trigger evaluator の複数箇所に影響するため、
既存 workflow の意味が変わらないことを最優先に回帰テストする。表現力を上げすぎると wi-153 が避けた
「汎用ワークフローエンジン化」に近づくため、型付き語彙の範囲に留める。
