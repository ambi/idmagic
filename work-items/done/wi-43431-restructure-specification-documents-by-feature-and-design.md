---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 仕様文書の書き方と、それを検査し描画する開発用の道具だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification: [SPECIFICATION_FORMAT.md]
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/workspace/src/workspace.ts
    - tools/check/src/specification-doc.ts
    - tools/check/src/gherkin-scenarios.ts
    - tools/check/src/specification-rules.ts
    - tools/check/src/check-specification-rules.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/canonical-document-set.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/work-item-references.ts
    - tools/brief/src/brief.ts
    - tools/spec-route/src/main.ts
    - tools/render-docs/src/main.ts
    - tools/render-docs/src/render.ts
  tests: [tools/check/src, tools/workspace/src, tools/render-docs/src]
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "仕様文書の配置と書式を定める規約、検査、生成器だけを変える。既存の REQ と EX の ID、規則文、例のステップ、状態遷移の表、TypeSpec の契約は変えない。旧形式の Context は従来どおり検査に通る。"
---

# 仕様文書を、機能仕様と内部設計を軸にした多層の構造へ改める

## 動機

wi-96676 で IdManagement の暗黙の仕様を書き起こし、生成した HTML を確認したところ、仕様書としても設計書としても読みにくいことが分かった。
Tenancy も同じ状態である。

| 文書 | 問題 |
| --- | --- |
| `decisions.md` | 判断の箇条書きが延々と続く。どの規則や設計を正当化する判断なのかが分からない。ADR として読むには、背景と代替案が欠けている |
| `internals.md` | 一部の機構しか書かれておらず、Context の内部設計を網羅しているとは読めない。長い段落が続く |
| `scenarios.feature.md` | 仕様が Example の形でしか書かれていない。規則が多くなると全体を把握できない |
| Context のルート | 状態遷移、判断、内部設計、シナリオが、どの機能ノードにも入らなかった記述の寄せ集めになっている |

原因は個々の文書の書き方ではなく、`SPECIFICATION_FORMAT.md` が定め、検査が強制している構造にある。
この構造は、ノードごとに文書をファイル種別（状態遷移、判断、仕組み、シナリオ）で分割する。
そのため、一つの機能を理解するには四つのファイルを行き来する必要がある。
一つの機能に収まらない記述は、種別ごとにルートへ溜まる。

## 対象範囲

- `SPECIFICATION_FORMAT.md` を新しい構造へ改める。
  - 機能仕様の手本は SCIM（RFC 7643、RFC 7644）とする。
  - 内部設計の手本は arc42 の構成要素ビュー、実行時ビュー、横断的概念とする。
  - IEEE 1016 の設計視点は、内部設計の網羅を確かめる表に使う。
- `DOCUMENTATION_GUIDE.md`、`WORK_ITEM_FORMAT.md`、`.agents/skills/spec-change`、`.agents/skills/update-design` を、新しい構造に合わせて改める。
- 検査と生成器が、新旧の二つの形式を Context ごとに切り替えて扱えるようにする。
- 旧形式のまま残る Context を `tools/check/legacy-spec-layout.json` に列挙する。
  この一覧は減る方向にしか変えない。

## 対象外

- 各 Context の文書の移行。
  IdManagement は wi-21670、Tenancy は wi-39141、残りの Context は wi-26063 が扱う。
- REQ と EX の ID の付け替え。
- `practices/` への汎用部分の切り出し（wi-99632）。
  本項目は `SPECIFICATION_FORMAT.md` を書き換えるので、wi-99632 は本項目の後に、新しい内容を前提として分類する。

## 設計

### 文書の木

木は、システム、Context、機能群、機能の 4 層にする。
長大になった機能は、機能ノードの中で章ページへ分ける。

```text
docs/domain/<context>/
  README.md              目的、境界、モデルの全体図、機能地図
  glossary.md
  standards.md
  design/                Context の内部設計
    README.md            構成要素、実行時の流れ、データと永続化、横断的概念の索引、障害と復旧、設計視点の網羅表
    decisions.md         重要な設計判断
    <concept>.md         横断的概念
  <group>/               機能群
    README.md            境界と機能の索引
    <feature>/           機能ノード
      README.md          機能仕様
      <chapter>.md       機能仕様の章
      design.md          機能の内部設計
      examples.feature.md  例の付録
```

機能仕様（機能ノードの `README.md`）は、次の H2 をこの順に置く。
該当する内容のない節は置かない。

| 節 | 内容 |
| --- | --- |
| 概要 | 機能の責務、境界、行為者 |
| モデル | 扱う Aggregate と項目、保つ性質 |
| 状態遷移 | `### <機械名>` の下に、現行の状態の表と遷移の表を置く |
| 操作 | 操作ごとに H3 を置き、その下に規則を H4 で置く |
| エラー | 操作をまたいで共通するエラーの意味 |
| セキュリティ上の考慮 | 認可、テナント境界、情報の開示 |

章ページは、操作の節を分けたものである。
章ページにも H4 の規則を置ける。

### 規則と例

規則は仕様本文で宣言する。

```markdown
#### REQ-IDMANAGEMENT-042 管理者が作成する User のユーザー名とパスワード

- （規則文。一行に一つの義務）
- **判断**：（理由。重要な判断なら design/decisions.md の見出しへリンクする）
- **担保手段**：`usecases.CreateUser`
- **例**：EX-IDMANAGEMENT-042-01
- **要判断**：…
```

例は、同じ機能ノードの `examples.feature.md` に Gherkin で書く。
付録の `## Rule: REQ-… 題名` は、仕様本文で宣言した規則を参照する見出しであり、宣言ではない。
有効な規則には、付録の例が一つ以上必要である。
テストから引く対象と `//spec:covers` の形は変えない。

現行の欄 **理由** は **判断** に改める。
**判断** には理由そのものを書く。
代替案を比べた重要な判断の場合は、その判断の見出しへリンクする。

### 設計判断

判断の大半は、それが正当化する規則、モデル、設計要素の注記にする。
独立した記録は、代替案、成立条件、再検討の条件が必要な少数の判断に限る。
これらの判断は `design/decisions.md` に置く。

`design/decisions.md` では、判断ごとに次の H3 を置く。

1. 背景
2. 決定
3. 検討した代替案（案、利点、欠点、採らない理由の表）
4. 結果と再検討の条件
5. 関連する規則

arc42 は第 9 章で、重要な判断だけをアーキテクチャ全体で一か所にまとめる。
SCIM の RFC、WHATWG の標準、IEEE 1016 は、理由を該当する節や設計要素へ添える。
Context ごとに判断のディレクトリを置く手本はない。
この構造は、二つの方式を合わせたものである。

### 内部設計の網羅

`design/README.md` の末尾には、IEEE 1016 の設計視点（文脈、構成、論理、依存、情報、インターフェース、相互作用、状態の動態、アルゴリズム、資源）を行とする表を置く。
各行には、その視点を記述した場所を書くか、「該当なし」とその理由を書く。
この表は、書き漏らした視点を読み手と書き手の両方に見せるためのものである。

現行の規約は、コードから復元できる内容を内部設計に書くことを禁じている。
この禁止は次のように改める。
責務単位の構成表（機能スライスと層ごとの責務、公開するポート、依存先）は書いてよい。
ファイルとパッケージの列挙は、引き続き書かない。

### 生成する一覧

機能仕様の「規則一覧」と「未決事項」、Context の README の「機能地図」は、手で保守しない。
render-docs が規則の見出しと **要判断** の欄から生成して、ページへ差し込む。
手で書いた一覧は、規則を追加したときに更新が漏れても、どの検査にも検出されない。

### 移行中の共存

Context が新しい形式かどうかは、`docs/domain/<context>/design/README.md` があるかで決める。
道具の多くは、履歴のリビジョンや一つのファイルだけを読む。
そうした道具でも、設定ファイルを読まずに同じ判定ができるようにするためである。
`legacy-spec-layout.json` は、この判定に対する歯止めとして使う。
`design/README.md` を持たない Context は、この一覧に載っていなければならない。
`design/README.md` を持つ Context は、この一覧から外さなければならない。
どちらも `mise run check` で検査する。

新しい形式の Context では、ファイルを次のように判定する。

| 置き場所 | 置けるファイル | 種別 |
| --- | --- | --- |
| Context の直下 | `README.md`、`glossary.md`、`standards.md` | 現行どおり |
| `design/` | `README.md`、`decisions.md`、任意の名前の横断的概念 | 内部設計。`decisions.md` だけは判断の骨格を検査する |
| 機能群（子のディレクトリを持つ段） | `README.md` | 境界と索引。規則を宣言できない |
| 機能ノード（Context から 1 段か 2 段下の、子のディレクトリを持たない段） | `README.md`、`design.md`、`examples.feature.md`、任意の名前の章 | `README.md` と章は機能仕様、`examples.feature.md` は付録 |

