---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-094 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-064 }
---

# 誰の代理でどの Agent が何を実行できるかを一覧で確認する

## 動機

Agent の権限は、本人の付与、クライアントのスコープ、資源の規則、ReBAC の関係に分かれている。
現在は管理者がそれぞれの画面を見比べないと、実効の権限を説明できない。
レビューと障害対応の前提になるので、一つの Agent について一覧で確認できるようにする。

起票後、Agent の詳細に「リソース」の画面（`AdminAgentResourcesPage`）が加わり、接続先ごとの操作割当と対象の範囲（REQ-AUTHORIZATION-017、REQ-IDMANAGEMENT-105）を表示と編集できるようになった（2026-11-13 に確認）。
残る不足は、次の要素を同じ Agent について並べて見られないことである。

| 要素 | 所有者 | 現在見られる場所 |
| --- | --- | --- |
| 操作割当と対象の範囲 | AccessControl | Agent の「リソース」画面 |
| 代行を付与した利用者と期限 | OAuth2 の同意 | テナント全体の同意の画面（`AdminConsentsPage` の代行の一覧） |
| 束縛したクライアントのスコープ | OAuth2 のクライアント | クライアントの画面 |
| 規則の条件、承認の要否、上限 | OAuth2 の登録 | MCP リソースサーバーの編集画面 |
| ガードレール | OAuth2 | Agent の詳細のガードレールの画面（`AdminAgentGuardrailEditPage`） |
| 最終の発行と棚卸しの所見 | IdManagement | 棚卸し（REQ-IDMANAGEMENT-094） |

## 対象範囲

- Agent を起点に、代行を付与した利用者、資源、スコープ、規則の操作、最終の発行を一覧する読み取り。
- 管理コンソールの Agent 詳細への表示。
- 表示が保証する範囲（評価の結果そのものではないこと）の明示。

## 対象外

- 権限の変更操作。
- 全 Agent と全資源の直積の事前計算。

## 設計

OAuth2、IdManagement、AccessControl を読むので、組み立ての位置を境界の判断手順で決める。
第一候補は読み取りの合成を組み立て地点に置き、新しい表を作らない方式である。
特定の利用者と資源の可否は、既存の判定の説明を使い、一覧で推測しない。
既存の「リソース」画面を拡張するか、別の要約の画面を置くかを T001 で決める。

この一覧は、[[wi-213-access-certification-campaigns]] で Agent の所有者が再確認するときの判断材料と、[[wi-60346-request-time-bound-agent-access]] の期限の表示の置き場所になり得る。
両項目の着手時に、この一覧の単位を再利用できるかを確かめる。

## 計画

1. 表示する単位と所有者を決める。
2. API と画面を実装する。

## タスク

- [ ] T001 [Design] 境界の判断の記録を書き、表示の単位を確定する。
- [ ] T002 [Spec] 読み取りの仕様と TypeSpec を追加する。
- [ ] T003 [Acceptance] 付与と規則の変更が一覧へ反映され、他テナントが現れないことの RED を作る。
- [ ] T004 [App] API を実装する。
- [ ] T005 [UI] 管理コンソールへ表示する。
- [ ] T006 [Verify] 件数の多いテナントでの応答を確かめる。

## 検証

- `mise run check-spec`、`mise run check-boundaries`。
- 画面の変更には `mise run test-ui-e2e-file` を使う。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

一覧を実効の許可と誤解すると、判断を誤る。
条件と関係の判定を含まない項目を画面で区別する。
