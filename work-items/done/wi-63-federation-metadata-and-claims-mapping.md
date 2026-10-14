---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Federation metadata 公開と claim 発行ルールエンジンを WS-Fed / WS-Trust で共有する

ADR-062 を追加し、AD FS 互換の federation metadata 公開範囲を確定した。
`/{realm}/federationmetadata/2007-06/federationmetadata.xml` は tenant issuer、
WS-Fed passive endpoint、WS-Trust active endpoint、MEX endpoint、federation 署名証明書を
`SecurityTokenServiceType` / `ApplicationServiceType` の RoleDescriptor として広告する。
`/{realm}/trust/mex` は `usernamemixed` endpoint と UsernameToken 前提の policy を広告する。
Claim mapping は既存の宣言的 `ClaimMappingPolicy` / `IssueClaims` を継続利用し、RP 管理 API
から NameID / claim rule を管理できる。未マップ属性は出力せず、required source 欠落は拒否する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