旧形式の名前（`states.md`、`decisions.md`、`internals.md`、`scenarios.feature.md`）は、新しい形式の機能ノードと機能群には置けない。

機能仕様の検査は次のとおりである。

- 規則の宣言は `### REQ-…` または `#### REQ-…` の見出しとする。
  廃止した規則は、見出しの末尾に `(superseded by REQ-…)` を付ける。
- 有効な規則には **担保手段** の欄が必要である。
  **理由** の欄は使えず、**判断** を使う。
- 機能ノードの `README.md` の H2 は、設計の節の表の語彙をその順で使い、同じ節を二度置かない。
- 付録の `## Rule:` の題名は、仕様本文の見出しの題名と一致しなければならない。

spec-diff は、新しい形式の規則の事実を `'spec'`、題名、付録の例のステップ、`--`、本文の順に並べて作る。
旧形式の `'gherkin'` とは形式の印が異なるので、移行のコミットでは題名だけを比べる。
移行で題名を変えなければ、規則は変更として報告されない。
spec-diff は新形式の規則に別の形式の印 `'spec'` を与える。
形式の印が異なる二つの版では、題名だけを比べる既存の分岐を使う。
これにより、移行のコミットで規則が変更や削除として報告されないようにする。

完了済みの work item の `affected_spec.path` は書き換えない。
新設する `tools/check/relocated-spec-paths.json` で、旧パスを新パスへ読み替える。
未完了の work item のパスは、移行の際に書き換える。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| IdManagement と Tenancy だけを検査の対象から外す | 二つの Context だけが無検査になり、規則の ID の重複や被覆の欠落を検出できなくなる |
| `scenarios.feature.md` を規則の置き場として残し、節立てだけを改める | 仕様を例の形で読む構造が残り、問題の一つを解かない |
| 例を廃止し、テストを例の代わりにする | 規則が意図する代表例と境界を、テストを読まずに確かめられなくなる |
| 判断をすべて ADR の形の独立文書にする | 判断と、それが正当化する規則が離れる。手本のどれにもない形である |

## 計画

1. `SPECIFICATION_FORMAT.md` と関連文書を改める。
2. `tools/workspace/src/document-layout.ts` に新しい種別と、旧形式の一覧による切り替えを入れる。
3. 規則の宣言と付録の解析を改める（`specification-doc.ts`、`gherkin-scenarios.ts`、`specification-rules.ts`、`check-specification-rules.ts`）。
4. spec-diff、work item の参照の解決、`brief`、`spec-route`、`spec-review-candidates`、`security-controls` を新形式に対応させる。
5. render-docs に新しいページの種類と、生成する一覧を加える。
6. 旧形式の一覧に全 Context を載せ、既存の文書がすべて従来どおり検査に通ることを確かめる。

## タスク

