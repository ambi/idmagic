---
status: completed
---

# Cross-App Access (Identity Assertion JWT Authorization Grant) でエージェントのアプリ間アクセスを仲介する

`draft-ietf-oauth-identity-assertion-authz-grant-05` に固定して、ID-JAG の発行（REQ-OAUTH2-066〜068）、償還（REQ-OAUTH2-069〜071）、許可の管理（REQ-OAUTH2-072）を追加した。`jwt-bearer` と `jwt-dpop` のグラント、`IdJagSigning` の RS256 の鍵（REQ-SIGNINGKEYS-013）、Authorization Server Metadata の広告、管理 API 6 操作と管理コンソールを含む。ID Token には積極的な判定のため `token_use: id_token` を載せる。

旧計画から方針を変えた。`-05` が処理を定めない `actor_token` と `act` は両ホップで拒否し、許可は Application ではなく OAuth2 が所有し、受け入れ側の信頼は SSO の `IdentityProviderConnection` と `FederatedIdentity` に載せた。管理 API の API トークンのスコープは `mcp-resource-servers:*` を流用した。

確認の限界: 2 テナントの E2E はループバックを拒否する JWKS の HTTP 取得だけを Echo の読み取りへ差し替えており、実際の `jwks_uri` の取得は通していない。外部の製品との相互運用は未確認で、draft である以上主張しない。mTLS の束縛はユースケースの表で確かめ、E2E は DPoP だけを通した。残るリスクは draft の改訂（特に個人 draft に依存する `jwt-dpop`）と、許可の削除の後も発行済みの ID-JAG が最長 300 秒、アクセストークンが最長 600 秒有効なことである。SAML とリフレッシュトークンの `subject_token`、JIT、ステップアップは対象外とした。
