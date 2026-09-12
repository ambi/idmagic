---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: テストの追跡性を補い、公開仕様、利用者向けの振る舞い、運用手順を変更しない。
  references: []
initial_context:
  specification:
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-001
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-002
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-006
    - docs/contexts/seeding/scenarios.feature.md#REQ-SEEDING-010
  typespec: []
  source:
    - backend/cmd/internal/bootstrap/seeding.go
    - backend/seeding/manifests_yaml/loader.go
    - backend/seeding/usecases/plan.go
  tests:
    - backend/cmd/internal/bootstrap/seeding_test.go
    - backend/seeding/manifests_yaml/loader_test.go
    - backend/seeding/usecases/plan_test.go
  stop_before_reading:
    - frontend
    - docs/contexts/oauth2
---

# Seeding が宣言する具体例 5 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/contexts/seeding/scenarios.feature.md` が宣言する 5 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 2 件だけ**だというものである。残る 14 件は、既存テストへ新しい観測を足すか、テスト自体を書く必要があった。件数は作業量の目安にならない。

## Scope

- 5 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-SEEDING-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Verification

- `mise run check-spec` が、`docs/contexts/seeding/scenarios.feature.md` の 5 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Design

既存の `domain.Plan`、`seedusecases.Run`、`bootstrap.Seed` を変更しない。

テストは各具体例が指定する入力、戻り値、永続状態を観測し、対応する `EX-SEEDING-*` 注記を付ける。

部分失敗のテストでは、状態を持つテスト用 `Contributor` を使い、完了済み論理キーと未完了論理キーを区別して再試行の収束を観測する。

## Plan

1. 台帳上の 5 件を確認する。既存の負債は `check-spec` の失敗条件ではないため、Acceptance RED は適用しない。
2. 既存テストへ具体例の観測内容を注記し、`EX-SEEDING-006-01` と `EX-SEEDING-010-01` の不足した観測を追加する。
3. テスト、追跡性検査、最終検証を実行し、完了証拠を記録する。

## Tasks

- [x] T001 [Acceptance] 台帳にある 5 件を確認し、既存負債には Acceptance RED が適用されないことを記録する。
- [x] T002 [Unit] `EX-SEEDING-001-01`、`EX-SEEDING-002-01`、`EX-SEEDING-002-02` の既存テストへ観測内容を対応付ける。
- [x] T003 [Unit] `EX-SEEDING-006-01` の全操作 `noop` と不変な永続状態を観測する。
- [x] T004 [Unit] `EX-SEEDING-010-01` の部分失敗後の再試行が完了済みと未完了の論理キーを正しく扱うことを観測する。
- [x] T005 [Verify] 台帳を更新し、対象パッケージと最終ゲートを検証する。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は main との差分に規範仕様の変更がないことを報告した。
  5 件の既存の Seeding 具体例を、それぞれの入力、計画、永続状態を観測するテストへ対応付け、被覆台帳から削除した。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 既存の具体例被覆負債をテストへ対応付ける保守作業であり、利用者が観測する新しい境界を変更しない。
  - **Requirement**: N/A: 規範の追加または変更はない。
  - **Observed Failure**: N/A: 既存の負債は `check-spec` の失敗条件ではないため、Acceptance RED は成立しない。
  - **Detection Reason**: 台帳の 5 エントリを読んで対象を固定し、削除後の `check-spec` で具体例を名指しするテストの被覆数が 322 件から 327 件へ増えたことを確認した。
- **Unit RED Evidence**:
  - **Test**: N/A: 実装済みの振る舞いの観測を強化する作業であり、製品コードを変更しない。
  - **Requirement**: N/A: 規範の追加または変更はない。
  - **Observed Failure**: N/A: 既存実装は強化後のテストを満たした。
  - **Detection Reason**: `TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges`、`TestLoadStrictlyDecodesAndMergesContainedIncludes`、`TestRunCanBeRetriedAfterApplyFailure` が、それぞれ dry run、マニフェスト統合、部分失敗後の再試行を観測する。
- **Change-Resistance Results**:
  低リスクのテスト追跡性変更であり、追加の耐性試験は適用しない。
- **Verification Results**:
  - `mise run test-go-package -- ./backend/seeding/usecases` - passed
  - `mise run test-go-package -- ./backend/seeding/manifests_yaml` - passed
  - `mise run test-go-package -- ./backend/cmd/internal/bootstrap` - passed
  - `mise run test-go-changed` - passed
  - `mise run check-spec` - passed
  - `mise run lint-go` - passed
  - `mise run verify` - `wi-462` と仕様文書の同時変更に由来する `check` の失敗により failed。今回の 5 ファイルは対象外であり、修正しない。
