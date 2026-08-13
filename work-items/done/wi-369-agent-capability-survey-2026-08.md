---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: 2026-08 時点の AI エージェント向け機能の棚卸しと採否判断の記録であり、仕様も実装も変更しない。
---

# 2026-08 時点の AI エージェント向け IdP/IdM 機能を棚卸しし、採否と見送り理由を記録する

## Motivation

見送った候補が「検討されなかった」のか「検討したうえで見送った」のかは、記録が無いと
後から区別できない。区別できなければ同じ調査を繰り返し、同じ結論に別の労力を払うことになる。

2026 年に入り、AI エージェント向け ID 基盤は「新しい標準を待つ段階」から「既存の OAuth / OIDC を
どう組み合わせるか」の段階に移った。この転換点で idmagic の現状を外部動向と突き合わせ、
何を実装し、何を恒久的に追わず、何を条件付きで先送りするかを 1 箇所にまとめる。

判断は時点に依存する。何を見て判断したかを版数付きで残すことが、この記録の主目的である。

## Scope

- 調査時点 (2026-08) の外部動向を、参照した一次情報の版数とともに記録する。
- idmagic のエージェント関連実装の現状を記録する。
- 本棚卸しから起票した work item を記録する。
- 恒久的に追わないものと、条件付きで先送りするものを、理由と再評価条件とともに記録する。

## Out of Scope

- 仕様・実装の変更。本 work item は記録であり、導かれた作業は個別の work item が持つ。
- 棚卸しから起票した work item の中身の設計。各 work item に閉じる。

## Design

### 調査時点の外部動向 (2026-08)

- **OpenID Foundation "Identity Management for Agentic AI" (2025-10)** — 既存の OAuth 2.0 /
  OIDC でエージェントの多くのユースケースは守れると結論しつつ、3 つのギャップを指摘:
  ① 自律実行モードと on-behalf-of モードの切り替えを追跡できない、
  ② 標準の不在によりプロプライエタリ ID 基盤が乱立する、
  ③ 再帰的委譲が原理的な上限を持たない。
- **NIST NCCoE 概念文書 (2026-02-05)** "Accelerating the Adoption of Software and AI Agent
  Identity and Authorization" — MCP / OAuth 2.0-2.1 / OIDC / SPIFFE-SPIRE / SCIM を
  エージェント ID の候補標準として列挙。NIST AI Agent Standards Initiative も 2026-02 発足。
- **OpenID AuthZEN WG Drafts (2026)** — **AARP** (Access Request and Approval Profile:
  認可が「まだ」下せないとき = 承認・同意・委譲権限・アテステーション・リスク評価が
  前提条件となるときの、要求 → 追跡 → 充足 → 再評価の相互運用パターン) と
  **COAZ** (MCP ツール認可の AuthZEN プロファイル) が WG Draft に昇格。
- **IETF** — `draft-araut-oauth-transaction-tokens-for-agents-01` (2026-05-12、Informational、
  個人 draft)、`draft-liu-oauth-a2a-profile-00`。
  `draft-ietf-oauth-identity-assertion-authz-grant` は `-04` (2026-05-21、Standards Track、未 RFC 化)。
- **MCP 2026-07-28 改訂** — stateless core 化 + 認可ハードニング 6 SEP。idmagic 側の影響は
  [[wi-322-mcp-authorization-spec-repin-2026-07-audit]] で監査済みで**破壊的差分なし**。
  重要な点として、MCP は**エージェント ID・リクエスト単位認可・委譲の証跡・監査をプロトコル外と
  明言**しており、これらは認可サーバー (= idmagic) 側の責務である。
- **Linux Foundation Agentic AI Infrastructure Foundation (AAIF)** 発足。

### idmagic の現状 (2026-08)

エージェント ID の**アイデンティティ層とトークン層は実装済み**である。

- `Agent` を User / OAuth2Client と並ぶ第三の principal として定義
  ([[wi-49-agent-identity-first-class-principal]])。`AgentKind` (Autonomous / Supervised)、
  `AgentStatus` (Active / Disabled / Killed は一方向終端)、`owner_user_id` 必須。
  Agent 自身は資格情報を持たず `AgentCredentialBinding` で既存 OAuth2Client に束縛する。
- RFC 8693 Token Exchange は **delegation-only** (`sub` を保持し `act` を必ず入れ子)、
  `may_act` 強制、深さ上限あり ([[wi-50-token-exchange-delegation-actor-chain]])。
- RFC 9396 RAR はテナント登録済みの検証スキーマを持ち、単調な縮小のみを許す
  ([[wi-51-rich-authorization-requests-agent-scopes]])。
