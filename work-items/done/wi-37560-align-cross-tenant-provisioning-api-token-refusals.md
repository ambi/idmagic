---
status: completed
archived_from: 186c416f6f5d0b136b4f7242a3dafe2539145437
---

# テナントの一致しない Provisioning API トークンが実際に受ける拒否を、具体例に書く

`mise run spec-diff` が示す規範の差分は REQ-PROVISIONING-001 だけであり、`EX-PROVISIONING-001-03` の `Then` が
「操作は `AccessDeniedError` で拒否される」から「操作は 401 の `InvalidAccessTokenError` で拒否される」へ変わったことである。
具体例が、TypeSpec が既に宣言していた `IdMagic.Contract.AuthenticationRequiredResponse` と、
共通認証境界 `backend/shared/http/support_http/auth.go` が実際に返す拒否に一致した。
製品のコードと TypeSpec は変わっていない。この具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
記録は当初 403 へ寄せる bugfix として起票したが、Application、SAML、OAuth2 の既存の判断と RFC 7662 の非開示に合わせて、規範を実装へ寄せる方針へ改めた。

全文は `git show 186c416f6f5d0b136b4f7242a3dafe2539145437:<このファイルのパス>` で読む。
