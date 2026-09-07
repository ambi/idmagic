# WI-453: Return RFC 6750 invalid_token responses from UserInfo

作業項目は `wi-453-userinfo-invalid-token-401` である。

WI-453 は、GET と POST の `/userinfo` が無効、期限切れ、改ざん済み、または失効済みのアクセストークンを受けたとき、`invalid_token` を HTTP 401 と `WWW-Authenticate: Bearer` challenge で返すようにする。

拒否レスポンスには `sub` その他のユーザークレームを含めない。`/authorize` と `/token` の OAuth 認可サーバー用エラー写像は変更しない。

規範上の条件は [REQ-OAUTH2-020](../../contexts/oauth2/scenarios.feature.md#rule-req-oauth2-020-失効したアクセストークンによる-userinfo-の取得は-invalid_token-で拒否される) と [RFC6750-INVALID-TOKEN](../../contexts/oauth2/standards.md#the-oauth-20-authorization-framework-bearer-token-usage) が定める。
