---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-19
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発方法論と文書形式の表現を日本語へ揃えるだけで、利用者へ知らせる製品変更はない。"
  references: []
initial_context:
  specification:
    - .claude/rules/japanese-writing.md
    - SPECIFICATION_FORMAT.md
    - WORK_ITEM_FORMAT.md
    - docs/development/specification-first-workflow.md
  typespec: []
  source: []
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - spec
    - docs/releases
spec_impact:
  kind: none
  reason: "文章の言語と表見出しの語を揃えるだけで、規範要素の意味も機械検査が読む形も変えない。"
---

# 人が読む文書を日本語で書く規則を、いま外れている文書へ適用する

## Motivation

`.claude/rules/japanese-writing.md` は、リポジトリ内の人が読む文章を日本語で書くと定める。
Markdown はタイトル、見出し、表見出し、リンクラベルを含めて日本語とする、とも定める。
英語で書く対象は API エラーメッセージ、ログメッセージ、CLI ヘルプ、コミットメッセージ、`en` ロケールの UI 文言だけであり、方法論文書は例外に挙がっていない。

規則から外れている散文が 3 件ある。

| 文書 | 行数 | 日本語の比率 | 状態 |
| --- | --- | --- | --- |
| `SPECIFICATION_FORMAT.md` | 409 | 0% | 全体が英語 |
| `docs/development/specification-first-workflow.md` | 470 | 1% | 節見出しだけ日本語で、本文は英語 |
| `WORK_ITEM_FORMAT.md` | 215 | 13% | 前半は日本語、`affected_spec is required for` 以降が英語 |

同じ体系の `DOCUMENTATION_GUIDE.md`（935 行）は日本語で書かれている。
方法論文書だから英語である、という区別は成り立っておらず、書いた順序の違いがそのまま残っている。
`specification-first-workflow.md` のように見出しだけ日本語で本文が英語の形は、読み手が節を選んでから言語が切り替わるため、混在の中でも負担が大きい。

表見出しにも同じ揺れがある。

| 表 | 日本語の見出し | 英語の見出し |
| --- | --- | --- |
| 用語集 | `docs/domain/glossary.md` の 1 件 | Context の用語集 21 件（`Term \| Definition \| Aliases`） |

同じ役割の表が、`docs/domain/` の直下と配下で違う言語の見出しを使っている。

索引表の見出し語も 3 種類に割れている。
`docs/domain/` の Context が `文書 \| 内容`、`docs/design/` と `docs/requirements/` と `docs/design/verification/` が `文書 \| 責務`、`docs/domain/README.md` と `docs/operations/README.md` が `文書 \| 定めるもの` を使う。
どれも「そのディレクトリのファイルを一行ずつ挙げる表」であり、役割は同じである。

## Scope

- `SPECIFICATION_FORMAT.md` の散文を日本語で書き直す。
- `docs/development/specification-first-workflow.md` の散文を日本語で書き直す。
- `WORK_ITEM_FORMAT.md` の英語部分を日本語で書き直す。
- Context の用語集 21 件の表見出しを `用語 \| 定義 \| 別名` へ揃える。
- 索引表の見出し語を 1 種類へ揃える。
- 揃えた表見出しを `SPECIFICATION_FORMAT.md` の該当箇所へ書き、以降の文書が従う形にする。

## Out of Scope

- 機械検査が読む表見出し。`specification-doc.ts` は `| ID | Adoption | Strength | Statement |` と `| From | Event | Guard | To | Effects |` を文字列として突き合わせており、これらは外部契約で定められた名前と同じ扱いにする。
- `standards.md` の `Statement` 列に残る英語。標準仕様の語をそのまま引いている行があり、原表記を保つ判断と日本語化を分けて読む必要がある。別に扱う。
- 生成物の `CONFIGURATION.md` と `ROUTE_PRIORITY.md`。生成元の Go のコードに書かれた文言であり、文書の書き換えでは変わらない。
- `docs/releases/` の既存の記録。書かれた時点の記録として残す。
- 規約文書の重複の解消。[[wi-629-one-source-for-the-document-layout-and-format-rules]] が扱う。本項目は wi-629 が残すと決めた文だけを書き換える。

## Design

翻訳ではなく書き直しとする。
英語の構文をなぞった日本語は、原文より読みにくいものになる。
節ごとに何を定めているかを読み取り、`japanese-tech-writing` スキルの規範に従って日本語の文として組み直す。

原表記を保つ対象を先に決める。
識別子、frontmatter のキー、`REQ-*` と `EX-*`、`risk-based-v3` のような規約の名前、`mise` タスク名、ファイルパス、TypeSpec のシンボル、`Adoption` と `Strength` の値はそのまま残す。
完了記録の `Completed At`、`Acceptance RED Evidence`、`Primary Use Case Evidence` など、機械検査がフィールド名として読むラベルも原表記を保つ。
これらは文章の言語とは別の軸にあり、書き換えると検査と文書の対応が切れる。

索引表の見出しは `文書 \| 内容` を採る。
`責務` は、その文書が負う義務を挙げる表に見えるが、実際に並んでいるのは各文書が扱う対象である。
`定めるもの` は、規約を定めない文書（内部設計や用語集）の行で意味が合わない。
`内容` が 21 件と最も多く、どのディレクトリでも意味が合う。

用語集の見出しは `用語 \| 定義 \| 別名` を採る。
`docs/domain/glossary.md` が既にこの形であり、Published Language と Context の用語集で見出しが揃う。

## Plan

