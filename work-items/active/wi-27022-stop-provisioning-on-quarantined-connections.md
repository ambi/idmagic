---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-25
risk_notes: |
  隔離しても送信が止まらないと、障害中の下流へ再試行が積み増され、隔離より前に作られた無効化と削除も届く。止め方を誤ると、解除後に止めたタスクが失われ、下流が乖離したまま残る。
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-011 }
---

# 隔離した接続では、作成済みのプロビジョニングタスクを下流へ送らず、フル同期も始めない

## 動機

用語集は Quarantine を「管理者が `ResumeProvisioningConnection` で解除するまで再開しない」状態と定め、TypeSpec の `ConnectionQuarantined` は「No further task is created.」と説明する。

実装では、隔離を見ているのはイベント同期（`capture.go`）とインクリメンタル同期（`ReconcileConnections`）だけである。

| 経路 | 隔離した接続での動作 |
| --- | --- |
| ディスパッチャー（`DispatchPendingTasks`） | `pending` のプロビジョニングタスクにジョブを関連付け、下流へ送らせる |
| ジョブハンドラー（`ExecuteTask`） | 関連付け済みのタスクをそのまま下流へ送る |
| `StartFullResync` | 対象ごとに `update` のタスクを作る |
| `ProvisionOnDemand` | タスクを作る |

隔離より前に作られていたタスクは、隔離の後も送られ続ける。
隔離は、新しく作るものを減らすだけで、下流への送信を止めていない。

[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] の readiness pass で見つかった。
誤削除ガードは作る前に止めるので、この欠陥の影響を受けない。一方、連続失敗による隔離は下流の障害を理由に止めるものであり、送り続けると障害中の下流へ再試行を積み増す。

## 対象範囲

- ディスパッチャーとジョブハンドラーが、隔離した接続のタスクを下流へ送らないようにする。送らなかったタスクを解除後に再開する方法を決める。
- 隔離した接続では、`StartFullResync` と `ProvisionOnDemand` の扱いを決める。拒否するなら公開契約へ拒否を宣言する。

## 対象外

- 誤削除ガードそのもの。[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] が扱う。

## 検証

- `mise run verify`
