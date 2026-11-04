---
status: completed
---

# 実在の MCP クライアントで再現できる執行の手順を公開する

公式の MCP TypeScript SDK v2（`@modelcontextprotocol/client` と `@modelcontextprotocol/server` の 2.3.1）で、改修していない MCP サーバー、`idmagic-mcp-gateway`、IdMagic を結ぶ手順を `examples/mcp-client-walkthrough/README.md` に置いた。
`mise run test-mcp-client-walkthrough` は、IdMagic とゲートウェイを現在のソースからビルドし、公開の管理 API と MCP だけで同じ手順を再現する。
確かめるのは 13 の確認で、内容は接続時の発見とトークン取得、許可、割当のない操作、条件の拒否、承認が必要な操作の安全側の停止、撤回、停止の後の拒否である。
拒否した呼び出しは MCP サーバーへ届かず、撤回と拒否では SDK がトークンを取り直さないことも見る。
CI では日次の定期実行で動く。

判断：委譲は Agent 自身の `client_credentials` で行い、利用者から Agent への委譲は未確認とした。承認後の実行は wi-83492 が扱う。拒否の 403 に `insufficient_scope` を付けない。拒否は割当、条件、関係で決まり、SDK の再認可では結果が変わらないためである。
試作で、IdMagic が `client_id` を省いたクライアントアサーションを拒否する欠陥を見つけ、wi-62545 で直した。

確認の限界：管理画面の操作は画面のコードで文言と入力の形を確かめたが、ブラウザーで手順どおりに操作してはいない。再現試験は同じ操作を管理 API で行う。DPoP、Envoy の背後での SDK、ほかの SDK とクライアント製品は確かめていない。
