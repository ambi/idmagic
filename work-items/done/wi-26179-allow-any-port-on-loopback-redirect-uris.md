---
status: completed
---

# ループバックの IP の `redirect_uri` で、要求時の任意のポートを許す

`standards.md` に RFC 8252 の節を加えた。
`RFC8252-LOOPBACK-PORT` は、`http` でホストが字句 `127.0.0.1` か `[::1]` の登録値について、ポートだけ、またはポートの有無だけが異なる認可要求の `redirect_uri` を受け入れ、示された値へ届ける。
`RFC8252-LOCALHOST-PORT` は `excluded` とし、`localhost` は完全一致のままとした（利用者の確認による）。
`RFC9700-REDIRECT-MATCH` と `CIMD02-REDIRECT-VALIDATE` に、この例外だけを許すことを書き足した。

照合は新しい `AuthorizationRedirectURIAllowed` が担い、`ResolveRedirectURI` を通る `/authorize`、PAR、CIMD のクライアントに効く。
`post_logout_redirect_uri` は RFC 8252 の対象外なので、`RedirectURIAllowed` の完全一致のまま残した。
例外は字句の分割と `url.Parse` の結果の両方でホストとポートを確かめ、userinfo、フラグメント、ホストに続く文字、範囲外や先頭がゼロのポートを拒否する。

確認した範囲は、`/authorize` からトークン交換までの受け入れ試験、拒否側の受け入れ試験、CIMD のユースケース試験、単体試験、`FuzzAuthorizationRedirectURIAllowed` の 60 秒の探索である。
CIMD の試験は、照合を完全一致へ戻す故障注入で失敗することを確かめた。
残るリスクとして、ファジングはループバックの形の入力に偏らないため、字句の分割の誤りは主に表の事例で守っている。
`backend/shared/spec/policy.go` の宣言的規則 `redirect_uri_registered` は完全一致のままだが、`authorize:initiate` を評価する本番の呼び出し元はない。
