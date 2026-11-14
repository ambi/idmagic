---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p3
group: agent-governance
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-094 }
  - { path: docs/modules/authorization/features/check.md, requirement: REQ-AUTHORIZATION-004 }
---

# 採否を判断したうえで Agent の権限を申請と承認と期限で管理する

## 動機

本人の代行の付与には期限があるが、Agent 自身の資源権限を責任者が申請し、承認を経て期限で失効させる流れはない。
[Entra の access package](https://learn.microsoft.com/en-us/entra/id-governance/agent-id-governance-overview) はこの流れを提供し、期限前に責任者へ通知する。
人向けの [[wi-214-self-service-access-request-and-approval]] と重なる範囲があるので、統合か分離かを先に決める。

## 対象範囲

- 判断を最初に行う。実際に期限を持たせる対象（関係タプル、スコープの割り当て）と、wi-214 との関係を決める。
- 責任者の申請、承認、期限切れの自動失効、期限前の通知。
- 失効が次の評価から効くこと。

## 対象外

- Agent 自身が権限を申請する経路。プロンプトの影響を受ける主体に申請を任せる危険があり、最初は採らない。
- Entra の blueprint からの権限の継承。

## 設計

T001 で次を比べる。

| 案 | 利点 | 懸念 |
| --- | --- | --- |
| wi-214 に主体として Agent を加える（第一候補） | 申請と承認の仕組みを一つにできる | wi-214 の完成に依存する |
| Agent 専用の小さな申請 | 先に提供できる | 申請の仕組みが二つになる |
| 関係タプルの期限だけを追加する | 小さい | 承認の流れと責任者への通知が残る |

2026-11-13 に現状を確かめた。

| 対象 | 期限の仕組み |
| --- | --- |
| 関係タプル（Agent への直接の付与） | ない |
| 操作割当（REQ-AUTHORIZATION-017、REQ-OAUTH2-083） | ない。取り消しは次の評価から効く |
| 本人から Agent への代行の付与 | 同意の行として、付与から 365 日の固定の `expires_at` を持つ。申請や延長の流れはない |
| クライアントのスコープ | ない |

したがって期限を持たせる対象は、操作割当と関係タプルが中心になる。
ReBAC の評価を単調に保つため、期限は読み取り時の除外か、期限切れの削除で実現し、どちらかを判断する。
[[wi-152-just-in-time-privileged-role-activation]] も読み取り時の期限の評価を計画しているので、期限の表現を揃えられるかを比べる。

第一候補の wi-214 は `priority: p3` で、この項目（p2）より低い。
第一候補を採るなら、wi-214 の優先順位を上げるか、この項目の優先順位を下げるかを T001 で合わせて決める。
期限と申請の表示は、[[wi-40606-show-effective-agent-access]] の一覧に置けるかを確かめる。

## 計画

1. 現状を確かめて案を決める。依存が生じたら `depends_on` へ追加する。
2. 仕様を更新し、失効までの試験を先に書く。

## タスク

- [ ] T001 [Decision] 対象、wi-214 との関係、期限の実現方法を確定する。
- [ ] T002 [Spec] 申請、承認、失効の規範を追加する。
- [ ] T003 [Acceptance] 承認、期限切れ、延長、拒否の RED を評価の入口で観測する。
- [ ] T004 [App] 実装する。
- [ ] T005 [UI] 責任者と承認者の画面を加える。
- [ ] T006 [Verify] 自己承認の拒否、テナント境界、並行の更新を検証する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`：期限切れの後の不許可を確かめる。
- `mise run check-boundaries`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

期限切れの処理が遅れると権限が残る。
読み取り時に期限を判定する案を優先して比べる。
