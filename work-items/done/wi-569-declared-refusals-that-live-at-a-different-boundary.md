---
status: completed
archived_from: 562fbf60c2a3d8baa4064cb56055c772647eeb9e
---

# 異なる境界で実施している OAuth2 の拒否を仕様と一致させる

`mise run spec-diff` が示す規範の差分は、REQ-OAUTH2-026 と REQ-OAUTH2-035 の具体例、および TypeSpec に追加した
`OAuth2ClientNotFoundError` である。
`EX-OAUTH2-026-01` の最後の When と Then は、「public クライアントの登録を `InvalidRequestError` で拒否する」から、
「`client_credentials` を宣言して登録済みの public クライアントがトークンを要求すると `UnauthorizedClientError` で拒否され、
トークンは発行されない」へ変わった。RFC 6749 §4.4 が制限するのはグラントの利用であり、`Token` の 400 union は
`UnauthorizedClientError` を既に宣言していた。
`EX-OAUTH2-035-02` の Then は、「`InvalidRequestError` で拒否される」から、「別テナントの管理者による参照、更新、削除は
どれも `OAuth2ClientNotFoundError` で拒否され、応答は存在しない `client_id` と同じであり、`acme` のクライアントは変更も
削除もされない」へ変わった。TypeSpec は `GetAdminOAuth2Client`、`UpdateAdminOAuth2Client`、`DeleteAdminOAuth2Client` に
404 `OAuth2ClientNotFoundError` を宣言し、更新と削除の doc にも別テナントのクライアントを存在しないものとして扱うことを書いた。
製品のコードは変わっていない。二つの具体例は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。
公開契約に 404 が加わるため、`docs/releases/changes/wi-569.md` にリリースノートを置いた。

全文は `git show 562fbf60c2a3d8baa4064cb56055c772647eeb9e:<このファイルのパス>` で読む。
