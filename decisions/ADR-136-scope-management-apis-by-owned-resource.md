---
status: accepted
authors: [tn]
created_at: 2026-07-23
---

# ADR-136: 管理 API scope を所有 resource ごとに分割する

## コンテキスト

Application は OIDC、SAML、WS-Fed の binding を aggregate 内に持つ一方、各 protocol
context は raw client / service provider / relying party を独立した管理 API として公開する。
Provisioning も Application に帰属する connection 操作と tenant 横断 read model を異なる
path prefix で公開している。URL の第一 segment だけで scope を決めると、aggregate の不変条件と
bounded context の所有権のどちらかを崩し、将来の path 変更が権限変更になってしまう。

## 決定

現行の管理 API path は維持し、scope は操作対象 resource の所有権で分割する。

- Application aggregate とその protocol binding、category、assignment、sign-in policy は
  `applications:read` / `applications:write` とする。ただし tenant default sign-in policy は
  tenant 設定なので既存 `settings:read` / `settings:write` とする。
- OAuth2 の raw client、authorization detail type、MCP resource server は、それぞれ
  `oauth-clients:*`、`authorization-detail-types:*`、`mcp-resource-servers:*` とする。
- SAML service provider、WS-Fed relying party / Entra federation、outbound provisioning は
  `saml:*`、`wsfed:*`、`provisioning:*` とする。Provisioning は Application 配下の URL でも
  Provisioning context が所有するため `provisioning:*` を要求する。
- GET は `*:read`、変更操作と action endpoint は `*:write` に対応させる。

この決定は `spec/contexts/api-tokens.yaml` の `models.ApiTokenScope`、各 context の
`authorization.ManagementApiClient` 系 policy、管理 interface の `access.policies` に反映する。

## 却下した代替案

- すべてを `applications:*` に集約する案: raw protocol resource や Provisioning の独立した
  ライフサイクルを隠し、必要以上に広い権限を要求するため採用しない。
- binding も protocol scope に分ける案: Application aggregate の更新を複数 scope に分断し、
  aggregate が守る不変条件と管理 UI の操作単位を一致させられないため採用しない。
- path を全面的に Application 配下へ移す案: 所有権は変わらず、既存 UI・client・contract test
  だけを破壊するため採用しない。
- OAuth2 context 全体を `oauth2:*` にまとめる案: client、RAR schema、MCP resource server の
  運用責務と最小権限を分離できないため採用しない。

## 影響

- API token 発行画面と domain enum に 14 個の scope 値を追加する。
- Application / OAuth2 / Saml / WsFederation / Provisioning は ApiTokens の公開
  `ApiTokenScope` に依存し、session administrator と management API client の双方を SCL 上で許可する。
- 管理 API の PAT 認証、CSRF 除外、監査 actor 帰属、runtime scope enforcement は横断認証
  カーネルで結線する。本 ADR はその実装方式を決めず、各 endpoint が要求する scope を決める。
