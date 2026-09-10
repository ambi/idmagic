# WI-524: 不完全な `id_token_hint` によるログアウトを拒否する

作業項目は `wi-524-reject-incomplete-logout-id-token-hints` である。

WI-524 は、`/end_session` が `id_token_hint` からログアウト対象を決めるときの条件を締める。これまでは署名と発行者さえ通れば `aud`、`sub`、`sid` が空でも受理し、`sid` が空のときはブラウザー Cookie が示すセッションへ暗黙に降格していた。ヒントが名指ししていないセッションを失効させうる状態だった。

`sub` または `aud` を持たない `id_token_hint` は受理しなくなった。署名が正しくても、その token はどの主体もどのクライアントも指していない。

`sid` を持たない `id_token_hint` も受理しなくなった。`sid` はログアウト対象そのものであり、これが無いヒントは対象を名指ししていない。Cookie のセッションへの降格は、`id_token_hint` を付けなかった要求にだけ残る。

`sid` が示す LoginSession の主体と `sub` が食い違う `id_token_hint` も受理しなくなった。他人のセッションを名指ししたヒントを、そのままログアウト対象にしない。

いずれの拒否も `invalid_request` の 400 であり、LoginSession も同じ `sid` の RefreshTokenRecord も失効しない。

`id_token_hint` の有効期限の扱いは変わっていない。`exp` を過ぎたヒントは、それだけを理由に拒否されない。CIBA が `id_token_hint` から利用者を解決する経路も変わっていない。CIBA は `sid` を持たない ID Token をヒントに使うため、`sid` の必須化はログアウトの経路だけに置いた。

規範上の条件は [REQ-OAUTH2-024](../../contexts/oauth2/scenarios.feature.md#rule-req-oauth2-024-rp-initiated-logout-は-id_token_hint-からセッションとクライアントを特定する) と [OIDC-LOGOUT-ID-TOKEN-HINT](../../contexts/oauth2/standards.md#openid-connect-rp-initiated-logout-10) が定める。
