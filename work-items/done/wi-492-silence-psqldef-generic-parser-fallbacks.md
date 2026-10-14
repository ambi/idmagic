---
status: completed
archived_from: 938471ca31934434bbdc57c2272678ac17ad7bb4
---

# psqldef の汎用パーサーが postgres.sql で代替パーサーへ落ちる箇所を減らす

`mise run spec-diff` は規範の差分なしを報告する。`postgres.sql` の DDL の書き方だけが変わり、列の意味、制約の論理、
外部から観測できる振る舞いはどれも変わっていない。`psqldef` の汎用パーサーが代替へ落ちる箇所は 3 種類から 1 種類
(`UNLOGGED` の 9 表) へ減り、`check-schema` の出力の警告は 6 件から 4 件になった。

全文は `git show 938471ca31934434bbdc57c2272678ac17ad7bb4:<このファイルのパス>` で読む。
