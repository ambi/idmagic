# WorkloadIdentity State Transitions

## WorkloadTrustBundleLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配下の関連付けを交換に使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなくレコードそのものを取り除く終端操作であり、配下の関連付けもカスケード削除する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配下の関連付けを交換に使える |
| disabled | — | 配下の関連付けを交換に使えない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |

## AgentWorkloadBindingLifecycle

作成時は `enabled` とする。無効化すると `disabled` に遷移し、それ以降の交換には使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作である。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 交換に使える |
| disabled | — | 交換に使えない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |
