---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent-observation.md, requirement: REQ-IDMANAGEMENT-095 }
  - { path: docs/modules/identity-management/features/agent-observation.md, requirement: REQ-IDMANAGEMENT-101 }
---

# Agent の検出信号と Web Bot Auth の採否を判断して接続する

## 動機

[Ping](https://press.pingidentity.com/2026-03-24-Ping-Identity-Defines-the-Runtime-Identity-Standard-for-Autonomous-AI) は bot 認証と行動の信号を認可判断へ渡し、Okta と Entra も未登録 Agent の発見を提供する。
IdMagic は外部の観測の突合だけを持ち、ブラウザーを操作する Agent を実行時に見分けられない。
一方、[Web Bot Auth](https://developers.google.com/crawling/docs/crawlers-fetchers/web-bot-auth) は実験的な draft で、指紋や行動の分析を自製する価値も確定していない。

## 対象範囲

- 判断を最初に行う。Web Bot Auth の署名検証、外部の検出器からの信号の受け取り、見送りの三つを比べる。
- 採用する場合、信号をサインインのポリシーや評価の入力へ渡す最小の接続。

## 対象外

- 端末やブラウザーの発見器、指紋と行動の分析の自製。
- 信号による権限の付与。信号は制限にだけ使い、REQ-IDMANAGEMENT-101 の判断を維持する。

## 設計

判断の材料は、draft の状況、主要な Agent の署名の実装状況、顧客の検出製品からの要望である。
接続先は [[wi-150-risk-based-authentication-and-adaptive-sign-in]] のリスク入力を第一候補にする。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を調査して採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と接続先を確定する。
- [ ] T002 [Spec] 採用した信号の規範を書く。
- [ ] T003 [Acceptance] 署名の偽造、未知の鍵、信号の欠落の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

署名した Agent の正体は操作の許可を意味しない。
識別と認可を混同しない仕様にする。
