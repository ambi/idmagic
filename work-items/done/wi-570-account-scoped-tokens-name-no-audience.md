---
status: completed
archived_from: 14aa33664a730d09e8e785a5cc213e3d2b1148c1
---

# account スコープのトークンが、宛先のリソースサーバーを名乗らない

`mise run spec-diff` が示す規範の差分は、REQ-OAUTH2-001 の具体例、追加した `RFC9068-DEFAULT-AUDIENCE`、
変更した `RFC8707-MCP-RESOURCE-BINDING` である。
RFC 9068 §3 に従い、`resource` を含まない要求では `scope` からデフォルトの資源を推定して `aud` に入れることにした。
`account:` で始まるスコープを含むトークンは、レルムの IdMagic API（レルムの発行者識別子）を `aud` に持つ。
含まないトークンは従来どおり `client_id` を持つ。
推定は、デフォルトの audience を決める唯一の場所である `tokens_jose.JWTSigner.SignAccessToken` に置いた。
認可コード、リフレッシュ、デバイス認可、CIBA の各発行経路は変えていない。
account リソースサーバー（`support_http.Authenticator`）は、account スコープを持つトークンの `aud` がリクエスト先
レルムの発行者識別子を含まなければ 401 `invalid_token` で拒否する。ポータル境界のスコープだけを持つトークンは検査しない。
`EX-OAUTH2-001-01` の audience を「レルムの IdMagic API（レルムの発行者識別子）」と具体化し、検証の振る舞いを
`EX-OAUTH2-001-04` として追加した。`RFC8707-MCP-RESOURCE-BINDING` の「`resource` が未指定であれば `client_id` を
`audience` とする」は、`RFC9068-DEFAULT-AUDIENCE` への参照に置き換えた。用語集の Audience と
`docs/design/security/authorization.md` にも同じ規則を書いた。
構造変更として、account スコープの判定を `spec.IsAccountScope` へ抽出し、`token_handler.go` の
`containsAccountScope` を置き換えた。
`EX-OAUTH2-001-01` は名指すテストを得て、`tools/check/example-coverage-debt.json` の被覆台帳から外れた。

全文は `git show 14aa33664a730d09e8e785a5cc213e3d2b1148c1:<このファイルのパス>` で読む。
