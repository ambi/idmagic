# Envoy の背後の MCP ゲートウェイ

既存の Envoy を執行の位置にし、`idmagic-mcp-gateway` を HTTP 版の `ext_authz` の判定の入口として使う設定例である。
振る舞いの仕様は [REQ-MCPGATEWAY-007](../../docs/modules/mcp-gateway/features/runtime.md#req-mcpgateway-007-envoy-の-ext_authz-の入口は判定だけを返し設定例は失敗時に通さない) が定める。
`mise run test-envoy-mcp-gateway` は、実在の Envoy でこの [`envoy.yaml`](envoy.yaml) そのものを試験する。
CI では毎日の定期実行で動くので、設定例を変えたら手元でも実行する。

## 手順

1. ゲートウェイの設定に `mode: envoy_ext_authz` を書き、`upstream` を書かない。
   ほかの項目は[設定](../../docs/modules/mcp-gateway/features/runtime.md#設定)に従い、`resource` は Envoy の公開 URL にする。
2. `envoy.yaml` の接続先、公開の経路、`max_request_bytes` を環境に合わせる。
   経路を変えたら、メタデータの経路の末尾も合わせる。
   TLS は Envoy か、その前段で終端する。
3. 上流の MCP サーバーへ Envoy からだけ到達できるように、ネットワークを制限する。
4. 下の検査を、配置した Envoy へ向けて行う。

## 迂回の検査

設定の誤りは執行の迂回になる。
配置と設定の変更のたびに、次の結果を確かめる。
`$MCP` は Envoy の公開 URL である。

| 検査 | 操作 | 期待する結果 |
| --- | --- | --- |
| トークンがなければ通さない | `curl -i -X POST "$MCP" -H 'Content-Type: application/json' -d '{}'` | 401 |
| クエリを通さない | 同じ要求の URL に `?a=b` を付ける | 404 |
| `Origin` を検査している | 同じ要求に `-H 'Origin: https://evil.example'` を付ける | 403。401 なら `allowed_headers` から `origin` が落ちている |
| ゲートウェイの停止で通さない | ゲートウェイを止めて、有効なトークンで要求する | 503 |
| 上流へ直接届かない | Envoy の外から上流の URL へ要求する | 接続できない |

上の三つでは、MCP サーバーのログに要求が現れないことも確かめる。

## 単体の構成との違い

上流へ接続できないときの応答は、ゲートウェイの 502 と `-32003` ではなく Envoy の 503 になる。
監査ログの結果は `forwarded` ではなく `permitted` で、上流の結果を含まない。
評価の流量制限で 503 を返すとき、Envoy がゲートウェイの応答を捨てるので、`Retry-After` は MCP クライアントへ届かない。
