---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
depends_on: [wi-409-evaluate-agentic-discipline-development-workflow]
change_kind: tooling
evidence_policy: risk-based-v1
approval:
  by: tn
  at: 2026-08-23
  scope: "wi-409 が提案した高リスク Go ロジック向けの限定的な変異試験を、独立した評価として実施する。"
  baseline: 3cb041f1d61007a3213ead7c1bba989d1d19824a
initial_context:
  specification: [DEVELOPMENT.md, WORK_ITEM_FORMAT.md]
  typespec: []
  source: [mise.toml, backend/jobs/domain/job.go]
  tests: [backend/jobs/domain/job_test.go, tools/check/src/mise-config.test.ts]
  stop_before_reading: [frontend, spec, docs/contexts]
spec_impact: { kind: none, reason: "開発時の変異試験を評価する道具と任意実行タスクだけを追加し、製品の外部契約、振る舞い、保証は変更しない。" }
---

# Go の差分変異試験を限定的に評価する

## Motivation

`wi-409` は、試験が変更した論理の欠陥を検出できることを確かめる案を採用した一方、全変更への一律な変異試験は費用と同値変異の評価前に導入しないと決めた。
形式と承認規約を変更する `wi-410` へ道具の評価まで併合せず、小さな Go パッケージで再現性、所要時間、検出結果を独立して記録する。

## Scope

- Gremlins を `mise` に固定し、一つの Go パッケージと任意の Git 差分へ対象を限定するタスクを追加する。
- 純粋な状態遷移を持つ `backend/jobs/domain` で一回試行し、検出、生存、未網羅、所要時間を記録する。
- 通常の `mise run verify` には追加せず、高リスクまたは重大リスクの純粋な変更論理で選択する証拠経路として評価する。

## Out of Scope

- 全 Go パッケージ、UI、外部アダプターへの変異試験。
- CRAP または変異得点の全体閾値。
- 生き残った変異を自動的に欠陥と判定すること。
- 製品コードの変更。

## Design

Gremlins `0.6.0` を `mise.toml` へ固定し、`mise run test-go-mutation-package -- <package> [git-ref]` だけを公開する。
作業中は一つのパッケージを、レビュー時は承認基準点からの差分を指定できるようにし、並列数と試験 CPU を一つへ固定して局所実行の負荷を抑える。
変異試験は標準の `verify` から外し、対象となる変更のリスクと純粋性に応じて選ぶ。

## Plan

道具と任意実行タスクを追加し、構成検査でバージョン固定と全体検証からの除外を確認する。
次に `backend/jobs/domain` で試行し、結果と限界を完了証拠へ記録する。

## Tasks

- [x] T001 [Tooling] Gremlins `0.6.0` とパッケージ限定の `mise` タスクを追加した。
- [x] T002 [Guard] 変異タスクが固定され、通常の `verify` に含まれないことを `mise-config.test.ts` で検査した。
- [x] T003 [Pilot] `backend/jobs/domain` で15変異を試行し、13件の検出、生存0件、未網羅2件を確認した。
- [x] T004 [Evaluate] 約24秒という局所実行時間と未網羅の算術変異2件を記録し、全体必須化を見送った。

## Verification

- `mise run test-tools`
- `mise run test-go-mutation-package -- ./backend/jobs/domain`
- `mise run verify`

## Risk Notes

Gremlins は `0.x` で後方互換性を保証していないため版を固定する。
変異試験は同値変異、未網羅、時間依存の揺らぎを含み得るため、数値だけを品質判定にせず、生存または未網羅の個別変異を試験目標として人間が評価する。

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Gremlins `0.6.0` を固定したパッケージ限定の変異試験タスクを追加し、`backend/jobs/domain` の小規模試行で費用と検出力を評価した。通常の全体検証には組み込まず、高リスク以上の純粋な変更論理で選択する経路に限定した。
- **RED Evidence**:
  - **Test**: `mise run test-go-mutation-package -- ./backend/jobs/domain`
  - **Requirement**: N/A: 製品要求ではなく試験の欠陥検出能力を評価する開発道具である
  - **Observed Failure**: Gremlins が生成した変異15件のうち13件で対象パッケージの試験が失敗した
  - **Detection Reason**: 条件反転、境界変更、増減変更という妥当な誤実装に既存試験が失敗することを直接観測する
- **Post-Approval Changes**: none。`mise run spec-diff 3cb041f1d61007a3213ead7c1bba989d1d19824a` は規範仕様の変更なしと報告した。
- **Independent Verification**: Laplace が `mise.toml` と `mise-config.test.ts` を読み取り専用で確認し、道具の固定と全体 `verify` からの除外に問題がないことを確認した。変異試行を `wi-410` から分ける指摘を本作業項目で解消した。
- **Change-Resistance Results**: Gremlins は13件を検出し、生存0件、未網羅2件、タイムアウト0件、実行不能0件、所要24.56秒、試験有効性100%、変異網羅率86.67%と報告した。未網羅は `job.go` 174行目と175行目の算術変異であり、全体ゲート化する前の具体的な試験候補として残す。
- **Verification Results**:
  - `mise run test-tools`：passed（181 tests）。
  - `mise run test-go-mutation-package -- ./backend/jobs/domain`：passed（13 killed、0 lived、2 not covered）。
  - `mise run verify`：passed。
