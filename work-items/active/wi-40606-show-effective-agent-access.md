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
