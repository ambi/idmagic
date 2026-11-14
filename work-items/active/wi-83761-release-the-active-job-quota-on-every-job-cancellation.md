---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-036 }
  - { path: docs/modules/jobs/features/admin.md, requirement: REQ-JOBS-013 }
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-041 }
---

# どの経路で Job を取り消しても、`active_jobs` の使用量を減らす

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、Job を終端へ移す経路のうち、`active_jobs` を減らすものが一部だけであることが分かった。

| 経路 | 呼ぶ操作 | 使用量の減算 |
| --- | --- | --- |
| ランナーの完了と失敗 | `Runner` が `Repo.Complete` と `Repo.Fail` の後に `decrementActiveJobsQuota` | あり |
| 監査エクスポートの取り消し | `CancelExport` が `Jobs.Cancel` の後に自分で減らす | あり |
| 管理者による取り消し（REQ-JOBS-013、015） | `CancelJobForAdmin` が `Repo.Cancel` | なし |
| データエクスポートの取り消し（REQ-IDMANAGEMENT-041） | `CancelDataExport` が `JobRepo.Cancel` | なし |

実行中の Job を取り消すと、ハンドラーの完了の報告はリースを失って失敗するので、ランナーの減算も走らない。
[wi-23517](../done/wi-23517-release-the-consent-quota-when-the-user-revokes-a-consent.md) と同じく、同じ終端への遷移を複数の経路が別々に書き、一部だけが減算を足していた。

## 対象範囲

- 管理者の取り消しとデータエクスポートの取り消しで、`active_jobs` を一つ減らす。
- 終端への遷移と使用量の減算を、Jobs の一つの操作が担う形へ移す。

## 対象外

- 既存のテナントでずれた使用量の補正。
- 投入の失敗で取り消す `Enqueue` の補償。加算に失敗した Job なので減算しない。

## 設計

### 所有の形

候補 A を採る。
Jobs のユースケースに終端へ移す操作（完了、失敗、取り消し）を置き、その中で使用量を減らす。
`JobRepository.Cancel` を IdManagement と Audit から呼ばず、Jobs が公開する取り消しの操作を通す。
`CancelExport` の個別の減算を消す。
二重の減算を避けるため、既に終端の Job の取り消しは減算しない（`Cancel` が終端の Job を拒否する現在の振る舞いを保つ）。

## タスク

- [ ] T001 [Test] 管理者とデータエクスポートの取り消しの後に使用量が減らないことを RED にする。
- [ ] T002 [App] 終端への遷移の操作を Jobs に集め、各経路をそこへ通す。

## 検証

- `mise run test-go-package -- ./backend/jobs/...`
- `mise run test-go-package -- ./backend/idmanagement/internal/usecases`
- `mise run test-go-package -- ./backend/audit/internal/usecases`

## リスク

- Jobs の公開操作を増やすので、IdManagement と Audit から Jobs への依存の形を[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md)で確かめる。
