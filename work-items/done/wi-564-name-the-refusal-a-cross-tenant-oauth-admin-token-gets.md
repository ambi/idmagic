---
status: completed
archived_from: 6a3d211dd2e645d2951278e2d04f6e0c84d22a28
---

# テナントの一致しない OAuth2 管理 API トークンが実際に受ける拒否を、具体例に書く

`mise run spec-diff` が示す規範の差分は REQ-OAUTH2-003 だけであり、`EX-OAUTH2-003-04` の `Then` が
「操作は `AccessDeniedError` で拒否される」から「操作を 401 の InvalidAccessTokenError で拒否する」へ変わったことである。
[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が SAML 側で下した判断（管理発行トークンの
照合は共通境界 `backend/shared/http/support_http/auth.go` を通り、RFC 7662 の非開示に従って 401 で拒否する）を
OAuth2 admin API へそのまま適用した。TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse`
（401、`InvalidAccessTokenError` を含む union）と、実際に返る拒否が一致した。製品のコードと TypeSpec は変わっていない。
この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
テストの置き場所は、work item が起票時に見込んでいた `backend/shared/http/server_http/oauth2_scope_examples_test.go`
ではなく、`backend/oauth2/handlers_http/oauth2_scope_examples_test.go`（`EX-OAUTH2-003-01`〜`03` が既にある場所）
になった。同じ `oauthAdminRequest` と `stack.OtherRealm` が揃っており、具体例を分割する理由がなかった。

全文は `git show 6a3d211dd2e645d2951278e2d04f6e0c84d22a28:<このファイルのパス>` で読む。
