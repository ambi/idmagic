---
id: idp-wi-49-agent-identity-first-class-principal
title: "AI エージェントを第一級の非人間プリンシパルとして導入する"
created_at: 2026-06-22
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
AI エージェント (LLM ベースの自律・半自律ソフトウェア) が企業内で API を呼び、
データを取得し、ユーザーに代わって行動する利用が広がっている。現代の IdP は
エージェントを「人間でもなければ従来のサービスアカウントでもない」第一級の
プリンシパル種別として扱い始めた。Microsoft は Entra Agent ID (2026-04 GA) で
エージェントにディレクトリ上の ID を自動付与し、Okta / Auth0・Google・Ping も
非人間 ID (Non-Human Identity, NHI) としてのエージェント管理を提供する。

idmagic は現状 User と OAuth2Client (machine 含む) しか持たず、エージェント固有の
「所有者 (人間 / 組織) との結びつき」「目的・用途の宣言」「ライフサイクルと即時
停止 (kill switch)」「人間の関与レベル (自律 / 監督下)」を一級概念として表現できない。
本 WI は後続の委譲 ([[wi-50-token-exchange-delegation-actor-chain]])・代行・ガバナンス
([[wi-59-agent-governance-guardrails-audit-inventory]]) の土台として、Agent 集約を
導入する。エージェントの資格情報そのものは既存の OAuth2Client (client_credentials)
を再利用し、Agent はその上位に立つ統制・所有・来歴のための集約とする。

# Scope
- **decision**: 新規 ADR [[ADR-048]]: Agent を User / OAuth2Client と別の第一級プリンシパルとする根拠、 所有者モデル (human owner / owning group / 組織)、エージェントと OAuth2Client の関係 (Agent は 1 つ以上の client 資格情報を所有)、status / kill-switch の セマンティクス、tenant 帰属を確定する。
- **scl**: 新規 model: Agent / AgentStatus / AgentOwner / AgentKind (autonomous / supervised) / AgentCredentialBinding。AccessTokenClaims に発行先が Agent で あることを示す principal type を追加する。, 新規 event: AgentRegistered / AgentUpdated / AgentDisabled / AgentEnabled / AgentDeleted / AgentOwnerChanged。, 新規 interface: admin の Agent CRUD と一覧 (registry)、kill-switch 操作。 permission `AdminAgentsManage`。
- **go**: Agent 集約と Postgres adapter (agents テーブル、owner / client への外部キー、tenant scope)。, token 発行経路で principal が Agent の場合に Agent の status を fail-closed で 確認し、disabled / killed のエージェントには発行しない。
- **http**: /admin/agents の CRUD と kill-switch エンドポイント。
- **ui**: [object Object]

# Out of Scope
- 委譲・代行トークン (Token Exchange) 自体の実装 ([[wi-50-token-exchange-delegation-actor-chain]] で扱う)。
- workload attestation によるエージェント起動時ブートストラップ ([[wi-54-workload-identity-federation-spiffe]])。
- エージェントの予算・レート上限などガードレール ([[wi-59-agent-governance-guardrails-audit-inventory]])。
- エージェント検証可能クレデンシャル / agent passport (将来 WI、[[wi-47-verifiable-credentials-oid4vci-oid4vp]] の延長)。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: エージェントを登録 → トークン発行 → kill-switch → 同じエージェントのトークン発行が拒否されることを確認する。

# Risk Notes
Agent をプリンシパルとして導入するとトークンの主体種別が増え、認可判断 (AuthZEN)
と監査の前提が広がる。エージェントの資格情報は新設せず既存 OAuth2Client に集約し、
Agent は所有・統制・来歴の層に限定することで攻撃面の二重化を避ける。status / kill
の確認は発行経路で fail-closed とし、判定漏れがあっても「発行しない」側に倒す。

# Completion
- **Completed At**: 2026-06-22
- **Summary**:
  ADR-048 に基づき、Agent を User / OAuth2Client と別の第一級プリンシパルとして
  spec-first で end-to-end 実装した。SCL に Agent 系 model / enum (AgentStatus =
  Active/Disabled/Killed, AgentKind)、9 イベント、AgentLifecycle 状態機械、10 admin
  interface、AdminAgentsManage 権限、vocabulary を追加し、Authentication component の
  owns_* と Go の policy action / interface マッピングを更新した。Go は domain 型・
  検証・repository port・Postgres / in-memory adapter・migration 0013・admin CRUD
  use case / HTTP ハンドラ (register / update / disable・enable / kill / delete /
  credential bind・unbind) を実装。トークン発行経路 (client_credentials) に、束縛
  client から Agent を解決し !IsActive() なら invalid_client で拒否する fail-closed
  ゲートを入れ、束縛時は agent_id / principal_type=agent クレームを付与する。管理 UI
  (一覧・登録・所有者表示・無効化 / kill / 削除・資格情報束縛) は groups を踏襲して追加した。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - manual: register → token 発行 → kill → 同一エージェントの token 発行拒否 の live e2e は本環境 (DB 未起動) では未実行。
