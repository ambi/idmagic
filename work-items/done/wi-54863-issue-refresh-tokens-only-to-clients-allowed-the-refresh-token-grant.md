---
status: completed
---

# refresh_token のグラントを許可していないクライアントにリフレッシュトークンを発行しない

REQ-OAUTH2-021 の発行条件を、`refresh_token` のグラントを許可したクライアントの `offline_access` へ変え、REQ-OAUTH2-005 と REQ-OAUTH2-027 の交換の条項をそれに合わせた。
グラントのないクライアントの認可コードと `device_code` の交換は、リフレッシュトークンを返さず、応答とアクセストークンの `scope`、`AccessTokenIssued` のスコープから `offline_access` を除く（EX-OAUTH2-021-03）。
判定は `ExchangeScopes(grants, scopes) (granted, refresh)` に一つだけ置き、二つの交換が共有する。
交換の記録のスコープと、`device_code` の交換でリソース指示子の許可スコープと照合するスコープは、要求したスコープのまま残した。
