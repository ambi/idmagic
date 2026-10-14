---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Postgres schema の参照整合性制約を補完する

SCL に tenant-local な参照整合性保証を明記し、Postgres 宣言的 schema に
tenant / user / client / agent / application category / application ordering の
外部キーと enum 的 CHECK を追加した。ApplicationAssignment の user/group 多態 subject と
Application.category_ids / bindings の配列・JSON 参照は、複雑な DB trigger を置かず
既存の use case validation で fail-closed に守る方針とした。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
