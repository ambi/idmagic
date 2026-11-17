---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-17
priority: p3
group: agent-governance
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-074 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-055 }
  - { path: docs/modules/oauth2/standards.md, requirement: AIMS00-AGENT-CLIENT-AUTH }
---

# 採否を判断したうえで、一つのクライアントの下の Agent インスタンスを区別する

## 動機

IdMagic は Agent と OAuth クライアントを一対一に束縛する（REQ-IDMANAGEMENT-074）。
一つのプラットフォームが一つの `client_id` の下で多数の Agent を動かす場合、資源側と監査は個々のインスタンスを区別できない。

個人提案の [AI agent instance profile](https://datatracker.ietf.org/doc/html/draft-mcguinness-oauth-ai-agent-instance-00) は、Client Instance Assertion か Attestation-Based Client Authentication で `agent_instance_id` を示し、トークンの `act.sub` か `sub` へ載せる方式を提案している。
Entra Agent ID も blueprint の単一インスタンスと複数インスタンスの方式を preview にした。

## 対象範囲

- 判断を最初に行う。現在の一対一の束縛を維持する案、インスタンスを Agent として自動登録する案、インスタンスの識別子をトークンと監査だけに載せる案を比べる。
- 採用した案の最小の実装と、停止と失効の判定への接続。

## 対象外

- ハードウェアに根差したアテステーションの検証基盤。
- 個人提案の全体への適合の表示。

## 設計

判断の材料は、Client Attestation の WG 文書と個人提案の進展、多数のインスタンスを動かす利用者の要望、停止の保証との整合である。
インスタンスの識別子を自己申告として受けるなら、トークンへ載せても認可の根拠にはしない。
停止の単位を Agent からインスタンスへ細かくするかどうかを、判断の中心の問いにする。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を比べて採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と方式を確定する。
- [ ] T002 [Spec] 採用した方式の規範を書く。
- [ ] T003 [Acceptance] 偽の識別子、別インスタンスの鍵、停止したインスタンスの RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

自己申告の識別子を認可に使うと、インスタンスのなりすましで別の権限を得られる。
識別と認可を混同しない仕様にする。
