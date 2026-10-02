---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: []
change_kind: tooling
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

検査と生成器は、Context が `legacy-spec-layout.json` に載っているかで形式を切り替える。
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

- [ ] T001 [Spec] `SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`WORK_ITEM_FORMAT.md`、関連 skills を改める。
- [ ] T002 [App] 文書の種別の一覧と、旧形式の切り替えを実装する。
- [ ] T003 [App] 規則の宣言、付録、欄、節の骨格の検査を新形式に対応させる。
- [ ] T004 [App] spec-diff と ID の解決を新形式に対応させ、`relocated-spec-paths.json` を設ける。
- [ ] T005 [App] render-docs に新しいページと生成する一覧を加える。
- [ ] T006 [Verify] 新形式の fixture でテストし、既存の文書がすべて検査に通ることを確かめる。

## 検証

- `mise run test-tools`、`mise run typecheck-tools`、`mise run lint-tools`
- `mise run check-spec`、`mise run check`
- 新形式の fixture で、規則の宣言、付録の参照、欄、節の骨格、判断の章の骨格の検査が、違反を検出することを確かめる。

## リスク

検査の切り替えを誤ると、旧形式の Context の検査が黙って緩む。
既存の文書に対する検査の結果が変わらないことを、改定の前後で比べて確かめる。
