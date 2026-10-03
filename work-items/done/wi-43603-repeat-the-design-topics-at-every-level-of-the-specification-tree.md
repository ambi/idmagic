---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-43431-restructure-specification-documents-by-feature-and-design]
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 仕様文書の書き方と、それを検査し描画する開発用の道具と、システムの設計文書の索引だけを変える。製品の利用者が観測する振る舞い、API、設定は変わらない。
  references: []
initial_context:
  specification: [SPECIFICATION_FORMAT.md, work-items/done/wi-43431-restructure-specification-documents-by-feature-and-design.md]
  typespec: []
  source: [tools/check/src/feature-layout.acceptance.test.ts]
  tests: [tools/check/src/feature-layout.acceptance.test.ts]
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "仕様文書の配置と書式を定める規約、検査、生成器と、システムの設計文書の索引だけを変える。既存の REQ と EX の ID、規則文、例のステップ、状態遷移の表、TypeSpec の契約は変えない。旧形式の Context は従来どおり検査に通る。"
---

# 仕様文書の各段で、システムの設計と同じ話題の語彙を繰り返す

## 動機

wi-43431 で、仕様文書を機能仕様と内部設計を軸にした構造へ改めた。
wi-21670 で IdManagement の user を見本として移したところ、次の問題が分かった。

| 問題 | 原因 |
| --- | --- |
| 機能仕様の「操作」の節で、規則が ID 順に並ぶだけになり、整理されて見えない | 操作の節に、規則を置く枠の型がない。一つの H3 に別の操作（管理者による作成とフェデレーションによる作成）が混ざっても、それを見せる一覧がない |
| Context の内部設計に、解決戦略、品質要件の割り当て、リスクと技術的負債を書く場所がない | 内部設計の手本を arc42 の構成要素ビュー、実行時ビュー、横断的概念に限った |
| Context の内部設計（`design/README.md`）の節の名前が、システムの設計（`docs/design/`）の話題の名前と対応しない | 段ごとに別の語彙で構成した。読み手は、段が変わるたびに探す場所を覚え直す |
| Context が他の Context へ約束する内容（注入を受けるポート、発行するイベント）を書く場所がない | Context の README を境界の宣言と索引に限った |

arc42 は、大きなシステムでは共通部分とサブシステムごとの文書に分け、判断、品質要件、解決策を上から下へ伝播させることを推奨する（arc42 FAQ J-1）。
構成要素ビューも、白箱と黒箱を段階的に開く階層構造である（arc42 の 5 章）。
本項目はこの考え方を採る。
ただし、arc42 の章番号と章立ては採らない。
`docs/design/` がすでに話題別（architecture、data、security など）に構成されており、その語彙を全段で繰り返す方が読み手の負担が小さいからである。

## 対象範囲

- `SPECIFICATION_FORMAT.md` を次の構造へ改める。
  - 各段（システム、Context、機能）の文書を、仕様（外へ約束すること）と設計（その実現方法）に分ける。
  - 設計は、`docs/design/` と同じ話題の語彙で構成する。段が下がるごとに、話題の入れ物をディレクトリ、ファイル、H2 の節へ小さくする。
  - 仕様も、段をまたいで同じ話題（目的と範囲、機能、品質、用語、採用標準、受け入れ例）で構成する。
  - arc42 の 1 章から 12 章と話題の対応表を一度だけ置く。
  - 各段の設計の入口に「話題の索引」を置き、全話題について記述した場所か、該当しない理由を書く。
    IEEE 1016 の設計視点の網羅表は、この索引に置き換える。
  - 機能仕様の「操作」の節では、H3 を一つの操作に限り、各 H3 の冒頭に決まった型の要約表（入力、成功時の作用、拒否、冪等性）を置き、規則を正常、状態による分岐、拒否の順に並べる。
  - 機能仕様の H2 の語彙に「品質」を加える。
  - Context の直下に、割り当てを受けた品質要件の `quality.md` を置けるようにする。
  - Context の README に「公開する契約」を置く。
- `DOCUMENTATION_GUIDE.md`、`WORK_ITEM_FORMAT.md`、`.agents/skills/spec-change`、`.agents/skills/update-design` を合わせて改める。
- 検査を次のように改める。
  - Context の直下で `quality.md` を受け入れる。
  - 機能仕様の H2 の語彙と順序に「品質」を加える。
  - 新しい形式の Context の `design/README.md`、機能の `design.md`、`docs/design/README.md` が、話題の索引を全話題について持つことを検査する。
