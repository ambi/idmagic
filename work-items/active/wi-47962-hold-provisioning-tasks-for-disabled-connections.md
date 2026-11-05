---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/provisioning/features/task.md, requirement: REQ-PROVISIONING-020 }
---

# 無効にした接続へ、無効化の前に積んだタスクを送らない

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、接続の `status=disabled` を読む経路が、タスクの取り込みだけであることが分かった。

| 経路 | 隔離（`Health=quarantined`） | 無効（`Status=disabled`） |
| --- | --- | --- |
| 取り込み（`capture.go`）、再同期（`reconcile.go`） | 止める | 止める |
| ディスパッチャー（`dispatcher.go`） | 止める | 止めない |
| 送信（`ExecuteTask`） | 止める | 止めない |

無効化の前に積んだ `pending` のタスクと、実行待ちのジョブは、無効化の後も下流へ送られる。
隔離では [wi-27022](../done/wi-27022-stop-provisioning-on-quarantined-connections.md) が送信点の確認を入れ、[REQ-PROVISIONING-020](../../docs/modules/provisioning/features/task.md) が保持と再開を定めた。
無効化では、積んだタスクの扱いを規範が定めていない。
規範の取りこぼしである。

## 対象範囲

- 無効の接続のタスクの扱い（保持して再有効化の後に再開するか、取り消すか）を規範に書く。
- ディスパッチャーと送信点で、無効の接続へ送らない。

## 対象外

- 接続の削除。

## 設計

### 所有の形

候補 A を採る。
接続のドメインに、送ってよいかを返す一つの判定（隔離と無効の両方を読む）を置き、取り込み、再同期、ディスパッチャー、送信点がそれを使う。
送信点の `ExecuteTask` は、すべての下流への送信が通る唯一の箇所であり、ここでの判定が約束を担う。
ほかの経路の判定は、不要なジョブを作らないための前段に留める。

保持か取り消しかは、REQ-PROVISIONING-020 の状態遷移表に `disabled` の列を足して決める。

## タスク

- [ ] T001 [Spec] 無効の接続のタスクの遷移を規範に書く。
- [ ] T002 [Test] 無効化の後にディスパッチャーがタスクを送ることを RED にする。
- [ ] T003 [App] 接続の判定を一つにし、各経路で使う。

## 検証

- `mise run test-go-package -- ./backend/provisioning/...`
- `mise run check-spec`

## リスク

- 取り消しを選ぶと、再有効化の後に下流との差が残る。保持を第一に比べる。
