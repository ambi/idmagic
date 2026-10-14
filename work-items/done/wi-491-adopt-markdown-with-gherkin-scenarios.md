---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 規範シナリオを Markdown with Gherkin に置き換え、具体例ごとにテストへ追跡する

`mise run spec-diff` は `no normative specification change against main` を報告する。308 件の規則 id、その本文が名指しするエラー型 43 件、旧経路の条件と結果の断片はすべて残り、追加、削除、退役、改名はない。変わったのは正本の形式と追跡の粒度だけである。規範シナリオの正本は 22 ファイルの `scenarios.md` から `scenarios.feature.md` へ移り、306 件の生きた規則が `Rule` に、正常経路と 405 件の `ALT` が 711 件の `Example` と `Examples` 行に分かれ、各例が `EX-<CONTEXT>-<REQ-NNN>-<sequence>` の住所を得た。被覆の宣言集合は規則から例へ移り、テストが名指さない例は理由付き負債に載る。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