- render-docs が、機能仕様の操作の H3 と、その下の規則の見出しから「操作の一覧」を生成して差し込む。
  規則一覧と同じく、手で書いた一覧は規則を加えたときに更新が漏れても検出されないからである。
- システムの段に、arc42 の内容で欠けている三つの文書を加える。
  - `docs/design/architecture/constraints.md`：技術と組織の制約（arc42 の 2 章）
  - `docs/design/architecture/strategy.md`：解決戦略（arc42 の 4 章）
  - `docs/design/architecture/risks.md`：脅威以外のリスクと技術的負債（arc42 の 11 章）
- `docs/design/README.md` に話題の索引を置く。

## 対象外

- 各 Context の文書の移行。
  IdManagement は wi-21670、Tenancy は wi-39141、残りの Context は wi-26063 が扱う。
- 規則の単位の切り直しと、題名の型の統一。
  規則の ID と題名を変えるので仕様の変更になり、移行と分けて扱う。
- システムの段の既存の設計文書の移動と、章番号の付与。
- `practices/` への汎用部分の切り出し（wi-99632）。

## 設計

### 話題の語彙

話題の集合は次のとおりとし、`SPECIFICATION_FORMAT.md` で一度だけ定める。

| 話題 | システム | Context | 機能 | arc42 の章 |
| --- | --- | --- | --- | --- |
| アーキテクチャ | `design/architecture/` | `design/architecture.md` | `design.md` の H2 | 3 文脈と範囲、4 解決戦略、5 構成要素、6 実行時、7 配置 |
| 制約 | `design/architecture/constraints.md` | 必要な場合だけ `design/architecture.md` の節 | — | 2 |
| 設計判断 | `design/architecture/decisions.md` | `design/decisions.md` | 規則の判断の欄 | 9 |
| アプリケーション | `design/application/` | 必要な場合だけ `design/application.md` | — | 8（API、UI） |
| データ | `design/data/` | `design/data.md` | `design.md` の H2 | 8（永続化） |
| セキュリティ | `design/security/` | `design/security.md` | `design.md` の H2 | 8、11（脅威） |
| 信頼性 | `design/reliability/` | `design/reliability.md` | `design.md` の H2 | 8（エラー処理）、11 |
| 性能 | `design/performance/` | `design/performance.md` | `design.md` の H2 | 10、8 |
| オブザーバビリティ | `design/observability/` | 必要な場合だけ | — | 8 |
| 検証 | `design/verification/` | 必要な場合だけ | — | 10（品質シナリオの確かめ方） |
| インフラストラクチャ | `design/infrastructure/` | — | — | 7 |
| リスク | `design/architecture/risks.md` | `design/risks.md` | `design.md` の H2 | 11 |
| 横断的概念 | 上の各話題 | `design/<concept>.md` | — | 8 |

仕様の側は次のとおりである。

| 話題 | システム | Context | 機能 | arc42 の章 |
| --- | --- | --- | --- | --- |
| 目的と範囲 | `requirements/product-overview.md` | `README.md` の「責務と境界」 | `README.md` の「概要」 | 1、3 |
| 機能 | `requirements/functional.md` | `README.md` の「機能」 | `README.md` の「操作」と規則 | 1（規則の本体は arc42 の外） |
| 公開する契約 | TypeSpec | `README.md` の「公開する契約」 | `README.md` の「操作」 | 3、5（黒箱のインターフェース） |
| 品質 | `requirements/quality.md` | `quality.md` | `README.md` の「品質」 | 10 |
| 用語 | `domain/glossary.md` | `glossary.md` | — | 12 |
| 採用標準 | `domain/standards.md` | `standards.md` | — | arc42 の外 |
| 受け入れ例 | `domain/scenarios.feature.md` | — | `examples.feature.md` | arc42 の外 |

下の段は、上の段からの割り当てと例外だけを書く。
品質要件、判断、横断的概念は上で一度だけ書く。

### 話題の索引

各段の設計の入口に、次の表を置く。

```markdown
| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [アーキテクチャ](architecture.md) |
| オブザーバビリティ | 該当なし：システムのログ設計に従い、この Context に固有の信号はない |
```

検査は、表の行が話題の集合とちょうど一致し、各セルがリンクか「該当なし：」で始まる理由であることを確かめる。
機能の `design.md` は、Context の索引からの差分だけを書くので、「該当なし：」の理由に「Context の設計に従う」と書いてよい。

### 操作の節

