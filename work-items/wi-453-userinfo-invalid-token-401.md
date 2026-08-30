---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
change_kind: bugfix
priority: p2
depends_on: []
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.UserInfo }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.PostUserInfo }
---

# `/userinfo` の `invalid_token` を RFC 6750 のとおり 401 と `WWW-Authenticate` で返す

## Motivation

`wi-386` の総なめで、契約と実装のどちらが正しいかが実装側にある例が 1 件残った。

`/userinfo` は `UserInfoError401` を宣言し、その本体は `invalid_token` である。しかし handler は失敗をすべて `writeOAuthError` に渡し、`writeOAuthError` は `invalid_client` と `server_error` 以外をすべて 400 にする。したがって `/userinfo` に無効な token を出すと 400 が返り、`WWW-Authenticate` も付かない。

RFC 6750 §3.1 は、保護資源が `invalid_token` を返すときは 401 と `WWW-Authenticate` challenge を要求する。契約が正しく、サーバーが規格に従っていない。同じ判断は `PostUserInfo` にも及ぶ。

`wi-386` はこれを検出できていない。`writeOAuthError` はエラー値から応答を決める写像なので、`wi-386` の検査は辿らず、`/userinfo` を「読み残しあり」として数えている。人手で handler を読んで確かめた 1 件である。

## Scope

- `/userinfo` と `/userinfo` (POST) が `invalid_token` を 401 で返し、`support_http.SetBearerChallenge` と同じ形の `WWW-Authenticate` を付ける。
- 同じ判断が及ぶ他の保護資源接点 (Bearer token を検証して `invalid_token` を返す経路) を数え上げ、同じ形に揃える。
- 拒否が「何を残さなかったか」まで見る検査を置く。無効な token で `/userinfo` を叩いたとき、claim が 1 つも漏れていないことを確かめる。

## Out of Scope

- `writeOAuthError` の写像表そのものの作り直し。`/authorize` や `/token` は RFC 6749 §5.2 の側にあり、400 が正しい。接点ごとに規格が違うので、共通関数を一律に変えない。

## Verification

- `mise run check-spec`
- `mise run verify`
- `mise run check-status-drift`

## Risk Notes

400 から 401 への変更は、`/userinfo` の失敗を分岐している既存クライアントを壊しうる。壊れる側は規格に反した実装に依存していたことになるが、変更は 1 接点ずつ行い、監査ログで実際の呼び出し側を確かめてから広げる。
