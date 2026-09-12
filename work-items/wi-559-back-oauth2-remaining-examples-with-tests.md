---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。実装が具体例のとおりに振る舞っていない件が見つかった場合、それは欠陥として個別の work item に切り出す。" }
---

# OAuth2 が宣言する残り 99 件の具体例にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取り、REQ-OAUTH2-002 から REQ-OAUTH2-004 までの 6 件を消化して 1 件あたりの所要を測った。**6 件のうち注記だけで済んだものは 1 件も無かった。** 親項目の測定（16 件中 2 件）より悪い。

その所要を根拠に、wi-538 は残る 99 件を 5 つの子 work item へ割った。

**その分割は取り消した。** [[wi-565-make-backing-declared-examples-cheap]] が測り直したところ、所要が高い原因は具体例の側ではなく道具の側にあった。94 個のテストファイルが 44 フィールドの `Deps{}` を手で組み立てていること、具体例 id から観測点を引く索引が無いこと、台帳が読んだ結果を保持しないことである。wi-565 がこの 3 つを直す。原因が消えるなら分割の根拠も消えるので、5 記録を本記録 1 つへ畳んだ。5 つに割ることは readiness と Design と Completion の固定費を 5 回払わせるだけだった。

本項目は残る 99 件すべてを引き取る。

## Scope

- 99 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 読む順は `mise run spec-route -- <id>` で決める。出力は読む順の材料であり、台帳から外す根拠にはしない。
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。既存 fixture を触る必要が出た範囲だけ、同じ基盤へ移す。全面移行はしない。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出し**、台帳の当該行へ `blocked_by` と `finding` を書いて残す。
- 変更した Go の変異は `mise run test-go-mutation -- <package>` で読む。手書きの故障注入は、変異器が表現できない「配線を外す」種類だけに残す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `EX-OAUTH2-003-04`。[[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- `backend/shared/http/testing_stack` への既存 94 fixture の全面移行。wi-565 の Out of Scope でもある。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Verification

- `mise run check-spec` が、対象の 99 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。wi-538 の測定では、6 件のうち注記だけで済んだものは 1 件も無かった。
- **`spec-route` の出力を、テストがある証拠として読む。** 候補 operation は契約から導出した読む順の材料である。`report-coverage-debt` の分類が根拠にならないのと同じ理由で、これも根拠にはならない。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `EX-OAUTH2-005-01` は 5 つのイベント発行を並べ、`EX-OAUTH2-005-07` は応答とファミリー失効とイベント 2 種を並べている。`Then` の数だけ観測が要る。
- **並行交換を、逐次の 2 回で代用する。** `EX-OAUTH2-042-05` は「並行に 2 回交換する」と言っている。逐次に 2 回叩くテストは `042-04` の再交換であって、`042-05` ではない。
- **委譲深さの上限を、境界の片側だけで観測する。** `EX-OAUTH2-048-02` は上限以内、`048-03` は上限超えを言う。片側だけでは比較演算子の向きを取り違えた実装を見分けられない。
- **Agent の種別を、既知の値だけで観測する。** `EX-OAUTH2-050-03` は「既知のどの値でもない」場合を言う。列挙を網羅するテストはこの件を通してしまう。
- **99 件を、また量を理由に割る。** 割ってよいのは意味が 2 つあるときだけである。所要が下がらないと分かったら、下がらない原因を測って道具の側を直す。それが wi-565 のやり方である。