```markdown
## 操作

（生成）操作の一覧：操作、規則の数、未決事項の数

### 管理者による作成

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | … |
| 成功時の作用 | … |
| 拒否 | … |
| 冪等性 | … |

#### REQ-… （正常 → 状態による分岐 → 拒否 の順）
```

要約表の行の集合は検査しない。
操作によって冪等性が意味を持たないなど、行の要否が変わるからである。
要約表を置くことと、その順序はレビューで確かめる。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| arc42 の 12 章と章番号を全段の骨格にする | 規則、例、状態遷移、採用標準は arc42 の外にあり、拡張すると章番号の外に足すことになる。`docs/design/` の 35 文書の移動も要る |
| 段ごとに別の語彙で構成する（現行） | 読み手は段が変わるたびに探す場所を覚え直す |
| 操作の一覧を手で書く | 規則を加えたときに更新が漏れても、どの検査にも見つからない |
| IEEE 1016 の網羅表を残し、話題の索引と併用する | 網羅を確かめる仕組みが二つになり、片方だけが更新される |

### 見本

wi-21670 の作業ツリーで、IdManagement の user と Context の `design/` をこの構造で組み直し、render-docs で生成した HTML を利用者に確認してもらう。
見本は wi-21670 の変更であり、本項目には含めない。
構造の誤りを、検査と生成器を作った後に見つけると、直す範囲が大きくなるからである。

## 計画

1. `SPECIFICATION_FORMAT.md` に話題の語彙と索引、操作の節の型を書く。
2. wi-21670 の作業ツリーで見本を組み直し、利用者の確認を受ける。
3. 検査（`quality.md`、「品質」の節、話題の索引）を実装する。
4. render-docs に操作の一覧の生成と、話題の索引の表示を加える。
5. システムの段の三つの文書と、話題の索引を書く。
6. 関連する文書と skills を改める。

## タスク

- [x] T001 [Spec] `SPECIFICATION_FORMAT.md` に話題の語彙、索引、操作の節の型を書く。
  N/A: 製品の規範 ID はない。`mise run check` の表記の検査が「容量」を RED として検出し、「キャパシティ」へ直して GREEN にした。
- [x] T002 [Spec] wi-21670 の作業ツリーで見本を組み直し、利用者の確認を受ける。
  N/A: 製品の規範 ID はない。IdManagement の Context の仕様と設計、user の機能仕様と設計を新しい構造で組み、render-docs で生成した HTML を利用者が確認し、進めることに同意した。
- [x] T003 [App] 検査を改める（`quality.md`、「品質」の節、話題の索引）。
  Acceptance RED：`check/src/feature-layout.acceptance.test.ts` に加えた 6 件（Context の `quality.md`、索引のない設計の入口、欠けた話題と未知の話題、リンクも理由もない話題、システムの設計への適用、「品質」の節）が、実装前に 19 件中 6 件の失敗として現れることを確かめてから実装した。
  実行：`mise run test-tools-file -- check/src/feature-layout.acceptance.test.ts`、`mise run test-tools`、`mise run check`。
- [x] T004 [App] render-docs に操作の一覧を加える。
  Unit RED：`render-docs/src/render.test.ts` の `renders a context in the feature layout with generated rule indexes` に加えた操作の一覧の表明が、実装前に `expect(received).toBeGreaterThan(expected)`（「操作の一覧」が見つからず -1）で失敗することを確かめてから実装した。
  実行：`mise run test-tools-file -- render-docs/src/render.test.ts`。wi-21670 の見本にこの生成器を一時的に当て、ユーザーの機能仕様に 9 つの操作が並ぶことと、Context の `quality.md` がページになることを確かめた。
- [x] T005 [Docs] `docs/design/architecture/` に制約、解決戦略、リスクの文書を加え、`docs/design/README.md` に話題の索引を置く。
  N/A: 製品の規範 ID はない。T003 の話題の索引の検査が `docs/design/README.md` を拒否する状態から、索引を書いて `mise run check` を通した。表記の検査（`documentation-quality.test.ts`）が「版」を RED として検出し、「バージョン」に直した。
- [x] T006 [Docs] `DOCUMENTATION_GUIDE.md`、`WORK_ITEM_FORMAT.md`、関連する skills を改める。
  N/A: 製品の規範 ID はない。`WORK_ITEM_FORMAT.md` には新しい構造に依存する記述がなく、変えていない。`mise run check` が通ることを確かめた。