- MCP 認可サーバーとして RFC 9728 PRM と RFC 8707 resource indicator を全グラント経路に
  fail-closed 適用 ([[wi-56-mcp-authorization-server]])。
- ワークロード ID 連携 (JWT-SVID / K8s SA トークン) を Token Exchange に接続
  ([[wi-54-workload-identity-federation-spiffe]])。
- CAEP / SSF によるエージェント失効 ([[wi-58-continuous-access-evaluation-agent-revocation]])。
- DPoP・mTLS 証明書束縛・private_key_jwt・CIMD ([[wi-341-oauth-client-id-metadata-document-cimd]])。

一方で**制御・ガバナンス層が丸ごとバックログに滞留している**:
[[wi-52-ciba-async-human-approval]] (CIBA)、[[wi-53-rebac-fine-grained-authorization]] (ReBAC)、
[[wi-55-token-vault-federated-connections]] (Token Vault)、
[[wi-57-cross-app-access-identity-assertion-grant]] (Cross-App Access)、
[[wi-59-agent-governance-guardrails-audit-inventory]] (ガードレール / インベントリ)。

特筆すべき点として、idmagic の delegation-only + 深さ上限 + `may_act` 強制は、
OIDF の指摘するギャップ ③ に対する数少ない実装済みの回答になっている。

### 本棚卸しから起票したもの

- [[wi-366-www-authenticate-resource-metadata-challenge]] — RFC 9728 の PRM 配信は実装済みなのに
  401 からそこへ誘導する `resource_metadata` が 0 件で、MCP クライアントが認可サーバーを
  発見できない。低コストで、MCP 認可サーバーという製品の中心的な主張の欠けを埋める。
- [[wi-367-dpop-ath-claim-verification]] — RFC 9449 が要求する `ath` 検証が 0 件で、DPoP proof が
  特定の access token に束縛されていない。長時間稼働し多数のリソースサーバーを横断する
  エージェントでは、proof 流用の窓が構造的に広い。
- [[wi-368-delegation-depth-policy-and-delegation-mode-claim]] — 委譲深さ上限がハードコードで
  仕様の "configurable" と乖離しており、また委譲モードが明示されていない。
  OIDF ギャップ ① と ③ に直接対応する。

あわせて [[wi-52-ciba-async-human-approval]] を最優先と判断した。`AgentKind.Supervised` を
宣言できるのに**人間の監督を実行時に強制する手段が一つも無く**、Supervised は現状ただの
ラベルである。依存はすべて完了済みでアンブロックされており、
[[wi-59-agent-governance-guardrails-audit-inventory]] が明示的にこれを待っている。

### 恒久的に追わないもの

| 追わないもの | 理由 |
|---|---|
| プロンプトインジェクション対策・出力フィルタ・コンテンツ安全性 | IdP の責務ではない。IdP が提供すべきは「エージェントが**何を取れるか**」の縮小 (RAR / scope / resource indicator / ReBAC) であり、LLM の入出力検査はアプリまたはゲートウェイ層の仕事。境界を跨ぐと両方が中途半端になる |
| LLM ゲートウェイ・モデルルーティング・トークン課金計測の基盤 | 同上。ただし「予算超過で認可を拒否する」というガードレールの**判断側**だけは [[wi-59-agent-governance-guardrails-audit-inventory]] のスコープとして正しい。計測基盤そのものは作らない |
| MCP プロトコル本体 (JSON-RPC・tools・server 実装) | idmagic は認可サーバーであって MCP サーバーではない。MCP 2026-07-28 自身が「エージェント ID・リクエスト単位認可・委譲の証跡・監査はプロトコル外」と明言しており、認可サーバー側に寄せる現在の設計が標準の意図と一致する |
| RFC 7592 (DCR 管理) / software statement | MCP 2026-07-28 で RFC 7591 DCR 自体が非推奨化され、CIMD がクライアント登録の優先順位で上位に格上げされた。CIMD は [[wi-341-oauth-client-id-metadata-document-cimd]] で実装済みであり、7592 に投資する理由が消えた |
| エージェント向け Verifiable Credentials ([[wi-47-verifiable-credentials-oid4vci-oid4vp]] のエージェント拡張) | NIST も OIDF もエージェント ID の候補標準に VC を挙げていない (MCP / OAuth / OIDC / SPIFFE / SCIM)。wi-47 は人間向け mDL / EUDI 文脈のまま残し、エージェント用途に広げない |
| 独自形式の「エージェントパスポート」「AI エージェントレジストリ」 | これは OIDF が挙げるギャップ ② (プロプライエタリの乱立) そのものである。標準が固まる前に独自形式を作ると後で捨てる資産になる。`act` チェーン + `agent_id` クレーム + `AgentCredentialBinding` で現時点の実需は満たせている |
| impersonation モードの Token Exchange | 設計だけあって意図的に未実装。エージェント文脈では `act` チェーンが消えて監査が壊れるため、**実装しないことが正しい判断**である。仕様に「将来 gated」と書いてある状態を維持する |
| ML / UEBA ベースのエージェント異常検知 | [[wi-150-risk-based-authentication-and-adaptive-sign-in]] が人間側で明示的にルールベースに限定した判断と揃えるべき。エージェント版だけ ML を持ち込むと、説明可能性と fail-closed の保証が壊れる |

