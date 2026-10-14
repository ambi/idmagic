---
status: completed
archived_from: d5c7188b2bbb4edce37928fb4a3f0bae03c778bc
---

# テナントの一致しない API アクセストークンが実際に受ける拒否を、具体例に書く

`mise run spec-diff` が示す規範の差分は REQ-SAML-005 だけであり、`EX-SAML-005-03` の `Then` が
「操作を AccessDeniedError で拒否する」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
具体例が、TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse`（401、`InvalidAccessTokenError` を含む union）と、
共通認証境界 `backend/shared/http/support_http/auth.go` が実際に返す拒否に一致した。
製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
`EX-WSFEDERATION-001-03` は同じ経路の食い違いを台帳へ残したままとし、Out of Scope のとおり個別の記録で扱う。

全文は `git show d5c7188b2bbb4edce37928fb4a3f0bae03c778bc:<このファイルのパス>` で読む。
