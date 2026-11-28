---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-29
priority: p3
group: mcp-gateway
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/hosted-mcp.md, requirement: REQ-OAUTH2-091 }
  - { path: docs/modules/oauth2/features/hosted-mcp.md, requirement: REQ-OAUTH2-092 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-087 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-083 }
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-086 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-003 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-013 }
  - { path: docs/modules/token-vault/features/connection.md, requirement: REQ-TOKENVAULT-007 }
  - { path: docs/modules/token-vault/features/execution.md, requirement: REQ-TOKENVAULT-016 }
---

# Cloudflare の MCP server portals と比較して MCP の提供形態を再検討し、必要な変更を実装する

## 動機

Cloudflare の [MCP server portals](https://developers.cloudflare.com/cloudflare-one/access-controls/ai-controls/mcp-portals/) を参考に、IdMagic の MCP ゲートウェイの設計と機能を再検討する。
既存ゲートウェイの改善、ゲートウェイとは別の MCP サーバーポータルの追加、実装変更なしのいずれも結論の候補とする。
採否の判断から始めるため、優先度は p3 とする。

IdMagic の現在仕様には、[マネージド型 Gateway](../../docs/modules/oauth2/features/hosted-mcp.md) の登録と転送、[セルフホスト型 Gateway](../../docs/modules/mcp-gateway/features/runtime.md)、[ツールカタログと公開設定の区別](../../docs/modules/oauth2/features/admin-access.md#管理者による-mcp-リソースサーバーの準備)、[TokenVault による本人の資格情報の仲介](../../docs/modules/token-vault/features/execution.md#req-tokenvault-016-マネージド型-mcp-の代理転送でも本人の接続と規則を守る) がある。
マネージド型の公開 URL は資源登録ごとに発行され、その登録が上流設定を持つ。
これらとの重複と、複数サーバーの発見や接続をまとめる価値を分けて調べ、ポータルという名前だけを理由に別の仕組みを増やさない。

## 対象範囲

- 管理者によるサーバー登録、公開する機能の選択、割当、監査と、利用者による発見、クライアントへの接続設定、上流認証、再認可、接続解除を、Cloudflare と現在の IdMagic で比較する。
- サーバーの一覧や接続案内を提供するポータルと、複数サーバーを一つの MCP エンドポイントへ集約するポータルを区別し、それぞれの必要性を判断する。
- ツールの一覧と実行、prompts、resources、別名と名前の衝突、上流の変更と障害の扱いを比較する。Code Mode などの検索や実行の集約も、必要性を判断する対象に含める。
- 既存ゲートウェイの改善、独立したポータル、実装変更なしを比較し、採る点と採らない点、その理由、再検討の条件を残す。
- 変更が必要なら、採用した最小の範囲の要件、TypeSpec、設計、管理画面またはアカウント画面、実装、検証をこの項目で仕上げる。

## 対象外

- 公開されたサーバーの定期取得と認証結果の保存は [[wi-19974-refresh-public-mcp-tool-lists-and-check-their-authentication]] が扱う。本項目はポータルで必要な発見と同期の範囲を判断し、重なる実装が必要なら着手前に両項目の分担を更新する。
- 実際の MCP クライアントの要求の記録と再生は [[wi-11142-record-real-mcp-client-requests-to-the-gateway]] が扱う。本項目では採用した経路の接続条件を確認し、記録基盤を重ねて作らない。
- セルフホスト型の配置と運用の手順書は [[wi-56570-document-operating-the-mcp-gateway]] が扱う。本項目は提供形態の判断と、採用した変更に必要な案内を担当する。
- 非 OAuth の資格情報の仲介の採否は [[wi-62382-decide-non-oauth-downstream-credentials-in-the-vault]]、管理 API 自体の MCP 公開は [[wi-49273-decide-exposing-admin-api-as-mcp-server]]、Enterprise-Managed Authorization の適合は [[wi-99524-conform-to-mcp-enterprise-managed-authorization]] が扱う。ポータルの追加に伴って一括で実装しない。
- Cloudflare のネットワーク製品、DLP、WAF 全般の再実装と、競合製品との機能数の一致。

## 設計

### 既存製品との比較

起票時の調査日は 2026-11-29 である。
以下は公式文書の本文を読んだ範囲であり、各製品の実環境で接続を確認した結果ではない。
着手時に更新を確認し、採用する振る舞いの比較を補う。

| 製品と出典 | 読んだ範囲と扱い | IdMagic で判断する点 |
| --- | --- | --- |
| [Cloudflare MCP server portals](https://developers.cloudflare.com/cloudflare-one/access-controls/ai-controls/mcp-portals/) | 概要、サーバー登録、上流 OAuth、同期、ポータル作成、ツールと prompts の管理、名前空間、接続とセッション管理、ポリシーの制約の本文。複数サーバーを一つの接続先へまとめ、公開対象の選択、別名、利用者の上流認証を扱う。デフォルトの全公開と allowlist の選択肢があり、上流の管理者資格情報を共用する設定もある。ポータル経由では一部の追加認証が適用されない。 | 集約と接続管理の価値を調べる。全公開、資格情報の共用、認証の制約は既存の保証と照合し、そのまま採用しない。 |
| [Azure API Management の既存 MCP サーバーの公開](https://learn.microsoft.com/en-us/azure/api-management/expose-existing-mcp-server) | 制約、登録手順、公開ツールの選択の本文。上流サーバーの URL と transport を登録し、経路の接頭辞と公開 URL を設ける。外部 MCP サーバーでは tools と resources を扱い、prompts は扱わない。 | サーバーごとの公開 URL と接続案内で利用目的を満たせるかを、集約する案と比べる。 |
| [IBM ContextForge の管理概念](https://ibm.github.io/mcp-context-forge/overview/ui-concepts/)と[仮想サーバーの実装](https://github.com/IBM/mcp-context-forge/blob/612778b87cefc96583c4af9375bca0ce436a1877/mcpgateway/services/server_service.py#L654) | Virtual Servers と Gateways の本文、および固定コミットの `register_server` の登録処理。仮想サーバーは選んだ tools、resources、prompts をまとめる。実装はサーバーとそれぞれの関連付けを保存する。 | 集約する公開単位と上流の登録を分ける価値を比較する。保存と更新の責務、機能の選択、削除時の整合を新たに持つ費用も数える。 |

起票時には、ポータルの追加や既存の規範の変更を確定しない。
既存の毎回の認可、カタログからの明示的な公開、本人の接続の解除、承認した操作の最大一回の転送を比較の基準にする。
Cloudflare の機能ごとに、既存機能で満たす、改善する、別のポータルで担う、採らない、の結論と根拠を設計に追記する。

### 代替案と判断基準

| 案 | 採る条件 | 確かめる費用と限界 |
| --- | --- | --- |
| 既存ゲートウェイと接続管理を改善する | 問題が登録、発見、接続案内、上流認証などの既存機能に収まり、独立した公開単位を要しない | OAuth2 と TokenVault の責務の拡張範囲、セルフホスト型との設定差、既存 URL と操作割当への影響 |
| ゲートウェイとは別のポータルを設ける | 一覧や接続管理、または用途別の集約に独立した状態と操作が必要で、既存画面の改善だけでは主要ユースケースを満たせない | 一覧だけか実行も仲介するか、状態の所有者、公開契約、同期、失効、障害隔離。別画面、別モジュール、別プロセスの要否はそれぞれ判断する |
| 実装を追加も変更もしない | 具体的な利用経路が既存機能で満たせるか、追加の価値が状態と運用と安全性の負担を上回らない | 満たせる経路を仕様で示し、満たさない用途を受容する理由と再検討の条件を残す |

集約する案では、公開名から上流の資源登録と操作への対応、トークンの宛先、送信者制約、承認の束縛先を実装前に確定する。
別名の変更や同名ツールの再登録によって既存の割当を流用せず、一覧から隠した機能を直接呼び出す場合の扱いも定める。
一部の上流が停止、未認可、失効、同期失敗である場合の一覧と呼び出しの結果を分け、別のサーバーや利用者の資格情報へ切り替えない。

### 境界の判断

入力は、[モジュールの責務表](../../docs/modules/README.md#モジュール)、[OAuth2](../../docs/modules/oauth2/README.md)、[McpGateway](../../docs/modules/mcp-gateway/README.md)、[TokenVault](../../docs/modules/token-vault/README.md) と前付けの規範要素である。
[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D9 で独立配布のセルフホスト型を先に区別し、D1 と D2 で資源登録と本人の接続の所有者を保つ案から比較する。
新しいポータルの状態が必要なら、D8 に従い、隠す判断と独立して変えたい操作を挙げてからモジュールの追加を決める。

着手時に「上流の追加と削除」「公開する機能と名前の変更」「本人の接続解除」「上流の部分障害」を変更シナリオとして、候補ごとの保存、公開操作、利用側、監査、検証への波及を記録する。
SQL と作用の流れ、原子性、依存と組み立ての実態を調べる前に所有者を変更しない。
既存の編成で足りるなら新しいモジュールは作らず、独立した意味を持つ状態が必要なら、その所有者と最小の契約を設計に追記する。

## 計画

1. 現在の仕様、実装、画面と隣接項目を確認し、管理者の準備から利用者の接続と解除までの比較を完成する。
2. 三案を判断基準と変更シナリオで比較する。ポータルの意味、認可の単位、対応する MCP の機能と transport、同期と失敗の扱い、提供形態を、実装前にすべて確定する。
3. 変更する場合は要件の差分と TypeSpec を先に更新する。追加の規範を `affected_spec` に加え、主要ユースケースと故障から `primary_use_cases`、検証境界、テスト参照を確定する。
4. 採用した経路の RED を確認し、必要な実装と案内を作る。認可、テナント境界、移行、既存構成との互換性、ロールバックの条件を設計へ残す。
5. 変更しない結論も検討の完了として扱う。根拠、受容する不足、再検討の条件を担当する現在状態の設計文書に残し、要約を書く。

## タスク

- [ ] T001 [Research] Cloudflare、商用製品、OSS と現在の IdMagic の利用経路を比較し、既存機能と不足を確かめる。
- [ ] T002 [Decision] 既存ゲートウェイの改善、独立したポータル、実装変更なしの採否と理由、境界、残る制約を確定する。
- [ ] T003 [Spec] 変更する場合は要件と TypeSpec、脅威モデル、設計を更新し、仕様影響と主要ユースケースの証拠を計画する。変更しない場合は判断と再検討の条件を残す。
- [ ] T004 [Acceptance] 変更する場合は利用経路と重要な拒否を、故障を検出できる境界で RED にする。
- [ ] T005 [App] 採用した範囲を実装し、現在状態の設計と利用者の案内を更新する。
- [ ] T006 [Verify] 結論に応じた検証を行い、実装しなかった候補と確認の限界を含めて完了する。

## 検証

- 起票時は `mise run check-work-items` と `mise run check-links` で、形式、規範参照、隣接項目へのリンクを検査する。
- 着手時は `mise run brief -- wi-32292-reconsider-mcp-gateway-and-server-portals --target <対象パス>` で候補を得る。境界、作用、契約、認可を変える場合は該当する `--concern` を付け、選んだタスクと対象、理由をこの節へ追記する。
- 実装する場合は、主要経路の最終結果に加え、別テナント、割当なし、接続解除後、上流へ直接接続する迂回、承認の再使用を独立に観測する。拒否した要求が上流へ届かないことを検査する。
- 集約する場合は、同名機能の振り分け、一覧と直接呼び出しの認可、部分障害、同期後の追加機能の公開、名前や接続先を変えた後の古い割当を検査する。
- コードを変えた場合は対象の `test-go-package` と `mise run verify-changed -- <着手時の基準リビジョン>`、画面を変えた場合は対象の `test-ui-e2e-file` を選ぶ。仕様や境界を変えた場合は `check-spec`、`check-boundaries`、`spec-diff` など該当する検査を加える。localhost の待受けを使う検査は初回からサンドボックスの外で実行する。
- 実装変更なしの場合は比較と採否の根拠をレビューし、変更した文書の形式とリンクを検査する。模擬相手による試験、実際のクライアントとの接続、外部サービスでの確認を区別して報告する。

## リスク

集約によって、公開名、トークンの宛先、上流の操作、本人の接続の対応を誤ると、別の操作や資格情報で実行する危険がある。
対応を明示したモデルと拒否時の無作用を先に定める。

ツールの同期や検索の便利さを理由に、自動公開や管理者の資格情報の共用を導入すると、既存の操作割当と本人の解除の保証を弱めうる。
既存の保証から差分を示し、外部製品のデフォルト値を採用の根拠にしない。

比較対象を増やすだけで結論が出ないか、ポータルの追加が目的になって範囲が広がるおそれがある。
管理と利用の具体的な経路で価値を判定し、採用した最小の変更か、変更不要という結論で完了する。
