---
status: accepted
authors: [tn]
created_at: 2026-07-23
---

# ADR-137: API access token の wire format を RFC 9068 JWT に統一する

本 ADR は [ADR-135](ADR-135-unify-scim-and-management-api-tokens.md) のうち、識別 prefix と
hash 保存による token representation の決定を置き換える。token モデルの統一と ApiTokens context
の所有権は ADR-135 を維持する。

## コンテキスト

ADR-135 で SCIM と管理 API の credential を単一の ApiTokens context へ統合したが、実装は
`idmagic_pat_` prefix の opaque token、通常の OAuth access token は RFC 9068 JWT となり、
同じ resource server に二つの検証器、introspection 経路、sender constraint、audience、error
mapping が必要になっていた。account scope を OAuth grant と管理発行の両方へ広げると、この
重複は認可差異と標準適用漏れを生む。

一方、管理発行 token は通常 token より長い lifecycle と個別失効を必要とする。wire format の
統一と lifecycle state の統一は別問題であり、形式を分けなくても管理 record の active 確認で
即時失効を実現できる。

## 決定

OAuth grant と管理画面のどちらから発行する access token も、同じ署名鍵・claim profile・検証器を
使う RFC 9068 JWT とする。違いは発行経路、有効期間、refresh 可否、管理 record の有無だけに
限定する。

管理発行 token は JWT 本文を保存せず、`jti`、発行 User、OAuth client、audience、scope、期限、
sender constraint、失効時刻を ApiTokens context が保持する。resource server は署名だけで許可せず、
管理発行 token の `jti` が active record に一致することを fail-closed で検証する。これにより共通
JWT 検証を維持しながら RFC 7009 の即時失効を満たす。

管理発行時には利用者へ OAuth client の事前登録を要求せず、各 realm の Authorization Server が所有する
built-in public client `idmagic-api-token` を `client_id` として JWT claim と管理 record に記録する。
個々の credential と利用者は `jti` / `sub` で識別する。アプリ固有の actor identity が必要な連携は、
専用 client を登録して通常の OAuth grant を使う。RFC 7009 revocation は built-in public client id と token
を提示する要求、または管理者の token 管理 API から行う。

通常の user-bound OAuth grant と管理発行 token は同じ account scope policy を使う。
client_credentials や User subject を持たない token exchange に account scope は発行しない。

## 却下した代替案

- OAuth token を JWT、管理発行 token を opaque のまま維持する: lifecycle の違いを wire format に
  持ち込み、認証・RFC 6750 error・RFC 7662・DPoP・audience enforcement を二重化するため採らない。
- 全 access token を opaque にする: 即時失効は単純になるが、既存の RFC 9068/JWKS による分散検証と
  外部 resource server の offline verification を失い、wi-275 を超える破壊的変更になる。
- 長寿命 JWT を署名と exp だけで検証する: 失効が期限まで反映されず、管理 token の要件を満たさない。
- JWT 本文を DB に保存する: 再取得要件がなく、credential 漏洩面と保存時保護の負担だけを増やす。
- 管理発行のたびに OAuth client 登録を必須にする: 単発の CLI / script 利用まで二段階の設定にし、
  personal API credential の価値を損なうため採らない。

## 影響

- 規範契約は `spec/contexts/api-tokens.yaml` の `standards.RFC9068`、`models.ApiToken`、
  `interfaces.IssueApiToken` / `AuthenticateApiToken` を正とする。
- OAuth 発行契約と account scope の subject guard は `spec/contexts/oauth2.yaml` の
  `interfaces.Token` と account scope scenario を正とする。
- ApiTokens の永続化は token hash から JWT `jti` の lifecycle record へ変わり、失効は物理削除ではなく
  tombstone となる。
- RFC 6750 / 7009 / 7662 / 8414 / 8707 / 9449 / 9700 / 9728 の共通経路を二形式へ重複実装しない。
