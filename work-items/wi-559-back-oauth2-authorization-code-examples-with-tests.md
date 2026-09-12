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

# OAuth2 の認可コードとトークン発行が宣言する具体例 20 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取り、最初の 1 規則群 (REQ-OAUTH2-001 から REQ-OAUTH2-004) を通しで消化して所要を測った。

測定の結果は、**6 件のうち注記だけで済んだものが 1 件も無かった**というものである。5 件はテストを新しく書く必要があり、うち 1 件は共有 fixture へ保存先を 2 つ足す必要があった。残る 1 件は規範と実装が食い違っていた。

この所要では 105 件は 1 つの記録に収まらないので、wi-538 は残りを規則群ごとの子 work item へ割った。本項目はそのうち REQ-OAUTH2-001 と REQ-OAUTH2-005 から REQ-OAUTH2-014 までが宣言する 20 件を引き取る。

REQ-OAUTH2-001 をこの群へ入れたのは、`EX-OAUTH2-001-01` が「ユーザーに紐づくグラントを `/token` で交換する」ところから account リソースサーバーの参照までを 1 本の `Then` の連なりで言っていて、REQ-OAUTH2-005 が要求する認可コード + PKCE の完全な流れと同じ組み立てを必要とするためである。同じ fixture を 2 つの記録で別々に建てるのは避ける。

## Scope

- 20 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の規則群が宣言する具体例。[[wi-560-back-oauth2-registration-and-logout-examples-with-tests]]、[[wi-561-back-oauth2-client-administration-examples-with-tests]]、[[wi-562-back-oauth2-backchannel-approval-examples-with-tests]]、[[wi-563-back-oauth2-delegation-and-agent-examples-with-tests]] が持つ。
- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Verification

- `mise run check-spec` が、対象の 20 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。wi-538 の測定では、6 件のうち注記だけで済んだものは 1 件も無かった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。分類は読む順の材料にとどめる。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `EX-OAUTH2-005-01` は 5 つのイベント発行を並べ、`EX-OAUTH2-005-07` は応答とファミリー失効とイベント 2 種を並べている。`Then` の数だけ観測が要る。
- **fixture が製品と違うスタックを観測する。** 認可コードの流れは署名器、セッション、認証文脈の解決を伴う。どれかを偽物で置き換えると、入口を通したという事実そのものが根拠にならなくなる。`backend/shared/http/server_http` の既存 fixture が `Register` を通している理由はそこにある。
