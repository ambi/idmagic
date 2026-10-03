---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-045 }
---

# User の復元を下流へ通知し、管理者がメールアドレスを変えたら確認済みの状態を戻す

## 動機

User の状態や連絡先が変わっても、それに伴う作用の一部が起きていない。

| 対象 | 今の挙動 | 起きる問題 |
| --- | --- | --- |
| 復元 | 削除の予約は下流のプロビジョニングへ User の削除として通知するが、復元は通知しない | 下流の宛先では、復元した User が削除されたまま残る |
| 管理者によるメールアドレスの変更 | 変更前に確認済みなら、`email_verified` が `true` のまま残る | 所有を確かめていない新しいアドレスを、確認済みとして扱う |

## 対象範囲

- 管理者が User を復元したとき、下流のプロビジョニングへ通知する。
- 管理者が User のメールアドレスを別の値へ変えたとき、`email_verified` を `false` にする。
- REQ-IDMANAGEMENT-049 と REQ-IDMANAGEMENT-045 の要件文と、それを表す例を改める。

## 対象外

- 本人によるメールアドレスの変更。
  確認のリンクを経由して確認済みにするので、変えない。
- 通知の失敗の扱いの統一。
  Group と User で異なる点は、別に扱う。

## 設計

復元の通知は、削除の予約と同じプロビジョニングの通知の仕組みで送る。
通知する種類（再有効化として送るか、作成として送るか）は、下流の宛先が削除をどう扱うかに合わせて着手時に決める。

## 計画

1. 要件の差分を書く。
2. 受け入れ境界で RED を確認し、実装する。

## タスク

- [ ] T001 [Spec] REQ-IDMANAGEMENT-049 と REQ-IDMANAGEMENT-045 の差分を書く。
- [ ] T002 [Acceptance] 復元の通知と、確認済みの状態のリセットの RED を確認する。
- [ ] T003 [App] 実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run verify`

## リスク

- 下流の宛先が、削除した User の再作成を受け付けない場合がある。
  通知の種類を決める前に、Provisioning の宛先ごとの扱いを確かめる。
