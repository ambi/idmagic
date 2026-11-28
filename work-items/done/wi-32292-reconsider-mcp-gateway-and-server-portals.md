---
status: completed
---

# Cloudflare の MCP server portals と比較して MCP の提供形態を再検討し、必要な変更を実装する

Cloudflare、Azure API Management、ContextForge と比較し、既存ゲートウェイの一覧、接続の配布と復旧を改善する構成を採用した。
REQ-MCPGATEWAY-015 と REQ-OAUTH2-099 を追加し、現在の公開・割当・スコープを満たすツールだけをマネージド型とカタログ proxy の一覧に残し、定義とページングを保ち、主体間のキャッシュ共有を防ぐ。
REQ-OAUTH2-100 と REQ-OAUTH2-101 を追加し、接続 URL と対応条件を再訪時にも取得できる案内、本人の接続不足と再認可の必要を区別した復旧情報を実装した。
REQ-OAUTH2-092、REQ-TOKENVAULT-005、REQ-TOKENVAULT-016 を変更し、有効な接続も本人が再連携でき、上流の 401 は資格情報の拒否として案内し、自動再送と接続の無効化を行わない。
TypeSpec は ResourceAccessEvaluationResponse の allowed_tools、HostedMcpFailure の復旧情報の文書、McpResourceServerPages の接続案内を変更した。
独立ポータルと集約は公開集合と認可の対応、上流 DCR と同期は共有資格情報と本人ごとの可視範囲、Code Mode は任意コードの実行と各操作の承認の責務を増やすため見送り、現在の設計判断に理由と再検討の条件を残した。
手動の公開とサーバーごとの URL 配布を維持し、上流の追加・改名は管理者が反映する負担を受け入れる。
対応範囲は MCP 2026-07-28 のツールを使う Agent であり、2025 版、legacy SSE、prompts、resources、MCP Apps と任意のデスクトップ製品への互換性は提供しない。
Envoy ext_authz と独自マッピングは一覧の絞り込みの対象外であり、Cloudflare のネットワーク基盤、DLP と WAF は再実装しない。
