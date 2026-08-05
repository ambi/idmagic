---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-062: Federation metadata 公開と claim mapping の所有境界を確定する

## コンテキスト

WS-Federation / WS-Trust の RP や Microsoft Entra domain federation は、IdP の issuer、
passive / active endpoint、MEX endpoint、署名証明書を federation metadata から取得して信頼を
確立する。claim mapping は RP trust ごとの設定であり、metadata は realm 単位の公開情報である。

idmagic では WS-Fed relying party が `ClaimMappingPolicy` を持ち、token 発行時は
claim 発行エンジンに委譲する。metadata 公開はその管理面とは別に、現在存在する endpoint と
federation 署名証明書を広告する派生物として扱う。

## 決定

[[wi-63-federation-metadata-and-claims-mapping]] の metadata 公開スライスの意思決定。
[[ADR-059]] の claim 発行エンジンと [[ADR-060]] の federation 署名証明書を前提に、WS-Federation
passive と WS-Trust active が共有する AD FS 互換 metadata の公開面を確定する。

realm 配下で AD FS 互換の `federationmetadata.xml` を公開し、WsFederation context の wire adapter が
生成する。署名証明書は OAuth/OIDC の JWK 形式を流用せず WS-* 用の X.509 証明書を広告し、鍵用途・
ローテーション・公開重複期間は `SigningKeys` の責務として分離する ([[ADR-064]])。claim mapping は
AD FS claim rule language を採らず、`ClaimMappingPolicy` による宣言的 policy を WS-Fed / WS-Trust /
将来 SAML で共有する。具体的な URL・XML 構造・MEX discovery の仕様は
[`backend/wsfederation/ARCHITECTURE.md`](../backend/wsfederation/ARCHITECTURE.md) に置く。

## 影響

- RP / Entra は realm ごとの metadata URL から issuer、endpoint、署名証明書を取得できる。
- WS-Fed / WS-Trust の token 発行は、claim mapping と metadata 生成を再実装しない。
- 文書署名と複数証明書掲載は未完了であり、鍵ローテーション WI で見直す。

## 却下した代替案

- **OAuth discovery に WS-* metadata を混ぜる。** OIDC と WS-* の trust metadata は形式も消費者も異なる。
- **AD FS claim rule language 互換。** 表現力に対して検証コストが高く、Entra/M365 に必要な claim は宣言的 mapping で満たせる。
- **OAuth JWK を metadata に載せる。** WS-* consumer は X.509 証明書を期待し、鍵用途の境界も異なる。
