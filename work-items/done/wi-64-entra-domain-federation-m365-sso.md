---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Microsoft Entra domain federation で Microsoft 365 サインインと無音 SSO に対応する

Entra domain federation profile (ADR-065) を WS-Federation context の RP preset として
実装した。管理 UI 「Entra ドメインフェデレーション」(/admin/federation/entra) と
`POST /api/admin/wsfed/entra-federation` で、検証済み domain・sourceAnchor 属性・IssuerUri を
受け取り、UPN / ImmutableID / persistent NameID を必須 claim とする WsFedRelyingParty を
upsert する。応答は Entra へ登録する PassiveLogOnUri / ActiveLogOnUri / MetadataExchangeUri と
署名証明書の入手先、Hybrid Join 非対応の既知制約を返す。sourceAnchor は objectGUID の GUID 文字列を
Microsoft byte order で base64 化 (既に base64 ならそのまま) して ImmutableID に正規化し、設定時は
既存 user の欠落・重複・変換不能を fail-closed で拒否、発行時 (passive wi-61 / active wi-62) も
ApplyEntraProfile → IssueClaims で required claim 未充足を fail-closed で拒否する。複数の検証済み
ドメインは domain ごとに issuerUri (既定 urn:idmagic:entra:<domain>) の RP として共存し、サインイン時は
wtrealm で正しい profile へ解決する。README に Update-MgDomainFederationConfiguration への値マッピング・
required claims・無音 SSO 前提 (wi-65)・Hybrid Join 範囲外と回避策を追記した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
