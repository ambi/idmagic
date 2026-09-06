---
depends_on: []
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-07
priority: p1
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/api-tokens/standards.md, requirement: RFC7662-API-TOKEN-INACTIVE }
---

# 管理コンソールから失効させた API アクセストークンを、`/introspect` が有効と申告する

## Motivation

[[wi-500-back-api-tokens-standards-rows-with-tests]] が `RFC7662-API-TOKEN-INACTIVE` にテストを対応付けようとして見つけた。**この行はいま製品と一致していないので、行を満たすテストが書けない。**

`docs/contexts/api-tokens/standards.md` の `RFC7662-API-TOKEN-INACTIVE` は「未知、失効済み、期限切れ、またはレルム不一致のトークンには `active=false` だけを返す」と宣言している。ところが `DELETE /api/admin/v1/api-tokens/{id}` で失効させたトークンを `/introspect` へ提示すると、次が返る。

```
{"active":true,"scope":"users:read","client_id":"idmagic-api-token","sub":"admin-1",
 "aud":["https://idp.example/realms/default"],"iat":...,"exp":...,"jti":...,
 "token_type":"access_token","delegation_mode":"direct"}
```

`backend/shared/http/server_http/routes.go:305` が作る `apiTokenService` は、`ApiTokenAuthenticator` と `ManagedTokenRevoker` としては配線されているが、`TokenIntrospector` としてはどこにも配線されていない。`/introspect` のハンドラー（`backend/oauth2/handlers_http/token_handler.go:350`）が使うのは `d.OAuth2.TokenIntrospector`、すなわち生の `JWTSigner` である。その検証は署名・発行者・有効期限と、`AccessTokenDenylist` および Agent の revocation epoch だけを見る。管理コンソールの失効は `apitoken/usecases.Service.Revoke` → `repository.Revoke` でライフサイクル記録に `revoked_at` を立てるだけなので、denylist には載らない。

**`apitoken/usecases.Service.IntrospectAccessToken` は、まさにこの重ね合わせのために書かれていて、production のどこからも呼ばれていない。** 「共通 JWT 検証結果に managed-token lifecycle record の active 判定を重ねる」と自身の doc コメントが言っているとおりのものが、配線だけ落ちている。

**影響は文書上の不一致では済まない。** `docs/contexts/api-tokens/internals.md` の「Signature validity is not sufficient」は、署名だけで受け入れる設計では「失効は署名鍵をローテーションするまで効かない」と書いている。リソースサーバーが自前で保護するのではなく `/introspect` に問い合わせる構成では、まさにその状態になる。管理者がコンソールでトークンを失効させても、内省を信頼するリソースサーバーはそのトークンを受け入れ続ける。失効の即時性という機能そのものが、この経路では成立していない。

自前の入口（`support_http.Authenticator`）は `ApiTokenAuthenticator` を通るので正しく拒否する。つまり**同じトークンについて、製品の 2 つの経路が別の答えを出している。**

## Scope

- `/introspect` が管理発行トークンについてライフサイクル記録の判定を通るようにする。
- 判定を持たない検証経路を残さない。`AccessTokenIsRevoked` が `support_http` と `/introspect` の両方で使われるようにしたのと同じ形（REQ-OAUTH2-047）を、管理発行トークンについても採る。
- 4 通り（未知、`/revoke` での失効、管理コンソールでの失効、期限切れ、レルム不一致）が同じ本文になることを、`/introspect` の入口から観測する。
- `apitoken/usecases.Service.IntrospectAccessToken` が production から呼ばれるか、呼ばれないなら消えるかのどちらかにする。
- 消化できた時点で `RFC7662-API-TOKEN-INACTIVE` を `tools/check/standards-coverage-debt.json` から外し、`backend/shared/http/server_http/api_token_standards_test.go` の該当テストに行の id を名指す注記を足す。

## Out of Scope

- 保護リソースにおける DPoP 証明の `htu` の形。[[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- API アクセストークン以外の内省。通常の OAuth アクセストークンとリフレッシュトークンの判定は変えない。

## Risk Notes

- **重ね合わせを `/introspect` だけに足すと、経路がまた 2 本になる。** 判定は 1 か所が持ち、両方の入口がそこを通る形にする。
- **`active=false` に理由を載せない。** RFC 7662 §2.2 は非活性の理由を漏らすことを禁じている。失効と未知が別の本文になってはいけない。

## Verification

- 管理コンソールから失効させたトークンの `/introspect` が `active=false` だけを返す。
- 未知、`/revoke` を通した失効、管理コンソールを通した失効、期限切れ、レルム不一致の 5 通りが、
  同じ 1 つの本文になる。
- `RFC7662-API-TOKEN-INACTIVE` が `tools/check/standards-coverage-debt.json` から消え、
  `backend/shared/http/server_http/api_token_standards_test.go` のテストがその id を名指す。
- `mise run check-spec`
- `mise run verify`
