---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-15
depends_on: [wi-374-specification-overview-exemplars-and-standards-columns]
change_kind: tooling
spec_impact:
  kind: none
  reason: 仕様の意味は変えず、表 3 種のソース表現と検証方法だけを比較評価する。採否は評価結果で決める。
---

# 正準文書の表 3 種を YAML フェンスへ移す案を 1 コンテキストで評価する

## Motivation

正準文書の Markdown ソースは、表のところだけ読めない。ソース最長行は `oauth2` の `glossary.md` で 782 文字、`authorization` で 339 文字、`authentication` で 284 文字ある。実際の閲覧は生成 HTML で行うため、ソースの役割は執筆と検証に寄っている。

そしてこれらの表は既にデータである。`tools/render-spec-docs/src/render.ts:203-234` は正準ヘッダー行 `| From | Event | Guard | To | Effects |` と区切り行の形を検証し、壊れていれば throw する。つまり Markdown は手書きパーサ付きの保存形式として使われている。wi-374 で追加した Standards の値集合検査も、本来 JSON Schema が担う種類の検証を正規表現で書いたものである。

SCL 廃止 (`1b7b2cef`、2026-08-11) を YAML 回帰の否定材料に使うのは正確でない。`wi-355` の Motivation は「手書き SCL は約 1.1 MB、27,409 行で、その約 73% を models/interfaces が占める」であり、主因は形式ではなく TypeSpec が持つべき契約の重複だった。その 73% は TypeSpec へ移って解決済みで、残り 27% を Markdown にした分は `tools/render-spec-docs` の 1,426 行として支払っている。

## Scope

- `spec/contexts/data-keys/` 1 件で、`glossary.md`・`standards.md`・`states.md` の 3 表を ` ```yaml ` フェンスへ移す試作を行う。
- レンダラーと検査の改修量、手書きテーブルパーサを JSON Schema 検証へ置換できるか、ソース最長行の変化、`just spec-diff` の出力の読みやすさを実測する。
- 採否と、採らない場合の再評価条件を Completion に記録する。試作は評価後に撤去する。

## Out of Scope

- **`scenarios.md` の YAML 化**。1 行 1 挙動の行文法で、`- WHEN …` が縦に並ぶことが読みやすさの本体である。ソース最長行も `oauth2` で 324 文字と表より短い。YAML にすると得るものより失うものが大きい。
- **`README.md`、`decisions.md`、`internals.md`**。散文であり、YAML フィールドに Markdown の塊を入れるのは Markdown より悪い。
- **文書の分割**。種類ごとのファイル分割は済んでおり、この評価では触らない。別ファイルの YAML にはせず、その種類を所有する Markdown 内のフェンスに置く。
- **全 Context への適用**。採用が決まってから別 work item で行う。

## Design

- 目的は品質改善ではなく執筆と検証の人間工学である。wi-374 で確認したとおり Glossary と Scenarios の内容品質は既に良く、YAML 化してもそこは変わらない。混同すると評価軸がぶれる。
- フェンスを選ぶのは、種類とファイルの対応とレンダラーの文書走査を保てるからである。フェンス処理は mermaid 用に既に存在する。
- 採用条件を着手前に決める。(1) 手書きのテーブルパーサと `specification-doc.ts` の表検査を JSON Schema 検証へ置き換えられ、正味のコード量が減ること。(2) 生成 HTML の出力が現在と同等であること。(3) ソース最長行が明確に短くなること。(4) `just spec-diff` が現在と同等以上に意味の差分を出せること。1 つでも満たさなければ採らない。
- 進め方は `wi-354` の Cedar 評価に倣う。試作を作り、受け入れ基準で測り、採否と再評価条件を記録して試作を撤去する。

## Plan

1. 3 表の JSON Schema を書く。
2. `data-keys` の 3 表をフェンスへ移し、レンダラーと検査を試作で分岐させる。
3. 受け入れ基準の 4 項目を測る。
4. 採否を決め、試作を撤去して結果を記録する。

## Tasks

- [ ] T001 [Design] 3 表の JSON Schema と受け入れ基準を確定する。
- [ ] T002 [Tooling] `data-keys` で試作し、レンダラーと検査の改修量を測る。
- [ ] T003 [Verify] 生成 HTML、ソース最長行、`just spec-diff` の出力を現行と比較する。
- [ ] T004 [Decision] 採否と再評価条件を記録し、試作を撤去する。

## Verification

- `just check-spec`
- `just spec-render`
- `just spec-diff`
- `just verify-spec`

## Risk Notes

試作がレンダラーと検査の両方に分岐を入れるため、撤去し損ねると 2 つの形式を並行して支えることになる。撤去を T004 の一部として扱い、採らない場合も試作を残さない。

評価対象を 1 コンテキストに絞ると、`oauth2` の Standards 35 節・Glossary 113 行のような規模での挙動が分からない。`data-keys` で採用条件を満たした場合は、全文書へ広げる work item の冒頭で `oauth2` を先に試す。
