---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 中断された PostgreSQL 接続の解放完了をテストで待つ

`mise run spec-diff` は main に対する規範仕様差分がないと報告した。
期限切れ接続の非同期破棄が完了するまで、対象テストが1秒の上限付きでプール統計を再評価するようになった。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
