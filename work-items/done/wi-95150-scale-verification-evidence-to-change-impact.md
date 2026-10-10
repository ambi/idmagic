---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回公開前であり、変えるのは作業項目の検査と開発文書だけである。製品の利用者と運用者が観測する振る舞い、設定、API、既存データは変わらない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - .agents/skills/implement-work-item/SKILL.md
    - tools/check/src/primary-use-case-evidence.ts
    - tools/check/src/work-item-changes.ts
    - tools/check/src/check-work-items.ts
    - tools/check/src/agent-guidance.ts
    - tools/check/schemas/work-item.schema.json
  tests:
    - tools/check/src/primary-use-case-evidence.test.ts
    - tools/check/src/check-work-items.test.ts
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "作業記録と検証手順の適用条件を変更する。製品の HTTP 応答、認証と認可、永続状態、イベント、通知と品質要件は維持する。"
---

# 変更の影響に合わせて証拠を選び、低リスクの作業へ失敗の記録を一律に要求しない

## 動機

[証拠の要件](../../docs/development/specification-first-workflow.md#4-証拠の要件)は、低リスクの文書変更にも RED または実際に失敗した代替検査を要求する。
既存検査が成功していても必要な改善に、記録を満たすための失敗を作る費用が生じる。
現在は未リリースであり、当面も未リリースを継続する。
リリースの証拠整備より、日常開発で必要な保証と手続きの量を対応させることを優先する。

## 対象範囲

- 仕様影響のない文書変更と構造変更に必要な検証と完了記録の縮小。
- 製品の振る舞い、検査器の判定、認可や永続化への影響に応じた RED と故障注入の適用条件。
- ワークフロー、作業項目の形式、スキル、検査の一致。

## 対象外

- 認可、テナント分離、永続化、暗号、プロトコルの独立した故障を検出する検証の削除。
- 低リスクという自己申告だけによる検証の免除。
- [仕様の反例探索](../active/wi-458-specification-adequacy-counterexample-evidence.md)の新しい証拠契約の実装。

## 設計

変更する観測結果と故障の種類から、必要な検証を選ぶ。
文書の表現変更では参照、意味、形式の確認を使い、構造変更では既存の振る舞いの検証と配置の確認を使う。
それらの検査が変更前に失敗することを、改善の必要条件にしない。
検査器の拒否条件を変える場合は、その誤判定を独立した入力で検証する。

着手時に、残す証拠と免除する条件を代表例で決める。
新しい自己申告欄を増やす案や、失敗を作るためだけのテストを追加する案は採らない。
過去の完了記録へ新しい証拠を遡って要求しない。

### 代表例と必要な証拠

免除の判定には、作業項目の自己申告ではなく、完了の変更の差分を使う。
差分は、main とのマージベースからの変更と作業ツリーの変更を合わせたパスの集合である。
`change_kind` は `refactor` の主張を差分と照合するためだけに読み、それだけで免除を与えない。

| 代表的な変更 | 免除の条件 | 完了時に求める証拠 |
| --- | --- | --- |
| 仕様影響のない文書の改善 | `spec_impact` を宣言し、差分のパスがすべて Markdown である | 受け入れ RED と単体 RED は求めない。参照、形式、意味の検査（`check-links`、`check-work-items`、`verify`）の成功を検証結果に記す |
| 仕様影響のない構造変更 | `change_kind: refactor` と `spec_impact` を宣言し、差分がテストのパスを含まない | 受け入れ RED と単体 RED は求めない。変更していない既存テストと配置の検査の成功を検証結果に記す |
| 検査器の拒否条件の変更 | 免除しない | 受け入れ RED と単体 RED、または理由を伴う該当なしと実際に失敗した代替検査。中リスク以上では変更耐性の結果 |
| 製品の振る舞いの変更（feature、bugfix、標準対応） | 免除しない | 主要ユースケースの証拠（従来どおり） |
| 文書変更と名乗りながらコードを変える | 差分が Markdown 以外を含むので免除しない | 受け入れ RED と単体 RED |
| 構造変更と名乗りながらテストを変える | 差分がテストのパスを含むので免除しない | 受け入れ RED と単体 RED |

テストのパスは、`_test.go`、`.test.ts(x)` と `.spec.ts(x)` などのテストファイル、`testdata/` と `tests/` の下のファイルとする。
構造変更ではテストを変えないことが振る舞いを保つ主張の検査であり、テストを変えた変更は振る舞いも変えたものとして扱う。
`affected_spec` を宣言した記録は規範要素を変えるので免除しない。
Git の外の workspace では差分を求められないので、免除せず従来どおり RED の証拠を求める。
完了の変更の中で、別の古い完了記録を書き換えた場合も、その記録は同じ差分で判定する。
古い記録は RED の証拠を持つので、この判定で新たな所見は生じない。

採らなかった案は次のとおりである。

| 案 | 採らない理由 |
| --- | --- |
| `change_kind: docs` だけで免除する | 自己申告だけで、コードを変える変更も免除できる |
| 免除を宣言する新しい欄を足す | 自己申告欄が増えるだけで、検証の量と対応しない |
| すべての作業に RED を求め続ける | 既存検査が成功する改善に、記録のための失敗を作る費用が残る |

### 型と操作

- `changedPaths(snapshot: WorkspaceSnapshot): ReadonlySet<string> | undefined`（`tools/check/src/work-item-changes.ts`）: 完了の変更が触れたリポジトリ相対パス。Git の外では `undefined`。`changedWorkItemRecords` はこの集合から作る。
- `PrimaryUseCaseEnvironment.changedPaths: ReadonlySet<string> | undefined`: 判定への入力。Git の呼び出しは環境の組み立てに留め、`verifyPrimaryUseCaseEvidence` は純粋な判断のままにする。
- `preservesBehavior(record, changedPaths): boolean`: 上の表の免除条件。

## 計画

1. 文書、構造、検査器、製品の振る舞いの代表的変更で必要な検証を比較する。
2. 規約と形式、エージェント手順、検査を同期する。
3. 適用条件と、残す検証の検出能力を確かめる。

## タスク

- [x] T001 [Design] 変更の影響と必要な証拠の対応を決める。設計の「代表例と必要な証拠」に記した。
- [x] T002 [Docs] 不要な失敗の記録を要求する規約と手順を縮小する。対象は `docs/development/specification-first-workflow.md` の証拠の要件とリファクタリング、`docs/formats/work-item-format.md` の完了記録、`implement-work-item` スキル。検査は `mise run check-links` と `agent-guidance` の検査（`mise run check`）。
- [x] T003 [Tooling] 作業項目の検査を新しい適用条件へ合わせる。単体 RED は `mise run test-tools-file -- check/src/primary-use-case-evidence.test.ts`、受け入れ RED は `mise run test-tools-file -- check/src/check-work-items.test.ts`（Git の fixture で差分から判定する）。
- [x] T004 [Verify] 文書変更の正常完了と、重要な故障の証拠の欠落の拒否を確かめる。差分の取得、Markdown の判定、テストのパスの判定、`refactor` の照合、`affected_spec` の除外へ故障を手で注入する。

## 検証

- 変更前から検査が成功する文書改善を、架空の RED なしで完了できる。
- 振る舞いの変更を文書変更と名乗るだけでは必要な検証を免除できない。
- 重要な独立した故障への検証と、既存の完了記録の解釈を維持する。
- mise run test-tools-file、mise run check-work-items、mise run check-links、mise run verify を実行する。

## リスク

記録の削減と検証の削減を混同すると、重要な故障を見逃す。
代表例の観測結果と必要な検証を先に決め、書く量と検出能力を別々に確認する。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
  変わったのは作業項目の検査、開発文書、`implement-work-item` スキルである。
  主要ユースケースの契約が適用されない作業のうち、振る舞いを保つ変更には受け入れ RED、単体 RED、変更耐性の結果を求めなくなった。
  振る舞いを保つ変更とは、`spec_impact` を宣言し差分がすべて Markdown である変更と、`change_kind: refactor` と `spec_impact` を宣言し差分がテストのパスを含まない変更である。
  判定には作業項目の申告ではなく、main とのマージベースからの変更と作業ツリーの変更を合わせた差分を使う。
  差分の取得は `tools/check/src/work-item-changes.ts` の `changedPaths` にまとめ、変わった記録の判定もこの集合から作る。
  `affected_spec` を宣言する記録、Git の外の workspace、検査器の変更、製品の振る舞いの変更には、従来どおりの証拠を求める。
  ワークフローに「振る舞いを保つ変更」の節を足し、リファクタリングの節と作業項目フォーマットの完了記録をこれに合わせた。
  利用者の指示により、`implement-work-item` スキルの日本語の段落を英語へ統一した。
  そのとき、スキーマがすでに受け付けない `risk-based-v3` の記録を扱う一文を削った。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/check-work-items.test.ts` の「Markdown だけを変えた完了を、RED の証拠なしで通す」。
  - **要件**: N/A: 製品の振る舞いを変えない作業項目の検査の変更であり、規範となる製品要件はない。
  - **観測した失敗**: 差分を検査へ渡す前の実装では、Markdown だけを変えて完了した Git の fixture に対し `completion.acceptance_red_evidence is required for non-applicable work` と `completion.unit_red_evidence is required for non-applicable work` を返して失敗した。
  - **検出できる理由**: fixture は実際の Git の差分を `checkWorkItems` へ通すので、差分を読まない実装と、差分から免除を判定する実装を所見の有無で区別できる。同じ記録でコードも変えた fixture と、ブランチでコミットした fixture が、免除しすぎる実装を拒否する。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/primary-use-case-evidence.test.ts` の「Markdown だけを変えた仕様影響のない完了を、RED の証拠なしで受理する」と「テストを変えない refactor の完了を、RED の証拠なしで受理する」。
  - **要件**: N/A: 製品要件のない検査の変更である。
  - **観測した失敗**: 免除を実装する前は、2 件とも RED の証拠を求める 2 件の所見を返して失敗した。
  - **検出できる理由**: 同じ記録に対し、Markdown 以外を含む差分、テストを含む差分、refactor 以外の差分、`affected_spec` の宣言、差分なしの各入力で RED の証拠を求めることを別のテストが固定するので、免除の条件を緩めても締めても失敗する。
- **変更耐性の結果**:
  TypeScript は Go の変異器の対象外なので、故障を手で注入し、二つのテストファイルを実行した。

  | 注入した故障 | 検出したテスト |
  | --- | --- |
  | 差分を検査へ渡さない | 受け入れの「Markdown だけを変えた完了を、RED の証拠なしで通す」 |
  | main とのマージベースからの差分を読まない | 受け入れの、ブランチでコミットした変更の 2 件 |
  | Markdown を一つでも含めば免除する | 単体 3 件と受け入れ 2 件 |
  | テストのパスを判定しない | 単体の「テストを変えた refactor の完了には RED の証拠を求める」 |
  | `spec_impact` の宣言を求めない | 単体の「規範要素を変える記録は、Markdown だけの変更でも RED の証拠を求める」 |
  | 差分を求められないときに免除する | 単体 2 件 |
  | refactor の照合を外す | 単体 2 件と受け入れ 2 件 |
  | `testdata/`、`tests/`、`.spec`、`_test.go` をそれぞれテストのパスから外す | 単体の「テストを変えた refactor の完了には RED の証拠を求める」 |

  最初の注入では、`tests/` と `.spec` を外す故障が生き残った。
  テストの入力 `frontend/tests/e2e/demo.spec.ts` が両方の条件に当たり、片方を外しても判定が変わらなかったからである。
  入力を `frontend/e2e/demo.spec.ts` と `frontend/tests/fixtures/users.json` に分け、両方の検出を確かめた。
- **検証結果**:
  - `mise run test-tools-file -- check/src/primary-use-case-evidence.test.ts` - 成功（18 件）
  - `mise run test-tools-file -- check/src/check-work-items.test.ts` - 成功（14 件）
  - `mise run test-tools-file -- check/src/agent-guidance.test.ts` - 成功
  - `mise run check-work-items` - 成功
  - `mise run check-links` - 成功
  - `mise run verify` - 成功
