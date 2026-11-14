---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-15
priority: p3
group: scim-provisioning
depends_on: [wi-47962-hold-provisioning-tasks-for-disabled-connections]
change_kind: feature
affected_spec:
  - { path: docs/modules/provisioning/features/connection.md, requirement: REQ-PROVISIONING-012 }
  - { path: docs/modules/provisioning/features/connection.md, requirement: REQ-PROVISIONING-013 }
---

# 無効な接続への手動同期を予約として受理するか、開始を拒否するかを定める

## 動機

wi-47962 は、無効な接続のタスクを保持し、再有効化後に実行する仕組みを実装した。
管理者の On-Demand Provision とフル同期は、隔離中の接続を拒否する一方、無効な接続にはタスクを作る。
作ったタスクは実行されず保持されるため、これを再開後の予約として使うことは可能だが、現在の要件はその意味を明示していない。
無効中の受理を欠陥と断定せず、管理者へ何を約束する操作かを先に決める必要がある。
既存操作の採否の判断から始めるため、優先度は p3 とする。

## 対象範囲

- 無効な接続への手動同期を受理するか拒否するかを決める。
- 受理した場合のレスポンス、タスクの保持、フル同期の進捗を仕様に明記する。
- 拒否する場合は、タスク、フル同期、イベントを作らない結果を定める。
- 接続の無効と隔離が併存する場合、および対象が 0 件の場合を揃える。

## 対象外

- 無効化前からあるタスクの保持と再開の変更。
- 接続削除と、実行中の外部通信の取り消し。

## 設計

採用案は未決定である。
既存 IdP の手動同期の受付条件を公式資料で確認し、次の二案を利用者へ相談してから実装する。

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| 再有効化後の予約として受理する | 停止中に作業を準備でき、現行の保持と再開を利用できる | 同期開始の成功と下流への送信を混同しないレスポンスと進捗の説明が必要になる |
| 無効中は開始を拒否する | 無効な接続に新しい手動作業が積まれず、操作の意味が分かりやすい | 再有効化してから操作をやり直す必要があり、停止中の準備には使えない |

[Entra の Off と On](https://learn.microsoft.com/en-us/entra/identity/app-provisioning/configure-automatic-user-provisioning-portal) は一時停止の参考になるが、この資料だけでは手動同期の受付条件を確定できない。
今回必要なのはこの操作の意味の明確化であり、無効なら必ず拒否するという結論を先に置かない。

## 計画

On-Demand Provision とフル同期それぞれの入力、成功の意味、状態、イベントを比較する。
利用者と受付方針を決め、既存の振る舞いを維持する場合も要件と検証で約束を明確にする。
レスポンスやエラーを変える場合は TypeSpec と管理画面への影響を確認する。

## タスク

- [ ] T001 [Research] 既存 IdP の手動同期の受付条件を調べ、二案を比較して相談する。
- [ ] T002 [Spec] 採用した受付条件と観測結果を要件に書く。
- [ ] T003 [App] 必要な実装と、選択した結果を観測する検証を揃える。
- [ ] T004 [Verify] 無効、隔離、再有効化、対象 0 件、拒否時の無作用を検証する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/...`：二つの手動入口と下流への送信結果。
- `mise run check-spec` と `mise run verify-changed -- main`：仕様と実装の整合。

## リスク

受理を成功とだけ説明すると、無効なままで下流に届かないことを管理者が見落とす。
一律拒否へ変えると、停止中に準備する既存の使い方を失う可能性がある。
