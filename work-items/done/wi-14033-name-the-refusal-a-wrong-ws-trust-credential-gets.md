---
status: completed
archived_from: 03d2c0da9cc6610c2c564e50fe33c845c8a50fb4
---

# WS-Trust の資格情報の誤りが実際に受ける拒否を、具体例に書く

`mise run spec-diff` が示す規範の差分は、`EX-WSFEDERATION-004-03` の `Then` が
「AccessDeniedError を返しトークンを発行しない」から「401 で拒否しトークンを発行しない」へ変わったことだけである。
具体例が、契約 `WsTrustIssue` の `WsTrustIssueError401` と製品が実際に返す拒否に一致した。
製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、被覆台帳から外れた。

全文は `git show 03d2c0da9cc6610c2c564e50fe33c845c8a50fb4:<このファイルのパス>` で読む。
