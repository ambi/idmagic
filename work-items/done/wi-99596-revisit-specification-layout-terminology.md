---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: [wi-53794-remove-the-legacy-specification-layout-paths]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 仕様文書の木の用語と付録のファイルの名前を改めるだけで、製品の振る舞い、公開 API、設定、運用手順は変えないので、リリースの読者へ知らせる差分がない。
  references: []
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - DOCUMENTATION_GUIDE.md
    - docs/domain/scenarios.feature.md
  typespec: []
  source:
    - tools/check/src/specification-rules.ts
    - tools/check/src/check-specification-rules.ts
    - tools/check/src/feature-slices.ts
    - tools/check/src/specification-doc.ts
    - tools/check/src/work-item-references.ts
    - tools/workspace/src/document-layout.ts
    - tools/render-docs/src/render.ts
  tests:
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/canonical-document-set.test.ts
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "仕様文書の木の用語とファイルの名前を見直すだけで、要件、例、標準、TypeSpec の内容は変えない。"
---

# 仕様文書の木の用語とファイルの名前を見直す

## 動機

仕様の規約には、読み手に誤解を与えかねない名前が二つある。

一つ目は「機能ノード」である。
`SPECIFICATION_FORMAT.md` は、一つの機能の仕様を置くディレクトリを「機能ノード」と呼び、コードの側の `backend/<context>/<feature>/` を「機能スライス」と呼び分けている。
「ノード」は仕様の木（システム、コンテキスト、機能群、機能）の葉という比喩だが、この repository の外の方法論の用語ではない。
コードと文書の両方を Vertical Slice Architecture の「機能スライス（feature slice）」にそろえたほうが、読み手が仕様と実装を行き来しやすい。

二つ目は `examples.feature.md` である。
新しい形式では、要件は機能仕様の見出しで宣言し、Gherkin のファイルには任意の具体例（`EX-*`）だけを置くので、付録であることを名前で示すために `scenarios.feature.md` から改めた。
しかし `examples` は「サンプルのファイル」の印象を与え、受け入れの例であることが伝わりにくい。
一方で `scenarios.feature.md` に戻すと、システムの階層に残る `docs/domain/scenarios.feature.md`（Context をまたぐ要件を `Rule` として宣言する規範の文書）と同じ名前で役割だけが異なることになる。

## 対象範囲

- 「機能ノード」の言い換えを決め、規約の文書（`SPECIFICATION_FORMAT.md`、`DOCUMENTATION_GUIDE.md`、`docs/development/specification-first-workflow.md`、`docs/development/specification-format-rationale.md`）、検査と生成器の診断と識別子、work item のスキーマ、文書の本文の言及をそろえる。
- 例の付録のファイルの名前を決め、変えるなら、すべての機能の付録、検査、生成器、`generate-spec-examples`、テストの準備データ、文書のリンクをそろえる。
- `docs/domain/scenarios.feature.md` の前書きの、旧形式のファイルへの言及を改める。
- 名前を変える場合は、完了した work item の旧パスを `tools/check/relocated-spec-paths.json` で解決する。

## 対象外

- 要件、例、状態表の内容の変更。
- 機能スライスのないコードの構成を、機能スライスへ再編すること。コードの判断として別に扱う。

## 設計

### 機能ノードと機能スライスの対応

文書のディレクトリとコードの機能スライスは、一致することを既定にする。
現行の規約でも、「すべての機能スライスに機能ノードを置く」は検査済みであり、対応のないスライスの一覧は wi-26063 で空になった。

一致しない場合が二つある。

| 場合 | 例 | 扱い |
| --- | --- | --- |
| コードにスライスがあり、文書のディレクトリがない | なし（wi-26063 で解消した） | 検査で拒否し続ける |
| 文書のディレクトリがあり、コードにスライスがない | 共有の仕組み（IdManagement の `admin-access`、Authentication の `sign-in`）、層構成の Context（Tenancy、Audit、Jobs など） | 許すが、理由を Context の設計の構成要素の表に書く |

