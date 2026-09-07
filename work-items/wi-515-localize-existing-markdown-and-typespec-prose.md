---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-08
priority: p2
depends_on: [wi-512-system-wide-top-down-documentation-architecture]
change_kind: docs
spec_impact:
  kind: none
  reason: "既存文書と TypeSpec の説明文を日本語へ移し、プロダクトの振る舞い、契約、規範 ID は変更しない。"
---

# 既存の Markdown と TypeSpec の説明文を日本語へ移行する

## Motivation

`.claude/rules/japanese-writing.md` は、リポジトリ内で人が読む文章を日本語とし、英語を API エラーメッセージ、ログメッセージ、CLI ヘルプ、コミットメッセージ、`en` UI 文言に限定する。
しかし、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md`、生成される設定資料、既存 TypeSpec の doc comment には英語が残っている。
TypeSpec の説明文は OpenAPI と仕様 HTML に流入するため、生成器の固定 UI だけを日本語化しても API リファレンス全体の言語は統一されない。

## Scope

- 既存 Markdown のタイトル、見出し、表見出し、リンクラベル、本文を日本語へ移す。
- TypeSpec の doc comment と `@doc` を日本語へ移し、生成 OpenAPI と仕様 HTML に反映する。
- work item の固定節名を日本語へ移せるよう、形式文書、解析器、検査、既存記録の互換性を整える。
- 生成文書に残る英語を検出する検査を追加し、識別子、規格名、プロトコル名などの許容語を明示する。

## Out of Scope

- API エラーメッセージ、ログメッセージ、CLI ヘルプ、コミットメッセージ、`en` UI 文言の日本語化。
- 識別子、リテラル、パス、コマンド、製品名、プロトコル名、規格名、頭字語、外部契約上の名前の翻訳。
- 規範の意味、API 契約、プロダクトの振る舞いの変更。

## Design

正本を日本語化し、生成 HTML だけを後処理で翻訳しない。
既存 work item は履歴として読み取れる必要があるため、解析器は移行期間中に旧英語節名と新日本語節名の両方を受け入れ、新規作成時は日本語だけを出力する。
規範文の翻訳では ID と強度を固定し、`mise run spec-diff` の差分を意味変更ではなく表記変更としてレビューできる単位に分割する。

## Plan

1. 生成サイトと Markdown、TypeSpec に残る英語の種類と件数を記録し、許容する原表記を分類する。
2. work item の節名と解析器を後方互換に保ったまま日本語へ移行する。
3. 方法論、生成資料、正準文書、TypeSpec の順に正本を日本語化する。
4. 生成 OpenAPI と仕様 HTML を再生成し、許容語以外の英語 UI と説明文が残らないことを検査する。

## Tasks

- [ ] T001 [Design] 許容する原表記と日本語化対象の棚卸しを作る。
- [ ] T002 [Tools] work item の日本語節名を出力し、旧英語節名も読める解析と検査へ更新する。
- [ ] T003 [Docs] 方法論、生成資料、既存 Markdown の文章を日本語へ移す。
- [ ] T004 [Spec] TypeSpec の説明文を日本語へ移し、規範の意味が変わっていないことを確認する。
- [ ] T005 [Tools] 生成 HTML の言語検査を追加する。
- [ ] T006 [Verify] 仕様サイトを再生成し、文書、仕様、リンク、API 互換性、全体検証を通す。

## Verification

- `mise run spec-diff`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run check-links`
- `mise run check-work-items`
- `mise run test-tools`
- `mise run verify`

## Risk Notes

単純な単語置換は、要求とリクエスト、応答とレスポンス、標準用語と一般語を混同し、規範の意味を変えるおそれがある。
規範 ID と TypeSpec シンボルを固定し、文書の所有単位ごとに翻訳して差分を確認する。
英語検出を ASCII の有無だけで判定すると識別子や規格名を誤検出するため、許容理由を分類した検査にする。
