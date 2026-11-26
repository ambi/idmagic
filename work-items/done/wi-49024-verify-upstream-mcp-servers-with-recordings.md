---
status: completed
---

# 上流の MCP サーバーへの前提を、記録の再生と模倣で確かめる

上流の MCP サーバーへの前提を、MCP 2026-07-28 の仕様と RFC 9728 の例を転記した記録の再生と、状態を持つ模倣で確かめるようにした。
記録と模倣は MCP Gateway と OAuth2 の両方が使うので `backend/shared/testing_mcptwin` に置き、記録は版ごとのファイルにした。
2026-07-28 版は `initialize` を持たないので、初期化の代わりに未対応の版の `-32022` を記録した。
SSE の `subscriptions/listen` を照合するため、`testing_httprecord.Fidelity` に本文の読み方を差し替える `Decode` を足した（別コミット）。

見つけた誤りを一つ直した（利用者に確認した）。

- REQ-MCPGATEWAY-011、REQ-OAUTH2-095：Gateway のカタログ収集と OAuth2 のツール取得は、空文字列の `nextCursor` を一覧の終わりと読み、二ページ目以降を取らずに `complete=true` にしていた。仕様のページングは空文字列を有効なカーソルとし、終わりとして扱うことを禁じる。`nextCursor` がないか `null` のページだけを終わりにし、空文字列は `cursor` に載せて次のページを要求する。

保証は仕様の例を解釈できることまでで、特定の製品の癖は含まない。
`clientInfo` を送らないこと（SHOULD）と、`resultType` が `complete` でない応答は未規定のまま残した。
