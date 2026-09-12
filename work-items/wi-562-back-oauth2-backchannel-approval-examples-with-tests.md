---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。実装が具体例のとおりに振る舞っていない件が見つかった場合、それは欠陥として個別の work item に切り出す。" }
---

# OAuth2 のバックチャネル認可と承認リクエストが宣言する具体例 23 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取り、最初の 1 規則群を通しで消化して所要を測った。**6 件のうち注記だけで済んだものは 1 件も無かった。** この所要では 105 件は 1 つの記録に収まらないので、残りを規則群ごとの子 work item へ割った。

本項目はそのうち REQ-OAUTH2-041 から REQ-OAUTH2-043 までが宣言する 23 件を引き取る。`REQ-OAUTH2-041` は 10 件を宣言していて、OAuth2 Context の単一の規則としては最多である。

この 3 規則は 1 つの状態機械を 3 方向から言っている。認可要求の受理 (041)、承認が成立していない要求の交換 (042)、要求を判断できる主体 (043) である。3 つを別々の記録へ割ると、同じ承認リクエストの状態遷移を 3 回組み立てることになるので、1 つの記録が持つ。

## Scope

- 23 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。
- 承認リクエストの状態遷移を表として Design に書く。23 件のうち大半が状態と遷移の話なので、表が無いと同じ状態を別の言葉で 2 回テストすることになる。

## Out of Scope

- 他の規則群が宣言する具体例。[[wi-559-back-oauth2-authorization-code-examples-with-tests]]、[[wi-560-back-oauth2-registration-and-logout-examples-with-tests]]、[[wi-561-back-oauth2-client-administration-examples-with-tests]]、[[wi-563-back-oauth2-delegation-and-agent-examples-with-tests]] が持つ。
- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Verification

- `mise run check-spec` が、対象の 23 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **入力の妥当性検査を、状態遷移のテストで代用する。** `EX-OAUTH2-041-02` から `041-08` までは要求パラメーターの拒否である。承認リクエストが 1 件も作られないことまで読まないと、作ってから拒否する実装を見分けられない。
- **並行交換を、逐次の 2 回で代用する。** `EX-OAUTH2-042-05` は「並行に 2 回交換する」と言っている。逐次に 2 回叩くテストは `042-04` の再交換であって、`042-05` ではない。
- **ステップアップ認証と CSRF の拒否を、応答だけで観測する。** `EX-OAUTH2-043-02` と `043-03` は判断そのものが成立しないことを言う。拒否のあとに承認リクエストの状態を読み直す。