- [x] T007 [Verify] 検査と生成器のテストと、既存の文書の検査結果が変わらないことを確かめる。
  `mise run test-tools`（807 件）、`mise run typecheck-tools`、`mise run lint-tools`、`mise run check`、`mise run verify` が通る。規範の件数（標準 158、規則 387、例 934）は main と同じであり、一次情報文書はシステムの 3 文書の追加の分だけ 260 から 263 に増えた。

## 検証

- `mise run test-tools`、`mise run typecheck-tools`、`mise run lint-tools`
- `mise run check-spec`、`mise run check`
- 新しい形式の fixture で、話題の索引の欠け、未知の話題、理由のない「該当なし」、Context の `quality.md`、機能仕様の「品質」の節の順序を検査が扱うことを確かめる。
- 旧形式の Context の検査結果（文書、規則、例の件数）が、改定の前後で変わらないことを確かめる。

## リスク

話題の索引の検査を `docs/design/README.md` にも適用するため、システムの段の索引の書き方を誤ると `mise run check` 全体が止まる。
T005 で索引を書いてから、T003 の検査をシステムの段へ有効にする。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff` は、main に対する規範仕様の差分を報告しない。
  仕様文書の各段（システム、Context、機能）で、仕様と設計を分け、設計を `docs/design/` と同じ話題の語彙（アーキテクチャ、設計判断、アプリケーション、データ、セキュリティ、信頼性、性能、オブザーバビリティ、検証、インフラストラクチャ、リスク）で構成する規約を `SPECIFICATION_FORMAT.md` に定めた。
  話題の集合と arc42 の 12 章の対応を一度だけ書き、各段の設計の入口に話題の索引を置くことにした。IEEE 1016 の設計視点の網羅表は廃止した。
  機能仕様の操作の節は、一つの H3 に一つの操作を置き、要約表の後に規則を正常、状態による分岐、拒否の順に並べる型にした。機能仕様の節に「品質」を、Context の直下に割り当てた品質要件の `quality.md` を加えた。
  検査は、話題の索引（全話題、語彙の順、リンクか「該当なし：理由」）を、システムの `docs/design/README.md`、Context の `design/README.md`、機能の `design.md` に課し、Context の `quality.md` と「品質」の節を受け入れる。
  render-docs は、機能仕様の「操作」の見出しの直後に、操作ごとの規則と未決事項の数を並べた操作の一覧を生成する。
  システムの段に、制約、解決戦略、リスクと技術的負債の文書を加え、`docs/design/README.md` の文書表を話題の索引に置き換えた。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/feature-layout.acceptance.test.ts`（`check/src/runner.ts` の `documents` と `specification-rules` を、新しい形式の作業ツリーに対して起動する）
  - **Requirement**: N/A: 開発用の道具と文書の規約の変更であり、製品の規範 ID はない。
  - **Observed Failure**: 実装前に 19 件中 6 件が失敗した。Context の `quality.md` は `not a canonical document` で拒否され、索引のない設計の入口、欠けた話題、未知の話題、リンクも理由もない話題、システムの設計の索引の欠けはいずれも検出されず、「品質」の節は `is not a section of a feature specification` で拒否された。
  - **Detection Reason**: 文書の種別の判定、索引の行の照合、セルの判定、節の語彙のどれかを外すと、対応する表明が失敗する。
- **Unit RED Evidence**:
  - **Test**: `tools/render-docs/src/render.test.ts` の `renders a context in the feature layout with generated rule indexes`
  - **Requirement**: N/A: 開発用の道具の変更であり、製品の規範 ID はない。
  - **Observed Failure**: 実装前に、生成したページに「操作の一覧」がなく、`expect(received).toBeGreaterThan(expected)` が -1 を受け取って失敗した。
  - **Detection Reason**: 一覧の生成、差し込む位置、章の操作の収集、規則の操作への帰属のどれかを外すと、位置または内容の表明が失敗する。
- **Change-Resistance Results**:
  変更は TypeScript の道具であり、`mise run test-go-mutation` の対象外なので、誤実装を手で二つ注入した。
  話題のセルの判定を無効にする誤実装（`validateTopicIndex` の `TOPIC_PLACE` の判定を常に偽にする）は、`rejects a topic that neither links to its design nor says why it does not apply` が検出した。
  操作の一覧を差し込まない誤実装（`insertAfterHeading` がページを変えずに返す）は、`renders a context in the feature layout with generated rule indexes` が検出した。
  どちらも検出を確かめた後に元へ戻した。
- **Verification Results**:
  - `mise run test-tools` - 成功（807 件）
  - `mise run typecheck-tools`、`mise run lint-tools` - 成功
  - `mise run check` - 成功
  - `mise run verify` - 成功
