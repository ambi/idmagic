---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.GroupMembershipPushed }
---

# 発行されない GroupMembershipPushed のイベントと遷移を消す

## 動機

`tools/check/event-emission-debt.json` は、宣言はあるが本番のコードのどこも値を作らないドメインイベントを列挙する。
4 件のうち、`FederationLinked` と `FederationUnlinked` は wi-45298、`SigningKeyRetired` は wi-96851 が扱う。
残る `GroupMembershipPushed` を扱う work item はない。

Group のメンバーシップは、Group のプロビジョニングタスクの中で、Group のリソースを送った後に `members` への増分の `add` として送っている（`backend/provisioning/usecases/execute_task.go` の `pushGroupMembers`）。
そのタスクは成功すると `GroupPushed` を発行する。
メンバーシップだけを送るタスクは存在しないので、`GroupMembershipPushed` を発行する経路はない。

それでも、タスクの状態遷移の表（`docs/modules/provisioning/task/README.md`）と `backend/provisioning/domain/task.go` には `in_flight` から `GroupMembershipPushed` で `succeeded` へ進む遷移があり、TypeSpec もイベントを宣言している。
監査の記録を読む側は、起きないイベントに備えることになる。

## 対象範囲

- `GroupMembershipPushed` を、TypeSpec、ドメインイベントの型、タスクの状態遷移の表とコード、Provisioning の README のイベント一覧から消す。
- `tools/check/event-emission-debt.json` から `GroupMembershipPushed` を消す。

## 対象外

- メンバーシップの除去の配信。
  `RFC7643-OUT-GROUP-RESOURCES` が送らない理由を定めている。

## 設計

採らない案は、メンバーシップの送信を別のタスクに分け、その成功で `GroupMembershipPushed` を発行することである。
メンバーシップの送信は Group のリソースの送信で得た下流の識別子を使うので、別のタスクにすると、Group の送信の完了を待つ順序の制御が新たに要る。
いまの一つのタスクで送る形は、`GroupPushed` が Group とメンバーシップの両方の送信の完了を表しており、監査の記録の利用者がメンバーシップの送信だけを区別する要件もない。

## タスク

- [ ] T001 [Contract] TypeSpec からイベントを消す。
- [ ] T002 [Spec] タスクの状態遷移の表と README のイベント一覧から消す。
- [ ] T003 [App] ドメインイベントの型と遷移を消す。
- [ ] T004 [Tooling] `tools/check/event-emission-debt.json` から消す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-event-contract`
- `mise run check-api-compat`
- `mise run verify`

## リスク

- イベントの宣言を消すと、OpenAPI の互換性の検査が後退として報告する。
  未リリースなので、基準を更新して受け入れる。
