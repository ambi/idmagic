---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/jobs/features/queue.md, requirement: REQ-JOBS-002, impact: conforms }
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-013, impact: conforms }
  - { path: docs/modules/identity-management/features/group-csv.md, requirement: REQ-IDMANAGEMENT-026, impact: conforms }
---

# CSV の再評価ジョブを原子性と Jobs の規則を保って投入する

## 動機

GroupImportRowCommitter は再評価ジョブを SQL で直接投入し、Jobs の投入操作が必須とする active_jobs の確認と JobEnqueued の発行を通らない。
wi-88544 の配置の変更だけでこの経路を公開すると、規範に反する投入経路を固定してしまう。

## 対象範囲

- Group の行と再評価ジョブを同じトランザクションで確定したまま、Jobs がレーン、既定値、重複排除、クォータを担う操作を公開する。
- 新しいジョブが確定した後だけ JobEnqueued を発行し、重複した投入とロールバックでは発行しない。
- クォータ拒否時に Group、監査、使用量、ジョブを一部も残さないことを検証する。

## 対象外

- 直接の SQL と同じ結果だけを返す、クォータとイベントを省く公開操作の追加。
- ジョブ以外の CSV の履歴と監査の所有者の変更。

## 設計

2026-10-18 の負債の整理では、この既存挙動を保つ refactor は見送る。
通常の Enqueuer を別トランザクションで呼ぶ案も、行が確定する前にイベントを発行する案も原子性を保たない。
着手時には Jobs と Tenancy の操作を同じ Tx へ結び、確定後のイベントだけを呼び出し側へ渡す案を比較する。
一般の投入経路も、重複ヒットでクォータを増やさず、確定しなかったジョブを発行しないことを維持する。

## タスク

- [ ] T001 [Design] 原子性、クォータ、確定後の発行を担う操作を決める。
- [ ] T002 [Acceptance] クォータ拒否と重複投入の不整合をテストで固定する。
- [ ] T003 [App] Jobs の公開操作を通す。
- [ ] T004 [Verify] 通常の投入と CSV の行の成功、拒否、ロールバックを検証する。

## 検証

- Jobs、Tenancy、IdManagement の PostgreSQL テストと、確定前のイベント発行の故障注入。
- `mise run check-boundaries` と `mise run test-go-race`。

## リスク

クォータの更新だけを別に確定すると、失敗した行が active_jobs を消費し続ける。
発行を遅らせる設計と、保存とイベントの間の停止は区別して扱う。
