---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: [wi-53794-remove-the-legacy-specification-layout-paths]
change_kind: docs
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

着手時に、上の対応の扱いと合わせて一つに決める。

### 例の付録の名前

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| `acceptance.feature.md` | 受け入れの例であることが伝わり、システムの `scenarios.feature.md` とも衝突しない | Gherkin の慣習の名前ではない |
| `scenarios.feature.md` に戻し、システムの階層の文書を改名する | 旧形式からの読み手に馴染みがある | 規範の文書であるシステムの側の名前を変えることになる |
| 現状の `examples.feature.md` を保つ | 変更が要らない。Gherkin の `Example` と `EX-` に合う | サンプルのファイルの印象を与える |

着手時に一つに決める。

## 計画

1. 用語と名前を決め、この記録の設計に決定を書く。
2. 規約の文書を改める。
3. 検査、生成器、スキーマ、テストの準備データを改める。
4. 文書の木のファイルの名前と本文の言及を一括で改め、旧パスの対応を加える。

## タスク

- [ ] T001 [Docs] 用語とファイルの名前を決める。
- [ ] T002 [Docs] 規約の文書を改める。
- [ ] T003 [Tooling] 検査、生成器、スキーマ、テストの準備データを改める。
- [ ] T004 [Docs] 文書の木のファイルの名前と言及を改める。
- [ ] T005 [Verify] 検査と生成器が同じ結果を出すことを確かめる。

## 検証

- `mise run check`
- `mise run test-tools`
- `mise run render-docs`
- `mise run spec-diff` で、規範の変更がないことを確かめる。

## リスク

ファイルの名前を変えると、外部から張られたリンクと、完了した work item の旧パスが切れる。
旧パスは `relocated-spec-paths.json` で解決し、文書の中のリンクは `check-links` で検出して直す。
