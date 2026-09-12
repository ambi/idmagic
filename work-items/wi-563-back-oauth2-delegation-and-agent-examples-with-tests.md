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

# OAuth2 の保護リソース、委譲、Agent が宣言する具体例 29 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取り、最初の 1 規則群を通しで消化して所要を測った。**6 件のうち注記だけで済んだものは 1 件も無かった。** この所要では 105 件は 1 つの記録に収まらないので、残りを規則群ごとの子 work item へ割った。

本項目はそのうち REQ-OAUTH2-044 から REQ-OAUTH2-050 までが宣言する 29 件を引き取る。子 work item のうち最多である。Bearer 保護リソースのエラー、DPoP Proof の `ath` 束縛、オフボードされた所有者を持つ Agent、管理コンソールとアカウントポータルの失効判定、委譲深さの上限、イントロスペクションと監査が示す委譲モード、`Supervised` な Agent の再発行が含まれる。

## Scope

- 29 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。
- 29 件を消化する途中で 1 件あたりの所要を測り直し、さらに分割するかどうかを判断する。判断は件数ではなく測定に従う。

## Out of Scope

- 他の規則群が宣言する具体例。[[wi-559-back-oauth2-authorization-code-examples-with-tests]]、[[wi-560-back-oauth2-registration-and-logout-examples-with-tests]]、[[wi-561-back-oauth2-client-administration-examples-with-tests]]、[[wi-562-back-oauth2-backchannel-approval-examples-with-tests]] が持つ。
- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。とくに Agent の生存期間そのものは `workloadidentity` と `idmanagement` が持つ。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Verification

- `mise run check-spec` が、対象の 29 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **失効判定を、失効させずに観測する。** `EX-OAUTH2-047-02` と `047-03` は jti の失効リストと revocation epoch を分けて言う。片方だけを観測すると、もう一方が効いていない実装を通す。
- **委譲深さの上限を、上限ちょうどで観測しない。** `EX-OAUTH2-048-02` は上限以内、`048-03` は上限超えを言う。境界の両側を読まないと、比較演算子の向きを取り違えた実装を見分けられない。
- **`act` の入れ子を、文字列としてだけ読む。** `EX-OAUTH2-049-*` はイントロスペクションと監査が同じ規則で委譲モードを示すことを言う。2 つの出力を別々に読み、同じ入力に対して一致することまで読む。
- **Agent の種別を、既知の値だけで観測する。** `EX-OAUTH2-050-03` は「既知のどの値でもない」場合を言う。列挙を網羅するテストはこの件を通してしまう。
