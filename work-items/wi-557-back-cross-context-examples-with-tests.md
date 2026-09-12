---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
---

# 横断シナリオが宣言する具体例 4 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決めたが、`docs/scenarios.feature.md` が宣言する 4 件はどの Context にも属さない。**複数の Context が協調して初めて成り立つ振る舞いなので、どの Context の子 work item に入れても片側の話にしかならない。** 本項目がこの 4 件を引き取る。

対象は `EX-PLATFORM-001-01`、`EX-PLATFORM-002-01`、`EX-PLATFORM-003-01`、`EX-PLATFORM-003-02` である。`REQ-PLATFORM-004` の 2 件は既に名指しを持つ。

**`EX-PLATFORM-001-01` は近いテストがあるが、具体例を検証していない。** `backend/shared/http/server_http/routes_e2e_test.go` の `TestDisabledUserLoginAndExistingSessionAreRejected` は、利用者の状態を repository へ直接書いて無効化しており、具体例の `When`（管理者が無効化する）を通っていない。そのため 5 つの `Then` のうち、Agent の `AgentRevocationEpoch` の前進と `AgentAccessRevoked` の発行という 2 つを観測できる位置にない。

## Scope

- 4 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-PLATFORM-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。
- 各具体例が名指す参加 Context のすべてを、製品の正式な入口から 1 つの経路として通す。片方の Context だけを観測するテストは、この 4 件の根拠にならない。
- `EX-PLATFORM-001-01` は管理者の無効化操作を入口とし、5 つの `Then` をすべて観測する。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。
- `REQ-PLATFORM-004` の 2 件。既に `backend/shared/http/support_http/csrf_test.go` などが名指している。

## Verification

- `mise run check-spec` が、`docs/scenarios.feature.md` の 4 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **片方の Context だけを観測して済ませる。** 4 件はいずれも複数の Context が協調して初めて成り立つ振る舞いであり、引き金を持つ Context と結果を観測する Context が違う。片側だけのテストは、この具体例の根拠にならない。`TestDisabledUserLoginAndExistingSessionAreRejected` が repository へ直接書いて無効化しているのが、その失敗の実例である。
- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `EX-PLATFORM-001-01` は 5 つの `Then` を持つ。`Then` の数だけ観測が要る。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。
