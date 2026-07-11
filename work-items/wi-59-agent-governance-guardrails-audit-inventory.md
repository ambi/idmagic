---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-22
---

# エージェントガバナンス (ガードレール・委譲チェーン監査・インベントリ)

## Motivation
エージェントを第一級プリンシパルとして運用するには、ID・委譲・認可だけでなく
「過剰行動を抑えるガードレール」「誰がどの権限で何をしたかの監査」「全エージェントの
棚卸し (インベントリ)」という統制層が要る。Microsoft Entra Agent ID / Agent 365、
Okta・Ping のエージェントガバナンスはいずれも、レート・予算上限、行動監査、
ライフサイクル可視化を中核に据えている。

idmagic は監査イベント基盤 ([[wi-44-authentication-event-store-and-search]] 等) と
rate limit ([[wi-27-endpoint-rate-limit-and-bot-mitigation]]) を持つが、エージェント
単位の予算・レート・行動上限や、委譲チェーン (act) を含む横断監査、エージェント
インベントリ画面を持たない。本 WI は [[wi-49-agent-identity-first-class-principal]] を
起点に、エージェント単位のガードレール、actor チェーン込みの監査・相関検索、
インベントリ / 統制ダッシュボードを束ねる。これは導入した一連のエージェント機能に
運用統制を被せる総仕上げにあたる。

## Scope
- **decision**:
  - 新規 ADR [[ADR-058]]: エージェント単位のガードレール種別 (レート / 予算・コスト / 行動回数 / 有効期限 / 許可リソース)、上限超過時の挙動 (拒否 / 要承認へ降格)、actor チェーンを 含む監査イベントの相関キー、インベントリの表示観点 (所有者・最終活動・付与権限) を確定する。
- **scl**:
  - 新規 model: AgentGuardrail / GuardrailKind / AgentActivitySummary / AgentAuditQuery。監査イベントに actor チェーン (act) と委譲深さを載せる。
  - 新規 event: GuardrailConfigured / GuardrailBreached / AgentActionAudited。
  - 新規 interface: ガードレール CRUD、エージェント別アクティビティ / 監査検索、 インベントリ一覧。permission AdminAgentGovernanceManage。
- **go**:
  - ガードレール評価 (トークン発行 / 行動経路でレート・予算・回数・有効期限を fail-closed に 強制)、actor チェーンを含む監査イベントの発火と相関検索 adapter を実装する。
- **http**:
  - ガードレール管理 API、エージェント監査検索 API、インベントリ API。
- **ui**:
  - admin: エージェントインベントリ / 統制ダッシュボード、ガードレール設定、委譲チェーン込み監査ビュー。

## Out of Scope
- 課金・コスト計測基盤そのものの構築 (予算上限の評価フックの提供まで)。
- 異常検知エンジン ([[wi-58-continuous-access-evaluation-agent-revocation]] のシグナル源)。
- SIEM への外部エクスポート (まず内部監査・検索)。

## Verification
- `just test-go`
  - reason: レート / 予算 / 回数 / 有効期限の上限強制、超過時挙動、actor チェーン監査の相関、tenant scope の境界。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just build-ui`
- 手動: エージェントにレート / 予算上限を設定 → 上限内は許可・超過は拒否 → 委譲チェーン込みで監査に残ることを確認する。

## Risk Notes
ガードレール評価を発行 / 行動経路に挟むため、判定の重さやすり抜けが課題になる。
上限判定は fail-closed (迷ったら拒否) とし、actor チェーンの監査は既存イベント基盤に
載せて二重化を避ける。予算・コストは外部計測のフックに留め、計測基盤は別途とする。