後者を禁じない理由は、仕様の粒度は利用者が観測する機能で決まり、コードの粒度は依存と変更の都合で決まるからである。
厳密に一致させると、リファクタリングのたびに仕様のディレクトリと要件 ID の置き場所が動き、work item の参照と旧パスの対応表が増える。

用語の候補は次のとおりである。

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| 文書の側も「機能スライス」と呼ぶ | コードと文書の語が一つになる | スライスのない文書のディレクトリを、同じ語で呼ぶことになる |
| 文書の側を「機能仕様のディレクトリ」または「機能」と呼ぶ | 木の比喩をやめつつ、コードのスライスと区別できる | 語が長くなる |
| 現状の「機能ノード」を保つ | 変更が要らない | 外部の方法論の語と結び付かない |

決定：文書の側も「機能スライス」と呼び、「機能ノード」を廃止する。
コードと文書の語を一つにし、読み手が仕様と実装を行き来しやすくすることを優先する。
スライスのない文書のディレクトリも同じ語で呼び、その理由は上の表のとおり Context の設計の構成要素の表に書く。
検査と生成器の識別子（`featureNode` など）も同じ語へそろえる。

### 例の付録の名前

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| `acceptance.feature.md` | 受け入れの例であることが伝わり、システムの `scenarios.feature.md` とも衝突しない | Gherkin の慣習の名前ではない |
| `scenarios.feature.md` に戻し、システムの階層の文書を改名する | 旧形式からの読み手に馴染みがある | 規範の文書であるシステムの側の名前を変えることになる |
| 現状の `examples.feature.md` を保つ | 変更が要らない。Gherkin の `Example` と `EX-` に合う | サンプルのファイルの印象を与える |

決定：`acceptance.feature.md` にする。
システムの `docs/domain/scenarios.feature.md` は Context をまたぐ要件を宣言する正本であり、機能の付録は機能の仕様本文が宣言した要件へ受け入れの具体例を付けるだけである。
`acceptance` は後者の役割を名前で示し、前者と衝突しない。
あわせて、`docs/domain/scenarios.feature.md` の前書きが旧形式の Context ごとの `scenarios.feature.md` を指している記述を改める。

### 実装で決めたこと

| 場所 | 決定 | 理由 |
| --- | --- | --- |
| `SPECIFICATION_FORMAT.md` の用語 | 機能スライスを「仕様のディレクトリと、対応するコードのディレクトリからなる一つの機能の単位」と定義する。コードのディレクトリはないこともある | 「機能スライス」はコードの側の説明（`コードは機能スライスを持たず` など）で広く使われており、この定義ならその記述をそのまま読める |
| 検査の識別子 | コードの側の `FeatureSlice` 型と `featureSlices` を `CodeSlice` と `codeSlices` へ、`verifyFeatureNodes` を `verifyFeatureSliceSpecifications` へ、`FeatureNodeDeclarations` を `FeatureSliceDeclarations` へ、`FEATURE_NODE_DOCUMENTS` を `FEATURE_SLICE_DOCUMENTS` へ改名する。`feature-nodes.ts` は `feature-slices.ts` へ、`feature-node-debt.json` は `feature-slice-debt.json` へ移す | 仕様の側を機能スライスと呼ぶと、コードの側の既存の名前と衝突する |
| 診断 | `has no feature node under` を `has no feature slice specification under` に、`must be declared in a feature node` を `must be declared in a feature slice` にする | 識別子と同じ語にそろえる |
| `specification-doc.ts` の `documentKind` | 改名前の `examples.feature.md` も付録として種別を返す | `spec-diff` が基準のリビジョンの付録を読めないと、すべての要件が変わったと報告された。作業ツリーに置くことは段の集合の検査が拒否する |
| `render-docs` | 付録のページの名前を `examples.html` から `acceptance.html` にする | ファイル名から導く規則をそのまま保つ |
| `relocated-spec-paths.json` | 項目を加えない | 完了した work item の `affected_spec` は `examples.feature.md` を参照しておらず、解決が要るパスがない |
| `SPECIFICATION_FORMAT.md` の仕様の木 | コードのディレクトリのない機能スライスは、どのコードが実装するかを Context の設計の構成要素の表に書く、と規約に加える | 上の対応の表の扱いを規約へ移す |

