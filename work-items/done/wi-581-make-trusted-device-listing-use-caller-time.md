---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# 信頼済みデバイス一覧の評価時刻を呼び出し側から渡す

`mise run spec-diff` は main に対する規範仕様差分がないと報告した。
`ListActiveByUser` は呼び出し側の評価時刻を受け取り、メモリ実装と PostgreSQL 実装が独自に実時刻を読まない構成になった。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
