---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: 製品挙動を変えず、agent skill の本文記述言語を英語へ揃えるだけである。
initial_context:
  source:
    - .agents/skills
  stop_before_reading:
    - backend
    - frontend
    - spec
    - tools
---

# agent skill の本文を英語に統一する

## Motivation

`.agents/skills/**/SKILL.md` は frontmatter の `description` が英語である一方、本文が日本語で書かれて
いる。方式ドキュメント（`DEVELOPMENT.md` / `SPECIFICATION_FORMAT.md` / `WORK_ITEM_FORMAT.md`）と
`README.md` は英語で統一されており、skill だけが混在している。skill は方式ドキュメントと同じく
「手順の正本」であり、記述言語が揺れる理由がない。

なお `AGENTS.md` の規則が日本語を要求しているのは AI コーディングエージェントの応答（状況報告・説明・
質問・要約・最終応答）であり、リポジトリ内の手順ドキュメントの記述言語ではない。

## Scope

- `.agents/skills/*/SKILL.md` 全 7 件の本文を英語に書き換える。
- 手順の意味・順序・強調点は変えない。翻訳であって再設計ではない。
- frontmatter（`name` / `description`）は既に英語のため変更しない。

## Out of Scope

- skill の追加・削除・統合。
- 手順内容の変更。ただし wi-359 以降で追加した規約（着手時の `initial_context`、`just spec-diff`、
  `just spec-where`）が既に反映済みである点はそのまま維持する。
- `work-items/**` や仕様本文の記述言語。

## Design

- 翻訳時に日本語特有の省略（主語や目的語の省略）を英語で復元するため、行数はやや増える。手順の粒度は
  変えない。
- コマンド名・ファイルパス・Skill 名は原文のまま保つ。

## Plan

1. 7 件の SKILL.md を順に書き換える。
2. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [Docs] `commit` / `new-work-item` / `implement-work-item` を英語化する。
- [x] T002 [Docs] `spec-change` / `spec-render` / `update-design` を英語化する。
- [x] T003 [Docs] `parallel-work-items` を英語化する。
- [x] T004 [Verify] `just check` と `just verify` を通す。
- [x] T005 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just check`
- `just verify`

## Risk Notes

翻訳で手順の強制力（「〜しない」「〜まで行わない」）が弱まると、push や baseline 更新のような
取り返しのつかない操作の抑止が効かなくなる。禁止事項は英語でも明示的な否定形で残す。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  `.agents/skills/**/SKILL.md` 全 7 件（commit / new-work-item / implement-work-item / spec-change /
  spec-render / update-design / parallel-work-items）の本文を英語に統一した。手順の順序と粒度は
  変えていない。push 禁止、生成物を commit しない、baseline を通常変更で更新しない、ADR を作らない
  といった禁止事項は、英語でも明示的な否定形（Do not …）で残した。
  `commit` skill の frontmatter `description` にある「コミットして」「これをコミット」は、ユーザーが
  入力する起動フレーズとして一致対象なのでそのまま残した。
- **Verification Results**:
  - `just check` - passed
  - `just verify` - passed
