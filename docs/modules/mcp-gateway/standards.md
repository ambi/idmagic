# McpGateway の標準仕様

## OAuth 2.0 RAR Metadata and Error Remediation

draft-ietf-oauth-rar-metadata-remediation-00 — https://www.ietf.org/archive/id/draft-ietf-oauth-rar-metadata-remediation-00.html

Internet-Draft である。`-00` に固定し、相互運用を主張しない。
明細の組み立ては [REQ-OAUTH2-065](../oauth2/features/resource-access.md#req-oauth2-065-承認が必要な操作は承認と同じ対象と引数で一度だけ許可する)、MCP クライアントへの応答は [REQ-MCPGATEWAY-003](features/runtime.md#req-mcpgateway-003-評価が許可した要求だけを上流へ送る) が定める。

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RARREM-ERROR-CODE | required | SHOULD | 承認がないか使えないことで実行を拒否するとき、`WWW-Authenticate: Bearer` に `error="insufficient_authorization"` を載せる。 |
| RARREM-REMEDIATION | required | SHOULD | `authorization_remediation` に、要求から組み立てた `authorization_details` を持つ JSON を base64url で符号化して載せる。符号化した値が 8 KiB を超えるときは、ヘッダーから除いて JSON-RPC の `error.data` だけで返す。 |
| RARREM-STATUS | partial | MUST | 401 ではなく 403 で返す。MCP は無効なトークンに 401、実行時の権限の不足に 403 を求め、MCP の SDK は 401 で認可をやり直すためである。 |
| RARREM-REFERENCE | required | MUST | 承認したトークンは一回限りなので、`authorization_reference` を省く。草案は一回限りのトークンでは省くよう求める。 |
