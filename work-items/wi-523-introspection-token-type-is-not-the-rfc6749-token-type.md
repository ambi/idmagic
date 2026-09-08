---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-09-09
priority: p3
change_kind: bugfix
affected_spec:
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.IntrospectionResponse }
---

# 内省の `token_type` が RFC 6749 §5.1 の種別ではなく、トークンの区分を名乗っている

## Motivation

[[wi-518-back-oauth2-token-presentation-standards-rows-with-tests]] が `RFC7662-INTROSPECT` にテストを対応付けているときに見つけた。

`backend/shared/security/tokens_jose/jwt_signer.go:257` は、アクセストークンの内省結果に `TokenType: "access_token"` を置く。`backend/oauth2/token/usecases/introspect_token.go:115` は、リフレッシュトークンの内省結果に `"refresh_token"` を置く。どちらも `/introspect` の応答の `token_type` としてそのまま出る。

RFC 7662 §2.2 は `token_type` を「Type of the token as defined in Section 5.1 of OAuth 2.0」と定めており、RFC 6749 §5.1 が定める値は `Bearer` である。DPoP による送信者制約付きトークンなら RFC 9449 §5 の `DPoP` になる。**製品が返しているのは、この語彙のどれでもない。**

同じ製品の中で `/token` の応答は正しい語彙を使っている。`backend/oauth2/handlers_http/token_handler.go:218` は `Bearer` と `DPoP` を出し分けている。**発行時と内省時で、同じ 1 本のトークンについて `token_type` が違う語彙で語られる。**

RFC 7662 に従うリソースサーバーは、内省の `token_type` を読んでトークンの提示形式を決める。`access_token` はどの提示形式でもないので、そのリソースサーバーは提示形式を決められない。送信者制約付きのトークンでは、`DPoP` を期待する経路が `access_token` を読むことになり、制約を無視する側へ倒れる。

`spec/contexts/oauth2/models.tsp:1019` は `token_type?: string` と宣言するだけで値を制約していないので、どの正典文書もこの語彙を定めていない。

## Scope

- `/introspect` の `token_type` を RFC 6749 §5.1 / RFC 9449 §5 の語彙 (`Bearer`、`DPoP`) に揃える。
- 揃えた語彙を `spec/contexts/oauth2/models.tsp` の `IntrospectionResponse.token_type` に宣言する。
- アクセストークンとリフレッシュトークンの区分を、リソースサーバーが読める形で残すかどうかを決める。RFC 7662 はこの区分を運ぶメンバーを定めていない。
- `backend/shared/http/server_http/token_presentation_standards_test.go` が現状の値を固定している注記を、決めた語彙へ書き換える。

## Out of Scope

- `/token` の応答の `token_type`。既に正しい語彙である。
- `token_type_hint` の語彙。RFC 7009 §2.1 が別に定めており、`access_token` / `refresh_token` で正しい。
- api-tokens コンテキストの内省。同じ関数を通るので影響は受けるが、その行は
  [[wi-500-back-api-tokens-standards-rows-with-tests]] が固定している。

## Risk Notes

- **公開済みの応答の値を変える。** リソースサーバーが `access_token` を読んで分岐していれば壊れる。未リリースなので移行は不要だが、値の意味は取り消せない。
- **区分の情報を落とすと、内省だけではアクセストークンとリフレッシュトークンを見分けられなくなる。** 落とすのか、別のメンバーへ移すのかを Design で決める。

## Verification

- 送信者制約なしのアクセストークンの内省が `Bearer` を返す。
- DPoP 束縛のアクセストークンの内省が `DPoP` を返す。
- 同じ 1 本のトークンについて、`/token` の応答と `/introspect` の応答の `token_type` が一致する。
- `mise run verify`
