# WI-453: Accept 401 for UserInfo invalid_token responses

作業項目は `wi-453-userinfo-invalid-token-401` である。

`/userinfo` の `invalid_token` を HTTP 400 として扱っていた呼び出し側は、HTTP 401 と `WWW-Authenticate: Bearer` challenge を処理するように変更する。

GET と POST の両 binding が同じ応答になる。レスポンス本文は引き続き OAuth エラー本文であり、`error` は `invalid_token` のままである。データ移行や設定変更は不要である。

互換性境界は [REQ-OAUTH2-020](../../contexts/oauth2/scenarios.feature.md#rule-req-oauth2-020-失効したアクセストークンによる-userinfo-の取得は-invalid_token-で拒否される) と [RFC6750-INVALID-TOKEN](../../contexts/oauth2/standards.md#the-oauth-20-authorization-framework-bearer-token-usage) が定める。
