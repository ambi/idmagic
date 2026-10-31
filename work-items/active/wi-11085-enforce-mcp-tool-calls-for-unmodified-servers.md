---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p1
depends_on: [wi-10407-measure-and-rate-limit-resource-access-evaluation]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-063 }
---

# 改修していない MCP サーバーの tools/call を実行前に認可する

## 動機

`examples/resource-server-pep` は業務処理をコードで包むので、MCP サーバーを変更できる場合にしか使えない。
[Ping](https://developer.pingidentity.com/blog/securing-mcp-servers-with-pings-mcp-gateway/) と [Okta](https://www.okta.com/blog/product-innovation/okta-agent-gateway-runtime-security/) はゲートウェイを一般提供し、[Entra](https://learn.microsoft.com/en-us/entra/global-secure-access/how-to-configure-mcp-firewall) も MCP firewall を preview で提供している。
IdMagic には既存の MCP サーバーを無改修で保護する経路がない。

## 対象範囲

- 単体のリバースプロキシとして動く、MCP 専用の執行部品。
- `tools/call` のツール名を `resource_type`、`relation`、`resource_id` へ写す宣言。
- 2026-07-28 版の MCP を対象に固定し、解釈できない要求と JSON-RPC のバッチを不許可にする。
- 評価が不許可か不能のとき上流へ送らない。
- 主体、Agent、ツール、判断理由、相関 ID の監査とツール別の指標。
- 製品の対象外の説明を「汎用 API Gateway は対象外、MCP 専用の執行部品は含む」と明確にする。

## 対象外

- Envoy との接続（[[wi-23415-attach-mcp-enforcement-to-envoy]]）。
- `tools/list` の絞り込み（[[wi-24789-filter-mcp-tool-lists-by-authorization]]）。
- 下流向けのトークン差し替え（[[wi-23211-exchange-agent-tokens-through-an-enforcement-gateway]]）。
- 汎用 API Gateway、WAF、プロンプトの検査。

## 設計

境界の判断手順を T001 で適用し、次の候補を比較して記録する。

| 候補 | 利点 | 懸念 |
| --- | --- | --- |
| `examples` の延長 | 製品の依存が増えない | 運用する配布物として保守の約束ができない |
| 同じリポジトリの独立した Go モジュールと実行ファイル（第一候補） | 公開契約だけで結べ、IdMagic の内部型を import しない | リリースと互換性の管理が増える |
| IdMagic 本体のサーバーへ内蔵 | 配置が単純 | 業務通信が認証基盤を通り、障害と負荷の範囲が広がる |

中核は `Decide(request, mapping) -> Evaluation` の決定論的な計算とし、HTTP の転送と評価の呼び出しを作用として分ける。
主体を要求から自己申告させず、提示トークンと DPoP 証明をそのまま評価へ渡す。
DPoP の `htu` は登録の `resource` の配下である必要があるので、プロキシの公開 URL と登録の関係を仕様に定める。

## 計画

1. 配置と配布を決め、製品の対象外の記述を更新する。
2. ツールの写像の形式と、MCP の方法ごとの既定の扱いを決める。
3. 実在の MCP SDK のサーバーを無改修で背後へ置く受け入れテストを先に書く。

## タスク

- [ ] T001 [Design] 境界の判断の記録を書き、配置と配布を確定する。
- [ ] T002 [Spec] 写像、拒否の応答、監査と指標を仕様へ書く。
- [ ] T003 [Acceptance] 許可、拒否、評価不能、未知の方法で上流の呼び出しの有無を観測する RED を作る。
- [ ] T004 [App] 中核とプロキシを実装する。
- [ ] T005 [Verify] ストリーミング、バッチ、再送、別 audience、DPoP の再利用を検証する。

## 検証

- 無改修の MCP サーバーへの呼び出し回数を観測する受け入れテスト。
- `mise run check-boundaries`：部品が IdMagic の内部型へ依存しないことを確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

経路上の部品の解釈の誤りは、許可の取り違えに直結する。
既定で不許可にし、未知の方法とバッチを通さない。
評価の障害が業務通信の停止になるので、[[wi-10407-measure-and-rate-limit-resource-access-evaluation]] の測定を前提にする。