1. 原表記を保つ語の一覧を作り、書き換えの前に固定する。
2. `WORK_ITEM_FORMAT.md` から書き直す。3 件のうち最も短く、規約の対象が明確である。
3. `SPECIFICATION_FORMAT.md`、`docs/development/specification-first-workflow.md` の順に書き直す。
4. 表見出しを一括で揃える。用語集 21 件と索引表は `sd` で置換できるが、置換後に `mise run check-spec` と `check-links` で、見出しに依存する検査が無いことを確かめる。
5. `japanese-writing.md` の例外の一覧に方法論文書が入っていないことを確認し、規則と実態が揃った状態にする。

## Tasks

- [x] T001 [Docs] 原表記を保つ語の一覧を決める。
- [x] T002 [Docs] `WORK_ITEM_FORMAT.md` の英語部分を日本語で書き直す。
- [x] T003 [Docs] `SPECIFICATION_FORMAT.md` を日本語で書き直す。
- [x] T004 [Docs] `docs/development/specification-first-workflow.md` を日本語で書き直す。
- [x] T005 [Docs] 用語集 21 件と索引表の見出しを揃え、`SPECIFICATION_FORMAT.md` へ採った形を書く。
- [x] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-spec`
- `mise run check-links`
- `mise run check-terminology`
- `mise run check-work-items`
- `mise run verify`

## Evidence Plan

- **Acceptance RED**: 対象三文書の行頭に残る英語散文、Context の用語集に残る `| Term | Definition | Aliases |`、索引表に残る `責務` と `定めるもの` の見出しを `rg` で列挙し、対象の英語表現と揺れが存在することを確認する。
- **Unit RED**: N/A。実行可能な単体境界はない。代わりに、対象三文書の冒頭の規則を目視で比較し、同じ文書体系の規則が日本語と英語に分かれていることを確認する。

## Risk Notes

`specification-first-workflow.md` は `.agents/skills/` の各スキルと `AGENTS.md` から節見出しで参照されている。
`check-agent-guidance` はスキルの文字列を突き合わせるため、見出しを変えると落ちる。
節見出しは既に日本語なので変えないことを前提とし、変える必要が出た場合は参照元を同じ変更に含める。

規約文書を書き直すと、その規約に従って書かれた 477 件の完了記録との対応が読みにくくなる恐れがある。
規約の内容は変えないので記録は有効なままであり、`mise run check-work-items` が全件を読むことで確かめる。

表見出しの一括置換は、同じ文字列が別の意味で使われている箇所へ当たる恐れがある。
`| Term | Definition | Aliases |` は行全体で一致させ、`sd -s` の結果を `git diff --stat` で着弾件数が 21 件であることから確かめる。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `WORK_ITEM_FORMAT.md`、`SPECIFICATION_FORMAT.md`、`docs/development/specification-first-workflow.md` の人が読む散文を日本語で書き直し、識別子、パス、機械検査が読む名前は原表記を保った。
  Context の用語集 21 件を `用語 | 定義 | 別名`、索引表 11 か所を `文書 | 内容` に揃え、採用した見出しを `SPECIFICATION_FORMAT.md` に明記した。
  開発方法論の参照は、Extreme Programming を Kent Beck の Tidy First?、Screaming Architecture を Jimmy Bogard の Vertical Slice Architecture に置き換えた。Tidy First? は、小さな整理、整理を先に行うリファクタリング、その反復によるインクリメンタルな設計として記述した。
  `mise run spec-diff` は main に対して規範の変更なしと報告した。
- **Acceptance RED Evidence**:
  - **Test**: `rg --count-matches '^[A-Za-z]' WORK_ITEM_FORMAT.md SPECIFICATION_FORMAT.md docs/development/specification-first-workflow.md` と、表見出しの完全一致検索
  - **Requirement**: N/A: 文書表現と開発方法論の参照だけの変更で、プロダクトの規範 ID は変更しない
  - **Observed Failure**: 着手時は英字から始まる行が順に 57、206、231 行あり、英語の用語集見出しが 21 か所、索引表の不統一な見出しが 11 か所あった
  - **Detection Reason**: 書き直し後は英字から始まる行が順に 17、11、18 行まで減り、残った行が識別子、パス、コード、機械検査が読むフィールドだけであることを確認した。旧表見出しと代表的な英語の冒頭文は完全一致検索で 0 件になった
- **Unit RED Evidence**:
  - **Test**: N/A: 実行可能な単体境界がないため、対象三文書の冒頭の規則を目視で比較した
  - **Requirement**: N/A: 文書表現だけの変更で、プロダクトの規範 ID は変更しない
  - **Observed Failure**: 着手時は同じ文書体系の規則が日本語と英語に分かれ、三文書の本文が英語で始まっていた
  - **Detection Reason**: 完了時は三文書の人が読む本文が日本語で始まり、固定した識別子と機械可読な名前だけが原表記で残ることを確認した
- **Change-Resistance Results**:
  `risk: low` の文書変更なので変異試験は行っていない。完全一致検索で表見出しの置換件数と旧表記の消失を確かめた。`mise run test-tools` は書き直し時に混入した不採用語「独自版」を検出し、表現を修正した後に 572 件すべて通過した。
- **Verification Results**:
  - `git diff --check` - passed
  - `mise run check-spec` - passed（175 canonical documents、157 standards、316 rules、771 examples）
  - `mise run check-links` - passed（881 文書）
  - `mise run check-terminology` - passed（231 文書）
  - `mise run check-agent-guidance` - passed（4 guidance files）
  - `mise run check-work-items` - passed（627 dependency records）
  - `mise run test-tools` - passed（572 tests）
  - `mise run spec-diff` - no normative specification change against main
  - `mise run lint-go` - passed（0 issues）
  - `mise run verify` - passed
