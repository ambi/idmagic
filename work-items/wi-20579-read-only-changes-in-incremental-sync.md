---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-25
risk_notes: |
  読み取りを差分に絞ると、基準値の扱いを誤ったときに変更を取りこぼし、下流が乖離したまま残る。特に割り当ての解除は User の更新時刻を変えないので、User だけを基準値の対象にするとスコープ外になった User を無効化できない。
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-019 }
---

# インクリメンタル同期が、前回の基準値以降に変わった User と割り当てだけを読む

## 動機

インクリメンタル同期は、`PROVISIONING_RECONCILE_INTERVAL`（デフォルト 5 分）ごとに、接続ごとにテナントの全 User、全 `RemoteResourceLink`、決着していない全プロビジョニングタスクを読む。
1 周期の読み取り量は「User 数 × 接続数」に比例し、変更が 1 件も無い周期でも同じだけ読む。
大きなテナントや接続の多いテナントでは、周期のたびに DB を全件走査することになり、現実的に運用できない。

[[wi-92540-guarantee-provisioning-deliveries-by-reconciliation]] は「読み取りの量が問題になってから導入する」として、差分の読み取りを先送りした。

Microsoft Entra ID のプロビジョニングは、初回サイクルの終わりに基準値（watermark）を保存し、以降の増分サイクルでは基準値より後に更新されたユーザーとグループだけを読む。

## 対象範囲

- 接続ごとに基準値を保存し、インクリメンタル同期が基準値より後に変わった User と割り当てだけを読むようにする。
- 割り当ての解除を拾う方法を決める。割り当ての解除では User の `updated_at` が変わらないため、割り当ての変更履歴も基準値の対象にする必要がある。
- 基準値の扱いと、フル同期との関係を `internals.md` に書く。

## 対象外

- Group のインクリメンタル同期。

## 検証

- `mise run verify`
