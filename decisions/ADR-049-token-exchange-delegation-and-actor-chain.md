---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-049: Token Exchange による委譲・代行と actor チェーンを確定する

## コンテキスト

AI エージェントの中核ユースケースは「ユーザーに代わって (on-behalf-of) API やデータへ
アクセスする」ことである。これを安全に表現する標準が OAuth 2.0 Token Exchange (RFC 8693)
で、subject token を別の token に交換し、`act` (actor) / `may_act` claim で「エージェント A が
ユーザー B を代行している」関係と、サブエージェントへ連鎖する委譲チェーンを表現できる。

ここには 3 つの危険がある。

1. **代行関係の喪失**: なりすまし (impersonation) で交換すると、結果トークンは `sub` が
   ユーザーそのものになり「誰が実際に行ったか」の痕跡が消える。
2. **トークンの再利用**: 交換後トークンの audience を絞らないと、あるツール向けの token を
   別ツールへ流用でき、権限が横展開する。
3. **委譲の無限連鎖**: サブエージェントへの再交換に上限と許可制御がないと、意図しない深さ・
   主体へ権限が伝播する。

idmagic は既に DPoP / sender constraint / cnf ([[ADR-005]])、impersonation セッション
イベント (`SessionImpersonationStarted` / `Ended`、[[ADR-041]]) の素地を持つ。本 ADR は
`/token` に token-exchange grant を実装するにあたり、上記 3 点を**保証義務 (fail-closed)** として
確定する。

## 決定

[[wi-50-token-exchange-delegation-actor-chain]] の意思決定を確定する。[[ADR-048]] (エージェント一級プリンシパル)・
[[ADR-005]] (DPoP を既定の sender constraint)・[[ADR-010]] (AuthZEN ポリシー)・
[[ADR-012]] (opaque / JWT access token)・[[ADR-004]] (refresh token rotation)・
[[ADR-018]] (監査) を前提に、AI エージェントがユーザーを代行するときの**委譲モデルと
トークン交換の安全境界**を確定する。本 ADR は RAR ([[wi-51-rich-authorization-requests-agent-scopes]])・
workload federation ([[wi-54-workload-identity-federation-spiffe]])・Cross-App Access
([[wi-57-cross-app-access-identity-assertion-grant]]) すべての交換基盤となる。

1. **token-exchange grant を実装する**。`/token` に `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
   (RFC 8693) を追加する。
2. **既定は delegation、impersonation は明示許可時のみ**。判定漏れは delegation 側
   (痕跡を残す側) に倒す。
3. **`act` チェーンで委譲を表現する**。actor の主体は [[ADR-048]] の `Agent` を担い手とする。
4. **`may_act` とポリシーで委譲を制御する**。いずれも満たさない交換は拒否する (fail-closed)。
5. **Resource Indicators (RFC 8707) で audience を必須限定する**。最小権限のため 1 トークン =
   1 resource を既定とする。
6. **最大委譲深さを設ける**。既定値を本 ADR で定め、`TenantSettings` で override 可とする。
7. **交換後トークンは短命・refresh なしを既定とする**。継続が必要なら再交換させる
   ([[ADR-004]] の rotation とは別経路)。
8. **sender constraint を引き継ぐ**。subject token が DPoP 束縛されている場合 ([[ADR-005]])、
   交換後トークンも要求元の鍵に `cnf` で束縛する。
9. **観測と監査**。`TokenExchanged` / `DelegationChainExtended` / `TokenExchangeRejected` を
   emit する ([[ADR-018]])。新規 permission `TokenGrantTokenExchange`。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 影響

- `/token` に token-exchange 経路が加わり、`AccessTokenClaims` / `IdTokenClaims` に `act` (ネスト可) /
  `may_act` / resource 限定 `aud` が入る。`act` は optional 拡張で後方互換を保つ。
- discovery に `grant_types_supported` の token-exchange と `resource` 対応を反映する。
- AuthZEN ポリシーに交換可否ルールが追加され、[[ADR-010]] の網羅性テストで rule 実装漏れを検知する。
- introspection / userinfo の出力に actor チェーンが現れ、監査・ガバナンス
  ([[wi-59-agent-governance-guardrails-audit-inventory]]) が委譲深さと来歴を読める。
- workload federation ([[wi-54]]) と Cross-App Access ([[wi-57]]) は本 grant を再利用し、
  subject token 種別 (workload attestation / identity assertion) を差し替える形で積む。

## 却下した代替案

- **impersonation を既定にする**: 実装は単純だが `sub` がユーザーに化け、誰が行ったかの痕跡が
  消える。委譲の追跡性 ([[wi-59]]) と失効の的確さ ([[wi-58]]) が損なわれる。delegation を既定とする。
- **独自の delegation claim を定義する**: RFC 8693 の `act` / `may_act` を使わず自前 claim にすると
  相互運用 (resource server / 外部 IdP / MCP) が崩れる。標準の actor チェーンに従う。
- **audience 限定を任意 / scope のみにする**: 交換後トークンが複数 resource に有効だと横展開を招く。
  Resource Indicators (RFC 8707) を必須にし 1 トークン 1 resource に絞る。
- **委譲深さ無制限**: サブエージェント連鎖が意図せず深くなり権限が拡散する。最大深さと
  `may_act` / ポリシー判定で bounded にする。
- **交換後に refresh token を発行する**: 長命化し失効が効きにくくなる。短命 + 再交換とし、
  継続性より失効容易性を優先する。
