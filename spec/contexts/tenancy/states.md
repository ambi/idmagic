# Tenancy State Transitions

## TenantLifecycle

テナントは `Active` で通常稼働し、`Disable` で全プロトコルルートを停止する。`Enable` で復帰できる。物理削除は対象外とする。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。全プロトコルルートが応答する |
| Disabled | — | 全プロトコルルートを停止している。`Enable` で復帰できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | TenantDisabled | — | Disabled |  |
| Disabled | TenantEnabled | — | Active |  |
