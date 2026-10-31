---
status: completed
---

# MCP ゲートウェイを Envoy へ接続する

`idmagic-mcp-gateway` に `mode: envoy_ext_authz` を追加し、既存の判定をそのまま Envoy の HTTP 版 `ext_authz` の入口として使えるようにした。
許可では本文のない 200 と資格情報の除去の指示だけを返し、転送は Envoy が行う。
設定例と迂回の検査の手順を `examples/envoy-mcp-gateway` に置いた。

仕様は REQ-MCPGATEWAY-007 を追加し、REQ-MCPGATEWAY-001（`mode` の検査）、003（カンマを含むトークンと DPoP を複数の値として評価せずに拒否。単体の構成にも適用）、006（結果 `permitted`）を変更した。

起票時の第一候補だった `ext_proc` は、Envoy v1.40.2 で試作と比べて採らなかった。
判定まで本文を止める点は同じで、書き換えの要件がなく、gRPC と制御プレーンの依存が増えるためである。
`ext_authz` ではクエリを除けないのでクエリのある要求を拒否し、Envoy が 200 を許可と読むので判定の呼び出しへメタデータを返さない。

検証は、設定例そのものを読み込む実在の Envoy の試験 `mise run test-envoy-mcp-gateway`（CI の `go-test` ジョブでも実行）、単体テスト、`verify-changed` である。
手動の故障注入として、`failure_mode_allow` の有効化、`origin` の欠落、上流の `X-Request-ID` の残存、資格情報の除去の欠落、メタデータの防御の削除を試し、すべて検出された。

確認の限界と残るリスクは次のとおりである。

- 評価の流量制限の `Retry-After` は MCP クライアントへ届かない。Envoy が判定の入口の 5xx の応答を捨てるためで、結果は不許可のままである。この挙動は実装後の試験で判明し、`ext_proc` では同じ条件を測っていない。
- 上流の障害は JSON-RPC の `-32003` ではなく Envoy の 503 になる。
- 試験の評価はダブルで、製品の評価を通す経路は単体の構成の受け入れテストだけが確かめている。実際の MCP SDK と本番の TLS 終端を Envoy の背後で確かめていない。
- 利用者が `allowed_headers` から `origin` を除いても検出できず、Envoy を通らずに上流へ届く配置は迂回になる。どちらも手順の迂回の検査で確かめる。
- CI の段階は未実行で、次の push で初めて Linux のランナーで動く。`host-gateway` の到達は手元の Docker だけで確かめた。
