---
status: completed
archived_from: a66edc485dba53da25b3f951cc78d7d321dab0be
---

# `$ref` の傍らに書かれた keyword を API 互換検査が読み落とす

`mise run spec-diff` は `no normative specification change against main` を返す。変更は
`tools/check/src/api-compat.ts` の `resolve()` だけで、製品にも規範文書にも届かない。
`$ref` を解決するときに、同じスキーマオブジェクトに並んで書かれた keyword を捨てなくなった。
**偽陰性が本体だった。** 見つかる切っ掛けは偽陽性（[[wi-536-group-display-name-source-selects-nothing]]
が `display_name_source` を列挙にしたときの 5 件の誤報）だったが、同じ読み落としは逆向きにも効く。
`$ref` の傍らの `default` が `"a"` から `"b"` へ変わっても報告は**ゼロ件**だった。baseline 側も現行側も
解決先だけを見るので、どちらも `default` を持たないことになり、差が消える。`default` の変更は
この検査器自身が破壊的と宣言している変更であり、生成された OpenAPI には `$ref` の傍らの `default` が
30 件、`description` が 89 件ある。**`$ref` を持つ項目については、宣言した検査が効いていなかった。**
**重ね方は keyword ごとに違う。** `required` は和を取る。上書きにすると、baseline が傍らで 1 つ足していた
だけで解決先の要求が消え、現行の項目が「新しく必須になった」という誤報になる。`properties` は名前ごとに
重ねる。上書きにすると、傍らで書き直していない解決先の項目が「消えた」という誤報になる。**この 2 つは
素朴な `{...resolved, ...beside}` では区別が付かない**ので、それぞれを落とす検査を別に置いた。
**誤報は 1 つの検査で止まらなかった。** `check-work-items` は破壊的変更の有無から
`documentation_impact` の下限を導くので、偽陽性が出ている間は無関係な `in_progress` の記録まで
「documentation_impact none is weaker than inferred upgrade_note」で落ちた。誤報を baseline の更新で
黙らせていたら、この波及ごと残った。

全文は `git show a66edc485dba53da25b3f951cc78d7d321dab0be:<このファイルのパス>` で読む。
