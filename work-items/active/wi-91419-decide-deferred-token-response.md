---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-17
priority: p3
group: federation-protocols
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-050 }
  - { path: docs/modules/oauth2/features/guardrail.md, requirement: REQ-OAUTH2-076 }
  - { path: docs/modules/oauth2/standards.md, requirement: CIBA-CORE-POLL-MODE }
---

# Deferred Token Response の採否を判断する

## 動機

OAuth WG は [Deferred Token Response](https://hidekazu-konishi.com/entry/where_ai_agent_authorization_standards_stand.html)（-00、2026年9月）を WG 文書にした。
トークンエンドポイントが、人の承認などの完了まで応答を保留し、後で結果を返す標準の形を定める。

IdMagic は、`Supervised` な Agent の発行（REQ-OAUTH2-050）や承認へ降格したガードレール（REQ-OAUTH2-076）を、CIBA の承認リクエストとポーリングで扱う。
クライアントは CIBA の手順を知っている必要があり、通常のグラントの途中で承認が必要になった場合の標準の応答はない。

## 対象範囲

- 判断を最初に行う。Deferred Token Response を採用する案、CIBA の経路を維持する案、両方を提供する案を比べる。
- 採用した場合の、承認が必要な発行での保留の応答と、承認後の取得。

## 対象外

- CIBA の `ping` と `push`。
- 承認の通知の新しい手段。

## 設計

判断の材料は、draft の進展、MCP クライアントや Agent フレームワークの実装状況、既存の承認リクエストのモデルを再利用できるかである。
保留した要求は既存の承認リクエストとして作り、状態遷移を増やさない案を第一候補にする。
WG 文書になったばかりで形が変わる可能性が高いので、判断は draft の改訂を一度見てから行う。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を比べて採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否を確定する。
- [ ] T002 [Spec] 採用した方式の標準表と規範を書く。
- [ ] T003 [Acceptance] 承認前の取得、拒否後の取得、期限切れの RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

保留中の要求が承認前にトークンを返すと、承認を迂回できる。
REQ-OAUTH2-042 と同じく、承認が成立していない限り発行しないことを試験する。
