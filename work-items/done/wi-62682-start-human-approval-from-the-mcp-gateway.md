---
status: completed
---

# MCP ゲートウェイが承認を起票し、汎用の MCP クライアントを人の承認へ導く

汎用の MCP クライアントを、トークンを差し替えずに人の承認へ導く経路を追加した。
REQ-OAUTH2-093 を新設し、評価の入口が `approval_start` で承認リクエストを起票して本人へ通知し、`approval_request_id` の参照で承認を照合して一回だけ許可するようにした。
参照できるのは同じ登録が同じ利用者、クライアント、Agent の評価で起票したものに限り、同じ操作の保留は再利用し、保留中は一人 10 件までとする。
起票した承認は `auth_req_id` を持たず、使用権の記録と同時に `Consumed` になる（`ApprovalRequestLifecycle` の意味と「評価による使用」の列を更新）。
REQ-MCPGATEWAY-014 を新設し、`elicitation.url` を宣言したクライアントに MCP 2026-07-28 の `InputRequiredResult` で承認の URL と AES-256-GCM で封じた `requestState` を返し、再送を検証して自分の状態を除いた本文を一回だけ転送するようにした。
REQ-OAUTH2-043 でポータルは承認の URL の承認リクエストを先頭に示し、起票した登録の名前を表示する。

- **判断**：T000 で MCP のステップアップ認可ではなく URL モードの elicitation を採った。Claude Code は 403 `insufficient_scope` で再認可せず、URL モードは Claude Code と VS Code が宣言する。
- **判断**：`elicitation.url` を宣言しないクライアントには、当初案の `-32021` ではなく従来の 403 と明細を返す。CIBA を呼ぶ Agent の経路（wi-34766）を保つためで、利用者が確認した。
- **判断**：`requestState` の鍵はクライアントシークレットから HKDF で導き、主体は入れない。主体の一致は IdMagic が参照の評価で確かめる。
- **判断**：通知の失敗では評価を止めない。承認リクエストは保存済みで、ポータルと URL から開ける。
- 対象外のまま：`envoy_ext_authz`、マネージド型、主体が Agent のトークン、既存の `-32000` 番台のコードの移行。実 Envoy の試験（`test-envoy-mcp-gateway`）と、ファジングと変異テストは実施していない。
