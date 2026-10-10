---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "初回公開前であり、変えるのは仕様付録の書き方と検査ツールだけで、既存データと実利用契約に影響しない。"
  references: []
initial_context:
  specification:
    - docs/formats/specification-format.md
    - docs/modules/identity-management/group-csv/acceptance.feature.md
    - docs/modules/authorization/check/acceptance.feature.md
  typespec: []
  source:
    - tools/check/src/gherkin-scenarios.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/security-controls.ts
  tests:
    - tools/check/src/gherkin-scenarios.test.ts
    - tools/check/src/spec-diff.test.ts
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "受け入れ例の重複した表現を整理する。既存の要件と例の意味、HTTP 応答、認可とテナント分離、永続状態、イベント、通知は維持する。"
---

# 受け入れ例の共通前提をまとめ、保証と検出能力を維持して読む量を減らす

## 動機

2026-10-10 の調査では、受け入れ例の文書は合計 8,293 行あった。
[グループ CSV の例](../../docs/modules/identity-management/group-csv/acceptance.feature.md)は、共通の前提とプレビューから適用までの手順を条件ごとに繰り返す。
前提を変えるたびに同じ説明を直す費用があり、違いを読むにも同じ文脈を繰り返し消費する。
現在は未リリースであり、当面も未リリースを継続するため、告知整備より現在の仕様の理解を優先する。

## 対象範囲

- 同じ操作と期待結果の形を持つ例の、共通前提、決定表、Scenario Outline による整理。
- 既存の例の識別子、独立した期待結果、テストの参照と検出する故障の維持。
- グループ CSV と OAuth2 トークンを代表例とした効果の確認と、同じ条件に合う文書への展開。

## 対象外

- 認可、拒否後の無作用、原子性、並行性、再試行など、独立した検証ケースの削除。
- 行数だけを理由に例を削除すること、異なる操作を一つの表へ押し込むこと。
- 製品の規範要件や例の意味の変更、全例を新しい表現へ強制移行すること。

## 設計

要件本文は約束を定め、付録は条件と結果の理解を補う。
共通の操作を一度示し、違いを表にする案と、少数の代表例と条件表を組み合わせる案を比べる。
既存の実行可能な例がある場合は、その一次情報と独立した期待結果を利用し、別のデータ体系を作らない。

着手時に、既存の EX ID と表の行の対応、参照と生成表示の検査方法を決める。
形式の変更が規範差分として検出される場合は、その原因を確認し、実質的な意味の変更があれば仕様影響を更新する。
差分検査を無効にして変更を隠さない。

### 調査で分かった重複

| 重複の種類 | 代表例 | 採る表現 |
| --- | --- | --- |
| 同じ `Rule` の全例が同じ `Given` で始まる | グループ CSV の各 `Rule` | `Rule` 直下の `Background` |
| 前提と操作が同じで、`But` の条件と `Then` の結果だけが異なる | OAuth2 承認の REQ-OAUTH2-041 | `Scenario Outline` と `Examples` の表 |
| 操作の順序や手順の数が例ごとに異なる | REQ-OAUTH2-006、REQ-IDMANAGEMENT-026-05 | 個別の `Example` のまま残す |

### 検査の変更

変更前の検査では、二つの表現が規範差分の比較から外れていた。

| 表現 | 変更前の扱い | 問題 |
| --- | --- | --- |
| `Rule` 直下の `Background` | 構文解析は受理するが、例の手順に含めない | 前提を変えても差分に現れず、前提へ移すと全例の差分になる |
| `Scenario Outline` の行 | プレースホルダーのままの手順と、全列の値を比べる | 個別の `Example` を `Outline` へ移すと、同じ経路でも差分になる |

`tools/check/src/gherkin-scenarios.ts` の `parseScenarioDocument` は、各例の `steps` を公式の pickle と同じ経路に展開する。

- `Background` の手順を例の手順の先頭に足す。`Background` に `When` または `Then` を置くと、検査が拒否する。
- `Outline` の行の値を、手順の本文、表、DocString の `<列名>` へ埋める。
- `ScenarioExample.parameters` には、どの手順にも現れない列の値だけを残す。

`spec-diff` の `exampleFact` は展開済みの手順と残りの値を比べ、空の値を値なしと同じに扱う。
展開した手順を読む `security-controls.ts` も、`Outline` の結果に書いたエラー型を読めるようになる。

採らなかった案は次のとおりである。

| 案 | 採らない理由 |
| --- | --- |
| 例を `testdata/*.examples.json` の実行可能な例へ移す | 例の大半は製品のテストデータを持たず、新しい一次情報を 80 を超える文書の分だけ作ることになる |
| 例の掲載だけを短くし、検査は変えない | 移行がすべて規範差分として現れ、意味を変えていないことを機械的に確かめられない |

