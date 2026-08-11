---
status: pending
authors: [tn]
risk: high
created_at: 2026-08-11
depends_on: [wi-355-replace-scl-architecture-ledgers-and-adrs]
change_kind: tooling
spec_impact:
  kind: none
  reason: Cedar の採否を評価する将来作業であり、この work item 自体は現行の製品仕様や認可挙動を変更しない。
---

# Cedar を IdMagic の実行時認可正本として採用するか再評価する

## Motivation

specification の認可記述を廃止した後も、認可要件と Go の実行時 evaluator の間に重複が残り得る。一方、
Cedar の導入は policy schema、entity projection、validator、運用、移行を伴うため、仕様体系の簡素化と
同時に採用を確定すると変更リスクと管理対象を増やす。

## Scope

- Cedar / cedar-go の実行時・検証時の成熟度を実装時点で再調査する。
- 代表的な一つの認可経路で、既存 Go evaluator を実際に置換する pilot を設計する。
- policy、schema、entity projection、テスト、運用負荷を比較し、採用または不採用を決める。

## Out of Scope

- specification置換作業中のCedar導入。
- 文書としてだけCedar policyを追加し、Go evaluatorと二重管理すること。

## Design

- Cedarを採る場合の必須条件は、対象経路の実行時判断をCedarへ移し、同じ意味のGo rule mapを削除できることとする。
- 採否は将来時点のvalidator安定性、性能、障害時のfail-closed挙動、ローカル/remote AuthZEN互換性で判断する。
- 不採用の場合も、認可requirementsと既存Goテストを正本/実現の関係として維持する。

## Plan

- 現行認可経路と重複を再計測する。
- pilot対象と受け入れ基準を確定する。
- runtime testを先に追加して比較実装する。
- 採否と撤去可能な既存コード量をCompletionに記録する。

## Tasks

- [ ] T001 [Research] Cedar実装とvalidatorの成熟度を再評価する。
- [ ] T002 [Design] runtime pilotとfail-closed受け入れ基準を確定する。
- [ ] T003 [Pilot] 一つの認可経路で既存evaluatorとの置換可能性を検証する。
- [ ] T004 [Decision] 採否、移行費用、残る二重管理を記録する。
- [ ] T005 [Verify] 認可境界テストと全体検証を実行する。

## Verification

- `just test-go`
- `just check`
- `just verify`

## Risk Notes

認可エンジンの置換はsecurity-criticalである。文書追加だけで採用済みにせず、runtime pilotとfail-closed
テストが成立した場合だけ本移行を提案する。
