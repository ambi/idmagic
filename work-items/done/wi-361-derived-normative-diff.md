---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: [wi-360-work-item-reference-integrity-check]
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、git 差分から正規仕様の変更点を導出して表示・検査するだけである。
---

# 正規仕様の差分を git から導出する spec-diff を追加する

## Motivation

ある変更が「規範を何に変えたか」は、現状コード差分と work item の散文 `Completion.Summary` からしか
読めない。レビュー時にもエージェントの自己確認にも、正規シナリオ・状態遷移・API 契約の差分だけを
取り出した表示が無い。

OpenSpec は変更ごとに `## ADDED / MODIFIED / REMOVED Requirements` の delta ファイルを人が書くことで
これを得ているが、正本を二重に持つ運用は本リポジトリが意図的に捨てたものである。同じ情報は git から
導出できるので、書かせずに出す。

## Scope

- `tools/check/src/spec-diff.ts` を追加し、`just spec-diff [ref]`（既定は `main`）で次を出力する。
  - 追加 / 削除 / 本文変更された正規シナリオ ID
  - 変更された状態遷移表の行
  - 追加 / 削除された TypeSpec 宣言・operation
- 出力はプレーンテキスト 1 画面に収める。work item の `Completion` に貼れる粒度にする。

## Out of Scope

- delta ファイル形式の導入。正本は `spec/` のままとする。
- 失敗させるゲートの追加。`spec_impact: none` の宣言と実際の差分の矛盾、`change_kind: feature` で
  差分ゼロ、`affected_spec` 未記載の差分などは検査し得るが、いずれも通常の実装では起きにくく、
  起きても影響が軽い。摩擦だけが残るため入れない。本 work item は表示コマンドに限る。
- 生成 HTML への差分ビュー追加。

## Design

- 新規パッケージは作らず `tools/check` 内の 1 モジュールとして実装する。既存の
  `specification-doc.ts`（シナリオ ID・状態遷移表の解析）と `typespec-catalog.ts`（宣言列挙）を
  そのまま両側のリビジョンに適用して集合差分を取るだけにし、独自の差分表現を導入しない。
- 比較対象のリビジョン内容は `git show <ref>:<path>` で取得する。作業ツリー側は現物を読む。
- `just check` には組み込まない。人とエージェントがレビュー・完了記録の作成時に呼ぶ道具に留める。

## Plan

1. `spec-diff.ts` と単体テストを追加する。
2. `justfile` に `spec-diff` を追加する。
3. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [Tooling] `spec-diff.ts` を実装し、追加・削除・変更を判定する単体テストを追加する。
- [x] T002 [Tooling] `justfile` に `spec-diff` recipe を追加する。
- [x] T003 [Docs] `DEVELOPMENT.md` の完了手順に `just spec-diff` の利用を 1 行で追記する。
- [x] T004 [Verify] `just check` と `just verify` を通す。
- [x] T005 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just test-tools`
- `just spec-diff`
- `just check`
- `just verify`

## Risk Notes

`git show` に依存するため、shallow clone や worktree 差異で比較対象が取得できない場合がある。取得
失敗時は黙って通さず、明示的なエラーにする。差分判定は行単位ではなくシナリオ単位で行い、表記ゆれに
よる誤検知を避ける。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  `just spec-diff [ref]`（既定 `main`）を追加した。git のリビジョンと作業ツリーの双方から正規シナリオ、
  状態遷移行、TypeSpec 宣言を抽出し、追加・削除・変更を集合差分として出す。シナリオ本文は空白と空行を
  正規化して比較するため、整形だけの変更は差分に出ない。operation ごとのトランスポートラッパ
  （`*HttpRequest` / `*HttpResponse` / `*Error<status>` / `*Success_<status>`）は宣言一覧から除外し、
  出力を読める分量に保つ。ゲートは一切追加していない。
  `just spec-diff HEAD~3` で、直近 3 コミットが追加した 3 シナリオと Group 属性まわりの宣言が
  出ることを確認した。
- **Verification Results**:
  - `just test-tools` - passed（93 tests）
  - `just spec-diff` / `just spec-diff HEAD~3` - 期待どおりの出力
  - `just verify` - passed
