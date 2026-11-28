---
status: completed
---

# 動的クライアント登録で、登録したメタデータと client_secret の有効期限を返す

REQ-OAUTH2-016 に二つの箇条を足し、動的クライアント登録の応答が保存した `client_name` と `require_pkce`、`client_id_issued_at`（UNIX 秒）を返し、`client_secret` を発行したら `client_secret_expires_at: 0` を返すと定めた。
TypeSpec では `ClientRegistrationResponse` に三つのフィールドと `require_pkce` を加え、`ClientRegistrationRequest.grant_types` を任意にした（省略は `authorization_code`）。
`ClientSecretCredential.expires_at` の説明は、登録時の最初の資格情報を期限なしの例外として明記した。
secret に期限を付けない判断は、Auth0、Okta、Ping、Curity などが動的登録で 0 を返し、期限を付ける Connect2id は RFC 7592 の自動更新を併せ持つという調査に基づく。IdMagic は RFC 7592 を持たないので、期限を付けると再登録しか手段がなくなる。RFC 7592 を実装するときに見直す。
