---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: []
change_kind: docs
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

1. ディレクトリごとに書き換え、`mise run check-links` と `mise run check-spec` を通す。

## Tasks

- [ ] T001 [Docs] `DOCUMENTATION_GUIDE.md` を書き換える。
- [ ] T002 [Docs] `docs/` を書き換える。
- [ ] T003 [Verify] 変更を検証する。

## Verification

- `mise run check-links`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

規範シナリオの文を書き換えると、`spec-diff` が規範の変更として報告する。
意味を変えていないことを、完了時に `spec-diff` の結果とともに記録する。
