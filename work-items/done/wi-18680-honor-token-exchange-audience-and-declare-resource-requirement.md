---
status: completed
---

# トークン交換の `audience` を捨てず、`resource` の必須を宣言と一致させる

アクセストークンへのトークン交換は、対象を `resource` だけで受ける。
`audience` を含む要求は、`resource` と同じ値でも解決せずに 400 と `invalid_target` で拒否し、`TokenExchangeRejected` を発行する。
これまでは `audience` を読まずに捨て、`resource` の対象で発行していた。
標準要件 `RFC8693-TARGET`（`partial`）を追加し、`resource` の欠落を `invalid_request` で拒否することもそこへ宣言した。
`RFC8707-MCP-RESOURCE-BINDING` は、未指定の `resource` を `RFC9068-DEFAULT-AUDIENCE` へ委ねる経路からトークン交換を除いた。
REQ-OAUTH2-048 に同じ二つの拒否と判断を加え、TypeSpec の `TokenRequest` の `audience` と `resource` の説明を合わせた。
`audience` を登録済みのリソースへ解決する案は、`resource` との併用と複数の値の規則が要り、対象を広げる経路になるため採らなかった。
ID-JAG の発行の `audience` は変えていない。
