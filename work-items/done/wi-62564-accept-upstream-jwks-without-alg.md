---
status: completed
---

# 上流の JWK Set で `alg` を省いた RSA 鍵を、ID Token の検証に使う

`OIDC-CORE-ID-TOKEN-KID-OPTIONAL` の署名に使える RSA 鍵の条件に、`alg` の省略または `RS256` を加えた。
外部 OIDC の callback は、`alg` を省いた RSA の署名鍵も RS256 の検証の候補にし、そうした JWK Set だけを公開する上流でもサインインを完了する。
`RS384` など RS256 以外の `alg` を持つ鍵と、小文字の `rs256` は引き続き候補から外す。
空文字列の `alg` は省略と区別せず候補に入れる（未規定）。
ID Token のヘッダーの `alg` は `OIDC-CORE-ID-TOKEN-RS256` のとおり RS256 だけを受け入れるので、`alg` を省いた鍵がほかの方式の検証に使われる経路はない。
