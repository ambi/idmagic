---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-24
risk_notes: |
  ガードが効かないと、誤った設定や取り込みによる一括の無効化と削除が下流へそのまま届き、外部システムのアカウントを失わせる。逆にガードが誤って効くと、正当な deprovision が止まり、退職者のアカウントが下流で有効なまま残る。
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
---

# full resync が誤削除ガードの閾値を超えたら、deprovision を実行せずに接続を隔離する

## 動機

`REQ-PROVISIONING-011` は、1 回の full resync で deactivate または delete の対象が `accidental_deletion_count_threshold` を超えたら、deprovision を実行せず `ConnectionQuarantined` を発行して接続を隔離し、`notification_email` へ通知すると宣言する。

`DeprovisionPolicy` は `accidental_deletion_count_threshold` と `accidental_deletion_percent_threshold` を保存し検証するが、それを読む処理は無い。
`StartFullResync` は対象範囲の User と Group へ `update` のプロビジョニングタスクを作るだけで、deprovision の件数を数えない。
そのため、閾値を設定しても誤った一括の無効化や削除は止まらず、下流のアカウントがそのまま無効化または削除され得る。

隔離の状態遷移と `ConnectionQuarantined` の発行は、連続失敗の経路で [[wi-85060-publish-provisioning-lifecycle-events]] が実装した。

## 対象範囲

- full resync が作る deactivate と delete の件数と割合を、閾値と比べる。
- 閾値を超えたら、その resync の deprovision を下流へ送らず、接続を隔離して `ConnectionQuarantined` を発行する。
- 隔離を `notification_email` へ通知する。

## 対象外

- 隔離の解除（`ResumeProvisioningConnection`）。既に実装されている。
- 照合によるプロビジョニングタスクの保証。[[wi-92540-guarantee-provisioning-deliveries-by-reconciliation]] が扱う。照合が大量の deprovision を作るときにこのガードを効かせるかは、その記録の未解決の問いである。

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run verify`

## リスク

閾値の判定を件数だけで行うと、小さなテナントでは割合の閾値が効かず、大きなテナントでは件数の閾値が正当な一括処理を止める。両方の閾値の組み合わせを具体例で固定する。
