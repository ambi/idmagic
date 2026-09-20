---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: []
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "現在状態の文書の意味を変えず、日本語の述語だけを明確にするため。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - docs/development/specification-first-workflow.md
    - WORK_ITEM_FORMAT.md
  typespec: []
  source: []
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - spec
spec_impact:
  kind: none
  reason: "文書の言い回しを直すだけで、規範要素の意味は変えない。"
---

# 文書の「〜が持つ」「〜が所有する」を、関係を表す動詞に置き換える

## Motivation

`DOCUMENTATION_GUIDE.md` は、英語の have、own、hold を直訳した「〜が持つ」「〜が所有する」で、文書の担当範囲、機能の有無、値の保存場所を言い表さないと定めている。
直訳の動詞では、何と何がどういう関係にあるのかを読み手が推し量らなければならない。
規則を定めた時点で、`docs/`（`releases/` を除く）に 146 か所、`DOCUMENTATION_GUIDE.md` 自身に 72 か所の使用が残っている。

## Scope

- `docs/`、`DOCUMENTATION_GUIDE.md`、ルートの文書の該当箇所を、「〜に書く」「〜で定める」「〜で扱う」「〜がある」「〜に保存する」などへ書き換える。
- 利用者がセッションを持つ、のように所有そのものを表す箇所は変えない。

## Out of Scope

- `docs/releases/` の文書。
- ソースコードのコメント。

## Design

機械的な置換はしない。
同じ「持つ」でも、文書の担当範囲、値の保存場所、機能の有無で置き換える動詞が違うので、箇所ごとに文脈を読んで選ぶ。
用語検査への追加は見送る。所有そのものを表す正しい用法と、直訳の用法を字面で区別できないためである。

## Plan

1. `DOCUMENTATION_GUIDE.md` の規則を三文に縮める。
2. `DOCUMENTATION_GUIDE.md`、ルート文書、`docs/` の順に、実際の所有と保持を除いて関係を表す動詞へ書き換える。
3. 触れた文と隣接文にある、主述関係や語順が不自然な表現も直す。
4. ディレクトリごとに `mise run check-links` と `mise run check-spec` を通す。

## Tasks

- [x] T001 [Docs] `DOCUMENTATION_GUIDE.md` を書き換える。
- [x] T002 [Docs] `docs/` を書き換える。
- [x] T003 [Verify] 変更を検証する。

## Verification

- `mise run check-links`
- `mise run check-spec`
- `mise run verify`

## Evidence Plan

- **Acceptance RED**: `rg -n --glob '*.md' --glob '!docs/releases/**' 'が持つ|所有する' docs README.md SPECIFICATION_FORMAT.md WORK_ITEM_FORMAT.md AGENTS.md` が、文書と内容の関係を曖昧にした用例を報告する。
- **Unit RED**: N/A。実行可能な単体境界はない。代わりに `DOCUMENTATION_GUIDE.md` の規則自体が長い説明と例示を含み、今回採用した三文の規則になっていないことを確認する。

## Risk Notes

規範シナリオの文を書き換えると、`spec-diff` が規範の変更として報告する。
意味を変えていないことを、完了時に `spec-diff` の結果とともに記録する。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `DOCUMENTATION_GUIDE.md` の規則を、具体的な動詞を使うこと、否定では何がないかを明示すること、実際の所有や保持は変えないことの三文に縮めた。
  ルート文書と `docs/` では、文書の担当範囲、規則の定義場所、実装場所、保存場所、機能の有無を表す「持つ」「所有する」を、文脈に応じて「定める」「書く」「扱う」「実装する」「保存する」「設けない」などへ書き換えた。
  変更箇所の周辺にあった不自然な主述関係、重複した表現、曖昧な否定も直した。利用者のロール、トークン、属性、Agent の所有者、外部の取り込み元による管理権など、実際の所有や保持を表す用法は残した。
  `docs/releases/` とソースコードのコメントは変更していない。`mise run spec-diff` は main に対して規範の変更なしと報告した。
- **Acceptance RED Evidence**:
  - **Test**: `rg -n --glob '*.md' --glob '!docs/releases/**' 'が持つ|所有する' docs README.md SPECIFICATION_FORMAT.md WORK_ITEM_FORMAT.md AGENTS.md`
  - **Requirement**: N/A: 文書表現だけの変更で、プロダクトの規範 ID は変更しない
  - **Observed Failure**: 着手時は `docs/` に 146 か所、`DOCUMENTATION_GUIDE.md` に 72 か所あり、文書と内容、実装と機能、構成と値の関係を曖昧にした用例を含んでいた
  - **Detection Reason**: 書き換え後の同じ検索結果を文脈ごとに確認し、規則自身の語と実際の所有・保持を表す用例だけが残ることを確かめた
- **Unit RED Evidence**:
  - **Test**: N/A: 実行可能な単体境界がないため、`DOCUMENTATION_GUIDE.md` の規則を目視で比較した
  - **Requirement**: N/A: 文書表現だけの変更で、プロダクトの規範 ID は変更しない
  - **Observed Failure**: 着手時の規則は、理由と二つの例を含む長い一項目で、今回採用した三文の最小規則ではなかった
  - **Detection Reason**: 完了時の規則が三文だけであり、具体的な動詞、否定、実際の所有・保持の例外をそれぞれ一文で定めることを確認した
- **Change-Resistance Results**:
  `risk: low` の文書変更なので変異試験は行っていない。対象語の検索結果を一件ずつ文脈で分類し、実際の所有・保持を表す用法を残した。`mise run test-tools` は置換時に混入した不採用語「正本」を検出し、「一次情報」へ修正した後に通過した。
- **Verification Results**:
  - `git diff --check` - passed
  - `mise run check-links` - passed（881 文書）
  - `mise run check-terminology` - passed（231 文書）
  - `mise run check-work-items` - passed（627 dependency records）
  - `mise run check-spec` - passed（175 canonical documents、157 standards、316 rules、771 examples）
  - `mise run spec-diff` - no normative specification change against main
  - `mise run lint-go` - passed（0 issues）
  - `mise run verify` - passed
