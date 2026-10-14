---
status: completed
archived_from: 4eb7dc41959991db0d1496625e328c3cbcecf4d9
---

# IdManagement の機能を `principals` と `common` という抽象的な機能群に入れず、ドメインの語で配置する

`mise run spec-diff` は main に対して規範の変更なしを報告する。
IdManagement の 11 の機能ノードを Context の直下へ移し、`principals`、`common`、`groups`、`bulk-transfer` の機能群を廃止した。
Context の `README.md` は、責務を User、Group、Agent の台帳として書き直し、機能の索引を 11 の機能の表にした。
旧パスは `relocated-spec-paths.json` で新しいパスへ対応させた。
`spec-diff` は規則の本文のリンクをラベルだけで比べるようになり、文書の移動で変わる相対パスを仕様の変更として報告しなくなった。

全文は `git show 4eb7dc41959991db0d1496625e328c3cbcecf4d9:<このファイルのパス>` で読む。