### 今回は見送るが将来ありうるもの

| 見送るもの | 再評価の条件 |
|---|---|
| `model Agent` 集約の TypeSpec 定義 | `identity-management/models.tsp` は `model User` / `model Group` を持つが `model Agent` が無く、Request / Response と event だけがある。集約は Go と SPECIFICATION.md の散文にしか存在しない。spec-first の整合としては直すべきだが実行時挙動を変えないため今回は外す。**Agent 集約に仕様変更が入る時に併せて解消する** |
| `ManagementApiClient` の横断的認可カーネル | [[wi-320-agent-management-api-scope-wiring]]、[[wi-324-sharedsignals-agent-revocation-followups]] の項目 (4)、[[wi-274-application-admin-api-restructure-and-scopes]] の Risk Notes の**3 箇所で追跡され続けている**。横断的な設計が必要で、高価値な最小セットの範囲を超える。**次にこの穴を踏む work item が出た時点で専用の work item に切り出す** |
| Transaction Tokens for Agents | `draft-araut-oauth-transaction-tokens-for-agents-01` は Informational かつ個人 draft、A2A profile も `-00`。内部サービス連鎖でユーザーとエージェントの文脈を伝播する構想自体は正しいが、現時点で実装すると仕様変更を丸ごと被る。**IETF OAuth WG に採択されたら再評価する** |
| AuthZEN AARP / COAZ の独立追随 | AARP の実体は CIBA でやろうとしていることの抽象化、COAZ は RAR + resource indicator でやっていることの MCP プロファイルであり、独立起票は粒度が合わない。**[[wi-52-ciba-async-human-approval]] の Design に位置づけとして書き込む形で吸収した**。AARP が WG Draft から先へ進み、CIBA と異なる要求が出た時点で再評価する |
| X.509-SVID / SPIRE サーバー同梱 | [[wi-54-workload-identity-federation-spiffe]] が既に将来送りと判断済みで、その判断は今も正しい。JWT-SVID で K8s・クラウドの実需はカバーでき、TLS をエッジ終端する現在の構成では X.509-SVID の価値が薄い。**TLS 終端の構成が変わったら再評価する** |

## Plan

記録のみで実装を伴わないため、起票と同時に完了とする。導かれた作業は
wi-366 / wi-367 / wi-368 と、既存 work item への追記が持つ。

## Tasks

- [x] T001 [Research] 2026-08 時点の外部動向を一次情報の版数付きで調査する。
- [x] T002 [Research] idmagic のエージェント関連実装の現状をコードで確認する。
- [x] T003 [Decision] 採否を判断し、恒久的に追わないものと条件付き先送りを理由とともに記録する。

## Verification

- `just check-work-items`
- `just check`

## Risk Notes

リスクは low。記録のみで仕様も実装も変更しない。

この記録自体の失効には注意する。外部動向は版数付きで書いてあるため、参照した draft が
WG 採択・RFC 化・破棄のいずれかに動いた時点で「再評価の条件」欄の前提が変わる。
判断そのものではなく**判断の前提**を読み直す形で使う。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  2026-08 時点の AI エージェント向け IdP/IdM 機能を棚卸しし、idmagic の現状 (アイデンティティ層と
  トークン層は実装済み、制御・ガバナンス層はバックログに滞留) を外部動向 (OIDF の 3 ギャップ、
  NIST NCCoE 2026-02、AuthZEN AARP / COAZ、MCP 2026-07-28、Transaction Tokens draft) と
  突き合わせた。結果として wi-366 / wi-367 / wi-368 を起票し、[[wi-52-ciba-async-human-approval]] を
  最優先と判断した。恒久的に追わないもの 8 件と、条件付きで先送りするもの 5 件を、
  理由および再評価条件とともに記録した。
- **Verification Results**:
  - `just check-work-items` - passed (368 件すべて OK、依存記録 368 件も解決)。
