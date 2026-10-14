---
status: completed
archived_from: 286146b7bb3687a69ec20a682ee67b9a362d57ac
---

# SAML・WS-Federation・SCIM context を SCL 3.0 に移行する

- SAML、WS-Federation/WS-Trust、SCIM の3 context を SCL 3.0へ意味移行した。旧
`invariants` / `permissions` / `user_experience` と旧 scenario field を廃止し、interface
`requires`/`ensures`、`authorization`、`scenarios` (`actor`/`given`/`main_success`/
`extensions`)、browser/admin navigation `flows` へ再分類した。
- 全30 interface (SAML 6 / WS-Federation 8 / SCIM 16) に public または protected access を明示し、
tenant administrator、UsernameToken subject、SCIM Bearer client と tenant resource の policy を
定義した。SAML/WS-Federation metadata と browser endpoint は public、trust/token 管理、WS-Trust
Issue、SCIM provisioning は tenant-scoped protected access とした。
- OASIS SAML/WS-* requirements の旧 `relates_to` mapを単一 `refs` へ移し、SCIM には RFC 7643/
RFC 7644 の resource、PATCH、Bearer authorization、protocol error requirements を追加した。
SCIM mapping/token identity は tenant_id を含む複合 identity として tenant isolation を明示した。

全文は `git show 286146b7bb3687a69ec20a682ee67b9a362d57ac:<このファイルのパス>` で読む。
