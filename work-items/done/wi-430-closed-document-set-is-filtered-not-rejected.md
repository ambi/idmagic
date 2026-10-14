---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 正本文書の閉じたファイル集合を、絞り込みではなく拒否として強制する

`mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は動いていない。変わったのは検査の強制力である。`docs/` 直下と `docs/domain/<context>/` 直下に置かれた Markdown のうち、配置が定めない名前を持つものが `mise run check-spec` を落とすようになった。それまでは、そのファイルは検証の対象から外れるだけで、終了コードにも出力にも現れなかった。落とすときは、両辺を小文字にそろえて測った編集距離が 2 以下の許可名を候補として示し、近い名前が無ければその段の許可名を並べる。`SPECIFICATION_FORMAT.md` の宣言も 2 点で実装に合わせた。「file set が checked」は必須ファイルの存在検査が元から無いため偽であり、`file names` に改めた。候補提示についても、無条件に最も近い文書を示すという書き方をやめ、近い名前があるときとないときを書き分けた。判定は `verifyCanonicalDocumentSet` という純関数が持ち、走査は `listCanonicalDirectories` が、終了コードは `check-workspace.ts --documents` が持つ。集める側の `discoverSpecificationDocuments` は同じ走査結果を絞り込むだけで、絞り込みという性格は変わっていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