## 計画

1. 用語と名前を決め、この記録の設計に決定を書く。
2. 規約の文書を改める。
3. 検査、生成器、スキーマ、テストの準備データを改める。
4. 文書の木のファイルの名前と本文の言及を一括で改め、旧パスの対応を加える。

## タスク

- [x] T001 [Docs] 用語とファイルの名前を決める。
- [x] T002 [Docs] 規約の文書を改める。`spec-change` スキルの付録の名前も改める。
- [x] T003 [Tooling] 検査、生成器、スキーマ、テストの準備データを改める。検査：`mise run test-tools`、`mise run typecheck-tools`。
- [x] T004 [Docs] 文書の木のファイルの名前と言及を改める。84 個の付録を `acceptance.feature.md` へ移した。
- [x] T005 [Verify] 検査と生成器が同じ結果を出すことを確かめる。`mise run generate-spec-examples` は差を出さず、`mise run spec-diff -- main` は規範の差がないと報告した。

## 検証

- `mise run check`
- `mise run test-tools`
- `mise run render-docs`
- `mise run spec-diff` で、規範の変更がないことを確かめる。

## リスク

ファイルの名前を変えると、外部から張られたリンクと、完了した work item の旧パスが切れる。
旧パスは `relocated-spec-paths.json` で解決し、文書の中のリンクは `check-links` で検出して直す。

## 完了
- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。
  「機能ノード」を廃止し、仕様のディレクトリとコードのディレクトリからなる一つの機能の単位を「機能スライス」と定義した。規約の文書、検査と生成器の識別子と診断、work item のスキーマ、コメントをこの語へそろえた。
  例の付録 84 個を `examples.feature.md` から `acceptance.feature.md` へ移し、検査、生成器、`generate-spec-examples`、テストの準備データ、文書のリンク、`spec-change` スキルの言及をそろえた。
  `docs/domain/scenarios.feature.md` の前書きを、Context ごとの要件は機能仕様で宣言するという現在の規則に合わせた。
  完了した work item の `affected_spec` は旧い付録を参照していないので、旧パスの対応表には項目を加えなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run spec-diff -- main`
  - **Requirement**: N/A: 用語とファイルの名前の変更であり、製品の要件を変えない。
  - **Observed Failure**: 付録を改名した直後は、基準のリビジョンの `examples.feature.md` を付録と認識できず、すべての要件を「changed scenarios」と報告した。
  - **Detection Reason**: 実際の履歴と作業ツリーを比べるので、改名が仕様の変更として報告されることを検出する。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/specification-doc.test.ts` の `reads the appendix under its former name examples.feature.md`
  - **Requirement**: N/A: 用語とファイルの名前の変更であり、製品の要件を変えない。
  - **Observed Failure**: `documentKind` から旧名の分岐を外すと、このテストが `examples` を期待して `undefined` を受け取り失敗した。
  - **Detection Reason**: パスだけから種別を決める関数の入出力を確かめる。
- **Change-Resistance Results**:
  旧名の付録を作業ツリーに置くことを許す誤実装は、`rejects the appendix under its former name examples.feature.md` が検出する。
  履歴の旧名を読めなくする誤実装は、手で分岐を外して `reads the appendix under its former name examples.feature.md` が失敗することを確かめた。
  Go の変更はコメント 1 行だけなので、`test-go-mutation` は対象外である。
- **Verification Results**:
  - `mise run test-tools` - 成功（843 件）
  - `mise run typecheck-tools`、`mise run lint-tools`、`mise run check-spec` - 成功
  - `mise run generate-spec-examples` - 差なし
  - `mise run verify` - 成功
