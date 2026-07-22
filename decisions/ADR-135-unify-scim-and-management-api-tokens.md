---
status: accepted
authors: [tn]
created_at: 2026-07-23
---

# ADR-135: SCIM と管理 API のアクセストークンを統一する

## コンテキスト

SCIM 専用 token は tenant と有効期限だけを持ち、SCIM handler 内の認証に閉じていた。管理 API を token で公開する計画では操作単位の scope と複数 bounded context から再利用できる認証境界が必要であり、SCIM 専用モデルを並行維持すると token の生成・hash・失効・管理 UI が重複する。まだリリース前であり、既存 token の互換移行より単一モデルへ収束させる方を優先できる。

## 決定

SCIM 専用 token を独立モデルとして維持せず、SCIM と管理 API が共用する tenant-scoped API access token に統一する。API token の所有権は新しい ApiTokens bounded context に置き、利用側 context は公開された scope 語彙と authenticator port に依存する。SCIM は users / groups と read / write を分けた scope を要求する。

平文 token を再取得可能にする運用は採らず、識別可能な prefix を持つ値を発行時に一度だけ返し、永続化は hash のみにする。未リリースのため旧 SCIM token の migration は用意せず、置換時に既存値を無効化する。

現在の context / module 構成は [ARCHITECTURE.md](../ARCHITECTURE.md) の `ApiTokens`、規範契約は [ApiTokens SCL](../spec/contexts/api-tokens.yaml) の `models.ApiToken` / `models.ApiTokenScope` と `interfaces.AuthenticateApiToken` / `interfaces.IssueApiToken` / `interfaces.ListApiTokens` / `interfaces.RevokeApiToken` を正とする。

## 却下した代替案

- SCIM token と管理 API token を別モデルで維持する: hash 保存・有効期限・発行・失効・管理 UI が重複し、scope 認可の適用漏れを生みやすい。
- 旧 SCIM token に任意 scope を後付けする: SCIM context が横断的な管理 API の scope 語彙を所有することになり、bounded context の責務が逆転する。
- OAuth 2.0 access token だけを SCIM に受け入れる: 外部 SCIM connector が client credential flow を常に利用できるとは限らず、長寿命 credential の管理要件を別途解決できない。
- 旧 token を migration して互換維持する: 未リリース段階では移行複雑性と二重経路のリスクが便益を上回る。

## 影響

- `spec/contexts/api-tokens.yaml` が token、scope、発行・一覧・失効・認証契約を所有する。
- `spec/contexts/scim.yaml` の保護 interface は ApiTokens の SCIM scope を要求し、SCIM 専用 token 管理契約を持たない。
- `scim_tokens` と旧管理 endpoint は破壊的に廃止され、既存 SCIM token は利用できなくなる。
- ApiTokens context と module の現在地は `ARCHITECTURE.md` に同期する。