- [x] T001 [Spec] `SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`WORK_ITEM_FORMAT.md`、関連 skills を改める。
  N/A: 製品の規範 ID はない。配置図の整合検査（`document-layout-format.test.ts` の新しい段の表明）と `mise run check` のリンク検査が、旧いアンカーへのリンクを RED として検出した。
- [x] T002 [App] 文書の種別の一覧と、旧形式の切り替えを実装する。
- [x] T003 [App] 規則の宣言、付録、欄、節の骨格の検査を新形式に対応させる。
  Acceptance RED：`check/src/feature-layout.acceptance.test.ts` の 13 件中 12 件が失敗することを確かめてから実装した。
- [x] T004 [App] spec-diff と ID の解決を新形式に対応させ、`relocated-spec-paths.json` を設ける。
  `spec-diff.test.ts` に、旧形式から機能仕様と付録へ題名を変えずに移すと規則の差分が出ず、本文か例を変えると差分が出ることを固定した。
- [x] T005 [App] render-docs に新しいページと生成する一覧を加える。
  Unit RED：`render.test.ts` の新しい形式の描画テストが失敗することを確かめてから実装した。一時的な Context で `render-docs/src/main.ts` が新しい形式の文書を集めることも確かめ、その Context は削除した。
- [x] T006 [Verify] 新形式の fixture でテストし、既存の文書がすべて検査に通ることを確かめる。
  `mise run test-tools`（801 件）、`mise run typecheck-tools`、`mise run lint-tools`、`mise run check`、`mise run verify` が通る。既存の Context の検査結果（260 件の文書、387 件の規則、934 件の例）は改定の前後で変わらない。

## 検証

- `mise run test-tools`、`mise run typecheck-tools`、`mise run lint-tools`
- `mise run check-spec`、`mise run check`
- 新形式の fixture で、規則の宣言、付録の参照、欄、節の骨格、判断の章の骨格の検査が、違反を検出することを確かめる。

## リスク

検査の切り替えを誤ると、旧形式の Context の検査が黙って緩む。
既存の文書に対する検査の結果が変わらないことを、改定の前後で比べて確かめる。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff` は、規範仕様の差分を報告しない。
  仕様文書の構造を、機能仕様と内部設計を軸にした形式へ改める規約と道具を整えた。
  新しい形式の Context は `design/README.md` の有無で判定し、旧形式の Context は `tools/check/legacy-spec-layout.json` に列挙した（現時点は全 21 件）。
  検査は、機能仕様の `#### REQ-…` 見出しを規則の宣言として読み、付録 `examples.feature.md` の例と照合し、担保手段の欄を必須とし、理由の欄を判断の欄へ改め、機能仕様の節の順序と `design/decisions.md` の判断の骨格を確かめる。
  spec-diff、work item の参照の解決、brief、spec-route、セキュリティ統制の検査、render-docs も新しい形式を読む。
  render-docs は、機能仕様に規則一覧と未決事項を、Context の README に機能地図を生成して差し込む。
  完了した work item の旧パスは `tools/check/relocated-spec-paths.json` で読み替える。現時点では空である。
  セキュリティ統制の検査が固定のパスで読む `api-tokens/scenarios.feature.md` の扱いは、wi-26063 の対象範囲に加えた。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/feature-layout.acceptance.test.ts`（`check/src/runner.ts` の `documents` と `specification-rules` を、新しい形式の作業ツリーに対して起動する）
  - **Requirement**: N/A: 開発用の道具の変更であり、製品の規範 ID はない。
  - **Observed Failure**: 実装前に 13 件中 12 件が失敗した。新しい形式の作業ツリーは `docs/domain/demo/design/README.md: not a canonical specification document` で拒否され、付録のない規則、宣言のない付録の参照、題名の不一致、機能群での宣言、判断の記録の欠けた節、旧形式の一覧との食い違い、担保手段の欠落、理由の欄、節の順序のいずれも検出されなかった。
  - **Detection Reason**: 文書の種別、ファイルをまたぐ照合、欄と節の検査のどれかを外すと、対応する表明が失敗する。
- **Unit RED Evidence**:
  - **Test**: `tools/render-docs/src/render.test.ts` の `renders a context in the feature layout with generated rule indexes`、`tools/check/src/spec-diff.test.ts` の `reports nothing when a rule moves into a feature specification with an appendix`
  - **Requirement**: N/A: 開発用の道具の変更であり、製品の規範 ID はない。
  - **Observed Failure**: render のテストは、実装前に `domain/demo/design/index.html` が生成されないため `expect(received).toBeDefined()` で失敗した。spec-diff のテストは実装と同時に書いたため、RED を観測していない。
  - **Detection Reason**: 段の割り当て、入れ子の案内、生成する一覧、状態図、付録の装飾、トレーサビリティのどれかを外すと render のテストが失敗する。spec-diff のテストは、本文と例のステップのどちらの変更も差分として報告することを固定する。
- **Change-Resistance Results**:
  変更は TypeScript の道具であり、`mise run test-go-mutation` の対象外なので、誤実装を手で二つ注入した。
  付録の例のない規則を見逃す誤実装（`feature-nodes.ts` の例の有無の判定を無効にする）は、`rejects a rule that has no example in the appendix` が検出した。
  新しい形式の規則に旧形式と同じ形式の印を与える誤実装（`spec-diff.ts` の `'spec'` を `'gherkin'` にする）は、移行で本文が比べられて変更と報告されるため、`reports nothing when a rule moves into a feature specification with an appendix` が検出した。
  どちらも検出を確かめた後に元へ戻した。
