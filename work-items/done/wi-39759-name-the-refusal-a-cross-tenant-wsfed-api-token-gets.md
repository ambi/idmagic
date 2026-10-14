---
status: completed
archived_from: 01bf203420b8559f3d6a500eb8f25b80546d485d
---

# テナントの一致しない API アクセストークンが WS-Federation の管理 API で受ける拒否を、具体例に書く

`mise run spec-diff` が示す規範の差分は REQ-WSFEDERATION-001 だけであり、`EX-WSFEDERATION-001-03` の `Then` が
「操作を AccessDeniedError で拒否する」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
`Given` と `When` も `EX-SAML-005-03` と同じ形へ揃え、トークンが発行元テナントでは有効であることと、提示先が別テナントの 4 つの操作であることを明示した。
具体例が、TypeSpec が既に宣言していた 401 の `IdMagic.Contract.AuthenticationRequiredResponse` と、共通認証境界が実際に返す拒否に一致した。
製品のコードと TypeSpec は変わっていない。
この具体例は名指すテストを得て `tools/check/example-coverage-debt.json` から外れ、台帳の `untested` は空になった。

全文は `git show 01bf203420b8559f3d6a500eb8f25b80546d485d:<このファイルのパス>` で読む。