### 展開の範囲

各文書の移行は、`spec-diff` が規範差分を出さないことで意味の維持を確かめる。
移行は、まとめると行数が減る箇所に限る。
`Background` は、見出しと空行の費用より繰り返しが長い場合に置く。
`Outline` は、個別の例の合計より表のほうが短く、例の題名が表のいずれかの値と一致する場合に置く。
題名は比較の対象ではないので、表に残らない題名を持つ例はまとめない。
`通常経路` の例は代表例として個別に残す。

## 計画

1. 代表例の重複と独立した故障を列挙する。
2. 前提をまとめる表現を選び、要件、例、テストの対応を比較する。
3. 意味と検出能力を維持できる範囲だけを展開する。

## タスク

- [x] T001 [Inventory] 共通前提と、残す独立した条件と結果を特定する。
- [x] T002 [Tooling] `Background` と `Outline` の行を展開した経路で例を読み、比較する。
  - RED: `mise run test-tools-file -- check/src/gherkin-scenarios.test.ts`、`mise run test-tools-file -- check/src/spec-diff.test.ts`
  - 規則: `docs/formats/specification-format.md` の例の節に、`Rule` 直下の `Background` と展開した経路での比較を足す。
- [x] T003 [Docs] グループ CSV を手で整理し、ほかの付録へ同じ規則で展開する。
- [x] T004 [Verify] 例とテストの対応、規範の意味、読む量を比較する。

## 検証

- 変更前後で要件、例の条件と期待結果、独立した故障の対応を比較する。
- 参照される EX ID と拒否後の無作用、原子性の検証を失わない。
- mise run spec-diff、mise run check-spec、mise run check-links、mise run check-work-items を実行する。
- 対象の例の検査と製品のテストを対応する mise タスクで実行する。

## リスク

短縮のために異なる条件をまとめると、例が規範を補う役割を失う。
独立した条件と結果を先に固定し、共通の説明だけをまとめる。

## 完了

- **完了日**: 2026-10-11
- **要約**:
  `mise run spec-diff` は main に対して規範仕様の差分を報告しない。
  例の検査と `spec-diff` は、`Rule` 直下の `Background` の手順を各例の先頭に足し、`Scenario Outline` の行の値を手順へ埋めた経路で例を読む。
  この比較の上で、受け入れ例の付録 49 文書の共通の前提を `Background` へ、値だけ異なる例を `Scenario Outline` へまとめ、付録の合計を 8,293 行から 7,710 行へ減らした。
  グループ CSV は 429 行から 330 行、OAuth2 承認は 301 行から 213 行、OAuth2 トークンは 427 行から 398 行になった。
  EX ID は 942 件のまま変わらず、テストの参照もそのまま解決する。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/spec-diff.test.ts` の「共通の前提を Background に、値だけ異なる例を Outline にまとめても差分にしない」。
  - **要件**: N/A: 仕様の書き方と検査ツールの変更であり、製品の規範要件を持たない。
  - **観測した失敗**: 同じ経路を `Background` と `Outline` で書き直した文書について、`changedScenarios` が空ではなく `REQ-DEMO-001` を返した。
  - **検出できる理由**: 表現だけを変えた付録が規範差分として現れる誤りと、`Background` の前提を変えても差分に現れない誤りを、同じテストの二つの表明で区別する。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/gherkin-scenarios.test.ts` の「runs a Rule Background before every example of that Rule only」「rejects a Rule Background that holds an action or an outcome」「substitutes the row values into outline steps and keeps only the columns no step names」。
  - **要件**: N/A: 仕様の書き方と検査ツールの変更であり、製品の規範要件を持たない。
  - **観測した失敗**: 例の手順に `Background` の `Given` が含まれず、`Background` の `When` を拒否せず、`Outline` の手順に `<condition>` が残った。
  - **検出できる理由**: 公式の pickle と同じ経路を各例の手順として返すことを、前提、拒否、値の埋め込みの三つで別々に固定する。
- **変更耐性の結果**:
  - 具体例の手順へ `Background` を足す処理を外すと、「runs a Rule Background before every example of that Rule only」が失敗した。
  - `Outline` の行の値を手順へ埋める処理を外すと、値の埋め込みのテストと `spec-diff` の二つのテストが失敗した。
  - `Background` の手順の種別の検査を常に偽にすると、「rejects a Rule Background that holds an action or an outcome」が失敗した。
  - 移行後のグループ CSV で `Outline` の一つのセルを変えると、`mise run spec-diff` が REQ-IDMANAGEMENT-028 を変更として報告した。
- **検証結果**:
  - `mise run test-tools` - 成功
  - `mise run lint-tools` - 成功
  - `mise run spec-diff` - 規範仕様の差分なし
  - `mise run check-spec` - 成功
  - `mise run verify` - 成功
